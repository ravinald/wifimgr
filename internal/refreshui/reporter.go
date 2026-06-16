// Package refreshui renders the progress of a multi-API cache refresh. It offers
// two reporters behind one interface: a linear reporter that prints one whole
// line per stage (safe to pipe, the default), and a live "status board" that
// repaints a row per API in place — the docker-compose look — when stdout is an
// interactive terminal.
package refreshui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// Reporter receives progress events for a cache refresh. doRefreshAPI drives the
// per-stage calls; the orchestrator marks terminal success/failure. A single
// Reporter is shared across the concurrent per-API goroutines, so every
// implementation must be safe for concurrent use.
type Reporter interface {
	APIStart(label, vendor, siteID string)  // a per-API refresh began
	Stage(label, stage string)              // a named step started, e.g. "Fetching sites"
	StageResult(label, summary string)      // the step closed with a short summary, e.g. "5 sites"
	Progress(label string, done, total int) // determinate count within the active step (config fetch)
	APIDone(label string, dur time.Duration)
	APIError(label string, err error)
}

// New returns a Reporter and a teardown func. When interactive, it starts the
// live board (one repainting row per label) and the teardown paints the final
// frame and releases the terminal; otherwise it returns the linear reporter and
// a no-op teardown. Always call the teardown — defer it.
func New(labels []string, interactive bool) (Reporter, func()) {
	if !interactive || len(labels) == 0 {
		return NewLinear(), func() {}
	}
	b := newBoard(labels)
	b.start()
	return b.reporter(), b.stop
}

// Interactive reports whether stdout can host the live board: a real terminal
// that isn't the dumb fallback. A pipe, redirect, or TERM=dumb falls back to
// linear text so captured output stays free of cursor-control escapes.
func Interactive() bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd())) // #nosec G115 -- fds are small non-negative ints
}

// sharedLinear backs Resolve so callers that pass a nil Reporter (single-API
// refreshes) still get linear output without each allocating one. It is
// stateless beyond a per-label pending map guarded by its own mutex.
var sharedLinear = NewLinear()

// Resolve returns r, or the shared linear reporter when r is nil. doRefreshAPI
// uses it so a refresh invoked without a reporter keeps the original behavior.
func Resolve(r Reporter) Reporter {
	if r == nil {
		return sharedLinear
	}
	return r
}

// linearReporter prints one whole line per stage. It buffers the in-progress
// stage per label and emits the line atomically on StageResult, so concurrent
// per-API goroutines can't splice their output mid-line the way bare fmt.Printf
// did. Progress is a no-op — counts surface only in the final stage summary.
type linearReporter struct {
	mu      sync.Mutex
	w       io.Writer
	pending map[string]string // label -> active stage text awaiting its summary
}

// NewLinear returns a linear reporter writing to stdout.
func NewLinear() Reporter { return NewLinearWriter(os.Stdout) }

// NewLinearWriter returns a linear reporter writing to w. Used in tests to
// capture output.
func NewLinearWriter(w io.Writer) Reporter {
	return &linearReporter{w: w, pending: make(map[string]string)}
}

func (l *linearReporter) APIStart(label, vendor, siteID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if siteID != "" {
		_, _ = fmt.Fprintf(l.w, "  [%s] Refreshing %s API (site %s)...\n", label, vendor, siteID)
		return
	}
	_, _ = fmt.Fprintf(l.w, "  [%s] Refreshing %s API...\n", label, vendor)
}

func (l *linearReporter) Stage(label, stage string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pending[label] = stage
}

func (l *linearReporter) StageResult(label, summary string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	stage := l.pending[label]
	delete(l.pending, label)
	if summary == "" {
		_, _ = fmt.Fprintf(l.w, "    %s...\n", stage)
		return
	}
	_, _ = fmt.Fprintf(l.w, "    %s... %s\n", stage, summary)
}

func (l *linearReporter) Progress(string, int, int) {}

func (l *linearReporter) APIDone(label string, dur time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(l.w, "  [%s] Complete in %dms\n", label, dur.Milliseconds())
}

func (l *linearReporter) APIError(label string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// The command layer prints the aggregated error block; keep the inline trace
	// terse so a piped log still shows where a refresh died.
	_, _ = fmt.Fprintf(l.w, "  [%s] Failed: %v\n", label, err)
}
