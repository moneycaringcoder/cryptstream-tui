package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen   = lipgloss.Color("#00ff88")
	colorRed     = lipgloss.Color("#ff4444")
	colorDim     = lipgloss.Color("#555555")
	colorSep     = lipgloss.Color("#333333")
	colorCursor  = lipgloss.Color("#1a1a2e")
	colorFooter  = lipgloss.Color("#666666")
	colorDotGreen  = lipgloss.Color("#00ff88")
	colorDotYellow = lipgloss.Color("#ffaa00")

	colorFlashGreen = lipgloss.Color("#1a3a2a")
	colorFlashRed   = lipgloss.Color("#3a1a1a")

	styleHeader = lipgloss.NewStyle().
			Foreground(colorDim).
			Bold(true)

	stylePositive = lipgloss.NewStyle().Foreground(colorGreen)
	styleNegative = lipgloss.NewStyle().Foreground(colorRed)
	styleNeutral  = lipgloss.NewStyle().Foreground(colorDim)

	styleFlashPositive = lipgloss.NewStyle().
				Background(colorFlashGreen).
				Foreground(colorGreen)

	styleFlashNegative = lipgloss.NewStyle().
				Background(colorFlashRed).
				Foreground(colorRed)

	styleCursorRow = lipgloss.NewStyle().
			Background(colorCursor)

	styleSep = lipgloss.NewStyle().Foreground(colorSep)

	styleFooter = lipgloss.NewStyle().Foreground(colorFooter)

	styleDotConnected    = lipgloss.NewStyle().Foreground(colorDotGreen)
	styleDotReconnecting = lipgloss.NewStyle().Foreground(colorDotYellow)
)
