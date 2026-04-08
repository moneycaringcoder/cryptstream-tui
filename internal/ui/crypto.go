package ui

import (
	"math"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	tuikit "github.com/moneycaringcoder/tuikit-go"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
	"github.com/moneycaringcoder/cryptstream-tui/internal/defiyields"
	"github.com/moneycaringcoder/cryptstream-tui/internal/feargreed"
	"github.com/moneycaringcoder/cryptstream-tui/internal/funding"
	"github.com/moneycaringcoder/cryptstream-tui/internal/liquidation"
	"github.com/moneycaringcoder/cryptstream-tui/internal/news"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
	"github.com/moneycaringcoder/cryptstream-tui/internal/watchlist"
)

// Message types for external data.

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

// defiMsg carries DeFi yield pool data.
type defiMsg []defiyields.Pool

// newsMsg carries news articles.
type newsMsg []news.Article

// SortCol identifies which column is used for sorting.
type SortCol int

const (
	SortVolume SortCol = iota
	SortPrice
	SortChange
	SortSymbol
	SortCorrelation
	sortColCount
)

// FilterMode controls which subset of tickers to display.
type FilterMode int

const (
	FilterAll     FilterMode = iota
	FilterGainers
	FilterLosers
	filterModeCount
)

// MarketStats holds precomputed aggregate market data for the panel.
type MarketStats struct {
	TotalVolume  float64
	GainerCount  int
	LoserCount   int
	AvgChange    float64
	BtcDominance float64
	TopGainers   []ticker.Ticker
	TopLosers    []ticker.Ticker
	VolSpikes    []ticker.Ticker
	Pinned       []ticker.Ticker
}

// CryptoView is the main crypto dashboard component.
// Implements tuikit.Component.
type CryptoView struct {
	Cfg           *config.Config
	styles        Styles
	tickers       map[string]ticker.Ticker
	sorted        []ticker.Ticker
	priceHistory  map[string][]float64
	volumeHistory map[string][]float64
	Watchlist     *watchlist.Watchlist
	connected     bool
	width         int
	height        int
	visibleRows   int
	cursor        int
	offset        int
	sortCol       SortCol
	sortAsc       bool
	filterMode    FilterMode
	searching     bool
	searchQuery   string
	panelOn       bool
	fundingRates  map[string]funding.Info
	fearGreed     feargreed.Index
	recentLiqs    []liquidation.Liq
	liqFlash      map[string]time.Time
	correlations  map[string]float64
	marketStats   MarketStats
	defiPools     []defiyields.Pool
	showDefi      bool
	defiCursor    int
	defiScroll    int
	newsArticles  []news.Article
	newsOn        bool
	newsFlash     int
	focused       bool

	DetailOverlay *tuikit.DetailOverlay[ticker.Ticker]

	secVolSpikes *tuikit.CollapsibleSection
	secFunding   *tuikit.CollapsibleSection
	secLiqs      *tuikit.CollapsibleSection

	fundingPoller *tuikit.Poller
	fngPoller     *tuikit.Poller
	defiPoller    *tuikit.Poller
	newsPoller    *tuikit.Poller
}

// NewCryptoView creates a CryptoView pre-populated with initial ticker data.
func NewCryptoView(initial []ticker.Ticker, cfg *config.Config) *CryptoView {
	c := &CryptoView{
		Cfg:           cfg,
		styles:        NewStyles(*cfg),
		tickers:       make(map[string]ticker.Ticker, len(initial)),
		priceHistory:  make(map[string][]float64, len(initial)),
		volumeHistory: make(map[string][]float64, len(initial)),
		Watchlist:     watchlist.New(),
		sortCol:       parseSortCol(cfg.DefaultSort),
		sortAsc:       cfg.SortAscending,
		filterMode:    parseFilterMode(cfg.DefaultFilter),
		panelOn:       parsePanelOn(cfg.PanelLayout),
		liqFlash:      make(map[string]time.Time),
		correlations:  make(map[string]float64),
		newsOn:        true,
		secVolSpikes:  tuikit.NewCollapsibleSection("VOL SPIKES"),
		secFunding:    tuikit.NewCollapsibleSection("FUNDING RATES"),
		secLiqs:       tuikit.NewCollapsibleSection("LIQUIDATIONS"),
	}
	for _, t := range initial {
		c.tickers[t.Symbol] = t
		c.priceHistory[t.Symbol] = []float64{t.LastPrice}
		c.volumeHistory[t.Symbol] = []float64{t.QuoteVolume}
	}
	c.rebuildSorted()

	c.fundingPoller = tuikit.NewPoller(5*time.Minute, func() tea.Cmd { return fetchFundingCmd() })
	c.fngPoller = tuikit.NewPoller(30*time.Minute, func() tea.Cmd { return fetchFngCmd() })
	c.defiPoller = tuikit.NewPoller(5*time.Minute, func() tea.Cmd { return fetchDefiCmd() })
	c.newsPoller = tuikit.NewPoller(5*time.Minute, func() tea.Cmd { return fetchNewsCmd() })

	return c
}

