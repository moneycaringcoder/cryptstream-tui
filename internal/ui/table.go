package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// colCount is the number of data columns.
const colCount = 9

// columnHeaders are displayed at the top of the table.
var columnHeaders = []string{"#", "SYMBOL", "PRICE", "CHANGE", "VOLUME", "HIGH", "LOW", "BID", "ASK"}

// ColWidths returns column widths that fit within termWidth.
// Proportions are tuned for readability at common terminal widths.
func ColWidths(termWidth int) []int {
	// proportions out of 100
	props := []int{4, 8, 12, 9, 9, 11, 11, 11, 11}
	widths := make([]int, colCount)
	for i, p := range props {
		widths[i] = termWidth * p / 100
	}
	return widths
}

// RenderHeader returns the column header row as a string.
func RenderHeader(widths []int) string {
	var sb strings.Builder
	for i, h := range columnHeaders {
		sb.WriteString(styleHeader.Render(padRight(h, widths[i])))
	}
	return sb.String()
}

// RenderSeparator returns a full-width separator line.
func RenderSeparator(termWidth int) string {
	return styleSep.Render(strings.Repeat("─", termWidth))
}

// RenderRow renders a single ticker row. rank is 1-based.
func RenderRow(rank int, t ticker.Ticker, widths []int) string {
	flashing := time.Now().Before(t.FlashUntil)

	cells := []string{
		fmt.Sprintf("%d", rank),
		t.DisplaySymbol(),
		ticker.FormatPrice(t.LastPrice),
		formatChange(t.PriceChangePercent),
		ticker.FormatVolume(t.QuoteVolume),
		ticker.FormatPrice(t.HighPrice),
		ticker.FormatPrice(t.LowPrice),
		ticker.FormatPrice(t.BidPrice),
		ticker.FormatPrice(t.AskPrice),
	}

	var sb strings.Builder
	for i, cell := range cells {
		padded := padRight(cell, widths[i])
		if flashing {
			sb.WriteString(styleFlashRow.Render(padded))
		} else if i == 3 {
			sb.WriteString(changeStyle(t.PriceChangePercent).Render(padded))
		} else {
			sb.WriteString(padded)
		}
	}
	return sb.String()
}

// RenderFooter renders the bottom status bar.
func RenderFooter(pairCount int, connected bool, termWidth int) string {
	dot := styleDotConnected.Render("●")
	status := "connected"
	if !connected {
		dot = styleDotReconnecting.Render("●")
		status = "reconnecting..."
	}
	text := fmt.Sprintf("q quit  •  %d pairs  •  %s %s", pairCount, dot, status)
	return styleFooter.Render(text)
}

func formatChange(pct float64) string {
	if pct >= 0 {
		return fmt.Sprintf("+%.2f%%", pct)
	}
	return fmt.Sprintf("%.2f%%", pct)
}

func changeStyle(pct float64) lipgloss.Style {
	switch {
	case pct > 0.05:
		return stylePositive
	case pct < -0.05:
		return styleNegative
	default:
		return styleNeutral
	}
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
