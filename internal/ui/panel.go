package ui

const (
	topN = 10
)

// tableWidth returns the width available for the table.
// DualPane handles the sidebar split, so the table uses the full component width.
func (c *CryptoView) tableWidth() int {
	return c.width
}

// newsHeight returns the number of lines the news band occupies.
func (c *CryptoView) newsHeight() int {
	if !c.newsOn || len(c.newsArticles) == 0 {
		return 0
	}
	return 7 // top separator + 5 headline lines + bottom separator
}

// tableVisibleRows returns the number of visible rows.
func (c *CryptoView) tableVisibleRows() int {
	rows := c.height - 2 - c.newsHeight() // header + header separator; status bar is handled by App
	if rows < 0 {
		rows = 0
	}
	return rows
}
