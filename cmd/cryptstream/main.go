package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuikit "github.com/moneycaringcoder/tuikit-go"
	"github.com/moneycaringcoder/cryptstream-tui/internal/binance"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
	"github.com/moneycaringcoder/cryptstream-tui/internal/liquidation"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ui"
)

var version = "dev"

func main() {
	tuikit.CleanupOldBinary()
	config.EnsureExists()
	cfg := config.Load()

	initial, err := binance.FetchInitial(cfg.RestURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch initial data: %v\n", err)
		os.Exit(1)
	}

	cryptoView := ui.NewCryptoView(initial, &cfg)
	marketPanel := cryptoView.Panel

	configEditor := tuikit.NewConfigEditor(buildConfigFields(&cfg, cryptoView))

	commandBar := tuikit.NewCommandBar(buildCommands(cryptoView))

	detailOverlay := tuikit.NewDetailOverlay(tuikit.DetailOverlayOpts[ticker.Ticker]{
		Title: "Coin Detail",
		Render: func(t ticker.Ticker, w, h int, theme tuikit.Theme) string {
			return renderCoinDetail(t, cryptoView, w)
		},
	})
	cryptoView.DetailOverlay = detailOverlay

	statusLeft := func() string {
		filterLabel := ""
		switch cryptoView.FilterMode() {
		case ui.FilterGainers:
			filterLabel = "  •  ↑ GAINERS"
		case ui.FilterLosers:
			filterLabel = "  •  ↓ LOSERS"
		}
		searchLabel := ""
		if q := cryptoView.SearchQuery(); q != "" {
			searchLabel = fmt.Sprintf("  •  /%s", q)
		}
		if cryptoView.IsSearching() {
			query := cryptoView.SearchQuery()
			if query == "" {
				query = "_"
			}
			return fmt.Sprintf(" / %s", query)
		}
		return fmt.Sprintf(" ? help  / search  p panel  q quit  •  %d pairs%s%s", cryptoView.PairCount(), filterLabel, searchLabel)
	}

	statusRight := func() string {
		if cryptoView.IsSearching() {
			return "esc cancel  enter confirm "
		}
		posStr := ""
		total := cryptoView.VisibleCount()
		if total > 0 {
			posStr = fmt.Sprintf("  %d/%d", cryptoView.CursorPos()+1, total)
		}
		btcPrice := cryptoView.BtcPrice()
		btc := ""
		if btcPrice > 0 {
			btc = fmt.Sprintf("BTC %s  •  ", ticker.FormatPrice(btcPrice))
		}
		now := time.Now().Format("15:04:05")
		dot := "●"
		status := "connected"
		if !cryptoView.Connected() {
			status = "reconnecting..."
		}
		return fmt.Sprintf("%s  %s%s  %s %s ", posStr, btc, now, dot, status)
	}

	app := tuikit.NewApp(
		tuikit.WithLayout(&tuikit.DualPane{
			Main:         cryptoView,
			Side:         marketPanel,
			SideWidth:    30,
			MinMainWidth: 70,
			SideRight:    true,
			ToggleKey:    "p",
		}),
		tuikit.WithFocusCycleKey(""),
		tuikit.WithHelp(),
		tuikit.WithOverlay("Settings", "c", configEditor),
		tuikit.WithOverlay("Command", ":", commandBar),
		tuikit.WithOverlay("Detail", "", detailOverlay),
		tuikit.WithStatusBar(statusLeft, statusRight),
		tuikit.WithTickInterval(100*time.Millisecond),
		tuikit.WithMouseSupport(),
		tuikit.WithAutoUpdate(tuikit.UpdateConfig{
			Owner:      "moneycaringcoder",
			Repo:       "cryptstream-tui",
			BinaryName: "cryptstream",
			Version:    version,
			Mode:       tuikit.UpdateNotify,
		}),
	)

	// Ticker stream
	ch := make(chan ticker.Ticker, 256)
	done := make(chan struct{})

	go binance.Stream(cfg.WsURL, ch, done, cfg.MaxBackoff.Unwrap())

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ch:
				app.Send(ui.TickerMsgFrom(t))
			}
		}
	}()

	// Liquidation stream
	liqCh := make(chan liquidation.Liq, 64)
	go liquidation.Stream(liqCh, done, cfg.LiqMinNotional)

	go func() {
		for {
			select {
			case <-done:
				return
			case l := <-liqCh:
				app.Send(ui.LiqMsgFrom(l))
			}
		}
	}()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	close(done)
}

