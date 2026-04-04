package ui

import (
	"strings"
)

// View renders the full TUI frame as a string.
func (m Model) View() string {
	if m.termW == 0 {
		return ""
	}

	widths := ColWidths(m.termW)
	var sb strings.Builder

	// Column headers
	sb.WriteString(RenderHeader(widths))
	sb.WriteByte('\n')

	// Separator
	sb.WriteString(RenderSeparator(m.termW))
	sb.WriteByte('\n')

	// Rows — cap to visibleRows
	limit := len(m.sorted)
	if m.visibleRows > 0 && limit > m.visibleRows {
		limit = m.visibleRows
	}
	for i, t := range m.sorted[:limit] {
		sb.WriteString(RenderRow(i+1, t, widths))
		sb.WriteByte('\n')
	}

	// Fill blank lines to push footer to bottom
	filled := limit + 2 // header row + separator row
	for filled < m.termH-1 {
		sb.WriteByte('\n')
		filled++
	}

	// Footer separator
	sb.WriteString(RenderSeparator(m.termW))
	sb.WriteByte('\n')

	// Footer
	sb.WriteString(RenderFooter(len(m.tickers), m.connected, m.termW))

	return sb.String()
}
