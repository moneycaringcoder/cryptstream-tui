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
	rows := c.height - 4 // 2-line app header + table header + separator; status bar handled by App
	if rows < 0 {
		rows = 0
	}
	return rows
}
