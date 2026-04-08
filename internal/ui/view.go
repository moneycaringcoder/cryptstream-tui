package ui

import (
	"strings"
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

	tableH := c.height
	c.table.SetSize(tableW, tableH)
	c.table.SetFocused(c.focused)
	sb.WriteString(c.table.View())

	return sb.String()
}