func (c *CryptoView) Init() tea.Cmd {
	return tea.Batch(
		ConnCmd(true),
		fetchFundingCmd(),
		fetchFngCmd(),
		fetchDefiCmd(),
		fetchNewsCmd(),
	)
}

func (c *CryptoView) Update(msg tea.Msg) (tuikit.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tuikit.TickMsg:
		return c.handleTick(msg)
	case tickerMsg:
		return c.handleTicker(msg)
	case fundingMsg:
		if msg != nil {
			c.fundingRates = map[string]funding.Info(msg)
		}
		return c, nil
	case fngMsg:
		idx := feargreed.Index(msg)
		if idx.Value > 0 {
			c.fearGreed = idx
		}
		return c, nil
	case defiMsg:
		if msg != nil {
			c.defiPools = []defiyields.Pool(msg)
		}
		return c, nil
	case newsMsg:
		if msg != nil {
			newArticles := []news.Article(msg)
			if len(c.newsArticles) == 0 || (len(newArticles) > 0 && newArticles[0].Title != c.newsArticles[0].Title) {
				c.newsFlash = 20
			}
			c.newsArticles = newArticles
			c.visibleRows = c.tableVisibleRows()
			c.clampCursor()
		}
		return c, nil
	case liqMsg:
		l := liquidation.Liq(msg)
		c.recentLiqs = append([]liquidation.Liq{l}, c.recentLiqs...)
		if len(c.recentLiqs) > 10 {
			c.recentLiqs = c.recentLiqs[:10]
		}
		c.liqFlash[l.Symbol] = time.Now().Add(2 * time.Second)
		return c, nil
	case connMsg:
		c.connected = msg.connected
		return c, nil
	case tea.KeyMsg:
		return c.handleKey(msg)
	case tea.MouseMsg:
		return c.handleMouse(msg)
	}
	return c, nil
}

