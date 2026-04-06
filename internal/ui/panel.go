package ui

import (
	"fmt"
	"strings"

	"github.com/moneycaringcoder/cryptstream-tui/internal/funding"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

const (
	panelWidthRight = 30
	minTermWForPanel = 100
	topN             = 10
)

// panelVisible returns whether the panel should be shown given current dimensions.
func (m Model) panelVisible() bool {
	if !m.panelOn {
		return false
	}
	return m.termW >= minTermWForPanel
}

// tableWidth returns the width available for the table, accounting for panel.
func (m Model) tableWidth() int {
	if m.panelVisible() {
		return m.termW - panelWidthRight - 1 // 1 for border
	}
	return m.termW
}

// tableVisibleRows returns the number of visible rows.
func (m Model) tableVisibleRows() int {
	rows := m.termH - 4 // header + separator + footer separator + footer
	if rows < 0 {
		rows = 0
	}
	return rows
}

// renderPanel renders the market pulse sidebar.
func (m Model) renderPanel() string {
	if !m.panelVisible() {
		return ""
	}

	s := m.styles
	ms := m.marketStats
	w := panelWidthRight
	inner := w - 2 // leave space for border char + padding

	var lines []string
	border := s.PanelBorder.Render("┃")

	// Pinned references (BTC, ETH, SOL + starred)
	for _, t := range ms.Pinned {
		fr := m.fundingRates[t.Symbol]
		lines = append(lines, border+" "+m.formatRefLine(t, inner, fr))
	}

	// Separator
	lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))

	// Aggregate stats
	lines = append(lines, border+" "+s.PanelLabel.Render("Volume")+
		padLeftPlain(ticker.FormatVolume(ms.TotalVolume), inner-7))
	lines = append(lines, border+" "+s.PanelLabel.Render("Avg Chg")+
		padLeftPlain(formatChange(ms.AvgChange), inner-8))

	gainerStr := s.Positive.Render(fmt.Sprintf("↑ %d", ms.GainerCount))
	loserStr := s.Negative.Render(fmt.Sprintf("↓ %d", ms.LoserCount))
	lines = append(lines, border+" "+gainerStr+"  "+loserStr)

	lines = append(lines, border+" "+s.PanelLabel.Render("BTC Dom")+
		padLeftPlain(fmt.Sprintf("%.1f%%", ms.BtcDominance), inner-8))

	// Vol Spikes (only if any are spiking)
	if len(ms.VolSpikes) > 0 {
		lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))
		lines = append(lines, border+" "+s.PanelLabel.Render("VOL SPIKES"))
		for _, t := range ms.VolSpikes {
			sym := padRight(t.DisplaySymbol(), 8)
			ratio := s.VolSpike.Render(fmt.Sprintf("%.1fx", t.VolumeSpikeRatio))
			lines = append(lines, border+"  "+sym+" "+ratio)
		}
	}

	// Separator
	lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))

	// Top Gainers
	lines = append(lines, border+" "+s.PanelLabel.Render("TOP GAINERS"))
	for _, t := range ms.TopGainers {
		sym := padRight(t.DisplaySymbol(), 8)
		chg := s.Positive.Render(formatChange(t.PriceChangePercent))
		lines = append(lines, border+"  "+sym+" "+chg)
	}
	for i := len(ms.TopGainers); i < topN; i++ {
		lines = append(lines, border)
	}

	// Separator
	lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))

	// Top Losers
	lines = append(lines, border+" "+s.PanelLabel.Render("TOP LOSERS"))
	for _, t := range ms.TopLosers {
		sym := padRight(t.DisplaySymbol(), 8)
		chg := s.Negative.Render(formatChange(t.PriceChangePercent))
		lines = append(lines, border+"  "+sym+" "+chg)
	}
	for i := len(ms.TopLosers); i < topN; i++ {
		lines = append(lines, border)
	}

	// Fill remaining height to match table
	totalNeeded := m.termH
	for len(lines) < totalNeeded {
		lines = append(lines, border)
	}
	if len(lines) > totalNeeded {
		lines = lines[:totalNeeded]
	}

	return strings.Join(lines, "\n")
}

// formatRefLine formats a pinned coin reference for the panel.
func (m Model) formatRefLine(t ticker.Ticker, maxWidth int, fr funding.Info) string {
	s := m.styles
	if t.Symbol == "" {
		return ""
	}
	sym := padRight(t.DisplaySymbol(), 4)
	price := ticker.FormatPrice(t.LastPrice)
	chg := formatChange(t.PriceChangePercent)
	chgStyled := changeStyle(s, t.PriceChangePercent).Render(chg)

	// Funding rate (if available)
	fundStr := ""
	if fr.Rate != 0 {
		rateStr := fmt.Sprintf("%.3f%%", fr.Rate)
		if fr.Rate < 0 {
			fundStr = " " + s.Positive.Render(rateStr)
		} else {
			fundStr = " " + s.Negative.Render(rateStr)
		}
	}

	line := sym + " " + price + " " + chgStyled + fundStr
	return line
}

// padLeftPlain pads a plain string to the left with spaces.
func padLeftPlain(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
