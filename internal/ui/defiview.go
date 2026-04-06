package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// renderDefiTable renders the DeFi yields overlay in the table area.
func (m Model) renderDefiTable(tableW int) string {
	s := m.styles
	var sb strings.Builder

	// Header
	title := " DeFi Yields — Pools by APY (TVL ≥ $1M)"
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	sb.WriteString(titleStyle.Render(padRight(title, tableW)))
	sb.WriteByte('\n')
	sb.WriteString(RenderSeparator(s, tableW))
	sb.WriteByte('\n')

	// Column widths (proportional)
	colRank := 4
	colProto := tableW * 20 / 100
	colPool := tableW * 30 / 100
	colChain := tableW * 15 / 100
	colAPY := tableW * 15 / 100
	colTVL := tableW - colRank - colProto - colPool - colChain - colAPY
	if colTVL < 8 {
		colTVL = 8
	}

	// Column header
	hdr := s.Header.Render(
		padRight("#", colRank) +
			padRight("PROTOCOL", colProto) +
			padRight("POOL", colPool) +
			padRight("CHAIN", colChain) +
			padLeft("APY", colAPY) +
			padLeft("TVL", colTVL),
	)
	sb.WriteString(hdr)
	sb.WriteByte('\n')

	// Rows
	visRows := m.visibleRows - 1 // one extra header line used by title
	pools := m.defiPools

	maxScroll := len(pools) - visRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.defiScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}

	end := scroll + visRows
	if end > len(pools) {
		end = len(pools)
	}

	for i := scroll; i < end; i++ {
		p := pools[i]
		isCursor := i == m.defiCursor
		rank := fmt.Sprintf("%d", i+1)
		apy := fmt.Sprintf("%.2f%%", p.APY)
		tvl := ticker.FormatVolume(p.TVL)

		row := padRight(rank, colRank) +
			padRight(truncateRunes(p.Protocol, colProto-2), colProto) +
			padRight(truncateRunes(p.Symbol, colPool-2), colPool) +
			padRight(truncateRunes(p.Chain, colChain-2), colChain) +
			padLeft(apy, colAPY) +
			padLeft(tvl, colTVL)

		if isCursor {
			sb.WriteString(s.CursorRow.Render(row))
		} else {
			// Style APY green on non-cursor rows
			plainRow := padRight(rank, colRank) +
				padRight(truncateRunes(p.Protocol, colProto-2), colProto) +
				padRight(truncateRunes(p.Symbol, colPool-2), colPool) +
				padRight(truncateRunes(p.Chain, colChain-2), colChain)
			sb.WriteString(plainRow)
			sb.WriteString(s.Positive.Render(padLeft(apy, colAPY)))
			sb.WriteString(padLeft(tvl, colTVL))
		}
		sb.WriteByte('\n')
	}

	// Fill remaining height
	newsH := m.newsHeight()
	filled := (end - scroll) + 3 // title + sep + col header
	targetH := m.termH - 2 - newsH // minus footer sep + footer + news
	for filled < targetH {
		sb.WriteByte('\n')
		filled++
	}

	// News band
	if newsH > 0 {
		sb.WriteString(m.renderNewsBand(s, tableW))
	}

	// Footer
	sb.WriteString(RenderSeparator(s, tableW))
	sb.WriteByte('\n')

	left := " d close  j/k scroll"
	right := fmt.Sprintf(" %d pools ", len(pools))
	gap := tableW - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}
	sb.WriteString(s.Footer.Render(left + strings.Repeat(" ", gap) + right))

	return sb.String()
}
