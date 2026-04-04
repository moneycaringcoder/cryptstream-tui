package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen   = lipgloss.Color("#00ff88")
	colorRed     = lipgloss.Color("#ff4444")
	colorDim     = lipgloss.Color("#555555")
	colorSep     = lipgloss.Color("#333333")
	colorFlash   = lipgloss.Color("#ffffff")
	colorFooter  = lipgloss.Color("#666666")
	colorDotGreen  = lipgloss.Color("#00ff88")
	colorDotYellow = lipgloss.Color("#ffaa00")

	styleHeader = lipgloss.NewStyle().
			Foreground(colorDim).
			Bold(true)

	stylePositive = lipgloss.NewStyle().Foreground(colorGreen)
	styleNegative = lipgloss.NewStyle().Foreground(colorRed)
	styleNeutral  = lipgloss.NewStyle().Foreground(colorDim)

	styleFlashRow = lipgloss.NewStyle().
			Background(colorFlash).
			Foreground(lipgloss.Color("#000000"))

	styleSep = lipgloss.NewStyle().Foreground(colorSep)

	styleFooter = lipgloss.NewStyle().Foreground(colorFooter)

	styleDotConnected    = lipgloss.NewStyle().Foreground(colorDotGreen)
	styleDotReconnecting = lipgloss.NewStyle().Foreground(colorDotYellow)
)
