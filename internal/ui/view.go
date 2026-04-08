package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// View renders the full TUI frame as a string.
func (c *CryptoView) View() string {
	if c.width == 0 {
		return ""
	}

	tableW := c.tableWidth()
	if c.showDefi {
		return c.renderDefiTable(tableW)
	}
	return c.renderTable(tableW)
}

// renderTable renders the main table content using the tuikit.Table component.
func (c *CryptoView) renderTable(tableW int) string {
	var sb strings.Builder

	header := c.renderHeader(tableW)
	sb.WriteString(header)
	sb.WriteByte('\n')

	tableH := c.height - 2 // reserve 2 lines for header
	c.table.SetSize(tableW, tableH)
	c.table.SetFocused(c.focused)
	sb.WriteString(c.table.View())

	return sb.String()
}

// renderHeader renders the 2-line header above the table.
func (c *CryptoView) renderHeader(w int) string {
	s := c.styles

	// Line 1: title + connection dot + BTC price right-aligned
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
	title := titleStyle.Render("cryptstream")

	dot := s.DotConnected.Render("●")
	if !c.connected {
		dot = s.DotReconnecting.Render("●")
	}

	btcStr := ""
	if btcPrice := c.BtcPrice(); btcPrice > 0 {
		btcStr = "BTC " + ticker.FormatPrice(btcPrice)
	}

	// right side: dot + btc price
	rightPlain := " ● " + btcStr
	rightStyled := dot + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render(btcStr)

	leftWidth := lipgloss.Width(title)
	rightWidth := len(rightPlain)
	gap := w - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	line1 := title + strings.Repeat(" ", gap) + rightStyled

	// Line 2: dim stats — pair count, filter, sort
	filterLabel := "all"
	switch c.filterMode {
	case FilterGainers:
		filterLabel = "gainers"
	case FilterLosers:
		filterLabel = "losers"
	}

	sortLabel := "vol"
	switch c.sortCol {
	case SortPrice:
		sortLabel = "price"
	case SortChange:
		sortLabel = "change"
	case SortSymbol:
		sortLabel = "symbol"
	case SortCorrelation:
		sortLabel = "βbtc"
	}
	if c.sortAsc {
		sortLabel += " ↑"
	} else {
		sortLabel += " ↓"
	}

	statsStr := fmt.Sprintf("%d pairs  filter:%s  sort:%s", len(c.tickers), filterLabel, sortLabel)
	dimStyle := lipgloss.NewStyle().Foreground(s.ColorDim)
	line2 := dimStyle.Render(statsStr)

	return line1 + "\n" + line2
}
