package metricstui

import (
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// New returns a new model with the given width and height as well as default
// key mappings.
func newViewport(width, height int) (m viewportModel) {
	m.Width = width
	m.Height = height
	m.setInitialValues()
	return m
}

// viewportModel is the Bubble Tea model for this viewport element.
type viewportModel struct {
	Width  int
	Height int
	KeyMap viewportKeyMap

	// Whether or not to respond to the mouse. The mouse must be enabled in
	// Bubble Tea for this to work. For details, see the Bubble Tea docs.
	MouseWheelEnabled bool

	// The number of lines the mouse wheel will scroll. By default, this is 3.
	MouseWheelDelta int

	// YOffset is the vertical scroll position.
	YOffset int

	// xOffset is the horizontal scroll position.
	xOffset int

	// horizontalStep is the number of columns we move left or right during a
	// default horizontal scroll.
	horizontalStep int

	// YPosition is the position of the viewport in relation to the terminal
	// window. It's used in high performance rendering only.
	YPosition int

	// Style applies a lipgloss style to the viewport. Realistically, it's most
	// useful for setting borders, margins and padding.
	Style lipgloss.Style

	// HighPerformanceRendering bypasses the normal Bubble Tea renderer to
	// provide higher performance rendering. Most of the time the normal Bubble
	// Tea rendering methods will suffice, but if you're passing content with
	// a lot of ANSI escape codes you may see improved rendering in certain
	// terminals with this enabled.
	//
	// This should only be used in program occupying the entire terminal,
	// which is usually via the alternate screen buffer.
	//
	// Deprecated: high performance rendering is now deprecated in Bubble Tea.
	HighPerformanceRendering bool

	initialized      bool
	lines            []string
	longestLineWidth int
}

func (m *viewportModel) setInitialValues() {
	m.KeyMap = viewPortDefaultKeyMap()
	m.MouseWheelEnabled = true
	m.MouseWheelDelta = 3
	m.initialized = true
}

// Init exists to satisfy the tea.viewportModel interface for composability purposes.
func (m viewportModel) Init() tea.Cmd {
	return nil
}

// AtTop returns whether or not the viewport is at the very top position.
func (m viewportModel) AtTop() bool {
	return m.YOffset <= 0
}

// AtBottom returns whether or not the viewport is at or past the very bottom
// position.
func (m viewportModel) AtBottom() bool {
	return m.YOffset >= m.maxYOffset()
}

// PastBottom returns whether or not the viewport is scrolled beyond the last
// line. This can happen when adjusting the viewport height.
func (m viewportModel) PastBottom() bool {
	return m.YOffset > m.maxYOffset()
}

// ScrollPercent returns the amount scrolled as a float between 0 and 1.
func (m viewportModel) ScrollPercent() float64 {
	if m.Height >= len(m.lines) {
		return 1.0
	}
	y := float64(m.YOffset)
	h := float64(m.Height)
	t := float64(len(m.lines))
	v := y / (t - h)
	return math.Max(0.0, math.Min(1.0, v))
}

// HorizontalScrollPercent returns the amount horizontally scrolled as a float
// between 0 and 1.
func (m viewportModel) HorizontalScrollPercent() float64 {
	if m.xOffset >= m.longestLineWidth-m.Width {
		return 1.0
	}
	y := float64(m.xOffset)
	h := float64(m.Width)
	t := float64(m.longestLineWidth)
	v := y / (t - h)
	return math.Max(0.0, math.Min(1.0, v))
}

// SetContent set the pager's text content.
func (m *viewportModel) SetContent(s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n") // normalize line endings
	m.lines = strings.Split(s, "\n")
	m.longestLineWidth = findLongestLineWidth(m.lines)

	if m.YOffset > len(m.lines)-1 {
		m.GotoBottom()
	}
}

// maxYOffset returns the maximum possible value of the y-offset based on the
// viewport's content and set height.
func (m viewportModel) maxYOffset() int {
	return max(0, len(m.lines)-m.Height+m.Style.GetVerticalFrameSize())
}

func (m viewportModel) maxXOffset() int {
	return m.longestLineWidth - m.Width + m.Style.GetHorizontalFrameSize()
}

// visibleLines returns the lines that should currently be visible in the
// viewport.
func (m viewportModel) visibleLines() (lines []string) {
	h := m.Height - m.Style.GetVerticalFrameSize()
	w := m.Width - m.Style.GetHorizontalFrameSize()

	if len(m.lines) > 0 {
		top := max(0, m.YOffset)
		bottom := clamp(m.YOffset+h, top, len(m.lines))
		lines = m.lines[top:bottom]
	}

	if (m.xOffset == 0 && m.longestLineWidth <= w) || w == 0 {
		return lines
	}

	cutLines := make([]string, len(lines))
	for i := range lines {
		cutLines[i] = ansi.Cut(lines[i], m.xOffset, m.xOffset+w)
	}
	return cutLines
}

// scrollArea returns the scrollable boundaries for high performance rendering.
//
// Deprecated: high performance rendering is deprecated in Bubble Tea.
func (m viewportModel) scrollArea() (top, bottom int) {
	top = max(0, m.YPosition)
	bottom = max(top, top+m.Height)
	if top > 0 && bottom > top {
		bottom--
	}
	return top, bottom
}

// SetYOffset sets the Y offset.
func (m *viewportModel) SetYOffset(n int) {
	m.YOffset = clamp(n, 0, m.maxYOffset())
}

// ViewDown moves the view down by the number of lines in the viewport.
// Basically, "page down".
//
// Deprecated: use [viewportModel.PageDown] instead.
func (m *viewportModel) ViewDown() []string {
	return m.PageDown()
}

// PageDown moves the view down by the number of lines in the viewport.
func (m *viewportModel) PageDown() []string {
	if m.AtBottom() {
		return nil
	}

	return m.ScrollDown(m.Height)
}

// ViewUp moves the view up by one height of the viewport.
// Basically, "page up".
//
// Deprecated: use [viewportModel.PageUp] instead.
func (m *viewportModel) ViewUp() []string {
	return m.PageUp()
}

// PageUp moves the view up by one height of the viewport.
func (m *viewportModel) PageUp() []string {
	if m.AtTop() {
		return nil
	}

	return m.ScrollUp(m.Height)
}

// HalfViewDown moves the view down by half the height of the viewport.
//
// Deprecated: use [viewportModel.HalfPageDown] instead.
func (m *viewportModel) HalfViewDown() (lines []string) {
	return m.HalfPageDown()
}

// HalfPageDown moves the view down by half the height of the viewport.
func (m *viewportModel) HalfPageDown() (lines []string) {
	if m.AtBottom() {
		return nil
	}

	return m.ScrollDown(m.Height / 2) //nolint:mnd
}

// HalfViewUp moves the view up by half the height of the viewport.
//
// Deprecated: use [viewportModel.HalfPageUp] instead.
func (m *viewportModel) HalfViewUp() (lines []string) {
	return m.HalfPageUp()
}

// HalfPageUp moves the view up by half the height of the viewport.
func (m *viewportModel) HalfPageUp() (lines []string) {
	if m.AtTop() {
		return nil
	}

	return m.ScrollUp(m.Height / 2) //nolint:mnd
}

// LineDown moves the view down by the given number of lines.
//
// Deprecated: use [viewportModel.ScrollDown] instead.
func (m *viewportModel) LineDown(n int) (lines []string) {
	return m.ScrollDown(n)
}

// ScrollDown moves the view down by the given number of lines.
func (m *viewportModel) ScrollDown(n int) (lines []string) {
	if m.AtBottom() || n == 0 || len(m.lines) == 0 {
		return nil
	}

	// Make sure the number of lines by which we're going to scroll isn't
	// greater than the number of lines we actually have left before we reach
	// the bottom.
	m.SetYOffset(m.YOffset + n)

	// Gather lines to send off for performance scrolling.
	//
	// XXX: high performance rendering is deprecated in Bubble Tea.
	bottom := clamp(m.YOffset+m.Height, 0, len(m.lines))
	top := clamp(m.YOffset+m.Height-n, 0, bottom)
	return m.lines[top:bottom]
}

// LineUp moves the view down by the given number of lines. Returns the new
// lines to show.
//
// Deprecated: use [viewportModel.ScrollUp] instead.
func (m *viewportModel) LineUp(n int) (lines []string) {
	return m.ScrollUp(n)
}

// ScrollUp moves the view down by the given number of lines. Returns the new
// lines to show.
func (m *viewportModel) ScrollUp(n int) (lines []string) {
	if m.AtTop() || n == 0 || len(m.lines) == 0 {
		return nil
	}

	// Make sure the number of lines by which we're going to scroll isn't
	// greater than the number of lines we are from the top.
	m.SetYOffset(m.YOffset - n)

	// Gather lines to send off for performance scrolling.
	//
	// XXX: high performance rendering is deprecated in Bubble Tea.
	top := max(0, m.YOffset)
	bottom := clamp(m.YOffset+n, 0, m.maxYOffset())
	return m.lines[top:bottom]
}

// SetHorizontalStep sets the default amount of columns to scroll left or right
// with the default viewport key map.
//
// If set to 0 or less, horizontal scrolling is disabled.
//
// On v1, horizontal scrolling is disabled by default.
func (m *viewportModel) SetHorizontalStep(n int) {
	m.horizontalStep = max(n, 0)
}

// SetXOffset sets the X offset.
func (m *viewportModel) SetXOffset(n int) {
	m.xOffset = clamp(n, 0, m.longestLineWidth-m.Width)
}

// ScrollLeft moves the viewport to the left by the given number of columns.
func (m *viewportModel) ScrollLeft(n int) {
	m.SetXOffset(m.xOffset - n)
}

// ScrollRight moves viewport to the right by the given number of columns.
func (m *viewportModel) ScrollRight(n int) {
	m.SetXOffset(m.xOffset + n)
}

// TotalLineCount returns the total number of lines (both hidden and visible) within the viewport.
func (m viewportModel) TotalLineCount() int {
	return len(m.lines)
}

// VisibleLineCount returns the number of the visible lines within the viewport.
func (m viewportModel) VisibleLineCount() int {
	return len(m.visibleLines())
}

// GotoTop sets the viewport to the top position.
func (m *viewportModel) GotoTop() (lines []string) {
	if m.AtTop() {
		return nil
	}

	m.SetYOffset(0)
	return m.visibleLines()
}

// GotoBottom sets the viewport to the bottom position.
func (m *viewportModel) GotoBottom() (lines []string) {
	m.SetYOffset(m.maxYOffset())
	return m.visibleLines()
}

// GoToLeft sets the viewport to the leftmost horizontal position.
func (m *viewportModel) GoToLeft() []string {
	m.SetXOffset(0)
	return m.visibleLines()
}

// GoToRight sets the viewport to the rightmost horizontal position.
func (m *viewportModel) GoToRight() []string {
	m.SetXOffset(m.maxXOffset())
	return m.visibleLines()
}

// Sync tells the renderer where the viewport will be located and requests
// a render of the current state of the viewport. It should be called for the
// first render and after a window resize.
//
// For high performance rendering only.
//
// Deprecated: high performance rendering is deprecated in Bubble Tea.
func Sync(m viewportModel) tea.Cmd {
	if len(m.lines) == 0 {
		return nil
	}
	top, bottom := m.scrollArea()
	return tea.SyncScrollArea(m.visibleLines(), top, bottom)
}

// ViewDown is a high performance command that moves the viewport up by a given
// number of lines. Use viewportModel.ViewDown to get the lines that should be rendered.
// For example:
//
//	lines := model.ViewDown(1)
//	cmd := ViewDown(m, lines)
//
// Deprecated: high performance rendering is deprecated in Bubble Tea.
func ViewDown(m viewportModel, lines []string) tea.Cmd {
	if len(lines) == 0 {
		return nil
	}
	top, bottom := m.scrollArea()

	// XXX: high performance rendering is deprecated in Bubble Tea. In a v2 we
	// won't need to return a command here.
	return tea.ScrollDown(lines, top, bottom)
}

// ViewUp is a high performance command the moves the viewport down by a given
// number of lines height. Use viewportModel.ViewUp to get the lines that should be
// rendered.
//
// Deprecated: high performance rendering is deprecated in Bubble Tea.
func ViewUp(m viewportModel, lines []string) tea.Cmd {
	if len(lines) == 0 {
		return nil
	}
	top, bottom := m.scrollArea()

	// XXX: high performance rendering is deprecated in Bubble Tea. In a v2 we
	// won't need to return a command here.
	return tea.ScrollUp(lines, top, bottom)
}

// Update handles standard message-based viewport updates.
func (m viewportModel) Update(msg tea.Msg) (viewportModel, tea.Cmd) {
	var cmd tea.Cmd
	m, cmd = m.updateAsviewportModel(msg)
	return m, cmd
}

// Author's note: this method has been broken out to make it easier to
// potentially transition Update to satisfy tea.viewportModel.
func (m viewportModel) updateAsviewportModel(msg tea.Msg) (viewportModel, tea.Cmd) {
	if !m.initialized {
		m.setInitialValues()
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.PageDown):
			lines := m.PageDown()
			if m.HighPerformanceRendering {
				cmd = ViewDown(m, lines)
			}

		case key.Matches(msg, m.KeyMap.PageUp):
			lines := m.PageUp()
			if m.HighPerformanceRendering {
				cmd = ViewUp(m, lines)
			}

		case key.Matches(msg, m.KeyMap.HalfPageDown):
			lines := m.HalfPageDown()
			if m.HighPerformanceRendering {
				cmd = ViewDown(m, lines)
			}

		case key.Matches(msg, m.KeyMap.HalfPageUp):
			lines := m.HalfPageUp()
			if m.HighPerformanceRendering {
				cmd = ViewUp(m, lines)
			}

		case key.Matches(msg, m.KeyMap.Down):
			lines := m.ScrollDown(1)
			if m.HighPerformanceRendering {
				cmd = ViewDown(m, lines)
			}

		case key.Matches(msg, m.KeyMap.Up):
			lines := m.ScrollUp(1)
			if m.HighPerformanceRendering {
				cmd = ViewUp(m, lines)
			}

		case key.Matches(msg, m.KeyMap.Left):
			m.ScrollLeft(m.horizontalStep)

		case key.Matches(msg, m.KeyMap.Right):
			m.ScrollRight(m.horizontalStep)
		}

	case tea.MouseMsg:
		if !m.MouseWheelEnabled || msg.Action != tea.MouseActionPress {
			break
		}
		switch msg.Button { //nolint:exhaustive
		case tea.MouseButtonWheelUp:
			if msg.Shift {
				// Note that not every terminal emulator sends the shift event for mouse actions by default (looking at you Konsole)
				m.ScrollLeft(m.horizontalStep)
			} else {
				lines := m.ScrollUp(m.MouseWheelDelta)
				if m.HighPerformanceRendering {
					cmd = ViewUp(m, lines)
				}
			}

		case tea.MouseButtonWheelDown:
			if msg.Shift {
				m.ScrollRight(m.horizontalStep)
			} else {
				lines := m.ScrollDown(m.MouseWheelDelta)
				if m.HighPerformanceRendering {
					cmd = ViewDown(m, lines)
				}
			}
		// Note that not every terminal emulator sends the horizontal wheel events by default (looking at you Konsole)
		case tea.MouseButtonWheelLeft:
			m.ScrollLeft(m.horizontalStep)
		case tea.MouseButtonWheelRight:
			m.ScrollRight(m.horizontalStep)
		}
	}

	return m, cmd
}