func (c *CryptoView) handleTick(msg tuikit.TickMsg) (tuikit.Component, tea.Cmd) {
	// Flash countdowns
	if c.newsFlash > 0 {
		c.newsFlash--
	}

	// Check pollers
	var cmds []tea.Cmd
	if cmd := c.fundingPoller.Check(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd := c.fngPoller.Check(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd := c.defiPoller.Check(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd := c.newsPoller.Check(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if len(cmds) > 0 {
		return c, tea.Batch(cmds...)
	}
	return c, nil
}

func (c *CryptoView) handleTicker(msg tickerMsg) (tuikit.Component, tea.Cmd) {
	t := ticker.Ticker(msg)
	t.FlashUntil = time.Now().Add(c.Cfg.FlashDuration.Unwrap())
	thresh := c.Cfg.FlashThreshold
	if prev, ok := c.tickers[t.Symbol]; ok {
		diff := t.LastPrice - prev.LastPrice
		t.PriceDelta = diff
		switch {
		case diff > thresh:
			t.Flash = ticker.FlashPositive
		case diff < -thresh:
			t.Flash = ticker.FlashNegative
		default:
			t.Flash = ticker.FlashNeutral
		}
	}

	maxHist := c.Cfg.SparklineLength
	h := c.priceHistory[t.Symbol]
	h = append(h, t.LastPrice)
	if len(h) > maxHist {
		h = h[len(h)-maxHist:]
	}
	c.priceHistory[t.Symbol] = h

	volWindow := c.Cfg.VolumeWindow
	if volWindow < 2 {
		volWindow = 2
	}
	vh := c.volumeHistory[t.Symbol]
	vh = append(vh, t.QuoteVolume)
	if len(vh) > volWindow {
		vh = vh[len(vh)-volWindow:]
	}
	c.volumeHistory[t.Symbol] = vh

	if len(vh) >= 2 {
		var sum float64
		for _, v := range vh[:len(vh)-1] {
			sum += v
		}
		avg := sum / float64(len(vh)-1)
		if avg > 0 {
			ratio := t.QuoteVolume / avg
			mult := c.Cfg.VolumeSpikeMultiplier
			if mult <= 0 {
				mult = 2.0
			}
			t.VolumeSpiking = ratio >= mult
			t.VolumeSpikeRatio = ratio
		}
	}
	c.tickers[t.Symbol] = t
	c.rebuildSorted()
	return c, nil
}

func (c *CryptoView) handleKey(msg tea.KeyMsg) (tuikit.Component, tea.Cmd) {
	// DeFi view keys
	if c.showDefi {
		switch msg.String() {
		case "d", "esc":
			c.showDefi = false
		case "j", "down":
			c.defiCursor++
			c.clampDefiCursor()
		case "k", "up":
			c.defiCursor--
			c.clampDefiCursor()
		case "g", "home":
			c.defiCursor = 0
			c.clampDefiCursor()
		case "G", "end":
			c.defiCursor = len(c.defiPools) - 1
			c.clampDefiCursor()
		}
		return c, tuikit.Consumed()
	}

	// Search mode
	if c.searching {
		switch msg.String() {
		case "esc":
			c.searching = false
			c.searchQuery = ""
			c.rebuildSorted()
			c.cursor = 0
			c.clampCursor()
		case "enter":
			c.searching = false
		case "backspace":
			if len(c.searchQuery) > 0 {
				c.searchQuery = c.searchQuery[:len(c.searchQuery)-1]
				c.rebuildSorted()
				c.cursor = 0
				c.clampCursor()
			}
		default:
			k := msg.String()
			if len(k) == 1 && k[0] >= 32 && k[0] <= 126 {
				c.searchQuery += k
				c.rebuildSorted()
				c.cursor = 0
				c.clampCursor()
			}
		}
		return c, tuikit.Consumed()
	}

	// Normal mode
	switch msg.String() {
	case "j", "down":
		c.cursor++
		c.clampCursor()
		return c, tuikit.Consumed()
	case "k", "up":
		c.cursor--
		c.clampCursor()
		return c, tuikit.Consumed()
	case "g", "home":
		c.cursor = 0
		c.clampCursor()
		return c, tuikit.Consumed()
	case "G", "end":
		c.cursor = len(c.sorted) - 1
		c.clampCursor()
		return c, tuikit.Consumed()
	case "ctrl+d":
		half := c.visibleRows / 2
		if half < 1 {
			half = 1
		}
		c.cursor += half
		c.clampCursor()
		return c, tuikit.Consumed()
	case "ctrl+u":
		half := c.visibleRows / 2
		if half < 1 {
			half = 1
		}
		c.cursor -= half
		c.clampCursor()
		return c, tuikit.Consumed()
	case "tab":
		c.sortCol = (c.sortCol + 1) % sortColCount
		c.rebuildSorted()
		c.clampCursor()
		return c, tuikit.Consumed()
	case "shift+tab":
		c.sortCol = (c.sortCol - 1 + sortColCount) % sortColCount
		c.rebuildSorted()
		c.clampCursor()
		return c, tuikit.Consumed()
	case "s":
		if c.cursor >= 0 && c.cursor < len(c.sorted) {
			sym := c.sorted[c.cursor].Symbol
			wasStarred := c.Watchlist.IsStarred(sym)
			c.Watchlist.Toggle(sym)
			var msg string
			if wasStarred {
				msg = "★ unstarred " + strings.TrimSuffix(sym, "USDT")
			} else {
				msg = "★ starred " + strings.TrimSuffix(sym, "USDT")
			}
			if !c.panelOn {
				c.panelOn = true
				c.Cfg.PanelLayout = "right"
				c.visibleRows = c.tableVisibleRows()
			}
			c.rebuildSorted()
			c.clampCursor()
			return c, tuikit.NotifyCmd(msg, 2*time.Second)
		}
		return c, tuikit.Consumed()
	case "f":
		c.filterMode = (c.filterMode + 1) % filterModeCount
		c.rebuildSorted()
		c.cursor = 0
		c.clampCursor()
		return c, tuikit.Consumed()
	case "/":
		c.searching = true
		c.searchQuery = ""
		return c, tuikit.Consumed()
	case "p":
		c.panelOn = !c.panelOn
		if c.panelOn {
			c.Cfg.PanelLayout = "right"
		} else {
			c.Cfg.PanelLayout = "off"
		}
		c.visibleRows = c.tableVisibleRows()
		c.clampCursor()
		return c, tuikit.Consumed()
	case "d":
		c.showDefi = true
		c.defiCursor = 0
		c.defiScroll = 0
		return c, tuikit.Consumed()
	case "n":
		c.newsOn = !c.newsOn
		c.visibleRows = c.tableVisibleRows()
		c.clampCursor()
		return c, tuikit.Consumed()
	case "1":
		c.secVolSpikes.Toggle()
		return c, tuikit.Consumed()
	case "2":
		c.secFunding.Toggle()
		return c, tuikit.Consumed()
	case "3":
		c.secLiqs.Toggle()
		return c, tuikit.Consumed()
	case "enter":
		if c.DetailOverlay != nil && c.cursor >= 0 && c.cursor < len(c.sorted) {
			c.DetailOverlay.Show(c.sorted[c.cursor])
			return c, tuikit.Consumed()
		}
	case "esc":
		if c.searchQuery != "" {
			c.searchQuery = ""
			c.rebuildSorted()
			c.cursor = 0
			c.clampCursor()
			return c, tuikit.Consumed()
		}
	}

	return c, nil
}

func (c *CryptoView) handleMouse(msg tea.MouseMsg) (tuikit.Component, tea.Cmd) {
	if msg.Action == tea.MouseActionRelease {
		return c, nil
	}

	tableW := c.tableWidth()

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if msg.X < tableW {
			if c.showDefi {
				c.defiCursor--
				c.clampDefiCursor()
			} else {
				c.cursor--
				c.clampCursor()
			}
		}
	case tea.MouseButtonWheelDown:
		if msg.X < tableW {
			if c.showDefi {
				c.defiCursor++
				c.clampDefiCursor()
			} else {
				c.cursor++
				c.clampCursor()
			}
		}
	case tea.MouseButtonLeft:
		x := msg.X
		y := msg.Y

		if x >= tableW {
			break
		}

		newsH := c.newsHeight()
		tableEnd := 2 + c.visibleRows
		newsStart := c.height - 2 - newsH
		if y >= 2 && y < tableEnd {
			if c.showDefi {
				row := c.defiScroll + (y - 3)
				if row >= 0 && row < len(c.defiPools) {
					c.defiCursor = row
					c.clampDefiCursor()
				}
			} else {
				row := c.offset + (y - 2)
				if row < len(c.sorted) {
					c.cursor = row
					c.clampCursor()
				}
			}
		} else if newsH > 0 && y >= newsStart && y < newsStart+newsH {
			lineIdx := y - newsStart - 1
			if lineIdx >= 0 && lineIdx < 5 && lineIdx < len(c.newsArticles) {
				url := c.newsArticles[lineIdx].URL
				if url != "" {
					return c, func() tea.Msg {
						openBrowser(url)
						return nil
					}
				}
			}
		}
	}
	return c, nil
}

func (c *CryptoView) KeyBindings() []tuikit.KeyBind {
	return []tuikit.KeyBind{
		{Key: "j/↓", Label: "Move down", Group: "NAVIGATION"},
		{Key: "k/↑", Label: "Move up", Group: "NAVIGATION"},
		{Key: "ctrl+d", Label: "Half page down", Group: "NAVIGATION"},
		{Key: "ctrl+u", Label: "Half page up", Group: "NAVIGATION"},
		{Key: "g/Home", Label: "Jump to top", Group: "NAVIGATION"},
		{Key: "G/End", Label: "Jump to bottom", Group: "NAVIGATION"},
		{Key: "mouse", Label: "Scroll / select", Group: "NAVIGATION"},
		{Key: "tab", Label: "Cycle sort column", Group: "DATA"},
		{Key: "shift+tab", Label: "Sort column back", Group: "DATA"},
		{Key: "s", Label: "Star/unstar symbol", Group: "DATA"},
		{Key: "f", Label: "Cycle filter", Group: "DATA"},
		{Key: "p", Label: "Toggle sidebar", Group: "DATA"},
		{Key: "n", Label: "Toggle news", Group: "DATA"},
		{Key: "d", Label: "DeFi yields", Group: "DATA"},
		{Key: "enter", Label: "Coin detail", Group: "DATA"},
		{Key: "1/2/3", Label: "Toggle panel sections", Group: "DATA"},
		{Key: "/", Label: "Search symbols", Group: "SEARCH"},
		{Key: "enter", Label: "Confirm search", Group: "SEARCH"},
		{Key: "esc", Label: "Cancel / clear", Group: "SEARCH"},
	}
}

func (c *CryptoView) SetSize(w, h int) {
	c.width = w
	c.height = h
	c.visibleRows = c.tableVisibleRows()
	c.clampCursor()
}

func (c *CryptoView) Focused() bool       { return c.focused }
func (c *CryptoView) SetFocused(f bool)    { c.focused = f }

// SelectedTicker returns the currently selected ticker, if any.
func (c *CryptoView) SelectedTicker() (ticker.Ticker, bool) {
	if c.cursor >= 0 && c.cursor < len(c.sorted) {
		return c.sorted[c.cursor], true
	}
	return ticker.Ticker{}, false
}

// FundingRate returns the funding info for a symbol.
func (c *CryptoView) FundingRate(symbol string) funding.Info {
	return c.fundingRates[symbol]
}

// PriceHistory returns the sparkline data for a symbol.
func (c *CryptoView) PriceHistory(symbol string) []float64 {
	return c.priceHistory[symbol]
}

// Sorting and filtering.

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

func parsePanelOn(s string) bool {
	switch strings.ToLower(s) {
	case "off", "false", "":
		return false
	default:
		return true
	}
}

func (c *CryptoView) rebuildSorted() {
	all := make([]ticker.Ticker, 0, len(c.tickers))
	for _, t := range c.tickers {
		all = append(all, t)
	}

	if c.searchQuery != "" {
		q := strings.ToUpper(c.searchQuery)
		filtered := all[:0]
		for _, t := range all {
			if strings.Contains(t.Symbol, q) {
				filtered = append(filtered, t)
			}
		}
		all = filtered
	}

	filterCount := c.Cfg.FilterCount
	switch c.filterMode {
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
		switch c.sortCol {
		case SortPrice:
			return all[i].LastPrice > all[j].LastPrice
		case SortChange:
			return all[i].PriceChangePercent > all[j].PriceChangePercent
		case SortCorrelation:
			return c.correlations[all[i].Symbol] > c.correlations[all[j].Symbol]
		case SortSymbol:
			return all[i].Symbol < all[j].Symbol
		default:
			return all[i].QuoteVolume > all[j].QuoteVolume
		}
	}

	sort.SliceStable(all, func(i, j int) bool {
		if c.sortAsc {
			return !lessVal(i, j)
		}
		return lessVal(i, j)
	})
	c.sorted = all
	c.computeMarketStats()
}

func (c *CryptoView) computeMarketStats() {
	var totalVol, totalChange float64
	var gainerCount, loserCount int

	all := make([]ticker.Ticker, 0, len(c.tickers))
	for _, t := range c.tickers {
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
	if btc, ok := c.tickers["BTCUSDT"]; ok && totalVol > 0 {
		btcDom = btc.QuoteVolume / totalVol * 100
	}

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

	var spikes []ticker.Ticker
	for _, t := range c.tickers {
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

	defaultPins := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	pinSet := make(map[string]bool, len(defaultPins))
	var pinned []ticker.Ticker
	for _, sym := range defaultPins {
		pinSet[sym] = true
		if t, ok := c.tickers[sym]; ok {
			pinned = append(pinned, t)
		}
	}
	for _, sym := range c.Watchlist.Symbols() {
		if !pinSet[sym] {
			pinSet[sym] = true
			if t, ok := c.tickers[sym]; ok {
				pinned = append(pinned, t)
			}
		}
	}

	c.marketStats = MarketStats{
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

	c.computeCorrelations()
}

func (c *CryptoView) computeCorrelations() {
	btcHist := c.priceHistory["BTCUSDT"]
	if len(btcHist) < 5 {
		return
	}
	for sym, hist := range c.priceHistory {
		if sym == "BTCUSDT" {
			c.correlations[sym] = 1.0
			continue
		}
		a, b := btcHist, hist
		if len(a) > len(b) {
			a = a[len(a)-len(b):]
		} else if len(b) > len(a) {
			b = b[len(b)-len(a):]
		}
		if len(a) < 5 {
			continue
		}
		c.correlations[sym] = pearson(a, b)
	}
}

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

func (c *CryptoView) clampCursor() {
	if len(c.sorted) == 0 {
		c.cursor = 0
		c.offset = 0
		return
	}
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= len(c.sorted) {
		c.cursor = len(c.sorted) - 1
	}
	if c.visibleRows > 0 {
		if c.cursor < c.offset {
			c.offset = c.cursor
		}
		if c.cursor >= c.offset+c.visibleRows {
			c.offset = c.cursor - c.visibleRows + 1
		}
	}
}

func (c *CryptoView) clampDefiCursor() {
	if len(c.defiPools) == 0 {
		c.defiCursor = 0
		c.defiScroll = 0
		return
	}
	if c.defiCursor < 0 {
		c.defiCursor = 0
	}
	if c.defiCursor >= len(c.defiPools) {
		c.defiCursor = len(c.defiPools) - 1
	}
	visRows := c.visibleRows - 1
	if visRows > 0 {
		if c.defiCursor < c.defiScroll {
			c.defiScroll = c.defiCursor
		}
		if c.defiCursor >= c.defiScroll+visRows {
			c.defiScroll = c.defiCursor - visRows + 1
		}
	}
}

// Fetch commands.

func fetchFundingCmd() tea.Cmd {
	return func() tea.Msg {
		rates, err := funding.Fetch()
		if err != nil {
			return fundingMsg(nil)
		}
		return fundingMsg(rates)
	}
}

func fetchFngCmd() tea.Cmd {
	return func() tea.Msg {
		idx, err := feargreed.Fetch()
		if err != nil {
			return fngMsg(feargreed.Index{})
		}
		return fngMsg(idx)
	}
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

func fetchNewsCmd() tea.Cmd {
	return func() tea.Msg {
		articles, err := news.Fetch(20)
		if err != nil {
			return newsMsg(nil)
		}
		return newsMsg(articles)
	}
}

// ConnCmd returns a Cmd that signals connection state.
func ConnCmd(connected bool) tea.Cmd {
	return func() tea.Msg {
		return connMsg{connected: connected}
	}
}

// TickerMsgFrom converts a ticker.Ticker into a tickerMsg.
func TickerMsgFrom(t ticker.Ticker) tea.Msg {
	return tickerMsg(t)
}

// LiqMsgFrom converts a liquidation.Liq into a liqMsg.
func LiqMsgFrom(l liquidation.Liq) tea.Msg {
	return liqMsg(l)
}

// SetSort changes the sort column by name: volume, price, change, symbol, correlation.
func (c *CryptoView) SetSort(col string) bool {
	sc := parseSortCol(col)
	if col != "" && sc == c.sortCol {
		c.sortAsc = !c.sortAsc // toggle direction if same column
	} else {
		c.sortCol = sc
	}
	c.rebuildSorted()
	c.clampCursor()
	return true
}

// SetFilter changes the filter mode by name: all, gainers, losers.
func (c *CryptoView) SetFilter(mode string) bool {
	fm := parseFilterMode(mode)
	c.filterMode = fm
	c.rebuildSorted()
	c.cursor = 0
	c.clampCursor()
	return true
}

// GoToSymbol scrolls to a symbol in the sorted list.
// Returns true if found.
func (c *CryptoView) GoToSymbol(sym string) bool {
	sym = strings.ToUpper(sym)
	if !strings.HasSuffix(sym, "USDT") {
		sym += "USDT"
	}
	for i, t := range c.sorted {
		if t.Symbol == sym {
			c.cursor = i
			c.clampCursor()
			return true
		}
	}
	return false
}

// ReapplyConfig re-derives styles and state from the current config.
func (c *CryptoView) ReapplyConfig() {
	c.styles = NewStyles(*c.Cfg)
	c.panelOn = parsePanelOn(c.Cfg.PanelLayout)
	c.sortCol = parseSortCol(c.Cfg.DefaultSort)
	c.sortAsc = c.Cfg.SortAscending
	c.filterMode = parseFilterMode(c.Cfg.DefaultFilter)
	c.visibleRows = c.tableVisibleRows()
	c.rebuildSorted()
	c.clampCursor()
}
