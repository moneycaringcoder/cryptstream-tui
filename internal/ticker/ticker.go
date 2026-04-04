package ticker

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FlashDir indicates the direction of a price change for flash coloring.
type FlashDir int8

const (
	FlashNeutral  FlashDir = 0
	FlashPositive FlashDir = 1
	FlashNegative FlashDir = -1
)

// Ticker holds the current market data for a single trading pair.
type Ticker struct {
	Symbol             string
	LastPrice          float64
	PriceChangePercent float64
	QuoteVolume        float64
	HighPrice          float64
	LowPrice           float64
	BidPrice           float64
	AskPrice           float64
	FlashUntil         time.Time
	Flash              FlashDir
	PriceDelta         float64 // change from previous update, shown briefly
}

// DisplaySymbol strips the USDT suffix for display (e.g. "BTCUSDT" → "BTC").
// Returns the symbol unchanged if it does not end in "USDT".
func (t Ticker) DisplaySymbol() string {
	return strings.TrimSuffix(t.Symbol, "USDT")
}

// FormatVolume formats a non-negative float as an abbreviated string: 4.2B, 892.4M, 12.3K.
func FormatVolume(v float64) string {
	if v >= 1e9 {
		return fmt.Sprintf("%.1fB", v/1e9)
	}
	if v >= 1e6 {
		s := fmt.Sprintf("%.1f", v/1e6)
		if s == "1000.0" {
			return fmt.Sprintf("%.1fB", v/1e9)
		}
		return s + "M"
	}
	if v >= 1e3 {
		s := fmt.Sprintf("%.1f", v/1e3)
		if s == "1000.0" {
			return fmt.Sprintf("%.1fM", v/1e6)
		}
		return s + "K"
	}
	return fmt.Sprintf("%.1f", v)
}

// FormatPrice formats a price with commas and appropriate decimal places.
func FormatPrice(p float64) string {
	if p >= 1 {
		// Format with 2 decimal places, then insert commas into the integer part.
		s := fmt.Sprintf("%.2f", p)
		dot := strings.Index(s, ".")
		intPart := s[:dot]
		fracPart := s[dot:] // includes "."
		return insertCommas(intPart) + fracPart
	}
	// Small values: use FormatFloat to avoid scientific notation
	return strconv.FormatFloat(p, 'f', -1, 64)
}

func insertCommas(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var result strings.Builder
	start := n % 3
	if start > 0 {
		result.WriteString(s[:start])
	}
	for i := start; i < n; i += 3 {
		if i > 0 {
			result.WriteByte(',')
		}
		result.WriteString(s[i : i+3])
	}
	return result.String()
}
