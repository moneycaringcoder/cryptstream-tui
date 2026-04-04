package binance

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/websocket"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

const (
	restURL = "https://api.binance.com/api/v3/ticker/24hr"
	wsURL   = "wss://stream.binance.com/stream?streams=!ticker@arr"
)

// FetchInitial calls the Binance REST 24hr ticker endpoint, filters to USDT
// pairs, and returns them sorted by QuoteVolume descending.
// url is injectable for testing; pass restURL for production use.
func FetchInitial(url string) ([]ticker.Ticker, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("binance REST fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance REST fetch: unexpected status %s", resp.Status)
	}

	var raw []RawTicker
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("binance REST decode: %w", err)
	}

	var tickers []ticker.Ticker
	for _, r := range raw {
		t, err := r.ToTicker()
		if err != nil {
			continue // skip malformed entries
		}
		if !IsUSDTPair(t) {
			continue
		}
		tickers = append(tickers, t)
	}

	sort.Slice(tickers, func(i, j int) bool {
		return tickers[i].QuoteVolume > tickers[j].QuoteVolume
	})

	return tickers, nil
}

// Stream connects to the Binance all-market WebSocket stream and sends
// USDT ticker updates to ch. It reconnects with exponential backoff on
// disconnect. Send on done to stop.
func Stream(url string, ch chan<- ticker.Ticker, done <-chan struct{}) {
	backoff := time.Second
	for {
		select {
		case <-done:
			return
		default:
		}

		if err := streamOnce(url, ch, done); err != nil {
			select {
			case <-done:
				return
			case <-time.After(backoff):
				backoff = time.Duration(math.Min(float64(backoff*2), float64(30*time.Second)))
			}
		} else {
			return
		}
	}
}

func streamOnce(url string, ch chan<- ticker.Ticker, done <-chan struct{}) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	defer conn.Close()

	// Close the connection when done is signalled so ReadMessage unblocks immediately.
	go func() {
		<-done
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// Check if we stopped intentionally
			select {
			case <-done:
				return nil
			default:
				return fmt.Errorf("ws read: %w", err)
			}
		}

		var envelope WsMessage
		if err := json.Unmarshal(msg, &envelope); err != nil {
			continue
		}

		for _, raw := range envelope.Data {
			t, err := raw.ToTicker()
			if err != nil {
				continue
			}
			if !IsUSDTPair(t) {
				continue
			}
			select {
			case ch <- t:
			case <-done:
				return nil
			}
		}
	}
}
