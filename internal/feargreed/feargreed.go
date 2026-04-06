package feargreed

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

const URL = "https://api.alternative.me/fng/"

// Index holds the current Fear & Greed value.
type Index struct {
	Value       int    // 0-100
	Label       string // e.g. "Extreme Fear", "Greed"
}

type rawResponse struct {
	Data []struct {
		Value               string `json:"value"`
		ValueClassification string `json:"value_classification"`
	} `json:"data"`
}

// Fetch retrieves the current Fear & Greed index.
func Fetch() (Index, error) {
	resp, err := http.Get(URL) //nolint:gosec
	if err != nil {
		return Index{}, fmt.Errorf("fng fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Index{}, fmt.Errorf("fng fetch: status %s", resp.Status)
	}

	var raw rawResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Index{}, fmt.Errorf("fng decode: %w", err)
	}

	if len(raw.Data) == 0 {
		return Index{}, fmt.Errorf("fng: no data")
	}

	val, err := strconv.Atoi(raw.Data[0].Value)
	if err != nil {
		return Index{}, fmt.Errorf("fng parse: %w", err)
	}

	return Index{
		Value: val,
		Label: raw.Data[0].ValueClassification,
	}, nil
}