func buildConfigFields(cfg *config.Config, cv *ui.CryptoView) []tuikit.ConfigField {
	save := func() {
		config.Save(*cfg)
		cv.ReapplyConfig()
	}
	return []tuikit.ConfigField{
		// Display
		{Group: "Display", Label: "Flash Duration", Hint: "How long price flashes last (e.g. 300ms, 1s)",
			Get: func() string { return time.Duration(cfg.FlashDuration).String() },
			Set: func(v string) error {
				d, err := time.ParseDuration(v)
				if err != nil {
					return err
				}
				cfg.FlashDuration = config.Duration(d)
				save()
				return nil
			}},
		{Group: "Display", Label: "Sparkline Length", Hint: "Number of price points in trend sparkline",
			Get: func() string { return strconv.Itoa(cfg.SparklineLength) },
			Set: func(v string) error {
				n, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				cfg.SparklineLength = n
				save()
				return nil
			}},
		{Group: "Display", Label: "Default Sort", Hint: "Initial sort column: volume, price, change, symbol",
			Get: func() string { return cfg.DefaultSort },
			Set: func(v string) error {
				v = strings.ToLower(v)
				switch v {
				case "volume", "price", "change", "symbol":
					cfg.DefaultSort = v
					save()
					return nil
				}
				return fmt.Errorf("must be volume, price, change, or symbol")
			}},
		{Group: "Display", Label: "Sort Ascending", Hint: "true = ascending, false = descending",
			Get: func() string { return strconv.FormatBool(cfg.SortAscending) },
			Set: func(v string) error {
				b, err := strconv.ParseBool(v)
				if err != nil {
					return err
				}
				cfg.SortAscending = b
				save()
				return nil
			}},

		// Behavior
		{Group: "Behavior", Label: "Default Filter", Hint: "Startup filter: all, gainers, or losers",
			Get: func() string { return cfg.DefaultFilter },
			Set: func(v string) error {
				v = strings.ToLower(v)
				switch v {
				case "all", "gainers", "losers":
					cfg.DefaultFilter = v
					save()
					return nil
				}
				return fmt.Errorf("must be all, gainers, or losers")
			}},
		{Group: "Behavior", Label: "Filter Count", Hint: "Max coins shown in gainers/losers filter",
			Get: func() string { return strconv.Itoa(cfg.FilterCount) },
			Set: func(v string) error {
				n, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				cfg.FilterCount = n
				save()
				return nil
			}},
		{Group: "Behavior", Label: "Flash Threshold", Hint: "Min price change ($) to trigger row flash",
			Get: func() string { return strconv.FormatFloat(cfg.FlashThreshold, 'f', -1, 64) },
			Set: func(v string) error {
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return err
				}
				cfg.FlashThreshold = f
				save()
				return nil
			}},

		// Detection
		{Group: "Detection", Label: "Volume Window", Hint: "Rolling window size for volume spike detection",
			Get: func() string { return strconv.Itoa(cfg.VolumeWindow) },
			Set: func(v string) error {
				n, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				if n < 2 {
					return fmt.Errorf("must be at least 2")
				}
				cfg.VolumeWindow = n
				save()
				return nil
			}},
		{Group: "Detection", Label: "Spike Multiplier", Hint: "Volume must be Nx avg to count as spike",
			Get: func() string { return strconv.FormatFloat(cfg.VolumeSpikeMultiplier, 'f', -1, 64) },
			Set: func(v string) error {
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return err
				}
				if f <= 0 {
					return fmt.Errorf("must be positive")
				}
				cfg.VolumeSpikeMultiplier = f
				save()
				return nil
			}},
		{Group: "Detection", Label: "Liq Min Notional", Hint: "Min liquidation size in USD to display",
			Get: func() string { return strconv.FormatFloat(cfg.LiqMinNotional, 'f', 0, 64) },
			Set: func(v string) error {
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return err
				}
				if f < 0 {
					return fmt.Errorf("must be non-negative")
				}
				cfg.LiqMinNotional = f
				save()
				return nil
			}},

		// Panel
		{Group: "Panel", Label: "Panel Layout", Hint: "Sidebar position: off or right",
			Get: func() string { return cfg.PanelLayout },
			Set: func(v string) error {
				v = strings.ToLower(v)
				switch v {
				case "off", "right":
					cfg.PanelLayout = v
					save()
					return nil
				}
				return fmt.Errorf("must be off or right")
			}},

		// Connection
		{Group: "Connection", Label: "WebSocket URL", Hint: "Binance live ticker WebSocket endpoint",
			Get: func() string { return cfg.WsURL },
			Set: func(v string) error { cfg.WsURL = v; save(); return nil }},
		{Group: "Connection", Label: "REST URL", Hint: "Binance REST endpoint for initial data",
			Get: func() string { return cfg.RestURL },
			Set: func(v string) error { cfg.RestURL = v; save(); return nil }},
		{Group: "Connection", Label: "Max Backoff", Hint: "Max reconnection delay (e.g. 30s, 1m)",
			Get: func() string { return time.Duration(cfg.MaxBackoff).String() },
			Set: func(v string) error {
				d, err := time.ParseDuration(v)
				if err != nil {
					return err
				}
				cfg.MaxBackoff = config.Duration(d)
				save()
				return nil
			}},

		// Theme
		{Group: "Theme", Label: "Green", Hint: "Hex color for positive values",
			Get: func() string { return cfg.Theme.Green }, Set: func(v string) error { cfg.Theme.Green = v; save(); return nil }},
		{Group: "Theme", Label: "Red", Hint: "Hex color for negative values",
			Get: func() string { return cfg.Theme.Red }, Set: func(v string) error { cfg.Theme.Red = v; save(); return nil }},
		{Group: "Theme", Label: "Dim", Hint: "Hex color for neutral/dim text",
			Get: func() string { return cfg.Theme.Dim }, Set: func(v string) error { cfg.Theme.Dim = v; save(); return nil }},
		{Group: "Theme", Label: "Separator", Hint: "Hex color for separator lines",
			Get: func() string { return cfg.Theme.Separator }, Set: func(v string) error { cfg.Theme.Separator = v; save(); return nil }},
		{Group: "Theme", Label: "Cursor", Hint: "Hex color for cursor row background",
			Get: func() string { return cfg.Theme.Cursor }, Set: func(v string) error { cfg.Theme.Cursor = v; save(); return nil }},
		{Group: "Theme", Label: "Footer", Hint: "Hex color for footer text",
			Get: func() string { return cfg.Theme.Footer }, Set: func(v string) error { cfg.Theme.Footer = v; save(); return nil }},
		{Group: "Theme", Label: "Flash Green BG", Hint: "Background color for positive price flash",
			Get: func() string { return cfg.Theme.FlashGreen }, Set: func(v string) error { cfg.Theme.FlashGreen = v; save(); return nil }},
		{Group: "Theme", Label: "Flash Red BG", Hint: "Background color for negative price flash",
			Get: func() string { return cfg.Theme.FlashRed }, Set: func(v string) error { cfg.Theme.FlashRed = v; save(); return nil }},
		{Group: "Theme", Label: "Star", Hint: "Hex color for star/watchlist indicator",
			Get: func() string { return cfg.Theme.Star }, Set: func(v string) error { cfg.Theme.Star = v; save(); return nil }},
	}
}

