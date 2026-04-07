package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderArticleView renders the full-screen article reader overlay.
func (m Model) renderArticleView() string {
	s := m.styles
	var sb strings.Builder

	if m.newsCursor < 0 || m.newsCursor >= len(m.newsArticles) {
		return ""
	}
	a := m.newsArticles[m.newsCursor]

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cccccc"))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5588cc")).Underline(true)

	w := m.termW
	contentW := w - 4 // 2 padding each side

	// Title bar
	titleBar := strings.Repeat("─", 3) + " ARTICLE " + strings.Repeat("─", w-12)
	sb.WriteString(s.Header.Render(titleBar))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Title (word-wrapped)
	titleLines := wordWrap(a.Title, contentW)
	for _, line := range titleLines {
		sb.WriteString("  " + titleStyle.Render(line))
		sb.WriteByte('\n')
	}
	sb.WriteByte('\n')

	// Meta line
	meta := a.Source
	if a.Author != "" {
		meta += " · " + a.Author
	}
	meta += " · " + timeAgo(a.Time) + " ago"
	sb.WriteString("  " + metaStyle.Render(meta))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Description (word-wrapped)
	rows := 5 + len(titleLines) // title bar + blank + title lines + blank + meta + blank
	if a.Description != "" {
		descLines := wordWrap(a.Description, contentW)
		for _, line := range descLines {
			sb.WriteString("  " + bodyStyle.Render(line))
			sb.WriteByte('\n')
			rows++
		}
	} else {
		sb.WriteString("  " + metaStyle.Render("(no description available)"))
		sb.WriteByte('\n')
		rows++
	}
	sb.WriteByte('\n')
	rows++

	// URL
	sb.WriteString("  " + urlStyle.Render(truncateRunes(a.URL, contentW)))
	sb.WriteByte('\n')
	rows += 2

	// Fill
	for rows < m.termH-1 {
		sb.WriteByte('\n')
		rows++
	}

	// Footer
	sb.WriteString(s.Sep.Render(strings.Repeat("─", w)))
	sb.WriteByte('\n')
	sb.WriteString(s.Footer.Render(" esc close  enter open in browser"))

	return sb.String()
}

// wordWrap splits text into lines that fit within maxWidth.
func wordWrap(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) <= maxWidth {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	lines = append(lines, current)
	return lines
}
