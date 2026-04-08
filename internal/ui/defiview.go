package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderDefiTable renders the DeFi yields overlay using the tuikit.Table component.
func (c *CryptoView) renderDefiTable(tableW int) string {
	s := c.styles
	var sb strings.Builder

	// Header
	title := " DeFi Yields — Pools by APY (TVL ≥ $1M)"
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true)
	sb.WriteString(titleStyle.Render(padRight(title, tableW)))
	sb.WriteByte('\n')

	// Table component handles columns, rows, cursor, scroll
	defiH := c.height - 1 // minus title line
	c.defiTable.SetSize(tableW, defiH)
	c.defiTable.SetFocused(true)
	sb.WriteString(c.defiTable.View())

	// Footer
	sb.WriteString("\n")
	left := " d close  j/k scroll"
	right := fmt.Sprintf(" %d pools ", len(c.defiPools))
	gap := tableW - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}
	sb.WriteString(s.Footer.Render(left + strings.Repeat(" ", gap) + right))

	return sb.String()
}