func renderCoinDetail(t ticker.Ticker, cv *ui.CryptoView, w int) string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	posStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff88"))
	negStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444"))

	chgStyle := posStyle
	if t.PriceChangePercent < 0 {
		chgStyle = negStyle
	}

	lines := []string{
		labelStyle.Render("Symbol:     ") + valStyle.Render(t.DisplaySymbol()),
		labelStyle.Render("Price:      ") + valStyle.Render(ticker.FormatPrice(t.LastPrice)),
		labelStyle.Render("Change:     ") + chgStyle.Render(fmt.Sprintf("%+.2f%%", t.PriceChangePercent)),
		labelStyle.Render("High 24h:   ") + valStyle.Render(ticker.FormatPrice(t.HighPrice)),
		labelStyle.Render("Low 24h:    ") + valStyle.Render(ticker.FormatPrice(t.LowPrice)),
		labelStyle.Render("Volume:     ") + valStyle.Render(ticker.FormatVolume(t.QuoteVolume)),
	}

	// Funding rate if available
	fr := cv.FundingRate(t.Symbol)
	if fr.Rate != 0 {
		frStyle := posStyle
		if fr.Rate > 0 {
			frStyle = negStyle
		}
		lines = append(lines, labelStyle.Render("Funding:    ")+frStyle.Render(fmt.Sprintf("%+.4f%%", fr.Rate)))
	}

	// Volume spike
	if t.VolumeSpiking {
		lines = append(lines, labelStyle.Render("Vol Spike:  ")+posStyle.Render(fmt.Sprintf("%.1fx avg", t.VolumeSpikeRatio)))
	}

	// Sparkline
	hist := cv.PriceHistory(t.Symbol)
	if len(hist) > 1 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Price Trend:"))
		lines = append(lines, renderSparkline(hist, w-2))
	}

	return strings.Join(lines, "\n")
}

