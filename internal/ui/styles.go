package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
)

// Styles holds all computed lipgloss styles, derived from config.
type Styles struct {
	ColorGreen      lipgloss.Color
	ColorRed        lipgloss.Color
	ColorDim        lipgloss.Color
	ColorSep        lipgloss.Color
	ColorCursor     lipgloss.Color
	ColorStar       lipgloss.Color
	ColorFlashGreen lipgloss.Color
	ColorFlashRed   lipgloss.Color

	Header         lipgloss.Style
	Positive       lipgloss.Style
	Negative       lipgloss.Style
	Neutral        lipgloss.Style
	FlashPositive  lipgloss.Style
	FlashNegative  lipgloss.Style
	CursorRow      lipgloss.Style
	Sep            lipgloss.Style
	Footer         lipgloss.Style
	Star           lipgloss.Style
	DotConnected   lipgloss.Style
	DotReconnecting lipgloss.Style
	PanelBorder    lipgloss.Style
	PanelLabel     lipgloss.Style
	VolSpike       lipgloss.Style
	ColorVolSpike  lipgloss.Color
}

// NewStyles creates a Styles from the given config theme.
func NewStyles(cfg config.Config) Styles {
	t := cfg.Theme
	s := Styles{
		ColorGreen:      lipgloss.Color(t.Green),
		ColorRed:        lipgloss.Color(t.Red),
		ColorDim:        lipgloss.Color(t.Dim),
		ColorSep:        lipgloss.Color(t.Separator),
		ColorCursor:     lipgloss.Color(t.Cursor),
		ColorStar:       lipgloss.Color(t.Star),
		ColorFlashGreen: lipgloss.Color(t.FlashGreen),
		ColorFlashRed:   lipgloss.Color(t.FlashRed),
	}

	s.Header = lipgloss.NewStyle().Foreground(s.ColorDim).Bold(true)
	s.Positive = lipgloss.NewStyle().Foreground(s.ColorGreen)
	s.Negative = lipgloss.NewStyle().Foreground(s.ColorRed)
	s.Neutral = lipgloss.NewStyle().Foreground(s.ColorDim)
	s.FlashPositive = lipgloss.NewStyle().Background(s.ColorFlashGreen).Foreground(s.ColorGreen)
	s.FlashNegative = lipgloss.NewStyle().Background(s.ColorFlashRed).Foreground(s.ColorRed)
	s.CursorRow = lipgloss.NewStyle().Background(s.ColorCursor)
	s.Sep = lipgloss.NewStyle().Foreground(s.ColorSep)
	s.Footer = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Footer))
	s.Star = lipgloss.NewStyle().Foreground(s.ColorStar)
	s.DotConnected = lipgloss.NewStyle().Foreground(s.ColorGreen)
	s.DotReconnecting = lipgloss.NewStyle().Foreground(s.ColorStar)
	s.PanelBorder = lipgloss.NewStyle().Foreground(s.ColorSep)
	s.PanelLabel = lipgloss.NewStyle().Foreground(s.ColorDim).Bold(true)
	s.ColorVolSpike = lipgloss.Color("#ff8800")
	s.VolSpike = lipgloss.NewStyle().Background(lipgloss.Color("#3a2a1a")).Foreground(s.ColorVolSpike)

	return s
}
