package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moneycaringcoder/cryptstream-tui/internal/binance"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ui"
)

func main() {
	// 1. Initial REST fetch — populates table before WebSocket connects
	initial, err := binance.FetchInitial(binance.RestURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch initial data: %v\n", err)
		os.Exit(1)
	}

	// 2. Channel for live updates
	ch := make(chan ticker.Ticker, 256)
	done := make(chan struct{})

	// 3. Start WebSocket stream in background
	go func() {
		binance.Stream(binance.WsURL, ch, done)
	}()

	// 4. Build initial model and start Bubble Tea program
	model := ui.New(initial)
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

	// Signal shutdown after the program exits so goroutines can stop cleanly.
	close(done)
}
