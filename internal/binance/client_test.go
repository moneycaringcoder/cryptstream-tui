package binance_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/moneycaringcoder/cryptstream-tui/internal/binance"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
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

func TestStreamSendsUSDTTickers(t *testing.T) {
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Mini ticker array — matches !miniTicker@arr format.
		tickers := []binance.WsRawTicker{
			{Symbol: "BTCUSDT", LastPrice: "67000", OpenPrice: "66000",
				HighPrice: "68000", LowPrice: "65000", QuoteVol: "5000000000"},
			{Symbol: "ETHBTC", LastPrice: "0.05", OpenPrice: "0.051",
				HighPrice: "0.052", LowPrice: "0.049", QuoteVol: "100000"},
		}
		b, _ := json.Marshal(tickers)
		conn.WriteMessage(websocket.TextMessage, b)
		time.Sleep(200 * time.Millisecond) // keep connection open long enough for client to read
	}))
	defer srv.Close()

	wsAddr := "ws" + strings.TrimPrefix(srv.URL, "http")
	ch := make(chan ticker.Ticker, 10)
	done := make(chan struct{})

	go func() {
		binance.Stream(wsAddr, ch, done)
	}()

	select {
	case tk := <-ch:
		if tk.Symbol != "BTCUSDT" {
			t.Errorf("expected BTCUSDT, got %s", tk.Symbol)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for ticker")
	}
	close(done)
}
