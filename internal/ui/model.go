package ui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// maxHistory is the number of recent prices kept per symbol for sparklines.
const maxHistory = 20

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
	tickers      map[string]ticker.Ticker // keyed by Symbol
	sorted       []ticker.Ticker          // current sorted view
	priceHistory map[string][]float64     // recent prices per symbol for sparklines
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
}

// New creates an initial Model pre-populated with tickers from the REST fetch.
func New(initial []ticker.Ticker) Model {
	m := Model{
		tickers:      make(map[string]ticker.Ticker, len(initial)),
		priceHistory: make(map[string][]float64, len(initial)),
		watchlist:    watchlist.New(),
		connected:    false,
		sortCol:      SortVolume,
		sortAsc:      false,
	}
	for _, t := range initial {
		m.tickers[t.Symbol] = t
		m.priceHistory[t.Symbol] = []float64{t.LastPrice}
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
	switch m.filterMode {
	case FilterGainers:
		sort.Slice(all, func(i, j int) bool {
			return all[i].PriceChangePercent > all[j].PriceChangePercent
		})
		if len(all) > 20 {
			all = all[:20]
		}
	case FilterLosers:
		sort.Slice(all, func(i, j int) bool {
			return all[i].PriceChangePercent < all[j].PriceChangePercent
		})
		if len(all) > 20 {
			all = all[:20]
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
		si := m.watchlist.IsStarred(all[i].Symbol)
		sj := m.watchlist.IsStarred(all[j].Symbol)
		if si != sj {
			return si // starred first
		}
		if m.sortAsc {
			return !lessVal(i, j)
		}
		return lessVal(i, j)
	})
	m.sorted = all
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
	// Adjust offset so cursor is always visible.
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

