package ui

import (
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// tickMsg fires on a regular interval to trigger flash expiry checks.
type tickMsg time.Time

// tickerMsg carries a live update from the Binance stream.
type tickerMsg ticker.Ticker

// connMsg signals a connection state change.
type connMsg struct{ connected bool }

// Model is the Bubble Tea application state.
type Model struct {
	tickers     map[string]ticker.Ticker // keyed by Symbol
	sorted      []ticker.Ticker          // sorted by QuoteVolume desc
	connected   bool
	termW       int
	termH       int
	visibleRows int
}

// New creates an initial Model pre-populated with tickers from the REST fetch.
func New(initial []ticker.Ticker) Model {
	m := Model{
		tickers:   make(map[string]ticker.Ticker, len(initial)),
		connected: false,
	}
	for _, t := range initial {
		m.tickers[t.Symbol] = t
	}
	m.rebuildSorted()
	return m
}

// rebuildSorted re-sorts the tickers map into the sorted slice by QuoteVolume desc.
func (m *Model) rebuildSorted() {
	all := make([]ticker.Ticker, 0, len(m.tickers))
	for _, t := range m.tickers {
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].QuoteVolume > all[j].QuoteVolume
	})
	m.sorted = all
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

