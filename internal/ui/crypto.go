package ui

import (
	"fmt"
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
	newsArticles  []news.Article
	newsOn        bool
	newsFlash     int
	focused       bool

	DetailOverlay *tuikit.DetailOverlay[ticker.Ticker]
	Panel         *MarketPanel

	table     *tuikit.Table
	defiTable *tuikit.Table

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
	}
	c.Panel = NewMarketPanel(c.styles)

	columns := []tuikit.Column{
		{Title: "#", Width: 5, Align: tuikit.Right},
		{Title: "SYMBOL", Width: 14, Sortable: true},
		{Title: "PRICE", Width: 20, Align: tuikit.Right, Sortable: true},
		{Title: "CHANGE", Width: 10, Align: tuikit.Right, Sortable: true},
		{Title: "TREND", Width: 22, MinWidth: 70},
		{Title: "βBTC", Width: 8, Align: tuikit.Right, MinWidth: 90, Sortable: true},
		{Title: "VOLUME", Width: 15, Align: tuikit.Right, Sortable: true},
	}

	cellRenderer := func(row tuikit.Row, colIdx int, isCursor bool, theme tuikit.Theme) string {
		if colIdx >= len(row) {
			return ""
		}
		s := c.styles
		symbol := ""
		if len(row) > 1 {
			symbol = row[1] + "USDT"
		}

		cell := row[colIdx]

		// TREND column: render sparkline
		if colIdx == 4 {
			sparkData := c.priceHistory[symbol]
			styled, _ := tuikit.Sparkline(sparkData, 20, &tuikit.SparklineOpts{
				UpStyle:      s.Positive,
				DownStyle:    s.Negative,
				NeutralStyle: s.Neutral,
			})
			return styled
		}

		// Determine flash state
		t := c.tickers[symbol]
		flashing := time.Now().Before(t.FlashUntil) && t.Flash != ticker.FlashNeutral
		liqFlashing := time.Now().Before(c.liqFlash[symbol])
		starred := c.Watchlist.IsStarred(symbol)
		corr := c.correlations[symbol]

		if flashing {
			return flashStyle(s, t.Flash).Render(cell)
		}
		if liqFlashing {
			return s.LiqFlash.Render(cell)
		}
		if isCursor {
			switch colIdx {
			case 3: // CHANGE
				return s.CursorRow.Foreground(changeColor(s, t.PriceChangePercent)).Render(cell)
			case 5: // βBTC
				return s.CursorRow.Foreground(corrColor(s, corr)).Render(cell)
			case 6: // VOLUME
				if t.VolumeSpiking {
					return s.CursorRow.Foreground(s.ColorVolSpike).Render(cell)
				}
			case 1: // SYMBOL
				if starred {
					runes := []rune(cell)
					if len(runes) > 1 {
						return s.CursorRow.Foreground(s.ColorStar).Render(string(runes[:1])) + s.CursorRow.Render(string(runes[1:]))
					}
				}
			}
			return s.CursorRow.Render(cell)
		}

		// Non-cursor styling
		switch colIdx {
		case 3: // CHANGE
			return changeStyle(s, t.PriceChangePercent).Render(cell)
		case 5: // βBTC
			return corrStyle(s, corr).Render(cell)
		case 6: // VOLUME
			if t.VolumeSpiking {
				return s.VolSpike.Render(cell)
			}
		case 1: // SYMBOL
			if starred {
				runes := []rune(cell)
				if len(runes) > 1 {
					return s.Star.Render(string(runes[:1])) + string(runes[1:])
				}
			}
		}
		return cell
	}

	sortFunc := func(a, b tuikit.Row, sortCol int, sortAsc bool) bool {
		// Sorting is handled externally by rebuildSorted, so this is a no-op
		// (rows are pre-sorted before being set on the table)
		return false
	}

	c.table = tuikit.NewTable(columns, nil, tuikit.TableOpts{
		CellRenderer: cellRenderer,
		SortFunc:     sortFunc,
	})

	defiColumns := []tuikit.Column{
		{Title: "#", Width: 4, Align: tuikit.Right},
		{Title: "PROTOCOL", Width: 20},
		{Title: "POOL", Width: 30},
		{Title: "CHAIN", Width: 15},
		{Title: "APY", Width: 15, Align: tuikit.Right, Sortable: true},
		{Title: "TVL", Width: 15, Align: tuikit.Right, Sortable: true},
	}

	defiCellRenderer := func(row tuikit.Row, colIdx int, isCursor bool, theme tuikit.Theme) string {
		if colIdx >= len(row) {
			return ""
		}
		cell := row[colIdx]
		s := c.styles
		if isCursor {
			return s.CursorRow.Render(cell)
		}
		if colIdx == 4 { // APY
			return s.Positive.Render(cell)
		}
		return cell
	}

	c.defiTable = tuikit.NewTable(defiColumns, nil, tuikit.TableOpts{
		Sortable:     true,
		CellRenderer: defiCellRenderer,
	})

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
			c.syncPanel()
		}
		return c, nil
	case fngMsg:
		idx := feargreed.Index(msg)
		if idx.Value > 0 {
			c.fearGreed = idx
			c.syncPanel()
		}
		return c, nil
	case defiMsg:
		if msg != nil {
			c.defiPools = []defiyields.Pool(msg)
			c.rebuildDefiRows()
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
		c.syncPanel()
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
		default:
			if c.defiTable != nil {
				c.defiTable.Update(msg)
			}
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

	// Normal mode — delegate navigation to table
	switch msg.String() {
	case "j", "down", "k", "up", "g", "home", "G", "end", "ctrl+d", "ctrl+u":
		if c.table != nil {
			c.table.Update(msg)
			c.cursor = c.table.CursorIndex()
		}
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
	case "d":
		c.showDefi = true
		if c.defiTable != nil {
			c.defiTable.SetCursor(0)
		}
		return c, tuikit.Consumed()
	case "n":
		c.newsOn = !c.newsOn
		c.visibleRows = c.tableVisibleRows()
		c.clampCursor()
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

	if c.showDefi {
		if c.defiTable != nil {
			c.defiTable.Update(msg)
		}
		return c, nil
	}

	// Delegate to table for scrolling and row selection
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		if c.table != nil {
			c.table.Update(msg)
			c.cursor = c.table.CursorIndex()
		}
	case tea.MouseButtonLeft:
		newsH := c.newsHeight()
		newsStart := c.height - newsH
		if newsH > 0 && msg.Y >= newsStart {
			lineIdx := msg.Y - newsStart - 1
			if lineIdx >= 0 && lineIdx < 5 && lineIdx < len(c.newsArticles) {
				url := c.newsArticles[lineIdx].URL
				if url != "" {
					return c, func() tea.Msg {
						tuikit.OpenURL(url)
						return nil
					}
				}
			}
		} else if c.table != nil {
			c.table.Update(msg)
			c.cursor = c.table.CursorIndex()
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
		{Key: "n", Label: "Toggle news", Group: "DATA"},
		{Key: "d", Label: "DeFi yields", Group: "DATA"},
		{Key: "enter", Label: "Coin detail", Group: "DATA"},
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
	c.syncPanel()
	c.rebuildRows()
}

func (c *CryptoView) rebuildRows() {
	if c.table == nil {
		return
	}
	rows := make([]tuikit.Row, len(c.sorted))
	for i, t := range c.sorted {
		corr := c.correlations[t.Symbol]
		corrStr := fmt.Sprintf("%.2f", corr)
		if t.Symbol == "BTCUSDT" {
			corrStr = "—"
		}
		price := ticker.FormatPrice(t.LastPrice)
		if time.Now().Before(t.FlashUntil) && t.PriceDelta != 0 {
			price += " " + fmt.Sprintf("%+.2f", t.PriceDelta)
		}
		vol := ticker.FormatVolume(t.QuoteVolume)
		if t.VolumeSpiking {
			vol += fmt.Sprintf(" %.1fx", t.VolumeSpikeRatio)
		}
		sym := t.DisplaySymbol()
		if c.Watchlist.IsStarred(t.Symbol) {
			sym = "★ " + sym
		}
		rows[i] = tuikit.Row{
			fmt.Sprintf("%d", i+1),
			sym,
			price,
			formatChange(t.PriceChangePercent),
			"", // TREND — rendered by CellRenderer
			corrStr,
			vol,
		}
	}
	c.table.SetRows(rows)
	c.table.SetCursor(c.cursor)
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

func (c *CryptoView) rebuildDefiRows() {
	if c.defiTable == nil {
		return
	}
	rows := make([]tuikit.Row, len(c.defiPools))
	for i, p := range c.defiPools {
		rows[i] = tuikit.Row{
			fmt.Sprintf("%d", i+1),
			p.Protocol,
			p.Symbol,
			p.Chain,
			fmt.Sprintf("%.2f%%", p.APY),
			ticker.FormatVolume(p.TVL),
		}
	}
	c.defiTable.SetRows(rows)
}

func (c *CryptoView) syncPanel() {
	if c.Panel == nil {
		return
	}
	c.Panel.SetMarketStats(c.marketStats)
	c.Panel.SetFundingRates(c.fundingRates)
	c.Panel.SetFearGreed(FearGreedData{Value: c.fearGreed.Value, Label: c.fearGreed.Label})
	c.Panel.SetRecentLiqs(c.recentLiqs)
	c.Panel.SetTickers(c.tickers)
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

// Accessor methods for status bar closures.

// FilterMode returns the current filter mode.
func (c *CryptoView) FilterMode() FilterMode { return c.filterMode }

// SearchQuery returns the current search query.
func (c *CryptoView) SearchQuery() string { return c.searchQuery }

// IsSearching returns whether search mode is active.
func (c *CryptoView) IsSearching() bool { return c.searching }

// PairCount returns the total number of tracked pairs.
func (c *CryptoView) PairCount() int { return len(c.tickers) }

// VisibleCount returns the number of visible (filtered/sorted) rows.
func (c *CryptoView) VisibleCount() int { return len(c.sorted) }

// CursorPos returns the current cursor position.
func (c *CryptoView) CursorPos() int { return c.cursor }

// BtcPrice returns the current BTC price.
func (c *CryptoView) BtcPrice() float64 {
	if btc, ok := c.tickers["BTCUSDT"]; ok {
		return btc.LastPrice
	}
	return 0
}

// Connected returns the current connection state.
func (c *CryptoView) Connected() bool { return c.connected }

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
	if c.Panel != nil {
		c.Panel.SetStyles(c.styles)
	}
	c.panelOn = parsePanelOn(c.Cfg.PanelLayout)
	c.sortCol = parseSortCol(c.Cfg.DefaultSort)
	c.sortAsc = c.Cfg.SortAscending
	c.filterMode = parseFilterMode(c.Cfg.DefaultFilter)
	c.visibleRows = c.tableVisibleRows()
	c.rebuildSorted()
	c.clampCursor()
}
