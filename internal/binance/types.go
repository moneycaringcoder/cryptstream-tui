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

// WsRawTicker matches the fields sent by Binance's !miniTicker@arr WebSocket stream.
// Mini tickers lack bid/ask/priceChangePercent — change % is derived from open & close.
type WsRawTicker struct {
	Symbol    string `json:"s"`
	LastPrice string `json:"c"`
	OpenPrice string `json:"o"`
	HighPrice string `json:"h"`
	LowPrice  string `json:"l"`
	QuoteVol  string `json:"q"`
}

// ToTicker parses the mini-ticker fields into a ticker.Ticker.
// PriceChangePercent is calculated as (close−open)/open×100.
// BidPrice and AskPrice are left zero; the caller should preserve them from existing data.
func (r WsRawTicker) ToTicker() (ticker.Ticker, error) {
	parse := func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	}
	last, err := parse(r.LastPrice)
	if err != nil {
		return ticker.Ticker{}, err
	}
	open, err := parse(r.OpenPrice)
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
	vol, err := parse(r.QuoteVol)
	if err != nil {
		return ticker.Ticker{}, err
	}

	var changePct float64
	if open != 0 {
		changePct = (last - open) / open * 100
	}

	return ticker.Ticker{
		Symbol:             r.Symbol,
		LastPrice:          last,
		PriceChangePercent: changePct,
		QuoteVolume:        vol,
		HighPrice:          high,
		LowPrice:           low,
		// BidPrice and AskPrice not available in mini ticker
	}, nil
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
