package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuikit "github.com/moneycaringcoder/tuikit-go"
	"github.com/moneycaringcoder/cryptstream-tui/internal/funding"
	"github.com/moneycaringcoder/cryptstream-tui/internal/liquidation"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// MarketPanel is the sidebar component showing market stats, funding,
// liquidations, and watchlist data. Implements tuikit.Component.
type MarketPanel struct {
	width   int
	height  int
	focused bool
	styles  Styles

	marketStats  MarketStats
	fundingRates map[string]funding.Info
	fearGreed    FearGreedData
	recentLiqs   []liquidation.Liq
	tickers      map[string]ticker.Ticker

	secVolSpikes *tuikit.CollapsibleSection
	secFunding   *tuikit.CollapsibleSection
	secLiqs      *tuikit.CollapsibleSection
}

// FearGreedData holds the Fear & Greed index for display.
type FearGreedData struct {
	Value int
	Label string
}

// NewMarketPanel creates a MarketPanel with default state.
func NewMarketPanel(styles Styles) *MarketPanel {
	return &MarketPanel{
		styles:       styles,
		fundingRates: make(map[string]funding.Info),
		tickers:      make(map[string]ticker.Ticker),
		secVolSpikes: tuikit.NewCollapsibleSection("VOL SPIKES"),
		secFunding:   tuikit.NewCollapsibleSection("FUNDING RATES"),
		secLiqs:      tuikit.NewCollapsibleSection("LIQUIDATIONS"),
	}
}

// Setter methods for CryptoView to push data.

// SetMarketStats updates aggregate market stats.
func (p *MarketPanel) SetMarketStats(ms MarketStats) { p.marketStats = ms }

// SetFundingRates updates funding rate data.
func (p *MarketPanel) SetFundingRates(fr map[string]funding.Info) { p.fundingRates = fr }

// SetFearGreed updates Fear & Greed index data.
func (p *MarketPanel) SetFearGreed(fg FearGreedData) { p.fearGreed = fg }

// SetRecentLiqs updates recent liquidation events.
func (p *MarketPanel) SetRecentLiqs(liqs []liquidation.Liq) { p.recentLiqs = liqs }

// SetTickers updates the ticker map for reference price lookups.
func (p *MarketPanel) SetTickers(t map[string]ticker.Ticker) { p.tickers = t }

// SetStyles updates the styles (e.g. after config change).
func (p *MarketPanel) SetStyles(s Styles) { p.styles = s }

func (p *MarketPanel) Init() tea.Cmd { return nil }

func (p *MarketPanel) Update(msg tea.Msg) (tuikit.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			p.secVolSpikes.Toggle()
			return p, tuikit.Consumed()
		case "2":
			p.secFunding.Toggle()
			return p, tuikit.Consumed()
		case "3":
			p.secLiqs.Toggle()
			return p, tuikit.Consumed()
		}
	}
	return p, nil
}

