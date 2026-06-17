package refreshui

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain strips ANSI styling so View assertions match on plain text.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

func TestBoardModelActiveCountedStage(t *testing.T) {
	m := newBoardModel([]string{"mist", "meraki"})

	if m.labelW != len("meraki") {
		t.Fatalf("labelW = %d, want %d", m.labelW, len("meraki"))
	}

	m.Update(startMsg{label: "mist", vendor: "mist"})
	m.Update(stageMsg{label: "mist", stage: "AP configs"})
	m.Update(progressMsg{label: "mist", done: 143, total: 231})

	r := m.rows["mist"]
	if r.state != rowActive || r.done != 143 || r.total != 231 {
		t.Fatalf("row = %+v, want active 143/231", r)
	}
	if v := m.View(); !strings.Contains(v, "AP configs 143/231") {
		t.Fatalf("view missing counted stage:\n%s", v)
	}
}

func TestBoardModelStageResetsCount(t *testing.T) {
	m := newBoardModel([]string{"mist"})
	m.Update(progressMsg{label: "mist", done: 5, total: 10})
	m.Update(stageMsg{label: "mist", stage: "Fetching WLANs"})

	if r := m.rows["mist"]; r.done != 0 || r.total != 0 {
		t.Fatalf("count not reset on new stage: %+v", r)
	}
}

func TestBoardModelDoneAndError(t *testing.T) {
	m := newBoardModel([]string{"mist", "meraki"})

	m.Update(doneMsg{label: "mist", dur: 952 * time.Millisecond})
	m.Update(errMsg{label: "meraki", err: errors.New("connection refused")})

	if r := m.rows["mist"]; r.state != rowDone || r.dur != 952*time.Millisecond {
		t.Fatalf("mist row = %+v, want done 952ms", r)
	}
	if r := m.rows["meraki"]; r.state != rowFailed {
		t.Fatalf("meraki row = %+v, want failed", r)
	}

	v := m.View()
	if !strings.Contains(v, "Started 952ms") {
		t.Fatalf("view missing done summary:\n%s", v)
	}
	if !strings.Contains(v, "Failed: connection refused") {
		t.Fatalf("view missing error summary:\n%s", v)
	}
}

func TestFriendlyError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "dial timeout keeps host",
			err:  errors.New(`failed to fetch sites: aruba-pina: POST /rest/login: Post "https://172.30.8.20:4343/rest/login": dial tcp 172.30.8.20:4343: i/o timeout`),
			want: "connection timed out (172.30.8.20:4343)",
		},
		{name: "refused", err: errors.New("dial tcp 10.0.0.1:443: connect: connection refused"), want: "connection refused (10.0.0.1:443)"},
		{name: "dns", err: errors.New(`Get "https://api.x.com": dial tcp: lookup api.x.com: no such host`), want: "host not found"},
		{name: "auth", err: errors.New("meraki: 401 Unauthorized"), want: "authentication failed"},
		{name: "unknown falls to innermost", err: errors.New("fetch sites: weird vendor explosion"), want: "weird vendor explosion"},
		{name: "nil", err: nil, want: "unknown error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := friendlyError(tc.err); got != tc.want {
				t.Fatalf("friendlyError = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestBoardRowsAlignAtBar guards the off-by-one fix: the bar (or dots) must
// start at the same column on active, done, and failed rows. The status glyphs
// (spinner, ✔, ✖) are all 3-byte runes, so equal byte offsets mean equal
// columns under the Ascii color profile (no escapes).
func TestBoardRowsAlignAtBar(t *testing.T) {
	m := newBoardModel([]string{"api"})

	m.Update(progressMsg{label: "api", done: 1, total: 2})
	active := strings.IndexRune(m.View(), '█')
	m.Update(doneMsg{label: "api", dur: time.Second})
	done := strings.IndexRune(m.View(), '█')
	m.Update(errMsg{label: "api", err: errors.New("x")})
	failed := strings.IndexRune(m.View(), '.')

	if active != done || active != failed {
		t.Fatalf("bar columns differ: active=%d done=%d failed=%d\n%s", active, done, failed, m.View())
	}
}

func TestBoardModelLineCountStable(t *testing.T) {
	// The rendered line count must equal the API count regardless of state, so
	// bubbletea repaints in place instead of scrolling.
	m := newBoardModel([]string{"a", "b", "c"})
	want := 3
	if got := strings.Count(m.View(), "\n"); got != want {
		t.Fatalf("initial line count = %d, want %d", got, want)
	}
	m.Update(doneMsg{label: "a", dur: time.Second})
	m.Update(errMsg{label: "b", err: errors.New("x")})
	if got := strings.Count(m.View(), "\n"); got != want {
		t.Fatalf("post-update line count = %d, want %d", got, want)
	}
}
