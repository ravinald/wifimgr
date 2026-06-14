// Package aruba implements the vendors.Client interface for standalone Aruba
// Instant (IAP) deployments via the device-local REST API on the Virtual
// Controller (https://<iap-ip>:4343).
//
// The IAP API splits into three families: a Monitoring API (/rest/show-cmd,
// which returns CLI `show` text inside a JSON envelope), a Configuration API
// (/rest/<object>, structured JSON with action: create|delete), and an Action
// API (/rest/<verb>, per-AP operations). Reads therefore require parsing CLI
// output; writes are structured. The REST API must be enabled on the device
// first with `allow-rest-api` + `commit apply` — it is off by default and
// cannot be turned on remotely.
package aruba

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/vendors"
)

// defaultMinInterval spaces requests to the IAP. The on-box web server keeps a
// small session table and serves REST only from the master, so wifimgr drives
// it sequentially rather than concurrently.
const defaultMinInterval = 150 * time.Millisecond

// defaultMemoTTL backstops the show-command memo. It comfortably covers a single
// refresh (seconds) while bounding how stale a reused client's reads can get.
const defaultMemoTTL = 60 * time.Second

// Client is the HTTP client for a single Instant AP Virtual Controller.
type Client struct {
	baseURL  string // https://<vc-ip>:4343
	host     string // <vc-ip>, used as iap_ip_addr in monitoring calls
	user     string
	passwd   string
	apiLabel string

	httpClient *http.Client

	mu          sync.Mutex
	sid         string
	lastReq     time.Time
	minInterval time.Duration

	// showMemo deduplicates identical `show` reads within a single operation: a
	// refresh fans the same handful of commands across every service, and the
	// device state is constant for that burst. Any write (PostObject) clears it,
	// so apply's read-after-write still sees fresh state. memoTTL bounds staleness
	// for a long-lived client; each CLI command otherwise runs in its own process.
	showMemo map[string]memoEntry
	memoTTL  time.Duration
}

type memoEntry struct {
	out string
	at  time.Time
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithInsecureSkipVerify disables TLS certificate verification. Instant APs
// ship a self-signed VC certificate; operators opt into skipping verification
// per API rather than wifimgr defaulting to it.
func WithInsecureSkipVerify(skip bool) ClientOption {
	return func(c *Client) {
		if tr, ok := c.httpClient.Transport.(*http.Transport); ok {
			tr.TLSClientConfig.InsecureSkipVerify = skip
		}
	}
}

// WithHTTPClient sets a custom HTTP client (used by tests with httptest).
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = hc }
}

// WithAPILabel records the registry label for error attribution.
func WithAPILabel(label string) ClientOption {
	return func(c *Client) { c.apiLabel = label }
}

// NewClient creates a client for the IAP at baseURL. baseURL must include the
// scheme and the VC IP (e.g. https://10.0.0.1:4343).
func NewClient(user, passwd, baseURL string, opts ...ClientOption) *Client {
	host := hostFromBaseURL(baseURL)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}

	c := &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		host:        host,
		user:        user,
		passwd:      passwd,
		apiLabel:    "aruba",
		httpClient:  &http.Client{Transport: transport, Timeout: 30 * time.Second},
		minInterval: defaultMinInterval,
		showMemo:    make(map[string]memoEntry),
		memoTTL:     defaultMemoTTL,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// apiEnvelope is the JSON wrapper every IAP REST response carries. Field names
// match the device's keys verbatim, including the embedded spaces.
type apiEnvelope struct {
	Status        string `json:"Status"`
	StatusCode    *int   `json:"Status-code"`
	SID           string `json:"sid"`
	Message       string `json:"message"`
	ErrorMessage  string `json:"Errormessage"`
	CommandOutput string `json:"Command output"`
	CLICommand    string `json:"CLI Command executed"`
	IAPIPAddress  string `json:"IAP IP address"`
}

