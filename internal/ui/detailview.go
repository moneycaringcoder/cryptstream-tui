package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// renderDetailView renders the coin detail overlay for the selected ticker.
func (m Model) renderDetailView() string {
	s := m.styles
	var sb strings.Builder

	if m.cursor < 0 || m.cursor >= len(m.sorted) {
		return ""
	}
	t := m.sorted[m.cursor]
	w := m.termW

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cccccc"))

	// Title bar
	titleBar := strings.Repeat("─", 3) + " " + t.DisplaySymbol() + " " + strings.Repeat("─", w-6-len(t.DisplaySymbol()))
	sb.WriteString(s.Header.Render(titleBar))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Price header
	priceStr := ticker.FormatPrice(t.LastPrice)
	chg := formatChange(t.PriceChangePercent)
	chgStyled := changeStyle(s, t.PriceChangePercent).Render(chg)
	sb.WriteString("  " + titleStyle.Render(priceStr) + "  " + chgStyled)
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	rows := 4
	col1W := 20

	// Detail rows
	details := []struct {
		label string
		value string
		style lipgloss.Style
	}{
		{"24h Volume", ticker.FormatVolume(t.QuoteVolume), valStyle},
		{"24h High", ticker.FormatPrice(t.HighPrice), s.Positive},
		{"24h Low", ticker.FormatPrice(t.LowPrice), s.Negative},
	}

	// Bid/Ask spread
	if t.BidPrice > 0 && t.AskPrice > 0 {
		spread := t.AskPrice - t.BidPrice
		spreadPct := 0.0
		if t.AskPrice > 0 {
			spreadPct = spread / t.AskPrice * 100
		}
		details = append(details,
			struct {
				label string
				value string
				style lipgloss.Style
			}{"Bid", ticker.FormatPrice(t.BidPrice), valStyle},
			struct {
				label string
				value string
				style lipgloss.Style
			}{"Ask", ticker.FormatPrice(t.AskPrice), valStyle},
			struct {
				label string
				value string
				style lipgloss.Style
			}{"Spread", fmt.Sprintf("%s (%.3f%%)", ticker.FormatPrice(spread), spreadPct), valStyle},
		)
	}

	// Funding rate
	if fr, ok := m.fundingRates[t.Symbol]; ok && fr.Rate != 0 {
		rateStr := fmt.Sprintf("%.4f%%", fr.Rate)
		rateStyle := s.Positive
		if fr.Rate > 0 {
			rateStyle = s.Negative
		}
		details = append(details, struct {
			label string
			value string
			style lipgloss.Style
		}{"Funding Rate", rateStr, rateStyle})
	}

	// Correlation
	if corr, ok := m.correlations[t.Symbol]; ok && t.Symbol != "BTCUSDT" {
		details = append(details, struct {
			label string
			value string
			style lipgloss.Style
		}{"βBTC Correlation", fmt.Sprintf("%.3f", corr), corrStyle(s, corr)})
	}

	// Volume spike
	if t.VolumeSpiking {
		details = append(details, struct {
			label string
			value string
			style lipgloss.Style
		}{"Volume Spike", fmt.Sprintf("%.1fx average", t.VolumeSpikeRatio), s.VolSpike})
	}

	for _, d := range details {
		sb.WriteString("  " + labelStyle.Render(padRight(d.label, col1W)) + d.style.Render(d.value))
		sb.WriteByte('\n')
		rows++
	}

	// Sparkline (full width)
	sb.WriteByte('\n')
	rows++
	sb.WriteString("  " + labelStyle.Render("Price Trend"))
	sb.WriteByte('\n')
	rows++
	sparkData := m.priceHistory[t.Symbol]
	sparkW := w - 4
	if sparkW > 0 && len(sparkData) >= 2 {
		spark, dw := renderSparkline(s, sparkData, sparkW)
		if dw < sparkW {
			spark += strings.Repeat(" ", sparkW-dw)
		}
		sb.WriteString("  " + spark)
		sb.WriteByte('\n')
		rows++

		// Min/max labels
		mn, mx := sparkData[0], sparkData[0]
		for _, v := range sparkData {
			mn = math.Min(mn, v)
			mx = math.Max(mx, v)
		}
		minMax := labelStyle.Render(fmt.Sprintf("  Lo %s", ticker.FormatPrice(mn))) +
			strings.Repeat(" ", sparkW-20) +
			labelStyle.Render(fmt.Sprintf("Hi %s", ticker.FormatPrice(mx)))
		sb.WriteString(minMax)
		sb.WriteByte('\n')
		rows++
	}

	// Fill
	for rows < m.termH-1 {
		sb.WriteByte('\n')
		rows++
	}

	sb.WriteString(s.Sep.Render(strings.Repeat("─", w)))
	sb.WriteByte('\n')
	sb.WriteString(s.Footer.Render(" esc close  s star/unstar"))

	return sb.String()
}
