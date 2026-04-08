package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/moneycaringcoder/cryptstream-tui/internal/funding"
	"github.com/moneycaringcoder/cryptstream-tui/internal/liquidation"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

const (
	panelWidthRight = 30
	minTermWForPanel = 100
	topN             = 10
)

// panelVisible returns whether the panel should be shown given current dimensions.
func (c *CryptoView) panelVisible() bool {
	if !c.panelOn {
		return false
	}
	return c.width >= minTermWForPanel
}

// tableWidth returns the width available for the table, accounting for panel.
func (c *CryptoView) tableWidth() int {
	if c.panelVisible() {
		return c.width - panelWidthRight - 1 // 1 for border
	}
	return c.width
}

// tableVisibleRows returns the number of visible rows.
// newsHeight returns the number of lines the news band occupies.
func (c *CryptoView) newsHeight() int {
	if !c.newsOn || len(c.newsArticles) == 0 {
		return 0
	}
	return 6 // separator + 5 headline lines
}

func (c *CryptoView) tableVisibleRows() int {
	rows := c.height - 4 - c.newsHeight() // header + separator + footer separator + footer + news
	if rows < 0 {
		rows = 0
	}
	return rows
}

// renderPanel renders the market pulse sidebar.
func (c *CryptoView) renderPanel() string {
	if !c.panelVisible() {
		return ""
	}

	s := c.styles
	ms := c.marketStats
	w := panelWidthRight
	inner := w - 2 // leave space for border char + padding

	var lines []string
	border := s.PanelBorder.Render("┃")

	// Pinned references (BTC, ETH, SOL + starred)
	for _, t := range ms.Pinned {
		fr := c.fundingRates[t.Symbol]
		lines = append(lines, border+" "+c.formatRefLine(t, inner, fr))
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

	// Market breadth bar (gainers vs losers visual)
	total := ms.GainerCount + ms.LoserCount
	if total > 0 {
		barW := inner - 1
		greenW := barW * ms.GainerCount / total
		if greenW > barW {
			greenW = barW
		}
		redW := barW - greenW
		bar := s.Positive.Render(strings.Repeat("█", greenW)) + s.Negative.Render(strings.Repeat("█", redW))
		lines = append(lines, border+" "+bar)
	}

	// Fear & Greed gauge
	if c.fearGreed.Value > 0 {
		lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))
		fg := c.fearGreed
		barW := inner - 1 // width for the gauge bar
		filled := barW * fg.Value / 100
		if filled > barW {
			filled = barW
		}
		// Color: red (0-25), yellow (25-50), yellow-green (50-75), green (75-100)
		var barColor string
		switch {
		case fg.Value < 25:
			barColor = "#ff4444"
		case fg.Value < 50:
			barColor = "#ffaa00"
		case fg.Value < 75:
			barColor = "#aaff00"
		default:
			barColor = "#00ff88"
		}
		barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor))
		dimBlock := lipgloss.NewStyle().Foreground(s.ColorDim)
		bar := barStyle.Render(strings.Repeat("█", filled)) + dimBlock.Render(strings.Repeat("░", barW-filled))
		label := fmt.Sprintf(" %s %d", fg.Label, fg.Value)
		labelStyled := barStyle.Render(label)
		lines = append(lines, border+" "+bar)
		lines = append(lines, border+labelStyled)
	}

	// Vol Spikes (collapsible, key: 1)
	if len(ms.VolSpikes) > 0 {
		lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))
		arrow := "▾"
		if c.secVolSpikes.Collapsed {
			arrow = "▸"
		}
		lines = append(lines, border+" "+s.PanelBorder.Render(arrow)+" "+s.PanelLabel.Render("VOL SPIKES"))
		if !c.secVolSpikes.Collapsed {
			for _, t := range ms.VolSpikes {
				sym := padRight(t.DisplaySymbol(), 8)
				ratio := s.VolSpike.Render(fmt.Sprintf("%.1fx", t.VolumeSpikeRatio))
				lines = append(lines, border+"  "+sym+" "+ratio)
			}
		}
	}

	// Funding rate extremes (collapsible, key: 2)
	if len(c.fundingRates) > 0 {
		type fundPair struct {
			sym  string
			rate float64
		}
		var pairs []fundPair
		for sym, info := range c.fundingRates {
			if info.Rate != 0 {
				pairs = append(pairs, fundPair{sym: strings.TrimSuffix(sym, "USDT"), rate: info.Rate})
			}
		}
		if len(pairs) > 0 {
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].rate > pairs[j].rate })
			lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))
			arrow := "▾"
			if c.secFunding.Collapsed {
				arrow = "▸"
			}
			lines = append(lines, border+" "+s.PanelBorder.Render(arrow)+" "+s.PanelLabel.Render("FUNDING RATES"))
			if !c.secFunding.Collapsed {
				show := 3
				for i := 0; i < show && i < len(pairs); i++ {
					p := pairs[i]
					sym := padRight(p.sym, 8)
					rate := s.Negative.Render(fmt.Sprintf("%+.3f%%", p.rate))
					lines = append(lines, border+"  "+sym+" "+rate)
				}
				for i := len(pairs) - 1; i >= 0 && i >= len(pairs)-show; i-- {
					p := pairs[i]
					if p.rate >= 0 {
						continue
					}
					sym := padRight(p.sym, 8)
					rate := s.Positive.Render(fmt.Sprintf("%+.3f%%", p.rate))
					lines = append(lines, border+"  "+sym+" "+rate)
				}
			}
		}
	}

	// Separator
	lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))

	// Gainers / Losers side by side
	colGap := 2
	colW := (inner - colGap) / 2 // width available per column
	lines = append(lines, border+" "+s.PanelLabel.Render(padRight("GAINERS", colW+colGap)+"LOSERS"))
	limit := 5
	for i := 0; i < limit; i++ {
		leftPad := strings.Repeat(" ", colW+colGap)
		rightStr := ""
		if i < len(ms.TopGainers) {
			g := ms.TopGainers[i]
			sym := g.DisplaySymbol()
			chg := fmt.Sprintf("%+.0f%%", g.PriceChangePercent)
			gap := colW - len(sym) - len(chg)
			if gap < 1 {
				gap = 1
			}
			leftPad = sym + strings.Repeat(" ", gap) + s.Positive.Render(chg) + strings.Repeat(" ", colGap)
		}
		if i < len(ms.TopLosers) {
			l := ms.TopLosers[i]
			sym := l.DisplaySymbol()
			chg := fmt.Sprintf("%.0f%%", l.PriceChangePercent)
			gap := colW - len(sym) - len(chg)
			if gap < 1 {
				gap = 1
			}
			rightStr = sym + strings.Repeat(" ", gap) + s.Negative.Render(chg)
		}
		lines = append(lines, border+" "+leftPad+rightStr)
	}

	// Liquidation feed (collapsible, key: 3)
	if len(c.recentLiqs) > 0 {
		lines = append(lines, border+s.PanelBorder.Render(strings.Repeat("─", w-1)))
		arrow := "▾"
		if c.secLiqs.Collapsed {
			arrow = "▸"
		}
		lines = append(lines, border+" "+s.PanelBorder.Render(arrow)+" "+s.PanelLabel.Render("LIQUIDATIONS"))
		if !c.secLiqs.Collapsed {
			liqColW := (inner - 1) / 2
			for i := 0; i < len(c.recentLiqs); i += 2 {
				left := c.formatLiqCell(s, c.recentLiqs[i], liqColW)
				right := ""
				if i+1 < len(c.recentLiqs) {
					right = c.formatLiqCell(s, c.recentLiqs[i+1], liqColW)
				}
				lines = append(lines, border+" "+left+right)
			}
		}
	}

	// Fill remaining height
	totalNeeded := c.height
	for len(lines) < totalNeeded {
		lines = append(lines, border)
	}

	if len(lines) > totalNeeded {
		lines = lines[:totalNeeded]
	}

	return strings.Join(lines, "\n")
}

// formatRefLine formats a pinned coin reference for the panel.
func (c *CryptoView) formatRefLine(t ticker.Ticker, maxWidth int, fr funding.Info) string {
	s := c.styles
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

// formatLiqCell renders a single liquidation entry padded to colW.
func (c *CryptoView) formatLiqCell(s Styles, l liquidation.Liq, colW int) string {
	sym := l.DisplaySymbol()
	sideStr := l.Side
	side := s.Negative.Render(sideStr)
	if l.Side == "SHORT" {
		side = s.Positive.Render(sideStr)
	}
	val := l.FormatNotional()
	plainLen := len(sym) + 1 + len(sideStr) + 1 + len(val)
	gap := colW - plainLen
	if gap < 0 {
		gap = 0
	}
	return sym + " " + side + " " + val + strings.Repeat(" ", gap)
}

// padLeftPlain pads a plain string to the left with spaces.
func padLeftPlain(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
