package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// column defines a table column with its properties.
type column struct {
	name       string
	minWidth   int // below this terminal width, hide the column
	rightAlign bool
}

// columns defines all columns in display order.
var columns = []column{
	{name: "#", minWidth: 0, rightAlign: true},
	{name: "SYMBOL", minWidth: 0, rightAlign: false},
	{name: "PRICE", minWidth: 0, rightAlign: true},
	{name: "CHANGE", minWidth: 0, rightAlign: true},
	{name: "TREND", minWidth: 70, rightAlign: false},
	{name: "VOLUME", minWidth: 0, rightAlign: true},
}

var colProps = []int{5, 14, 22, 12, 25, 16}

func visibleColumns(termWidth int) []int {
	var vis []int
	for i, c := range columns {
		if termWidth >= c.minWidth {
			vis = append(vis, i)
		}
	}
	return vis
}

func colWidths(termWidth int, vis []int) []int {
	totalProp := 0
	for _, i := range vis {
		totalProp += colProps[i]
	}
	widths := make([]int, len(vis))
	for j, i := range vis {
		widths[j] = termWidth * colProps[i] / totalProp
	}
	return widths
}

func sortIndicator(colIdx int, sortCol SortCol, sortAsc bool) string {
	var match SortCol
	switch colIdx {
	case 2:
		match = SortPrice
	case 3:
		match = SortChange
	case 5:
		match = SortVolume
	default:
		if colIdx == 1 {
			match = SortSymbol
		} else {
			return ""
		}
	}
	if sortCol != match {
		return ""
	}
	if sortAsc {
		return " ▲"
	}
	return " ▼"
}

// RenderHeader returns the column header row.
func RenderHeader(s Styles, termWidth int, sortCol SortCol, sortAsc bool) string {
	vis := visibleColumns(termWidth)
	widths := colWidths(termWidth, vis)

	var sb strings.Builder
	for j, colIdx := range vis {
		name := columns[colIdx].name + sortIndicator(colIdx, sortCol, sortAsc)
		inner := widths[j] - 2
		if inner < 1 {
			inner = 1
		}
		var cell string
		if columns[colIdx].rightAlign {
			cell = padLeft(name, inner)
		} else {
			cell = padRight(name, inner)
		}
		sb.WriteString(s.Header.Render(cell + "  "))
	}
	return sb.String()
}

// RenderSeparator returns a full-width separator line.
func RenderSeparator(s Styles, termWidth int) string {
	return s.Sep.Render(strings.Repeat("─", termWidth))
}

// RenderRow renders a single ticker row.
func RenderRow(s Styles, rank int, t ticker.Ticker, termWidth int, isCursor bool, sparkData []float64, starred bool) string {
	vis := visibleColumns(termWidth)
	widths := colWidths(termWidth, vis)
	flashing := time.Now().Before(t.FlashUntil) && t.Flash != ticker.FlashNeutral

	var sb strings.Builder
	for j, colIdx := range vis {
		inner := widths[j] - 2
		if inner < 1 {
			inner = 1
		}

		// TREND column is pre-styled per-character.
		if colIdx == 4 {
			styled, dw := renderSparkline(s, sparkData, inner)
			if dw < inner {
				styled += strings.Repeat(" ", inner-dw)
			}
			sb.WriteString(styled + "  ")
			continue
		}

		cell := cellValue(colIdx, rank, t, sparkData)
		if colIdx == 1 && starred {
			cell = "★ " + cell
		}

		var padded string
		if columns[colIdx].rightAlign {
			padded = padLeft(cell, inner)
		} else {
			padded = padRight(cell, inner)
		}
		padded += "  "

		if flashing {
			sb.WriteString(flashStyle(s, t.Flash).Render(padded))
		} else if isCursor {
			if colIdx == 3 {
				sb.WriteString(s.CursorRow.Foreground(changeColor(s, t.PriceChangePercent)).Render(padded))
			} else if colIdx == 1 && starred {
				runes := []rune(padded)
				sb.WriteString(s.CursorRow.Foreground(s.ColorStar).Render(string(runes[:1])) + s.CursorRow.Render(string(runes[1:])))
			} else {
				sb.WriteString(s.CursorRow.Render(padded))
			}
		} else if colIdx == 3 {
			sb.WriteString(changeStyle(s, t.PriceChangePercent).Render(padded))
		} else if colIdx == 1 && starred {
			runes := []rune(padded)
			sb.WriteString(s.Star.Render(string(runes[:1])) + string(runes[1:]))
		} else {
			sb.WriteString(padded)
		}
	}
	return sb.String()
}

