package ui

import (
	"math"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
	"github.com/moneycaringcoder/cryptstream-tui/internal/defiyields"
	"github.com/moneycaringcoder/cryptstream-tui/internal/feargreed"
	"github.com/moneycaringcoder/cryptstream-tui/internal/news"
	"github.com/moneycaringcoder/cryptstream-tui/internal/funding"
	"github.com/moneycaringcoder/cryptstream-tui/internal/liquidation"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
	"github.com/moneycaringcoder/cryptstream-tui/internal/watchlist"
)

// tickMsg fires on a regular interval to trigger flash expiry checks.
type tickMsg time.Time

// tickerMsg carries a live update from the Binance stream.
type tickerMsg ticker.Ticker

// connMsg signals a connection state change.
type connMsg struct{ connected bool }

// fundingMsg carries updated funding rate data.
type fundingMsg map[string]funding.Info

// liqMsg carries a single liquidation event.
type liqMsg liquidation.Liq

// fngMsg carries the Fear & Greed index.
type fngMsg feargreed.Index

// fngTickMsg triggers a re-fetch of Fear & Greed.
type fngTickMsg time.Time

// defiMsg carries DeFi yield pool data.
type defiMsg []defiyields.Pool

// defiTickMsg triggers a re-fetch of DeFi yields.
type defiTickMsg time.Time

// newsMsg carries news articles.
type newsMsg []news.Article

// newsTickMsg triggers a re-fetch of news.
type newsTickMsg time.Time

// SortCol identifies which column is used for sorting.
type SortCol int

const (
	SortVolume SortCol = iota
	SortPrice
	SortChange
	SortSymbol
	SortCorrelation
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
	fundingRates map[string]funding.Info
	fearGreed    feargreed.Index
	recentLiqs   []liquidation.Liq     // rolling feed, newest first, max 5
	liqFlash     map[string]time.Time  // symbol -> flash expiry for liq indicator
	correlations map[string]float64 // Pearson correlation to BTC
	marketStats  MarketStats
	defiPools    []defiyields.Pool
	showDefi     bool
	defiCursor   int
	defiScroll   int
	newsArticles []news.Article
	newsScroll   int  // unused now, kept for compatibility
	newsOn       bool // news band visible
	newsFlash    int  // countdown ticks for new article flash
	starFlash    int  // countdown ticks for star confirmation flash
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
	case "correlation":
		return SortCorrelation
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
		liqFlash:     make(map[string]time.Time),
		correlations: make(map[string]float64),
		newsOn:       true,
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
		case SortCorrelation:
			return m.correlations[all[i].Symbol] > m.correlations[all[j].Symbol]
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

	m.computeCorrelations()
}

// computeCorrelations computes Pearson correlation of each symbol's price history vs BTC.
func (m *Model) computeCorrelations() {
	btcHist := m.priceHistory["BTCUSDT"]
	if len(btcHist) < 5 {
		return
	}
	for sym, hist := range m.priceHistory {
		if sym == "BTCUSDT" {
			m.correlations[sym] = 1.0
			continue
		}
		// Align lengths (use the shorter tail)
		a, b := btcHist, hist
		if len(a) > len(b) {
			a = a[len(a)-len(b):]
		} else if len(b) > len(a) {
			b = b[len(b)-len(a):]
		}
		if len(a) < 5 {
			continue
		}
		m.correlations[sym] = pearson(a, b)
	}
}

// pearson computes the Pearson correlation coefficient between two equal-length slices.
func pearson(x, y []float64) float64 {
	n := float64(len(x))
	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}
	num := n*sumXY - sumX*sumY
	den := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))
	if den == 0 {
		return 0
	}
	return num / den
}

// PriceHistory returns the sparkline data for a symbol.
func (m *Model) PriceHistory(symbol string) []float64 {
	return m.priceHistory[symbol]
}

// clampDefiCursor keeps defi cursor and scroll within valid bounds.
func (m *Model) clampDefiCursor() {
	if len(m.defiPools) == 0 {
		m.defiCursor = 0
		m.defiScroll = 0
		return
	}
	if m.defiCursor < 0 {
		m.defiCursor = 0
	}
	if m.defiCursor >= len(m.defiPools) {
		m.defiCursor = len(m.defiPools) - 1
	}
	visRows := m.visibleRows - 1 // title row takes one extra
	if visRows > 0 {
		if m.defiCursor < m.defiScroll {
			m.defiScroll = m.defiCursor
		}
		if m.defiCursor >= m.defiScroll+visRows {
			m.defiScroll = m.defiCursor - visRows + 1
		}
	}
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

// Init starts the 100ms tick command, signals connection ready, and fetches external data.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), ConnCmd(true), fetchFundingCmd(), fetchFngCmd(), fetchDefiCmd(), fetchNewsCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchFundingCmd() tea.Cmd {
	return func() tea.Msg {
		rates, err := funding.Fetch()
		if err != nil {
			return fundingMsg(nil)
		}
		return fundingMsg(rates)
	}
}

func fundingTickCmd() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return fundingTickMsg(t)
	})
}

// fundingTickMsg triggers a re-fetch of funding rates.
type fundingTickMsg time.Time

func fetchFngCmd() tea.Cmd {
	return func() tea.Msg {
		idx, err := feargreed.Fetch()
		if err != nil {
			return fngMsg(feargreed.Index{})
		}
		return fngMsg(idx)
	}
}

func fngTickCmd() tea.Cmd {
	return tea.Tick(30*time.Minute, func(t time.Time) tea.Msg {
		return fngTickMsg(t)
	})
}

func fetchDefiCmd() tea.Cmd {
	return func() tea.Msg {
		pools, err := defiyields.Fetch(100, 1_000_000)
		if err != nil {
			return defiMsg(nil)
		}
		return defiMsg(pools)
	}
}

func defiTickCmd() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return defiTickMsg(t)
	})
}

func fetchNewsCmd() tea.Cmd {
	return func() tea.Msg {
		articles, err := news.Fetch(20)
		if err != nil {
			return newsMsg(nil)
		}
		return newsMsg(articles)
	}
}

func newsTickCmd() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return newsTickMsg(t)
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

// LiqMsgFrom converts a liquidation.Liq into a liqMsg for sending to the program.
func LiqMsgFrom(l liquidation.Liq) tea.Msg {
	return liqMsg(l)
}
