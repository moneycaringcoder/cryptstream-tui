package funding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const URL = "https://fapi.binance.com/fapi/v1/premiumIndex"

// Info holds funding rate data for a single symbol.
type Info struct {
	Rate            float64
	NextFundingTime time.Time
}

// rawPremiumIndex is the JSON shape from Binance.
type rawPremiumIndex struct {
	Symbol          string `json:"symbol"`
	LastFundingRate string `json:"lastFundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
}

// Fetch retrieves funding rates for all USDT-margined futures.
func Fetch() (map[string]Info, error) {
	resp, err := http.Get(URL) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("funding fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("funding fetch: status %s", resp.Status)
	}

	var raw []rawPremiumIndex
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("funding decode: %w", err)
	}

	rates := make(map[string]Info, len(raw))
	for _, r := range raw {
		rate, err := strconv.ParseFloat(r.LastFundingRate, 64)
		if err != nil {
			continue
		}
		rates[r.Symbol] = Info{
			Rate:            rate * 100, // convert to percentage
			NextFundingTime: time.UnixMilli(r.NextFundingTime),
		}
	}
	return rates, nil
}
