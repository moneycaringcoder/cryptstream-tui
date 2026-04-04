package binance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

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