// throttle spaces outbound requests by minInterval. Callers must not hold mu.
func (c *Client) throttle() {
	c.mu.Lock()
	wait := c.minInterval - time.Since(c.lastReq)
	c.lastReq = time.Now().Add(maxDuration(0, wait))
	c.mu.Unlock()
	if wait > 0 {
		time.Sleep(wait)
	}
}

// login authenticates and stores the session id. The caller holds no lock.
func (c *Client) login(ctx context.Context) error {
	body, _ := json.Marshal(map[string]string{"user": c.user, "passwd": c.passwd})

	env, _, err := c.do(ctx, http.MethodPost, "/rest/login", nil, body)
	if err != nil {
		return err
	}
	if env.SID == "" || !strings.EqualFold(env.Status, "Success") {
		return &vendors.AuthError{APILabel: c.apiLabel, Reason: firstNonEmpty(env.ErrorMessage, env.Message, "login failed")}
	}

	c.mu.Lock()
	c.sid = env.SID
	c.mu.Unlock()
	logging.Debugf("[aruba] %s logged in to %s", c.apiLabel, c.host)
	return nil
}

// Logout invalidates the session. Safe to call when not logged in.
func (c *Client) Logout(ctx context.Context) error {
	c.mu.Lock()
	sid := c.sid
	c.sid = ""
	c.mu.Unlock()
	if sid == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"sid": sid})
	_, _, err := c.do(ctx, http.MethodPost, "/rest/logout", nil, body)
	return err
}

// currentSID returns the active session id, logging in if needed.
func (c *Client) currentSID(ctx context.Context) (string, error) {
	c.mu.Lock()
	sid := c.sid
	c.mu.Unlock()
	if sid != "" {
		return sid, nil
	}
	if err := c.login(ctx); err != nil {
		return "", err
	}
	c.mu.Lock()
	sid = c.sid
	c.mu.Unlock()
	return sid, nil
}

// withSession runs fn with a valid sid, re-authenticating once if the device
// reports the session expired (status code 1).
func (c *Client) withSession(ctx context.Context, fn func(sid string) (*apiEnvelope, error)) (*apiEnvelope, error) {
	sid, err := c.currentSID(ctx)
	if err != nil {
		return nil, err
	}

	env, err := fn(sid)
	if err == nil {
		return env, nil
	}
	if !isExpiredSession(err) {
		return nil, err
	}

	// Session lapsed (15-minute idle timeout). Drop it and retry once.
	c.mu.Lock()
	c.sid = ""
	c.mu.Unlock()
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	c.mu.Lock()
	sid = c.sid
	c.mu.Unlock()
	return fn(sid)
}

// ShowCommand runs a monitoring `show` command and returns its CLI output text.
// cmd is the plain CLI form, e.g. "show running-config"; spaces are encoded as
// %20 as the device requires.
func (c *Client) ShowCommand(ctx context.Context, cmd string) (string, error) {
	if out, ok := c.memoGet(cmd); ok {
		return out, nil
	}

	env, err := c.withSession(ctx, func(sid string) (*apiEnvelope, error) {
		q := url.Values{}
		q.Set("iap_ip_addr", c.host)
		q.Set("sid", sid)
		// The device wants %20 for spaces, not '+'; build cmd separately.
		raw := "iap_ip_addr=" + url.QueryEscape(c.host) +
			"&cmd=" + encodeShowCmd(cmd) +
			"&sid=" + url.QueryEscape(sid)
		env, _, err := c.do(ctx, http.MethodGet, "/rest/show-cmd?"+raw, nil, nil)
		return env, err
	})
	if err != nil {
		return "", err
	}

	out := stripCLIPrefix(env.CommandOutput)
	c.memoPut(cmd, out)
	return out, nil
}

