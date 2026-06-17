package refreshui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// barWidth is the cell width of the determinate progress bar and the dotted
// placeholder shown before a counted stage begins, so a row's geometry doesn't
// jump when it switches between them.
const barWidth = 18

type rowState int

const (
	rowActive rowState = iota
	rowDone
	rowFailed
)

// row is the live state of one API's refresh, mutated by board messages.
type row struct {
	label   string
	state   rowState
	stage   string
	done    int
	total   int
	dur     time.Duration
	failErr error
}

// boardModel is the bubbletea model: a fixed, ordered set of rows repainted in
// place. The row set never grows after construction, so the rendered line count
// is stable and bubbletea can repaint without scrolling.
type boardModel struct {
	order  []string
	rows   map[string]*row
	spin   spinner.Model
	bar    progress.Model
	labelW int
}

func newBoardModel(labels []string) *boardModel {
	rows := make(map[string]*row, len(labels))
	labelW := 0
	for _, l := range labels {
		rows[l] = &row{label: l, stage: "waiting"}
		if len(l) > labelW {
			labelW = len(l)
		}
	}
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	return &boardModel{
		order:  labels,
		rows:   rows,
		spin:   sp,
		bar:    progress.New(progress.WithWidth(barWidth), progress.WithoutPercentage()),
		labelW: labelW,
	}
}

func (m *boardModel) Init() tea.Cmd { return m.spin.Tick }

func (m *boardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Let the operator escape a wedged render; the background refresh keeps
		// running and its summary still prints once teardown returns.
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case startMsg:
		if r := m.rows[msg.label]; r != nil {
			r.state = rowActive
			r.stage = "starting"
		}
	case stageMsg:
		if r := m.rows[msg.label]; r != nil {
			r.stage = msg.stage
			r.done, r.total = 0, 0
		}
	case progressMsg:
		if r := m.rows[msg.label]; r != nil {
			r.done, r.total = msg.done, msg.total
		}
	case doneMsg:
		if r := m.rows[msg.label]; r != nil {
			r.state = rowDone
			r.dur = msg.dur
		}
	case errMsg:
		if r := m.rows[msg.label]; r != nil {
			r.state = rowFailed
			r.failErr = msg.err
		}
	}
	return m, nil
}

var (
	doneStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true)
	failStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	labelStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Faint(true)
)

func (m *boardModel) View() string {
	var b []byte
	for _, label := range m.order {
		r := m.rows[label]
		name := labelStyle.Render(fmt.Sprintf("%-*s", m.labelW, label))
		b = append(b, '[')
		b = append(b, name...)
		b = append(b, ']', ' ')
		// One space after the status glyph in every branch so the bar column
		// lines up — the spinner and the ✔/✖ glyphs are each one cell wide.
		switch r.state {
		case rowDone:
			b = append(b, doneStyle.Render("✔")...)
			b = append(b, ' ')
			b = append(b, m.bar.ViewAs(1)...)
			b = append(b, fmt.Sprintf("  Started %dms", r.dur.Milliseconds())...)
		case rowFailed:
			b = append(b, failStyle.Render("✖")...)
			b = append(b, ' ')
			b = append(b, dimStyle.Render(dots(barWidth))...)
			b = append(b, fmt.Sprintf("  Failed: %s", friendlyError(r.failErr))...)
		default: // rowActive
			b = append(b, m.spin.View()...)
			b = append(b, ' ')
			if r.total > 0 {
				b = append(b, m.bar.ViewAs(float64(r.done)/float64(r.total))...)
				b = append(b, fmt.Sprintf("  %s %d/%d", r.stage, r.done, r.total)...)
			} else {
				b = append(b, dimStyle.Render(dots(barWidth))...)
				b = append(b, ' ', ' ')
				b = append(b, r.stage...)
			}
		}
		b = append(b, '\n')
	}
	return string(b)
}

