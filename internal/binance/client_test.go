package binance_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moneycaringcoder/cryptstream-tui/internal/binance"
)

func TestFetchInitial(t *testing.T) {
	tickers := []binance.RawTicker{
		{Symbol: "BTCUSDT", LastPrice: "67000.00", PriceChangePercent: "1.5",
			QuoteVolume: "5000000000", HighPrice: "68000", LowPrice: "66000",
			BidPrice: "66999", AskPrice: "67001"},
		{Symbol: "ETHBTC", LastPrice: "0.05", PriceChangePercent: "-0.1",
			QuoteVolume: "100000", HighPrice: "0.051", LowPrice: "0.049",
			BidPrice: "0.05", AskPrice: "0.051"},
		{Symbol: "ETHUSDT", LastPrice: "3500.00", PriceChangePercent: "-0.5",
			QuoteVolume: "2000000000", HighPrice: "3600", LowPrice: "3400",
			BidPrice: "3499", AskPrice: "3501"},
	}
	body, err := json.Marshal(tickers)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	result, err := binance.FetchInitial(srv.URL + "/api/v3/ticker/24hr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only USDT pairs
	if len(result) != 2 {
		t.Errorf("expected 2 USDT pairs, got %d", len(result))
	}
	// Sorted by volume descending: BTC first
	if result[0].Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT first, got %s", result[0].Symbol)
	}
}
