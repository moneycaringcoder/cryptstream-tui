package ui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
	"github.com/moneycaringcoder/cryptstream-tui/internal/watchlist"
)

// tickMsg fires on a regular interval to trigger flash expiry checks.
type tickMsg time.Time

// tickerMsg carries a live update from the Binance stream.
type tickerMsg ticker.Ticker

// connMsg signals a connection state change.
type connMsg struct{ connected bool }

// SortCol identifies which column is used for sorting.
type SortCol int

const (
	SortVolume SortCol = iota
	SortPrice
	SortChange
	SortSymbol
	sortColCount // sentinel for wrap-around
)

// parsePanelOn converts a config string to a bool.
func parsePanelOn(s string) bool {
	switch strings.ToLower(s) {
	case "off", "false", "":
		return false
	default:
		return true
	}
}

// MarketStats holds precomputed aggregate market data for the panel.
type MarketStats struct {
	TotalVolume  float64
	GainerCount  int
	LoserCount   int
	AvgChange    float64
	BtcDominance float64
	TopGainers   []ticker.Ticker
	TopLosers    []ticker.Ticker
	VolSpikes    []ticker.Ticker // top 5 volume spikes by ratio
	Pinned       []ticker.Ticker // BTC, ETH, SOL + starred symbols
}

// FilterMode controls which subset of tickers to display.
type FilterMode int

const (
	FilterAll     FilterMode = iota
	FilterGainers            // top gainers by change%
	FilterLosers             // top losers by change%
	filterModeCount
)

// Model is the Bubble Tea application state.
type Model struct {
	cfg          config.Config
	styles       Styles
	tickers      map[string]ticker.Ticker // keyed by Symbol
	sorted       []ticker.Ticker          // current sorted view
	priceHistory  map[string][]float64     // recent prices per symbol for sparklines
	volumeHistory map[string][]float64    // recent volumes per symbol for spike detection
	watchlist    *watchlist.Watchlist
	connected    bool
	termW        int
	termH        int
	visibleRows  int
	cursor       int        // selected row index within sorted
	offset       int        // scroll offset (first visible row)
	sortCol      SortCol    // active sort column
	sortAsc      bool       // ascending if true
	filterMode   FilterMode // current filter
	searching    bool       // search input mode active
	searchQuery  string     // current search text
	panelOn      bool
	marketStats  MarketStats
	configUI     configState
	showHelp     bool
}

// parseSortCol converts a config string to a SortCol.
func parseSortCol(s string) SortCol {
	switch strings.ToLower(s) {
	case "price":
		return SortPrice
	case "change":
		return SortChange
	case "symbol":
		return SortSymbol
	default:
		return SortVolume
	}
}

// parseFilterMode converts a config string to a FilterMode.
func parseFilterMode(s string) FilterMode {
	switch strings.ToLower(s) {
	case "gainers":
		return FilterGainers
	case "losers":
		return FilterLosers
	default:
		return FilterAll
	}
}

// New creates an initial Model pre-populated with tickers from the REST fetch.
func New(initial []ticker.Ticker, cfg config.Config) Model {
	m := Model{
		cfg:          cfg,
		styles:       NewStyles(cfg),
		tickers:       make(map[string]ticker.Ticker, len(initial)),
		priceHistory:  make(map[string][]float64, len(initial)),
		volumeHistory: make(map[string][]float64, len(initial)),
		watchlist:    watchlist.New(),
		connected:    false,
		sortCol:      parseSortCol(cfg.DefaultSort),
		sortAsc:      cfg.SortAscending,
		filterMode:   parseFilterMode(cfg.DefaultFilter),
		panelOn:      parsePanelOn(cfg.PanelLayout),
	}
	for _, t := range initial {
		m.tickers[t.Symbol] = t
		m.priceHistory[t.Symbol] = []float64{t.LastPrice}
		m.volumeHistory[t.Symbol] = []float64{t.QuoteVolume}
	}
	m.rebuildSorted()
	return m
}

