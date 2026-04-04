package binance_test

import (
	"testing"

	"github.com/moneycaringcoder/cryptstream-tui/internal/binance"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

func TestRawTickerToTicker(t *testing.T) {
	raw := binance.RawTicker{
		Symbol:             "BTCUSDT",
		LastPrice:          "67432.10",
		PriceChangePercent: "2.41",
		QuoteVolume:        "4200000000.00",
		HighPrice:          "68100.00",
		LowPrice:           "65900.00",
		BidPrice:           "67431.00",
		AskPrice:           "67433.00",
	}
	tk, err := raw.ToTicker()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tk.Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", tk.Symbol)
	}
	if tk.LastPrice != 67432.10 {
		t.Errorf("expected 67432.10, got %f", tk.LastPrice)
	}
	if tk.PriceChangePercent != 2.41 {
		t.Errorf("expected 2.41, got %f", tk.PriceChangePercent)
	}
}

func TestRawTickerToTickerInvalidPrice(t *testing.T) {
	raw := binance.RawTicker{Symbol: "BTCUSDT", LastPrice: "not-a-number"}
	_, err := raw.ToTicker()
	if err == nil {
		t.Error("expected error for invalid price, got nil")
	}
}

func TestIsUSDTPair(t *testing.T) {
	if !binance.IsUSDTPair(ticker.Ticker{Symbol: "BTCUSDT"}) {
		t.Error("BTCUSDT should be a USDT pair")
	}
	if binance.IsUSDTPair(ticker.Ticker{Symbol: "ETHBTC"}) {
		t.Error("ETHBTC should not be a USDT pair")
	}
}
