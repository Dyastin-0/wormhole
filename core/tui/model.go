package tui

import (
	"net/http"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/core/tui/components/logdetail"
	"github.com/Dyastin-0/wormhole/core/tui/components/loglist"
	"github.com/Dyastin-0/wormhole/core/tui/components/metrics"
	"github.com/Dyastin-0/wormhole/core/tui/messages"
	"github.com/Dyastin-0/wormhole/core/tui/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	name    string
	spinner spinner.Model

	metrics   metrics.Model
	logList   loglist.Model
	logDetail logdetail.Model

	viewMode messages.ViewMode

	httpLogch chan *proto.HTTPLog
	requestch chan *http.Request

	width  int
	height int

	keys GlobalKeyMap
	help help.Model
}

func New(name string, hasMetrics, hasHTTPLogging bool) (Model, chan *proto.HTTPLog, chan *http.Request) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Subtle)

	httpLogch := make(chan *proto.HTTPLog, 16)
	requestch := make(chan *http.Request, 16)

	h := help.New()
	h.ShortSeparator = " • "
	h.Styles.ShortSeparator = styles.Label.Width(0)
	h.Styles.ShortKey = styles.HelpKey
	h.Styles.FullKey = styles.HelpKey
	h.Styles.ShortDesc = styles.Footer.Faint(true)
	h.Styles.FullDesc = styles.Footer.Faint(true)

	return Model{
		name:      name,
		spinner:   s,
		metrics:   metrics.New(hasMetrics),
		logList:   loglist.New(hasHTTPLogging),
		logDetail: logdetail.New(),
		viewMode:  messages.ViewModeList,
		httpLogch: httpLogch,
		requestch: requestch,
		keys:      DefaultGlobalKeyMap(),
		help:      h,
	}, httpLogch, requestch
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.metrics.Init(),
		m.logList.Init(),
		m.logDetail.Init(),
	)
}