// friendlyError reduces a wrapped refresh error to a short, operator-readable
// reason — the raw chain (e.g. `failed to fetch sites: ...: dial tcp HOST: i/o
// timeout`) is too long for a board row and gets truncated. The full error still
// prints in the command's trailing error block. When a dial target is present it
// is appended so the operator still knows which host failed.
func friendlyError(err error) string {
	if err == nil {
		return "unknown error"
	}
	msg := err.Error()
	low := strings.ToLower(msg)

	var reason string
	switch {
	case strings.Contains(low, "deadline exceeded"),
		strings.Contains(low, "timeout"),
		strings.Contains(low, "timed out"):
		reason = "connection timed out"
	case strings.Contains(low, "connection refused"):
		reason = "connection refused"
	case strings.Contains(low, "no such host"):
		reason = "host not found"
	case strings.Contains(low, "no route to host"),
		strings.Contains(low, "network is unreachable"):
		reason = "network unreachable"
	case strings.Contains(low, "tls"), strings.Contains(low, "certificate"):
		reason = "TLS handshake failed"
	case strings.Contains(low, "401"),
		strings.Contains(low, "403"),
		strings.Contains(low, "unauthorized"),
		strings.Contains(low, "forbidden"),
		strings.Contains(low, "invalid api key"),
		strings.Contains(low, "authentication"):
		reason = "authentication failed"
	default:
		reason = innermost(msg)
	}

	if host := dialTarget(msg); host != "" {
		return reason + " (" + host + ")"
	}
	return reason
}

// dialTarget extracts the host:port from a Go dial error
// (`... dial tcp 10.0.0.1:443: i/o timeout`), or "" when absent.
func dialTarget(msg string) string {
	const marker = "dial tcp "
	i := strings.Index(strings.ToLower(msg), marker)
	if i < 0 {
		return ""
	}
	rest := msg[i+len(marker):]
	if j := strings.Index(rest, ": "); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}

// innermost returns the last `: `-delimited segment of a wrapped error — the
// root cause — for errors that don't match a known class.
func innermost(msg string) string {
	parts := strings.Split(msg, ": ")
	for i := len(parts) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(parts[i]); s != "" {
			return s
		}
	}
	return strings.TrimSpace(msg)
}

// dots returns n dot runes — the indeterminate stand-in for the bar before a
// counted stage supplies a fraction.
func dots(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = '.'
	}
	return string(out)
}

// board owns the bubbletea program lifecycle and hands out a Reporter that feeds
// it. start launches the render loop; stop quits it and waits for the final
// frame to settle.
type board struct {
	prog *tea.Program
	done chan struct{}
	once sync.Once
}

func newBoard(labels []string) *board {
	m := newBoardModel(labels)
	p := tea.NewProgram(m, tea.WithOutput(os.Stdout))
	return &board{prog: p, done: make(chan struct{})}
}

func (b *board) start() {
	go func() {
		_, _ = b.prog.Run()
		close(b.done)
	}()
}

func (b *board) reporter() Reporter { return &boardReporter{prog: b.prog} }

func (b *board) stop() {
	b.once.Do(func() {
		b.prog.Quit()
		<-b.done
	})
}

// message types carry reporter events to the model over the program's queue.
type (
	startMsg    struct{ label, vendor, site string }
	stageMsg    struct{ label, stage string }
	progressMsg struct {
		label       string
		done, total int
	}
	doneMsg struct {
		label string
		dur   time.Duration
	}
	errMsg struct {
		label string
		err   error
	}
)

// boardReporter translates Reporter calls into program messages. Send is
// goroutine-safe, so the concurrent per-API refreshes share one reporter.
type boardReporter struct{ prog *tea.Program }

func (b *boardReporter) APIStart(label, vendor, site string) {
	b.prog.Send(startMsg{label: label, vendor: vendor, site: site})
}
func (b *boardReporter) Stage(label, stage string) {
	b.prog.Send(stageMsg{label: label, stage: stage})
}

// StageResult is unused by the board — the next Stage or the determinate count
// supersedes it — but the linear reporter needs it, so the interface keeps it.
func (b *boardReporter) StageResult(string, string) {}

func (b *boardReporter) Progress(label string, done, total int) {
	b.prog.Send(progressMsg{label: label, done: done, total: total})
}
func (b *boardReporter) APIDone(label string, dur time.Duration) {
	b.prog.Send(doneMsg{label: label, dur: dur})
}
func (b *boardReporter) APIError(label string, err error) {
	b.prog.Send(errMsg{label: label, err: err})
}