// rebuildSorted re-sorts the tickers map into the sorted slice using the active sort column.
// Starred symbols are pinned to the top. Filter mode is applied.
func (m *Model) rebuildSorted() {
	all := make([]ticker.Ticker, 0, len(m.tickers))
	for _, t := range m.tickers {
		all = append(all, t)
	}

	// Apply search filter
	if m.searchQuery != "" {
		q := strings.ToUpper(m.searchQuery)
		filtered := all[:0]
		for _, t := range all {
			if strings.Contains(t.Symbol, q) {
				filtered = append(filtered, t)
			}
		}
		all = filtered
	}

	// Apply filter
	filterCount := m.cfg.FilterCount
	switch m.filterMode {
	case FilterGainers:
		sort.Slice(all, func(i, j int) bool {
			return all[i].PriceChangePercent > all[j].PriceChangePercent
		})
		if len(all) > filterCount {
			all = all[:filterCount]
		}
	case FilterLosers:
		sort.Slice(all, func(i, j int) bool {
			return all[i].PriceChangePercent < all[j].PriceChangePercent
		})
		if len(all) > filterCount {
			all = all[:filterCount]
		}
	}

	lessVal := func(i, j int) bool {
		switch m.sortCol {
		case SortPrice:
			return all[i].LastPrice > all[j].LastPrice
		case SortChange:
			return all[i].PriceChangePercent > all[j].PriceChangePercent
		case SortSymbol:
			return all[i].Symbol < all[j].Symbol
		default: // SortVolume
			return all[i].QuoteVolume > all[j].QuoteVolume
		}
	}

	sort.SliceStable(all, func(i, j int) bool {
		if m.sortAsc {
			return !lessVal(i, j)
		}
		return lessVal(i, j)
	})
	m.sorted = all
	m.computeMarketStats()
}

// computeMarketStats derives aggregate stats from all tickers (unfiltered).
func (m *Model) computeMarketStats() {
	var totalVol, totalChange float64
	var gainerCount, loserCount int

	all := make([]ticker.Ticker, 0, len(m.tickers))
	for _, t := range m.tickers {
		all = append(all, t)
		totalVol += t.QuoteVolume
		totalChange += t.PriceChangePercent
		if t.PriceChangePercent > 0 {
			gainerCount++
		} else if t.PriceChangePercent < 0 {
			loserCount++
		}
	}

	avgChange := 0.0
	if len(all) > 0 {
		avgChange = totalChange / float64(len(all))
	}

	btcDom := 0.0
	if btc, ok := m.tickers["BTCUSDT"]; ok && totalVol > 0 {
		btcDom = btc.QuoteVolume / totalVol * 100
	}

	// Sort a copy to find top gainers/losers
	sort.Slice(all, func(i, j int) bool {
		return all[i].PriceChangePercent > all[j].PriceChangePercent
	})

	topGainers := make([]ticker.Ticker, 0, topN)
	for i := 0; i < topN && i < len(all); i++ {
		if all[i].PriceChangePercent > 0 {
			topGainers = append(topGainers, all[i])
		}
	}

	topLosers := make([]ticker.Ticker, 0, topN)
	for i := len(all) - 1; i >= 0 && len(topLosers) < topN; i-- {
		if all[i].PriceChangePercent < 0 {
			topLosers = append(topLosers, all[i])
		}
	}

	// Collect volume spikes sorted by ratio descending
	var spikes []ticker.Ticker
	for _, t := range m.tickers {
		if t.VolumeSpiking {
			spikes = append(spikes, t)
		}
	}
	sort.Slice(spikes, func(i, j int) bool {
		return spikes[i].VolumeSpikeRatio > spikes[j].VolumeSpikeRatio
	})
	if len(spikes) > 5 {
		spikes = spikes[:5]
	}

	// Build pinned list: BTC, ETH, SOL always, then starred symbols
	defaultPins := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	pinSet := make(map[string]bool, len(defaultPins))
	var pinned []ticker.Ticker
	for _, sym := range defaultPins {
		pinSet[sym] = true
		if t, ok := m.tickers[sym]; ok {
			pinned = append(pinned, t)
		}
	}
	for _, sym := range m.watchlist.Symbols() {
		if !pinSet[sym] {
			pinSet[sym] = true
			if t, ok := m.tickers[sym]; ok {
				pinned = append(pinned, t)
			}
		}
	}

	m.marketStats = MarketStats{
		TotalVolume:  totalVol,
		GainerCount:  gainerCount,
		LoserCount:   loserCount,
		AvgChange:    avgChange,
		BtcDominance: btcDom,
		TopGainers:   topGainers,
		TopLosers:    topLosers,
		VolSpikes:    spikes,
		Pinned:       pinned,
	}
}

// PriceHistory returns the sparkline data for a symbol.
func (m *Model) PriceHistory(symbol string) []float64 {
	return m.priceHistory[symbol]
}

// clampCursor keeps cursor and offset within valid bounds.
func (m *Model) clampCursor() {
	if len(m.sorted) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.sorted) {
		m.cursor = len(m.sorted) - 1
	}
	if m.visibleRows > 0 {
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
		if m.cursor >= m.offset+m.visibleRows {
			m.offset = m.cursor - m.visibleRows + 1
		}
	}
}

// Init starts the 100ms tick command and signals connection ready.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), ConnCmd(true))
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ConnCmd returns a Cmd that signals connection state.
func ConnCmd(connected bool) tea.Cmd {
	return func() tea.Msg {
		return connMsg{connected: connected}
	}
}

// TickerMsgFrom converts a ticker.Ticker into a tickerMsg for sending to the program.
func TickerMsgFrom(t ticker.Ticker) tea.Msg {
	return tickerMsg(t)
}
