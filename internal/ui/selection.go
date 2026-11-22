package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// noticeDuration is how long the "copied" notification stays on screen.
const noticeDuration = 2 * time.Second

// NoticeExpiredMsg clears the notification (and selection highlight) with the given sequence number.
type NoticeExpiredMsg struct{ Seq int }

// selection is a stream text selection over the rendered screen, in cell coordinates.
type selection struct {
	dragging       bool
	startX, startY int
	endX, endY     int
}

// empty reports whether the selection covers no cells.
func (s selection) empty() bool {
	return s.startX == s.endX && s.startY == s.endY
}

// ordered returns the selection endpoints in reading order.
func (s selection) ordered() (x1, y1, x2, y2 int) {
	if s.startY > s.endY || (s.startY == s.endY && s.startX > s.endX) {
		return s.endX, s.endY, s.startX, s.startY
	}
	return s.startX, s.startY, s.endX, s.endY
}

// columns returns the half-open column range [left, right) selected on a line.
func (s selection) columns(line, width int) (left, right int) {
	x1, y1, x2, y2 := s.ordered()
	left, right = 0, width
	if line == y1 {
		left = x1
	}
	if line == y2 {
		right = x2 + 1
	}
	return left, min(right, width)
}

// inRange reports whether a screen line is touched by the selection.
func (s selection) inRange(line int) bool {
	_, y1, _, y2 := s.ordered()
	return line >= y1 && line <= y2
}

// text extracts the plain text covered by the selection from rendered screen lines.
func (s selection) text(lines []string) string {
	var out []string
	for i, line := range lines {
		if !s.inRange(i) {
			continue
		}
		left, right := s.columns(i, ansi.StringWidth(line))
		if left >= right {
			out = append(out, "")
			continue
		}
		out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(line, left, right)), " "))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

var selectionStyle = lipgloss.NewStyle().Reverse(true)

// highlight renders the selection as reversed video over screen lines.
func (s selection) highlight(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = line
		if !s.inRange(i) {
			continue
		}
		width := ansi.StringWidth(line)
		left, right := s.columns(i, width)
		if left >= right {
			continue
		}
		picked := selectionStyle.Render(ansi.Strip(ansi.Cut(line, left, right)))
		out[i] = ansi.Cut(line, 0, left) + picked + ansi.Cut(line, right, width)
	}
	return out
}

// screenLines renders the current view as the lines the terminal displays.
func (m *Model) screenLines() []string {
	return strings.Split(m.renderCurrentView(), "\n")
}

// handleMouse implements drag-to-select with automatic copy on release.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			m.Notice = ""
			m.Selection = selection{dragging: true, startX: msg.X, startY: msg.Y, endX: msg.X, endY: msg.Y}
		}
	case tea.MouseMotionMsg:
		if m.Selection.dragging {
			m.Selection.endX, m.Selection.endY = msg.X, msg.Y
		}
	case tea.MouseReleaseMsg:
		return m.finishSelection(msg)
	case tea.MouseWheelMsg:
		return m.handleWheel(msg)
	}
	return m, nil
}

// finishSelection ends a drag and copies the selection if it covers any cells.
func (m *Model) finishSelection(msg tea.MouseReleaseMsg) (tea.Model, tea.Cmd) {
	if !m.Selection.dragging {
		return m, nil
	}
	m.Selection.dragging = false
	m.Selection.endX, m.Selection.endY = msg.X, msg.Y
	if m.Selection.empty() {
		return m, nil
	}
	return m.copySelection()
}

// handleWheel scrolls like the arrow keys, since mouse mode disables terminal alternate-scroll.
func (m *Model) handleWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseWheelUp:
		return m.handleKeyPress(tea.KeyPressMsg{Code: tea.KeyUp})
	case tea.MouseWheelDown:
		return m.handleKeyPress(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	return m, nil
}

// copySelection copies the selected text to the clipboard and shows a notification.
func (m *Model) copySelection() (tea.Model, tea.Cmd) {
	text := m.Selection.text(m.screenLines())
	if text == "" {
		m.Selection = selection{}
		return m, nil
	}
	m.noticeSeq++
	m.Notice = fmt.Sprintf("✓ Copied %d characters to clipboard", len([]rune(text)))
	seq := m.noticeSeq
	return m, tea.Batch(
		tea.SetClipboard(text),
		tea.Tick(noticeDuration, func(time.Time) tea.Msg { return NoticeExpiredMsg{Seq: seq} }),
	)
}

// handleNoticeExpired clears the notification unless a newer one replaced it.
func (m *Model) handleNoticeExpired(msg NoticeExpiredMsg) (tea.Model, tea.Cmd) {
	if msg.Seq == m.noticeSeq {
		m.Notice = ""
		m.Selection = selection{}
	}
	return m, nil
}

// decorateScreen overlays the selection highlight and the copy notification on the rendered screen.
func (m *Model) decorateScreen(content string) string {
	if m.Selection.empty() && m.Notice == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	if !m.Selection.empty() {
		lines = m.Selection.highlight(lines)
	}
	if m.Notice != "" && len(lines) > 0 {
		lines[len(lines)-1] = NoticeStyle.Width(m.Width).Render(m.Notice)
	}
	return strings.Join(lines, "\n")
}
