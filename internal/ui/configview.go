package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
)

// configField describes one editable setting.
type configField struct {
	group string
	label string
	hint  string // one-line description shown when cursor is on this field
	get   func(config.Config) string
	set   func(*config.Config, string) error
}

var configFields = []configField{
	// Display
	{group: "Display", label: "Flash Duration", hint: "How long price flashes last (e.g. 300ms, 1s)", get: func(c config.Config) string { return time.Duration(c.FlashDuration).String() }, set: func(c *config.Config, v string) error {
		d, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		c.FlashDuration = config.Duration(d)
		return nil
	}},
	{group: "Display", label: "Sparkline Length", hint: "Number of price points in trend sparkline", get: func(c config.Config) string { return strconv.Itoa(c.SparklineLength) }, set: func(c *config.Config, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		c.SparklineLength = n
		return nil
	}},
	{group: "Display", label: "Default Sort", hint: "Initial sort column: volume, price, change, symbol", get: func(c config.Config) string { return c.DefaultSort }, set: func(c *config.Config, v string) error {
		v = strings.ToLower(v)
		switch v {
		case "volume", "price", "change", "symbol":
			c.DefaultSort = v
			return nil
		}
		return fmt.Errorf("must be volume, price, change, or symbol")
	}},
	{group: "Display", label: "Sort Ascending", hint: "true = ascending, false = descending", get: func(c config.Config) string { return strconv.FormatBool(c.SortAscending) }, set: func(c *config.Config, v string) error {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		c.SortAscending = b
		return nil
	}},

	// Behavior
	{group: "Behavior", label: "Default Filter", hint: "Startup filter: all, gainers, or losers", get: func(c config.Config) string { return c.DefaultFilter }, set: func(c *config.Config, v string) error {
		v = strings.ToLower(v)
		switch v {
		case "all", "gainers", "losers":
			c.DefaultFilter = v
			return nil
		}
		return fmt.Errorf("must be all, gainers, or losers")
	}},
	{group: "Behavior", label: "Filter Count", hint: "Max coins shown in gainers/losers filter", get: func(c config.Config) string { return strconv.Itoa(c.FilterCount) }, set: func(c *config.Config, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		c.FilterCount = n
		return nil
	}},
	{group: "Behavior", label: "Flash Threshold", hint: "Min price change ($) to trigger row flash", get: func(c config.Config) string { return strconv.FormatFloat(c.FlashThreshold, 'f', -1, 64) }, set: func(c *config.Config, v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		c.FlashThreshold = f
		return nil
	}},

	// Detection
	{group: "Detection", label: "Volume Window", hint: "Rolling window size for volume spike detection", get: func(c config.Config) string { return strconv.Itoa(c.VolumeWindow) }, set: func(c *config.Config, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		if n < 2 {
			return fmt.Errorf("must be at least 2")
		}
		c.VolumeWindow = n
		return nil
	}},
	{group: "Detection", label: "Spike Multiplier", hint: "Volume must be Nx avg to count as spike", get: func(c config.Config) string { return strconv.FormatFloat(c.VolumeSpikeMultiplier, 'f', -1, 64) }, set: func(c *config.Config, v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		if f <= 0 {
			return fmt.Errorf("must be positive")
		}
		c.VolumeSpikeMultiplier = f
		return nil
	}},
	{group: "Detection", label: "Liq Min Notional", hint: "Min liquidation size in USD to display", get: func(c config.Config) string { return strconv.FormatFloat(c.LiqMinNotional, 'f', 0, 64) }, set: func(c *config.Config, v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		if f < 0 {
			return fmt.Errorf("must be non-negative")
		}
		c.LiqMinNotional = f
		return nil
	}},

	// Panel
	{group: "Panel", label: "Panel Layout", hint: "Sidebar position: off or right", get: func(c config.Config) string { return c.PanelLayout }, set: func(c *config.Config, v string) error {
		v = strings.ToLower(v)
		switch v {
		case "off", "right":
			c.PanelLayout = v
			return nil
		}
		return fmt.Errorf("must be off or right")
	}},

	// Connection
	{group: "Connection", label: "WebSocket URL", hint: "Binance live ticker WebSocket endpoint", get: func(c config.Config) string { return c.WsURL }, set: func(c *config.Config, v string) error {
		c.WsURL = v
		return nil
	}},
	{group: "Connection", label: "REST URL", hint: "Binance REST endpoint for initial data", get: func(c config.Config) string { return c.RestURL }, set: func(c *config.Config, v string) error {
		c.RestURL = v
		return nil
	}},
	{group: "Connection", label: "Max Backoff", hint: "Max reconnection delay (e.g. 30s, 1m)", get: func(c config.Config) string { return time.Duration(c.MaxBackoff).String() }, set: func(c *config.Config, v string) error {
		d, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		c.MaxBackoff = config.Duration(d)
		return nil
	}},

	// Theme
	{group: "Theme", label: "Green", hint: "Hex color for positive values", get: func(c config.Config) string { return c.Theme.Green }, set: func(c *config.Config, v string) error {
		c.Theme.Green = v
		return nil
	}},
	{group: "Theme", label: "Red", hint: "Hex color for negative values", get: func(c config.Config) string { return c.Theme.Red }, set: func(c *config.Config, v string) error {
		c.Theme.Red = v
		return nil
	}},
	{group: "Theme", label: "Dim", hint: "Hex color for neutral/dim text", get: func(c config.Config) string { return c.Theme.Dim }, set: func(c *config.Config, v string) error {
		c.Theme.Dim = v
		return nil
	}},
	{group: "Theme", label: "Separator", hint: "Hex color for separator lines", get: func(c config.Config) string { return c.Theme.Separator }, set: func(c *config.Config, v string) error {
		c.Theme.Separator = v
		return nil
	}},
	{group: "Theme", label: "Cursor", hint: "Hex color for cursor row background", get: func(c config.Config) string { return c.Theme.Cursor }, set: func(c *config.Config, v string) error {
		c.Theme.Cursor = v
		return nil
	}},
	{group: "Theme", label: "Footer", hint: "Hex color for footer text", get: func(c config.Config) string { return c.Theme.Footer }, set: func(c *config.Config, v string) error {
		c.Theme.Footer = v
		return nil
	}},
	{group: "Theme", label: "Flash Green BG", hint: "Background color for positive price flash", get: func(c config.Config) string { return c.Theme.FlashGreen }, set: func(c *config.Config, v string) error {
		c.Theme.FlashGreen = v
		return nil
	}},
	{group: "Theme", label: "Flash Red BG", hint: "Background color for negative price flash", get: func(c config.Config) string { return c.Theme.FlashRed }, set: func(c *config.Config, v string) error {
		c.Theme.FlashRed = v
		return nil
	}},
	{group: "Theme", label: "Star", hint: "Hex color for star/watchlist indicator", get: func(c config.Config) string { return c.Theme.Star }, set: func(c *config.Config, v string) error {
		c.Theme.Star = v
		return nil
	}},
}

