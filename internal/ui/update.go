package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// Update handles all incoming messages and returns the updated model + next cmd.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.termW = msg.Width
		m.termH = msg.Height
		m.visibleRows = msg.Height - 4 // header + separator + footer separator + footer
		if m.visibleRows < 0 {
			m.visibleRows = 0
		}
		m.clampCursor()
		return m, nil

	case tickMsg:
		return m, tickCmd()

	case tickerMsg:
		t := ticker.Ticker(msg)
		t.FlashUntil = time.Now().Add(300 * time.Millisecond)
		if prev, ok := m.tickers[t.Symbol]; ok {
			// Determine flash direction from price change.
			diff := t.LastPrice - prev.LastPrice
			switch {
			case diff > 0.0001:
				t.Flash = ticker.FlashPositive
			case diff < -0.0001:
				t.Flash = ticker.FlashNegative
			default:
				t.Flash = ticker.FlashNeutral
			}
		}
		m.tickers[t.Symbol] = t

		// Track price history for sparklines.
		h := m.priceHistory[t.Symbol]
		h = append(h, t.LastPrice)
		if len(h) > maxHistory {
			h = h[len(h)-maxHistory:]
		}
		m.priceHistory[t.Symbol] = h

		m.rebuildSorted()
		return m, nil

	case connMsg:
		m.connected = msg.connected
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.cursor++
			m.clampCursor()
		case "k", "up":
			m.cursor--
			m.clampCursor()
		case "g", "home":
			m.cursor = 0
			m.clampCursor()
		case "G", "end":
			m.cursor = len(m.sorted) - 1
			m.clampCursor()
		case "tab":
			m.sortCol = (m.sortCol + 1) % sortColCount
			m.rebuildSorted()
			m.clampCursor()
		case "shift+tab":
			m.sortCol = (m.sortCol - 1 + sortColCount) % sortColCount
			m.rebuildSorted()
			m.clampCursor()
		}
	}

	return m, nil
}