func renderSparkline(data []float64, width int) string {
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	span := max - min
	if span == 0 {
		span = 1
	}

	// Sample data to fit width
	n := len(data)
	if n > width {
		step := float64(n) / float64(width)
		sampled := make([]float64, width)
		for i := range sampled {
			sampled[i] = data[int(float64(i)*step)]
		}
		data = sampled
	}

	var sb strings.Builder
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ccff"))
	for _, v := range data {
		idx := int((v - min) / span * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		sb.WriteRune(blocks[idx])
	}
	return style.Render(sb.String())
}

func buildCommands(cv *ui.CryptoView) []tuikit.Command {
	return []tuikit.Command{
		{
			Name: "sort", Args: true, Hint: "Sort by column (volume, price, change, symbol)",
			Run: func(args string) tea.Cmd {
				col := strings.ToLower(strings.TrimSpace(args))
				cv.SetSort(col)
				return tuikit.NotifyCmd("sorted by "+col, 2*time.Second)
			},
		},
		{
			Name: "filter", Aliases: []string{"f"}, Args: true, Hint: "Filter: all, gainers, losers",
			Run: func(args string) tea.Cmd {
				mode := strings.ToLower(strings.TrimSpace(args))
				cv.SetFilter(mode)
				return tuikit.NotifyCmd("filter: "+mode, 2*time.Second)
			},
		},
		{
			Name: "go", Aliases: []string{"find", "g"}, Args: true, Hint: "Jump to symbol",
			Run: func(args string) tea.Cmd {
				sym := strings.TrimSpace(args)
				if cv.GoToSymbol(sym) {
					return tuikit.NotifyCmd("→ "+strings.ToUpper(sym), time.Second)
				}
				return tuikit.NotifyCmd(strings.ToUpper(sym)+" not found", 2*time.Second)
			},
		},
	}
}
