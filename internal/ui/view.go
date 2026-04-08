package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full TUI frame as a string.
func (c *CryptoView) View() string {
	if c.width == 0 {
		return ""
	}

	tableW := c.tableWidth()
	var tableStr string
	if c.showDefi {
		tableStr = c.renderDefiTable(tableW)
	} else {
		tableStr = c.renderTable(tableW)
	}

	if !c.panelVisible() {
		return tableStr
	}

	panelStr := c.renderPanel()
	return lipgloss.JoinHorizontal(lipgloss.Top, tableStr, panelStr)
}

// renderTable renders the main table content at the given width.
func (c *CryptoView) renderTable(tableW int) string {
	s := c.styles
	var sb strings.Builder

	sb.WriteString(RenderHeader(s, tableW, c.sortCol, c.sortAsc))
	sb.WriteByte('\n')

	sb.WriteString(RenderSeparator(s, tableW))
	sb.WriteByte('\n')

	visRows := c.visibleRows
	limit := len(c.sorted)
	if visRows > 0 && limit > c.offset+visRows {
		limit = c.offset + visRows
	}

	for i := c.offset; i < limit; i++ {
		t := c.sorted[i]
		isCursor := i == c.cursor
		spark := c.priceHistory[t.Symbol]
		starred := c.Watchlist.IsStarred(t.Symbol)
		liqFlash := time.Now().Before(c.liqFlash[t.Symbol])
		corr := c.correlations[t.Symbol]
		sb.WriteString(RenderRow(s, i+1, t, tableW, isCursor, spark, starred, liqFlash, corr))
		sb.WriteByte('\n')
	}

	filled := (limit - c.offset) + 2
	newsH := c.newsHeight()
	targetH := c.height - 2 - newsH
	for filled < targetH {
		sb.WriteByte('\n')
		filled++
	}

	if newsH > 0 {
		sb.WriteString(c.renderNewsBand(s, tableW))
	}

	sb.WriteString(RenderSeparator(s, tableW))
	sb.WriteByte('\n')

	btcPrice := 0.0
	if btc, ok := c.tickers["BTCUSDT"]; ok {
		btcPrice = btc.LastPrice
	}
	sb.WriteString(RenderFooter(s, len(c.tickers), c.connected, tableW, btcPrice, c.filterMode, c.searching, c.searchQuery, c.cursor, len(c.sorted)))

	return sb.String()
}

// renderNewsBand renders the news ticker band.
func (c *CryptoView) renderNewsBand(s Styles, w int) string {
	var sb strings.Builder

	articles := c.newsArticles
	if len(articles) == 0 {
		return ""
	}

	sb.WriteString(s.Sep.Render(strings.Repeat("─", w)))
	sb.WriteByte('\n')

	newsLines := 5
	agoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	srcStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cccccc"))
	flashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Background(lipgloss.Color("#2a2000"))

	for i := 0; i < newsLines; i++ {
		if i >= len(articles) {
			sb.WriteByte('\n')
			continue
		}
		a := articles[i]

		ago := timeAgo(a.Time)
		agoPad := padLeft(ago, 3)
		src := a.Source
		dot := " · "

		usedPlain := 1 + 3 + 1 + len(src) + len(dot)
		remaining := w - usedPlain - 1
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
		title = padRight(title, remaining)

		if i == 0 && c.newsFlash > 0 {
			plainLine := " " + agoPad + " " + src + dot + title + " "
			sb.WriteString(flashStyle.Render(plainLine))
		} else {
			sb.WriteString(" " + agoStyle.Render(agoPad) + " " + srcStyle.Render(src) + dotStyle.Render(dot) + titleStyle.Render(title) + " ")
		}
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
