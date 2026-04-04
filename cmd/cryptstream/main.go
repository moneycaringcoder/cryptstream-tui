package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moneycaringcoder/cryptstream-tui/internal/binance"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ui"
)

func main() {
	// Load or create config
	config.EnsureExists()
	cfg := config.Load()

	// 1. Initial REST fetch
	initial, err := binance.FetchInitial(cfg.RestURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch initial data: %v\n", err)
		os.Exit(1)
	}

	// 2. Channel for live updates
	ch := make(chan ticker.Ticker, 256)
	done := make(chan struct{})

	// 3. Start WebSocket stream in background
	go func() {
		binance.Stream(cfg.WsURL, ch, done, cfg.MaxBackoff.Unwrap())
	}()

	// 4. Build initial model and start Bubble Tea program
	model := ui.New(initial, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// 5. Pump ticker channel into Bubble Tea via p.Send
	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ch:
				p.Send(ui.TickerMsgFrom(t))
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	close(done)
}
