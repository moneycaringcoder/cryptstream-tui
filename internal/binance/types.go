package binance

import (
	"strconv"
	"strings"

	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// RawTicker matches the shape of Binance's 24hr ticker JSON (REST and WebSocket).
type RawTicker struct {
	Symbol             string `json:"symbol"`
	LastPrice          string `json:"lastPrice"`
	PriceChangePercent string `json:"priceChangePercent"`
	QuoteVolume        string `json:"quoteVolume"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	BidPrice           string `json:"bidPrice"`
	AskPrice           string `json:"askPrice"`
}

// WsMessage is the envelope Binance wraps stream data in for combined streams.
type WsMessage struct {
	Stream string      `json:"stream"`
	Data   []RawTicker `json:"data"`
}

// ToTicker parses string fields into a ticker.Ticker. Returns error if any
// numeric field cannot be parsed.
func (r RawTicker) ToTicker() (ticker.Ticker, error) {
	parse := func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	}

	last, err := parse(r.LastPrice)
	if err != nil {
		return ticker.Ticker{}, err
	}
	change, err := parse(r.PriceChangePercent)
	if err != nil {
		return ticker.Ticker{}, err
	}
	vol, err := parse(r.QuoteVolume)
	if err != nil {
		return ticker.Ticker{}, err
	}
	high, err := parse(r.HighPrice)
	if err != nil {
		return ticker.Ticker{}, err
	}
	low, err := parse(r.LowPrice)
	if err != nil {
		return ticker.Ticker{}, err
	}
	bid, err := parse(r.BidPrice)
	if err != nil {
		return ticker.Ticker{}, err
	}
	ask, err := parse(r.AskPrice)
	if err != nil {
		return ticker.Ticker{}, err
	}

	return ticker.Ticker{
		Symbol:             r.Symbol,
		LastPrice:          last,
		PriceChangePercent: change,
		QuoteVolume:        vol,
		HighPrice:          high,
		LowPrice:           low,
		BidPrice:           bid,
		AskPrice:           ask,
	}, nil
}

// IsUSDTPair returns true if the ticker's symbol ends in "USDT".
func IsUSDTPair(t ticker.Ticker) bool {
	return strings.HasSuffix(t.Symbol, "USDT")
}