func cellValue(colIdx, rank int, t ticker.Ticker, sparkData []float64) string {
	switch colIdx {
	case 0:
		return fmt.Sprintf("%d", rank)
	case 1:
		return t.DisplaySymbol()
	case 2:
		price := ticker.FormatPrice(t.LastPrice)
		if time.Now().Before(t.FlashUntil) && t.PriceDelta != 0 {
			price += " " + fmt.Sprintf("%+.2f", t.PriceDelta)
		}
		return price
	case 3:
		return formatChange(t.PriceChangePercent)
	case 4:
		return ""
	case 5:
		return ticker.FormatVolume(t.QuoteVolume)
	}
	return ""
}

// RenderFooter renders the bottom status bar.
func RenderFooter(s Styles, pairCount int, connected bool, termWidth int, btcPrice float64, filter FilterMode, searching bool, searchQuery string) string {
	dot := s.DotConnected.Render("●")
	status := "connected"
	if !connected {
		dot = s.DotReconnecting.Render("●")
		status = "reconnecting..."
	}

	now := time.Now().Format("15:04:05")
	btc := ""
	if btcPrice > 0 {
		btc = fmt.Sprintf("BTC %s  •  ", ticker.FormatPrice(btcPrice))
	}

	if searching {
		query := searchQuery
		if query == "" {
			query = "_"
		}
		left := fmt.Sprintf(" / %s", query)
		right := fmt.Sprintf("esc cancel  enter confirm ")
		gap := termWidth - len(left) - len(right)
		if gap < 1 {
			gap = 1
		}
		return s.Footer.Render(left + strings.Repeat(" ", gap) + right)
	}

	filterLabel := ""
	switch filter {
	case FilterGainers:
		filterLabel = "  •  ↑ GAINERS"
	case FilterLosers:
		filterLabel = "  •  ↓ LOSERS"
	}
	searchLabel := ""
	if searchQuery != "" {
		searchLabel = fmt.Sprintf("  •  /%s", searchQuery)
	}
	left := fmt.Sprintf(" ? help  / search  p panel  q quit  •  %d pairs%s%s", pairCount, filterLabel, searchLabel)
	right := fmt.Sprintf("%s%s  %s %s ", btc, now, dot, status)

	gap := termWidth - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}
	return s.Footer.Render(left + strings.Repeat(" ", gap) + right)
}

func renderSparkline(st Styles, data []float64, maxWidth int) (string, int) {
	if len(data) < 2 {
		return "", 0
	}
	if len(data) > maxWidth {
		data = data[len(data)-maxWidth:]
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	mn, mx := data[0], data[0]
	for _, v := range data {
		mn = math.Min(mn, v)
		mx = math.Max(mx, v)
	}
	spread := mx - mn
	if spread == 0 {
		spread = 1
	}

	var sb strings.Builder
	for i, v := range data {
		idx := int((v - mn) / spread * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		ch := string(blocks[idx])

		if i == 0 {
			sb.WriteString(st.Neutral.Render(ch))
		} else if v > data[i-1] {
			sb.WriteString(st.Positive.Render(ch))
		} else if v < data[i-1] {
			sb.WriteString(st.Negative.Render(ch))
		} else {
			sb.WriteString(st.Neutral.Render(ch))
		}
	}
	return sb.String(), len(data)
}

func formatChange(pct float64) string {
	if pct >= 0 {
		return fmt.Sprintf("+%.2f%%", pct)
	}
	return fmt.Sprintf("%.2f%%", pct)
}

func flashStyle(s Styles, dir ticker.FlashDir) lipgloss.Style {
	switch dir {
	case ticker.FlashPositive:
		return s.FlashPositive
	case ticker.FlashNegative:
		return s.FlashNegative
	default:
		return lipgloss.NewStyle()
	}
}

func changeColor(s Styles, pct float64) lipgloss.Color {
	switch {
	case pct > 0.05:
		return s.ColorGreen
	case pct < -0.05:
		return s.ColorRed
	default:
		return s.ColorDim
	}
}

func changeStyle(s Styles, pct float64) lipgloss.Style {
	switch {
	case pct > 0.05:
		return s.Positive
	case pct < -0.05:
		return s.Negative
	default:
		return s.Neutral
	}
}

func displayWidth(s string) int {
	return len([]rune(s))
}

func truncateRunes(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width])
}

func padRight(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return truncateRunes(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

func padLeft(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return truncateRunes(s, width)
	}
	return strings.Repeat(" ", width-w) + s
}
