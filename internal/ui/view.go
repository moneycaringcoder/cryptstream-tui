package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tuikit "github.com/moneycaringcoder/tuikit-go"
)

// View renders the full TUI frame as a string.
func (c *CryptoView) View() string {
	if c.width == 0 {
		return ""
	}

	tableW := c.tableWidth()
	if c.showDefi {
		return c.renderDefiTable(tableW)
	}
	return c.renderTable(tableW)
}

// renderTable renders the main table content using the tuikit.Table component.
func (c *CryptoView) renderTable(tableW int) string {
	s := c.styles
	var sb strings.Builder

	// Table component handles header, rows, cursor, scroll
	tableH := c.height - c.newsHeight()
	c.table.SetSize(tableW, tableH)
	c.table.SetFocused(c.focused)
	sb.WriteString(c.table.View())

	// News band sits between table and footer
	newsH := c.newsHeight()
	if newsH > 0 {
		sb.WriteString("\n")
		sb.WriteString(c.renderNewsBand(s, tableW))
		sb.WriteString("\n")
		sb.WriteString(s.Sep.Render(strings.Repeat("─", tableW)))
	}

	return sb.String()
}

// renderNewsBand renders the news ticker band.
func (c *CryptoView) renderNewsBand(s Styles, w int) string {
	articles := c.newsArticles
	if len(articles) == 0 {
		return ""
	}

	lines := make([]string, 0, 6)
	lines = append(lines, s.Sep.Render(strings.Repeat("─", w)))

	newsLines := 5
	agoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	srcStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cccccc"))
	flashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Background(lipgloss.Color("#2a2000"))

	for i := 0; i < newsLines; i++ {
		if i >= len(articles) {
			lines = append(lines, "")
			continue
		}
		a := articles[i]

		ago := tuikit.RelativeTime(a.Time, time.Now())
		agoPad := padLeft(ago, 7)
		src := a.Source
		dot := " · "

		usedPlain := 1 + 7 + 1 + len(src) + len(dot)
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
			lines = append(lines, flashStyle.Render(plainLine))
		} else {
			lines = append(lines, " "+agoStyle.Render(agoPad)+" "+srcStyle.Render(src)+dotStyle.Render(dot)+titleStyle.Render(title)+" ")
		}
	}

	return strings.Join(lines, "\n")
}