// configState holds ephemeral state for the config editor.
type configState struct {
	active      bool
	cursor      int
	editing     bool
	editBuf     string
	editErr     string
	dirty       bool // unsaved changes
	savedNotice int  // countdown ticks for "saved!" flash
}

// renderConfigView renders the full config editor screen.
func (m Model) renderConfigView() string {
	s := m.styles
	var sb strings.Builder

	// Title
	title := " SETTINGS "
	titleLine := strings.Repeat("─", 3) + title + strings.Repeat("─", m.termW-len(title)-3)
	sb.WriteString(s.Header.Render(titleLine))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	lastGroup := ""
	visibleIdx := 0
	rowsRendered := 0

	for i, f := range configFields {
		// Group header
		if f.group != lastGroup {
			if lastGroup != "" {
				sb.WriteByte('\n')
				rowsRendered++
			}
			sb.WriteString(s.Header.Render("  " + strings.ToUpper(f.group)))
			sb.WriteByte('\n')
			rowsRendered++
			lastGroup = f.group
		}

		isCursor := i == m.configUI.cursor
		label := fmt.Sprintf("  %-20s", f.label)
		val := f.get(m.cfg)

		if m.configUI.editing && isCursor {
			val = m.configUI.editBuf + "█"
		}

		// Add color preview swatch for theme fields
		colorPreview := ""
		if f.group == "Theme" && len(val) > 0 && val[0] == '#' {
			colorPreview = " " + lipgloss.NewStyle().Background(lipgloss.Color(val)).Render("   ")
		}

		line := label + "  " + val + colorPreview

		if isCursor {
			sb.WriteString(s.CursorRow.Render(padRight(line, m.termW)))
			sb.WriteByte('\n')
			rowsRendered++
			// Show hint below cursor row
			if f.hint != "" {
				hintLine := "  " + s.Neutral.Render(f.hint)
				sb.WriteString(hintLine)
				sb.WriteByte('\n')
				rowsRendered++
			}
		} else {
			sb.WriteString(padRight(line, m.termW))
			sb.WriteByte('\n')
			rowsRendered++
		}
		visibleIdx++
	}

	// Error message
	if m.configUI.editErr != "" {
		sb.WriteByte('\n')
		sb.WriteString(s.Negative.Render("  Error: " + m.configUI.editErr))
		sb.WriteByte('\n')
		rowsRendered += 2
	}

	// Fill to bottom
	filled := rowsRendered + 2
	for filled < m.termH-1 {
		sb.WriteByte('\n')
		filled++
	}

	// Footer
	sb.WriteString(s.Sep.Render(strings.Repeat("─", m.termW)))
	sb.WriteByte('\n')

	var footerLeft string
	if m.configUI.editing {
		footerLeft = " type value  •  enter save  •  esc cancel"
	} else {
		footerLeft = " j/k navigate  •  enter edit  •  r reset  •  esc close"
	}
	footerRight := ""
	if m.configUI.savedNotice > 0 {
		footerRight = "saved! "
	} else if m.configUI.dirty {
		footerRight = "unsaved changes "
	}

	gap := m.termW - len(footerLeft) - len(footerRight)
	if gap < 1 {
		gap = 1
	}
	sb.WriteString(s.Footer.Render(footerLeft + strings.Repeat(" ", gap) + footerRight))

	return sb.String()
}
