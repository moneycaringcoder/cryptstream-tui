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

	// Aggregate stats (compact 2-line layout)
	line1 := s.PanelLabel.Render("Vol ") + ticker.FormatVolume(ms.TotalVolume) + "  " + s.PanelLabel.Render("Avg ") + formatChange(ms.AvgChange)
	lines = append(lines, border+" "+line1)
	line2 := s.Positive.Render(fmt.Sprintf("↑%d", ms.GainerCount)) + " " +
		s.Negative.Render(fmt.Sprintf("↓%d", ms.LoserCount)) + "  " +
		s.PanelLabel.Render("BTC ") + fmt.Sprintf("%.1f%%", ms.BtcDominance)
	lines = append(lines, border+" "+line2)

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

	// Gainers / Losers side by side
	half := (inner - 1) / 2 // split inner width, -1 for gap
	lines = append(lines, border+" "+s.PanelLabel.Render(padRight("GAINERS", half))+s.PanelLabel.Render("LOSERS"))
	limit := 5
	for i := 0; i < limit; i++ {
		left := strings.Repeat(" ", half)
		right := ""
		if i < len(ms.TopGainers) {
			g := ms.TopGainers[i]
			left = padRight(g.DisplaySymbol(), 5) + s.Positive.Render(fmt.Sprintf("%+.0f%%", g.PriceChangePercent))
			left = padRight(left, half)
		}
		if i < len(ms.TopLosers) {
			l := ms.TopLosers[i]
			right = padRight(l.DisplaySymbol(), 5) + s.Negative.Render(fmt.Sprintf("%.0f%%", l.PriceChangePercent))
		}
		lines = append(lines, border+" "+left+right)
	}

	// Liquidation feed (if any)
	if len(m.recentLiqs) > 0 {
		lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))
		lines = append(lines, border+" "+s.PanelLabel.Render("LIQUIDATIONS"))
		for _, l := range m.recentLiqs {
			sym := padRight(l.DisplaySymbol(), 6)
			side := s.Negative.Render(l.Side)
			if l.Side == "SHORT" {
				side = s.Positive.Render(l.Side)
			}
			val := l.FormatNotional()
			lines = append(lines, border+"  "+sym+side+" "+val)
		}
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
