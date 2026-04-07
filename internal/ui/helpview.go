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
			{"ctrl+d", "Half page down"},
			{"ctrl+u", "Half page up"},
			{"g / Home", "Jump to top"},
			{"G / End", "Jump to bottom"},
			{"mouse wheel", "Scroll up/down"},
			{"click", "Select row / open article reader"},
			{"double-click", "Star / unstar selected coin"},
		},
	},
	{
		title: "DATA",
		entries: []helpEntry{
			{"tab", "Cycle sort column (vol → price → chg → sym → βBTC)"},
			{"shift+tab", "Cycle sort column backwards"},
			{"s", "Star / unstar selected symbol"},
			{"f", "Cycle filter (all → gainers → losers)"},
			{"p", "Toggle sidebar panel"},
			{"enter", "Open coin detail / read article"},
			{"n", "Focus / unfocus news band"},
			{"N", "Toggle news ticker band"},
			{"d", "Toggle DeFi yields overlay"},
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
		title: "COMMAND BAR",
		entries: []helpEntry{
			{":", "Open command bar"},
			{":q / :quit", "Quit"},
			{":w / :save", "Save config"},
			{":wq", "Save and quit"},
			{":sort <col>", "Sort by volume/price/change/symbol/corr"},
			{":filter <m>", "Filter: all/gainers/losers"},
			{":go <sym>", "Jump to symbol"},
			{":help", "Open help"},
			{":config", "Open settings"},
			{":defi", "Toggle DeFi yields"},
			{":news", "Toggle news band"},
			{":panel", "Toggle sidebar"},
		},
	},
	{
		title: "OTHER",
		entries: []helpEntry{
			{"c", "Open settings editor (r to reset field)"},
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