// View renders the viewport into a string.
func (m viewportModel) View() string {
	if m.HighPerformanceRendering {
		// Just send newlines since we're going to be rendering the actual
		// content separately. We still need to send something that equals the
		// height of this view so that the Bubble Tea standard renderer can
		// position anything below this view properly.
		return strings.Repeat("\n", max(0, m.Height-1))
	}

	w, h := m.Width, m.Height
	if sw := m.Style.GetWidth(); sw != 0 {
		w = min(w, sw)
	}
	if sh := m.Style.GetHeight(); sh != 0 {
		h = min(h, sh)
	}
	contentWidth := w - m.Style.GetHorizontalFrameSize()
	contentHeight := h - m.Style.GetVerticalFrameSize()
	contents := lipgloss.NewStyle().
		Width(contentWidth).      // pad to width.
		Height(contentHeight).    // pad to height.
		MaxHeight(contentHeight). // truncate height if taller.
		MaxWidth(contentWidth).   // truncate width if wider.
		Render(strings.Join(m.visibleLines(), "\n"))
	return m.Style.
		UnsetWidth().UnsetHeight(). // Style size already applied in contents.
		Render(contents)
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

func findLongestLineWidth(lines []string) int {
	w := 0
	for _, l := range lines {
		if ww := ansi.StringWidth(l); ww > w {
			w = ww
		}
	}
	return w
}

const spacebar = " "

type viewportKeyMap struct {
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	Down         key.Binding
	Up           key.Binding
	Left         key.Binding
	Right        key.Binding
}

// DefaultKeyMap returns a set of pager-like default keybindings.
func viewPortDefaultKeyMap() viewportKeyMap {
	return viewportKeyMap{
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", spacebar, "f"),
			key.WithHelp("f/pgdn", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("b/pgup", "page up"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
			key.WithHelp("u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("d", "½ page down"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
	}
}