// memoGet returns a cached show output if present and within TTL.
func (c *Client) memoGet(cmd string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.showMemo[cmd]
	if !ok || (c.memoTTL > 0 && time.Since(e.at) > c.memoTTL) {
		return "", false
	}
	return e.out, true
}

func (c *Client) memoPut(cmd, out string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.showMemo[cmd] = memoEntry{out: out, at: time.Now()}
}

// memoClear drops all cached reads. Called after a write, whose effect a cached
// read would otherwise miss.
func (c *Client) memoClear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.showMemo)
}

// PostObject sends a Configuration or Action API payload to /rest/<path> and
// validates the response envelope. path is the leaf, e.g. "ssid" or "hostname".
func (c *Client) PostObject(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("aruba: marshal %s payload: %w", path, err)
	}
	_, err = c.withSession(ctx, func(sid string) (*apiEnvelope, error) {
		q := url.Values{}
		q.Set("sid", sid)
		env, _, err := c.do(ctx, http.MethodPost, "/rest/"+strings.TrimPrefix(path, "/")+"?"+q.Encode(), nil, body)
		return env, err
	})
	// A write changes device state; drop cached reads so a later read-back (e.g.
	// apply's verify step) re-fetches rather than serving the pre-write snapshot.
	c.memoClear()
	return err
}

// do performs one HTTP round trip, decodes the envelope, and classifies it.
func (c *Client) do(ctx context.Context, method, path string, _ url.Values, body []byte) (*apiEnvelope, *http.Response, error) {
	c.throttle()

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("aruba: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	logging.Debugf("[aruba] %s %s", method, redactPath(path))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, &vendors.TransportError{APILabel: c.apiLabel, Op: method + " " + leafPath(path), Err: err, Retryable: true}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, &vendors.TransportError{APILabel: c.apiLabel, Op: leafPath(path), Status: resp.StatusCode, Err: err}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp, classifyHTTP(c.apiLabel, leafPath(path), resp.StatusCode, raw)
	}

	var env apiEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, resp, &vendors.TransportError{APILabel: c.apiLabel, Op: leafPath(path), Status: resp.StatusCode,
			Err: fmt.Errorf("decode response: %w", err)}
	}

	if cerr := classifyEnvelope(c.apiLabel, leafPath(path), &env); cerr != nil {
		return &env, resp, cerr
	}
	return &env, resp, nil
}

// hostFromBaseURL extracts the bare host (no port) from a base URL, tolerating
// a missing scheme.
func hostFromBaseURL(baseURL string) string {
	s := baseURL
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	if u, err := url.Parse(s); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return strings.TrimRight(baseURL, "/")
}

// encodeShowCmd escapes a CLI command for the cmd query parameter, using %20
// for spaces as the device requires (url encoding would emit '+').
func encodeShowCmd(cmd string) string {
	return strings.ReplaceAll(url.QueryEscape(cmd), "+", "%20")
}

// stripCLIPrefix removes the "cli output:" / "COMMAND=..." preamble the device
// prepends to monitoring output, leaving the raw command text.
func stripCLIPrefix(out string) string {
	s := strings.TrimPrefix(strings.TrimSpace(out), "cli output:")
	s = strings.TrimLeft(s, "\r\n")
	if idx := strings.Index(s, "COMMAND="); idx == 0 {
		if nl := strings.IndexAny(s, "\r\n"); nl >= 0 {
			s = s[nl+1:]
		}
	}
	return strings.TrimLeft(s, "\r\n")
}

func leafPath(path string) string {
	p := path
	if i := strings.Index(p, "?"); i >= 0 {
		p = p[:i]
	}
	return p
}

// redactPath keeps the sid out of debug logs.
func redactPath(path string) string {
	if i := strings.Index(path, "sid="); i >= 0 {
		end := strings.IndexByte(path[i:], '&')
		if end < 0 {
			return path[:i] + "sid=REDACTED"
		}
		return path[:i] + "sid=REDACTED" + path[i+end:]
	}
	return path
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
