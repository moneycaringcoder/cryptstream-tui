package ui

const (
	topN = 10
)

// tableWidth returns the width available for the table.
// DualPane handles the sidebar split, so the table uses the full component width.
func (c *CryptoView) tableWidth() int {
	return c.width
}

// tableVisibleRows returns the number of visible rows.
func (c *CryptoView) tableVisibleRows() int {
	rows := c.height - 2 // header + header separator; status bar is handled by App
	if rows < 0 {
		rows = 0
	}
	return rows
}
