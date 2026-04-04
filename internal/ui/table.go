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
	name      string
	minWidth  int  // below this terminal width, hide the column
	rightAlign bool
}

// columns defines all columns in display order.
// minWidth=0 means always visible.
var columns = []column{
	{name: "#", minWidth: 0, rightAlign: true},
	{name: "SYMBOL", minWidth: 0, rightAlign: false},
	{name: "PRICE", minWidth: 0, rightAlign: true},
	{name: "CHANGE", minWidth: 0, rightAlign: true},
	{name: "TREND", minWidth: 70, rightAlign: false},
	{name: "VOLUME", minWidth: 0, rightAlign: true},
}

// proportions out of 100 for each column.
var colProps = []int{5, 14, 22, 12, 25, 16}

// visibleColumns returns the indices of columns that fit at the given terminal width.
func visibleColumns(termWidth int) []int {
	var vis []int
	for i, c := range columns {
		if termWidth >= c.minWidth {
			vis = append(vis, i)
		}
	}
	return vis
}

// colWidths computes pixel widths for visible columns, distributing space proportionally.
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

// sortIndicator returns ▼ or ▲ if this column is the active sort, else "".
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
func RenderHeader(termWidth int, sortCol SortCol, sortAsc bool) string {
	vis := visibleColumns(termWidth)
	widths := colWidths(termWidth, vis)

	var sb strings.Builder
	for j, colIdx := range vis {
		name := columns[colIdx].name + sortIndicator(colIdx, sortCol, sortAsc)
		inner := widths[j] - 2 // reserve 2 char gap
		if inner < 1 {
			inner = 1
		}
		var cell string
		if columns[colIdx].rightAlign {
			cell = padLeft(name, inner)
		} else {
			cell = padRight(name, inner)
		}
		sb.WriteString(styleHeader.Render(cell + "  "))
	}
	return sb.String()
}

// RenderSeparator returns a full-width separator line.
func RenderSeparator(termWidth int) string {
	return styleSep.Render(strings.Repeat("─", termWidth))
}

// RenderRow renders a single ticker row.
// rank is 1-based, isCursor highlights the row, sparkData is the price history.
func RenderRow(rank int, t ticker.Ticker, termWidth int, isCursor bool, sparkData []float64, starred bool) string {
	vis := visibleColumns(termWidth)
	widths := colWidths(termWidth, vis)
	flashing := time.Now().Before(t.FlashUntil) && t.Flash != ticker.FlashNeutral

	var sb strings.Builder
	for j, colIdx := range vis {
		inner := widths[j] - 2 // reserve 2 char gap
		if inner < 1 {
			inner = 1
		}

		// TREND column is pre-styled per-character, handle separately.
		if colIdx == 4 {
			styled, dw := renderSparkline(sparkData, inner)
			if dw < inner {
				styled += strings.Repeat(" ", inner-dw)
			}
			sb.WriteString(styled + "  ")
			continue
		}

		// SYMBOL column: prepend yellow star if starred, reducing inner width.
		starPrefix := ""
		cellInner := inner
		if colIdx == 1 && starred {
			starPrefix = styleStar.Render("★") + " "
			cellInner -= 2 // star + space
			if cellInner < 1 {
				cellInner = 1
			}
		}

		cell := cellValue(colIdx, rank, t, sparkData, starred)
		var padded string
		if columns[colIdx].rightAlign {
			padded = padLeft(cell, cellInner)
		} else {
			padded = padRight(cell, cellInner)
		}

		if starPrefix != "" {
			padded = starPrefix + padded
		}
		padded += "  " // gap

		if flashing {
			sb.WriteString(flashStyle(t.Flash).Render(padded))
		} else if colIdx == 3 { // CHANGE column
			sb.WriteString(changeStyle(t.PriceChangePercent).Render(padded))
		} else if isCursor {
			sb.WriteString(styleCursorRow.Render(padded))
		} else {
			sb.WriteString(padded)
		}
	}
	return sb.String()
}

// cellValue returns the display string for a column.
func cellValue(colIdx, rank int, t ticker.Ticker, sparkData []float64, starred bool) string {
	switch colIdx {
	case 0: // #
		return fmt.Sprintf("%d", rank)
	case 1: // SYMBOL
		return t.DisplaySymbol()
	case 2: // PRICE
		price := ticker.FormatPrice(t.LastPrice)
		if time.Now().Before(t.FlashUntil) && t.PriceDelta != 0 {
			delta := fmt.Sprintf("%+.2f", t.PriceDelta)
			if t.PriceDelta > 0 {
				price += " " + delta
			} else {
				price += " " + delta
			}
		}
		return price
	case 3: // CHANGE
		return formatChange(t.PriceChangePercent)
	case 4: // TREND — handled separately in RenderRow
		return ""
	case 5: // VOLUME
		return ticker.FormatVolume(t.QuoteVolume)
	}
	return ""
}

// RenderFooter renders the bottom status bar with clock and BTC price.
func RenderFooter(pairCount int, connected bool, termWidth int, btcPrice float64, filter FilterMode, searching bool, searchQuery string) string {
	dot := styleDotConnected.Render("●")
	status := "connected"
	if !connected {
		dot = styleDotReconnecting.Render("●")
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
		text := left + strings.Repeat(" ", gap) + right
		return styleFooter.Render(text)
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
	left := fmt.Sprintf(" q quit  j/k scroll  tab sort  s star  f filter  / search  •  %d pairs%s%s", pairCount, filterLabel, searchLabel)
	right := fmt.Sprintf("%s%s  %s %s ", btc, now, dot, status)

	gap := termWidth - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}
	text := left + strings.Repeat(" ", gap) + right
	return styleFooter.Render(text)
}

// renderSparkline renders a heatmap-style sparkline where each bar is colored
// green (up from previous) or red (down from previous).
// Returns the styled string and its display width (number of characters).
func renderSparkline(data []float64, maxWidth int) (string, int) {
	if len(data) < 2 {
		return "", 0
	}

	// Truncate to fit available width.
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
			sb.WriteString(styleNeutral.Render(ch))
		} else if v > data[i-1] {
			sb.WriteString(stylePositive.Render(ch))
		} else if v < data[i-1] {
			sb.WriteString(styleNegative.Render(ch))
		} else {
			sb.WriteString(styleNeutral.Render(ch))
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

func flashStyle(dir ticker.FlashDir) lipgloss.Style {
	switch dir {
	case ticker.FlashPositive:
		return styleFlashPositive
	case ticker.FlashNegative:
		return styleFlashNegative
	default:
		return lipgloss.NewStyle()
	}
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
