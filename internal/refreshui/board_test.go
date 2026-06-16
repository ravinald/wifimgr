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
	m.Update(errMsg{label: "meraki", err: errors.New("login timeout")})

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
	if !strings.Contains(v, "Failed: login timeout") {
		t.Fatalf("view missing error summary:\n%s", v)
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
