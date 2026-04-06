package ui

import (
	"fmt"
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
	var tableStr string
	if m.showDefi {
		tableStr = m.renderDefiTable(tableW)
	} else {
		tableStr = m.renderTable(tableW)
	}

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
		corr := m.correlations[t.Symbol]
		sb.WriteString(RenderRow(s, i+1, t, tableW, isCursor, spark, starred, liqFlash, corr))
		sb.WriteByte('\n')
	}

	filled := (limit - m.offset) + 2
	newsH := m.newsHeight()
	targetH := m.termH - 2 - newsH // minus footer separator + footer + news
	for filled < targetH {
		sb.WriteByte('\n')
		filled++
	}

	// News band (above footer)
	if newsH > 0 {
		sb.WriteString(m.renderNewsBand(s, tableW))
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

// renderNewsBand renders the scrolling news ticker band.
func (m Model) renderNewsBand(s Styles, w int) string {
	var sb strings.Builder

	articles := m.newsArticles
	if len(articles) == 0 {
		return ""
	}

	sb.WriteString(s.Sep.Render(strings.Repeat("─", w)))
	sb.WriteByte('\n')

	newsLines := 4
	// Auto-scroll: rotate start index every ~30 ticks (3 seconds)
	startIdx := (m.newsScroll / 30) % len(articles)

	agoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	srcStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cccccc"))

	for i := 0; i < newsLines; i++ {
		idx := (startIdx + i) % len(articles)
		a := articles[idx]

		// Build plain text first, then style — avoids ANSI width issues
		ago := timeAgo(a.Time)
		agoPad := padLeft(ago, 3)
		src := a.Source
		dot := " · "

		// Remaining width for title
		usedPlain := 1 + 3 + 1 + len(src) + len(dot) // " " + ago(3) + " " + source + " · "
		remaining := w - usedPlain - 1                 // 1 trailing space
		if remaining < 0 {
			remaining = 0
		}
		title := a.Title
		titleRunes := []rune(title)
		if len(titleRunes) > remaining && remaining > 1 {
			title = string(titleRunes[:remaining-1]) + "…"
		} else if len(titleRunes) > remaining {
			title = string(titleRunes[:remaining])
		}

		// Pad title to fill remaining width
		title = padRight(title, remaining)

		line := " " + agoStyle.Render(agoPad) + " " + srcStyle.Render(src) + dotStyle.Render(dot) + titleStyle.Render(title) + " "
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	return sb.String()
}

// timeAgo returns a short human-readable time difference.
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
