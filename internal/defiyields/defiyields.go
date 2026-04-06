package defiyields

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

const URL = "https://yields.llama.fi/pools"

// Pool holds yield data for a single DeFi pool.
type Pool struct {
	Protocol string
	Symbol   string
	Chain    string
	APY      float64
	TVL      float64
}

type rawResponse struct {
	Data []rawPool `json:"data"`
}

type rawPool struct {
	Project string  `json:"project"`
	Symbol  string  `json:"symbol"`
	Chain   string  `json:"chain"`
	APY     float64 `json:"apy"`
	TVL     float64 `json:"tvlUsd"`
}

// Fetch retrieves the top pools by APY with TVL >= minTVL.
func Fetch(limit int, minTVL float64) ([]Pool, error) {
	resp, err := http.Get(URL) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("defi yields fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("defi yields fetch: status %s", resp.Status)
	}

	var raw rawResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("defi yields decode: %w", err)
	}

	// Filter by TVL and valid APY
	var filtered []rawPool
	for _, r := range raw.Data {
		if r.TVL >= minTVL && r.APY > 0 && r.APY < 10000 {
			filtered = append(filtered, r)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].APY > filtered[j].APY
	})

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	pools := make([]Pool, len(filtered))
	for i, r := range filtered {
		pools[i] = Pool{
			Protocol: r.Project,
			Symbol:   r.Symbol,
			Chain:    r.Chain,
			APY:      r.APY,
			TVL:      r.TVL,
		}
	}
	return pools, nil
}
