package ui

import (
	"strings"
)

// View renders the full TUI frame as a string.
func (m Model) View() string {
	if m.termW == 0 {
		return ""
	}

	var sb strings.Builder

	// Column headers with sort indicator
	sb.WriteString(RenderHeader(m.termW, m.sortCol, m.sortAsc))
	sb.WriteByte('\n')

	// Separator
	sb.WriteString(RenderSeparator(m.termW))
	sb.WriteByte('\n')

	// Determine visible window
	limit := len(m.sorted)
	if m.visibleRows > 0 && limit > m.offset+m.visibleRows {
		limit = m.offset + m.visibleRows
	}

	for i := m.offset; i < limit; i++ {
		t := m.sorted[i]
		isCursor := i == m.cursor
		spark := m.priceHistory[t.Symbol]
		sb.WriteString(RenderRow(i+1, t, m.termW, isCursor, spark))
		sb.WriteByte('\n')
	}

	// Fill blank lines to push footer to bottom
	filled := (limit - m.offset) + 2 // rows rendered + header + separator
	for filled < m.termH-1 {
		sb.WriteByte('\n')
		filled++
	}

	// Footer separator
	sb.WriteString(RenderSeparator(m.termW))
	sb.WriteByte('\n')

	// Footer with BTC price
	btcPrice := 0.0
	if btc, ok := m.tickers["BTCUSDT"]; ok {
		btcPrice = btc.LastPrice
	}
	sb.WriteString(RenderFooter(len(m.tickers), m.connected, m.termW, btcPrice))

	return sb.String()
}
