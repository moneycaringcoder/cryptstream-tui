package ui

import (
	"strings"
)

// View renders the full TUI frame as a string.
func (m Model) View() string {
	if m.termW == 0 {
		return ""
	}

	if m.configUI.active {
		return m.renderConfigView()
	}

	s := m.styles
	var sb strings.Builder

	sb.WriteString(RenderHeader(s, m.termW, m.sortCol, m.sortAsc))
	sb.WriteByte('\n')

	sb.WriteString(RenderSeparator(s, m.termW))
	sb.WriteByte('\n')

	limit := len(m.sorted)
	if m.visibleRows > 0 && limit > m.offset+m.visibleRows {
		limit = m.offset + m.visibleRows
	}

	for i := m.offset; i < limit; i++ {
		t := m.sorted[i]
		isCursor := i == m.cursor
		spark := m.priceHistory[t.Symbol]
		starred := m.watchlist.IsStarred(t.Symbol)
		sb.WriteString(RenderRow(s, i+1, t, m.termW, isCursor, spark, starred))
		sb.WriteByte('\n')
	}

	filled := (limit - m.offset) + 2
	for filled < m.termH-1 {
		sb.WriteByte('\n')
		filled++
	}

	sb.WriteString(RenderSeparator(s, m.termW))
	sb.WriteByte('\n')

	btcPrice := 0.0
	if btc, ok := m.tickers["BTCUSDT"]; ok {
		btcPrice = btc.LastPrice
	}
	sb.WriteString(RenderFooter(s, len(m.tickers), m.connected, m.termW, btcPrice, m.filterMode, m.searching, m.searchQuery))

	return sb.String()
}
