package ui

import (
	"fmt"
	"strings"

	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
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
		starred := m.watchlist.IsStarred(t.Symbol)
		sb.WriteString(RenderRow(i+1, t, m.termW, isCursor, spark, starred))
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
	sb.WriteString(RenderFooter(len(m.tickers), m.connected, m.termW, btcPrice, m.filterMode, m.searching, m.searchQuery))

	base := sb.String()

	// Detail popup overlay
	if m.showDetail && m.cursor >= 0 && m.cursor < len(m.sorted) {
		base = m.overlayDetail(base)
	}

	return base
}

// overlayDetail renders a detail popup centered on the screen.
func (m Model) overlayDetail(base string) string {
	t := m.sorted[m.cursor]
	spark := m.priceHistory[t.Symbol]

	boxW := 50
	if boxW > m.termW-4 {
		boxW = m.termW - 4
	}
	innerW := boxW - 4 // 2 border + 2 padding

	// Build popup lines
	var lines []string
	lines = append(lines, centerText(fmt.Sprintf("─── %s ───", t.DisplaySymbol()), innerW))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  Price:    %s", ticker.FormatPrice(t.LastPrice)))
	lines = append(lines, fmt.Sprintf("  Change:   %s", formatChange(t.PriceChangePercent)))
	lines = append(lines, fmt.Sprintf("  Volume:   %s", ticker.FormatVolume(t.QuoteVolume)))
	lines = append(lines, fmt.Sprintf("  High:     %s", ticker.FormatPrice(t.HighPrice)))
	lines = append(lines, fmt.Sprintf("  Low:      %s", ticker.FormatPrice(t.LowPrice)))
	lines = append(lines, "")

	// Large sparkline
	if len(spark) >= 2 {
		styled, _ := renderSparkline(spark, innerW)
		lines = append(lines, "  "+styled)
		// Direction label
		diff := spark[len(spark)-1] - spark[0]
		dir := "flat"
		if diff > 0 {
			dir = fmt.Sprintf("↑ +%.2f", diff)
		} else if diff < 0 {
			dir = fmt.Sprintf("↓ %.2f", diff)
		}
		lines = append(lines, centerText(dir, innerW))
	}
	lines = append(lines, "")
	lines = append(lines, centerText("[esc] close", innerW))

	boxH := len(lines) + 2 // top + bottom border
	startRow := (m.termH - boxH) / 2
	startCol := (m.termW - boxW) / 2

	// Split base into lines and overlay
	baseLines := strings.Split(base, "\n")
	for len(baseLines) < m.termH {
		baseLines = append(baseLines, "")
	}

	// Top border
	if startRow >= 0 && startRow < len(baseLines) {
		border := "┌" + strings.Repeat("─", boxW-2) + "┐"
		baseLines[startRow] = overlayAt(baseLines[startRow], border, startCol)
	}

	// Content lines
	for i, line := range lines {
		row := startRow + 1 + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		padded := padRight(line, innerW)
		content := "│ " + padded + " │"
		baseLines[row] = overlayAt(baseLines[row], content, startCol)
	}

	// Bottom border
	botRow := startRow + boxH - 1
	if botRow >= 0 && botRow < len(baseLines) {
		border := "└" + strings.Repeat("─", boxW-2) + "┘"
		baseLines[botRow] = overlayAt(baseLines[botRow], border, startCol)
	}

	return strings.Join(baseLines, "\n")
}

// overlayAt places overlay string on top of base string at the given column.
func overlayAt(base, overlay string, col int) string {
	baseRunes := []rune(base)
	overRunes := []rune(overlay)

	// Extend base if needed
	needed := col + len(overRunes)
	for len(baseRunes) < needed {
		baseRunes = append(baseRunes, ' ')
	}

	copy(baseRunes[col:], overRunes)
	return string(baseRunes)
}

func centerText(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return s
	}
	pad := (width - len(r)) / 2
	return strings.Repeat(" ", pad) + s
}
