package ui

import (
	"strings"
)

type helpEntry struct {
	key  string
	desc string
}

var helpSections = []struct {
	title   string
	entries []helpEntry
}{
	{
		title: "NAVIGATION",
		entries: []helpEntry{
			{"j / ↓", "Move cursor down"},
			{"k / ↑", "Move cursor up"},
			{"g / Home", "Jump to top"},
			{"G / End", "Jump to bottom"},
		},
	},
	{
		title: "DATA",
		entries: []helpEntry{
			{"tab", "Cycle sort column (volume → price → change → symbol)"},
			{"shift+tab", "Cycle sort column backwards"},
			{"s", "Star / unstar selected symbol"},
			{"f", "Cycle filter (all → gainers → losers)"},
		},
	},
	{
		title: "SEARCH",
		entries: []helpEntry{
			{"/", "Open search — type to filter symbols"},
			{"enter", "Confirm search"},
			{"esc", "Cancel search / clear active filter"},
		},
	},
	{
		title: "OTHER",
		entries: []helpEntry{
			{"c", "Open settings editor"},
			{"?", "Toggle this help screen"},
			{"q / ctrl+c", "Quit"},
		},
	},
}

func (m Model) renderHelpView() string {
	s := m.styles
	var sb strings.Builder

	title := " HELP "
	titleLine := strings.Repeat("─", 3) + title + strings.Repeat("─", m.termW-len(title)-3)
	sb.WriteString(s.Header.Render(titleLine))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	rows := 2
	for i, sec := range helpSections {
		if i > 0 {
			sb.WriteByte('\n')
			rows++
		}
		sb.WriteString(s.Header.Render("  " + sec.title))
		sb.WriteByte('\n')
		rows++

		for _, e := range sec.entries {
			line := "    " + s.Star.Render(padRight(e.key, 14)) + "  " + e.desc
			sb.WriteString(line)
			sb.WriteByte('\n')
			rows++
		}
	}

	// Fill to bottom
	for rows < m.termH-1 {
		sb.WriteByte('\n')
		rows++
	}

	sb.WriteString(s.Sep.Render(strings.Repeat("─", m.termW)))
	sb.WriteByte('\n')
	sb.WriteString(s.Footer.Render(" press ? or esc to close"))

	return sb.String()
}
