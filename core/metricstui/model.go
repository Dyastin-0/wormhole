package metricstui

import (
	"net/http"
	"time"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxBodySize = 5 * 1024 * 1024
	maxLogs     = 1024
)

type metricsModel struct {
	name           string
	spinner        spinner.Model
	metricsData    MetricsData
	startTime      time.Time
	logStore       *HTTPLogStore
	hasMetrics     bool
	hasHTTPLogging bool

	viewWidth   int
	viewHeight  int
	textYOffset int
	hexYOffset  int
	xOffset     int

	viewMode  ViewMode
	activeTab Tab

	httpLogch chan *proto.HTTPLog
	requestch chan *http.Request

	lastGPress    time.Time
	searchMode    bool
	searchQuery   string
	searchMatches []*matchLocation
	currentMatch  int

	stringContent string
	totalHexRows  int
	lineOffsets   []int
	maxLineLength int

	keys KeyMap
	help help.Model
}

type matchLocation struct {
	line  int
	start int
	end   int
}

func newMetricsModel(
	name string,
	hasMetrics,
	hasHTTPLogging bool,
) (chan *proto.HTTPLog, chan *http.Request, metricsModel) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(subtle)

	httpLogch := make(chan *proto.HTTPLog, 1)
	requestch := make(chan *http.Request, 1)

	h := help.New()
	h.ShortSeparator = " • "
	h.Styles.ShortSeparator = labelStyle.Width(0)
	h.Styles.ShortKey = helpKeyStyle
	h.Styles.FullKey = helpKeyStyle
	h.Styles.ShortDesc = footerStyle.Faint(true)
	h.Styles.FullDesc = footerStyle.Faint(true)
	h.Styles.ShortSeparator = footerStyle.Faint(true)

	return httpLogch, requestch, metricsModel{
		name:           name,
		spinner:        s,
		metricsData:    MetricsData{lastUpdate: time.Now()},
		startTime:      time.Now(),
		logStore:       NewHTTPLogStore(maxLogs, 10),
		hasMetrics:     hasMetrics,
		hasHTTPLogging: hasHTTPLogging,
		viewMode:       ViewModeList,
		activeTab:      TabResponseBody,
		httpLogch:      httpLogch,
		requestch:      requestch,
		keys:           DefaultKeyMap(),
		help:           h,
	}
}

func (m metricsModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *metricsModel) setSelectedLog() {
	log := m.logStore.GetSelected()
	if log == nil {
		return
	}

	var content string
	if m.activeTab == TabRequestBody {
		content = string(log.requestBody)
	} else {
		content = string(log.responseBody)
	}

	m.stringContent = content
	m.lineOffsets, m.maxLineLength = getLineOffsets(m.stringContent)
	m.totalHexRows = (len(m.stringContent) + hexColumnSize - 1) / hexColumnSize

	m.textYOffset = 0
	m.hexYOffset = 0
	m.xOffset = 0

	if m.searchQuery != "" {
		m.findMatches()
	} else {
		m.searchMatches = nil
		m.currentMatch = 0
	}
}

func StartTUI(
	name string,
	hasMetrics,
	hasHTTPLogging bool,
) (*tea.Program, chan any, chan *proto.HTTPLog, chan *http.Request) {
	metricsch := make(chan any, 16)
	httpLogch, requestch, model := newMetricsModel(name, hasMetrics, hasHTTPLogging)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	go func() {
		for msg := range metricsch {
			p.Send(msg)
		}
	}()

	return p, metricsch, httpLogch, requestch
}
