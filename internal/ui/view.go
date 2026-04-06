package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full TUI frame as a string.
func (m Model) View() string {
	if m.termW == 0 {
		return ""
	}

	if m.showHelp {
		return m.renderHelpView()
	}

	if m.configUI.active {
		return m.renderConfigView()
	}

	tableW := m.tableWidth()
	tableStr := m.renderTable(tableW)

	if !m.panelVisible() {
		return tableStr
	}

	panelStr := m.renderPanel()
	return lipgloss.JoinHorizontal(lipgloss.Top, tableStr, panelStr)
}

// renderTable renders the main table content at the given width.
func (m Model) renderTable(tableW int) string {
	s := m.styles
	var sb strings.Builder

	sb.WriteString(RenderHeader(s, tableW, m.sortCol, m.sortAsc))
	sb.WriteByte('\n')

	sb.WriteString(RenderSeparator(s, tableW))
	sb.WriteByte('\n')

	visRows := m.visibleRows
	limit := len(m.sorted)
	if visRows > 0 && limit > m.offset+visRows {
		limit = m.offset + visRows
	}

	for i := m.offset; i < limit; i++ {
		t := m.sorted[i]
		isCursor := i == m.cursor
		spark := m.priceHistory[t.Symbol]
		starred := m.watchlist.IsStarred(t.Symbol)
		liqFlash := time.Now().Before(m.liqFlash[t.Symbol])
		sb.WriteString(RenderRow(s, i+1, t, tableW, isCursor, spark, starred, liqFlash))
		sb.WriteByte('\n')
	}

	filled := (limit - m.offset) + 2
	targetH := m.termH - 2 // minus footer separator + footer
	for filled < targetH {
		sb.WriteByte('\n')
		filled++
	}

	sb.WriteString(RenderSeparator(s, tableW))
	sb.WriteByte('\n')

	btcPrice := 0.0
	if btc, ok := m.tickers["BTCUSDT"]; ok {
		btcPrice = btc.LastPrice
	}
	sb.WriteString(RenderFooter(s, len(m.tickers), m.connected, tableW, btcPrice, m.filterMode, m.searching, m.searchQuery))

	return sb.String()
}
