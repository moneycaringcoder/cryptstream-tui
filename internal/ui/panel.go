package ui

const (
	panelWidthRight  = 30
	minTermWForPanel = 100
	topN             = 10
)

// panelVisible returns whether the panel should be shown given current dimensions.
func (c *CryptoView) panelVisible() bool {
	if !c.panelOn {
		return false
	}
	return c.width >= minTermWForPanel
}

// tableWidth returns the width available for the table, accounting for panel.
func (c *CryptoView) tableWidth() int {
	if c.panelVisible() {
		return c.width - panelWidthRight - 1 // 1 for border
	}
	return c.width
}

// newsHeight returns the number of lines the news band occupies.
func (c *CryptoView) newsHeight() int {
	if !c.newsOn || len(c.newsArticles) == 0 {
		return 0
	}
	return 6 // separator + 5 headline lines
}

// tableVisibleRows returns the number of visible rows.
func (c *CryptoView) tableVisibleRows() int {
	rows := c.height - 4 - c.newsHeight() // header + separator + footer separator + footer + news
	if rows < 0 {
		rows = 0
	}
	return rows
}
