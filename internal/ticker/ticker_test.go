package ticker_test

import (
	"testing"

	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

func TestDisplaySymbol(t *testing.T) {
	tk := ticker.Ticker{Symbol: "BTCUSDT"}
	if tk.DisplaySymbol() != "BTC" {
		t.Errorf("expected BTC, got %s", tk.DisplaySymbol())
	}
}

func TestDisplaySymbolNonUSDT(t *testing.T) {
	tk := ticker.Ticker{Symbol: "ETHBTC"}
	if tk.DisplaySymbol() != "ETHBTC" {
		t.Errorf("expected ETHBTC, got %s", tk.DisplaySymbol())
	}
}

func TestFormatVolume(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{4_200_000_000, "4.2B"},
		{892_400_000, "892.4M"},
		{12_300, "12.3K"},
		{500, "500.0"},
		{999_950_000, "1.0B"},   // near-boundary: would show "1000.0M" without fix
		{999_950, "1.0M"},       // near-boundary: would show "1000.0K" without fix
	}
	for _, tt := range tests {
		got := ticker.FormatVolume(tt.input)
		if got != tt.expected {
			t.Errorf("FormatVolume(%v) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestFormatPrice(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{67432.10, "67,432.10"},
		{3521.44, "3,521.44"},
		{0.00045, "0.00045"},
	}
	for _, tt := range tests {
		got := ticker.FormatPrice(tt.input)
		if got != tt.expected {
			t.Errorf("FormatPrice(%v) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}
