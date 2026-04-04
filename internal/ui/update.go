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
		m.visibleRows = msg.Height - 4 // header + separator + footer + 1 padding
		if m.visibleRows < 0 {
			m.visibleRows = 0
		}
		return m, nil

	case tickMsg:
		return m, tickCmd()

	case tickerMsg:
		t := ticker.Ticker(msg)
		t.FlashUntil = time.Now().Add(300 * time.Millisecond)
		m.tickers[t.Symbol] = t
		m.rebuildSorted()
		return m, nil

	case connMsg:
		m.connected = msg.connected
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}