func (p *MarketPanel) View() string {
	if p.width == 0 {
		return ""
	}

	s := p.styles
	ms := p.marketStats
	w := p.width
	inner := w - 1 // 1 char padding

	var lines []string

	// Pinned references (BTC, ETH, SOL + starred)
	for _, t := range ms.Pinned {
		fr := p.fundingRates[t.Symbol]
		lines = append(lines, " "+p.formatRefLine(t, inner, fr))
	}

	// Separator
	lines = append(lines, s.PanelBorder.Render(strings.Repeat("─", w)))

	// Aggregate stats (compact 2-line layout)
	line1 := s.PanelLabel.Render("Vol ") + ticker.FormatVolume(ms.TotalVolume) + "  " + s.PanelLabel.Render("Avg ") + formatChange(ms.AvgChange)
	lines = append(lines, " "+line1)
	line2 := s.Positive.Render(fmt.Sprintf("↑%d", ms.GainerCount)) + " " +
		s.Negative.Render(fmt.Sprintf("↓%d", ms.LoserCount)) + "  " +
		s.PanelLabel.Render("BTC ") + fmt.Sprintf("%.1f%%", ms.BtcDominance)
	lines = append(lines, " "+line2)

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
		lines = append(lines, " "+bar)
	}

	// Fear & Greed gauge
	if p.fearGreed.Value > 0 {
		lines = append(lines, s.PanelBorder.Render(strings.Repeat("─", w)))
		fg := p.fearGreed
		barW := inner - 1
		filled := barW * fg.Value / 100
		if filled > barW {
			filled = barW
		}
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
		lines = append(lines, " "+bar)
		lines = append(lines, labelStyled)
	}

	// Vol Spikes (collapsible, key: 1)
	if len(ms.VolSpikes) > 0 {
		lines = append(lines, "")
		lines = append(lines, s.PanelBorder.Render(strings.Repeat("─", w)))
		arrow := "▾"
		if p.secVolSpikes.Collapsed {
			arrow = "▸"
		}
		lines = append(lines, " "+s.PanelBorder.Render(arrow)+" "+s.PanelLabel.Render("VOL SPIKES"))
		if !p.secVolSpikes.Collapsed {
			for _, t := range ms.VolSpikes {
				sym := padRight(t.DisplaySymbol(), 8)
				ratio := s.VolSpike.Render(fmt.Sprintf("%.1fx", t.VolumeSpikeRatio))
				lines = append(lines, "  "+sym+"  "+ratio)
			}
		}
	}

	// Funding rate extremes (collapsible, key: 2)
	if len(p.fundingRates) > 0 {
		type fundPair struct {
			sym  string
			rate float64
		}
		var pairs []fundPair
		for sym, info := range p.fundingRates {
			if info.Rate != 0 {
				pairs = append(pairs, fundPair{sym: strings.TrimSuffix(sym, "USDT"), rate: info.Rate})
			}
		}
		if len(pairs) > 0 {
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].rate > pairs[j].rate })
			lines = append(lines, "")
			lines = append(lines, s.PanelBorder.Render(strings.Repeat("─", w)))
			arrow := "▾"
			if p.secFunding.Collapsed {
				arrow = "▸"
			}
			lines = append(lines, " "+s.PanelBorder.Render(arrow)+" "+s.PanelLabel.Render("FUNDING RATES"))
			if !p.secFunding.Collapsed {
				show := 3
				for i := 0; i < show && i < len(pairs); i++ {
					fp := pairs[i]
					sym := padRight(fp.sym, 8)
					rate := s.Negative.Render(fmt.Sprintf("%+.3f%%", fp.rate))
					lines = append(lines, "  "+sym+"  "+rate)
				}
				for i := len(pairs) - 1; i >= 0 && i >= len(pairs)-show; i-- {
					fp := pairs[i]
					if fp.rate >= 0 {
						continue
					}
					sym := padRight(fp.sym, 8)
					rate := s.Positive.Render(fmt.Sprintf("%+.3f%%", fp.rate))
					lines = append(lines, "  "+sym+"  "+rate)
				}
			}
		}
	}

	// Separator
	lines = append(lines, "")
	lines = append(lines, s.PanelBorder.Render(strings.Repeat("─", w)))

	// Gainers / Losers side by side
	colGap := 2
	colW := (inner - colGap) / 2
	lines = append(lines, " "+s.PanelLabel.Render(padRight("GAINERS", colW+colGap)+"LOSERS"))
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
		lines = append(lines, " "+leftPad+rightStr)
	}

	// Liquidation feed (collapsible, key: 3)
	if len(p.recentLiqs) > 0 {
		lines = append(lines, "")
		lines = append(lines, s.PanelBorder.Render(strings.Repeat("─", w)))
		arrow := "▾"
		if p.secLiqs.Collapsed {
			arrow = "▸"
		}
		lines = append(lines, " "+s.PanelBorder.Render(arrow)+" "+s.PanelLabel.Render("LIQUIDATIONS"))
		if !p.secLiqs.Collapsed {
			liqColW := (inner - 1) / 2
			for i := 0; i < len(p.recentLiqs); i += 2 {
				left := p.formatLiqCell(s, p.recentLiqs[i], liqColW)
				right := ""
				if i+1 < len(p.recentLiqs) {
					right = p.formatLiqCell(s, p.recentLiqs[i+1], liqColW)
				}
				lines = append(lines, " "+left+right)
			}
		}
	}

	// Fill remaining height
	totalNeeded := p.height
	for len(lines) < totalNeeded {
		lines = append(lines, "")
	}

	if len(lines) > totalNeeded {
		lines = lines[:totalNeeded]
	}

	// Pad every line to fill the full panel width so JoinHorizontal aligns correctly
	for i, line := range lines {
		vis := lipgloss.Width(line)
		if vis < w {
			lines[i] = line + strings.Repeat(" ", w-vis)
		}
	}

	return strings.Join(lines, "\n")
}

// formatRefLine formats a pinned coin reference for the panel.
func (p *MarketPanel) formatRefLine(t ticker.Ticker, maxWidth int, fr funding.Info) string {
	s := p.styles
	if t.Symbol == "" {
		return ""
	}
	sym := padRight(t.DisplaySymbol(), 4)
	price := ticker.FormatPrice(t.LastPrice)
	chg := formatChange(t.PriceChangePercent)
	chgStyled := changeStyle(s, t.PriceChangePercent).Render(chg)

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
func (p *MarketPanel) formatLiqCell(s Styles, l liquidation.Liq, colW int) string {
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

func (p *MarketPanel) KeyBindings() []tuikit.KeyBind {
	return []tuikit.KeyBind{
		{Key: "1/2/3", Label: "Toggle panel sections", Group: "PANEL"},
	}
}

func (p *MarketPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *MarketPanel) Focused() bool       { return p.focused }
func (p *MarketPanel) SetFocused(f bool)    { p.focused = f }
