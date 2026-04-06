package news

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const url = "https://api.coingecko.com/api/v3/news?page=1"

// Article holds a single news headline.
type Article struct {
	Title   string
	Source  string
	URL     string
	Time    time.Time
}

type rawResponse struct {
	Data []rawArticle `json:"data"`
}

type rawArticle struct {
	Title     string `json:"title"`
	NewsSite  string `json:"news_site"`
	URL       string `json:"url"`
	CreatedAt int64  `json:"created_at"`
}

// Fetch retrieves the latest crypto news headlines.
func Fetch(limit int) ([]Article, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("news fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("news fetch: status %s", resp.Status)
	}

	var raw rawResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("news decode: %w", err)
	}

	if len(raw.Data) > limit {
		raw.Data = raw.Data[:limit]
	}

	articles := make([]Article, len(raw.Data))
	for i, r := range raw.Data {
		articles[i] = Article{
			Title:  r.Title,
			Source: r.NewsSite,
			URL:    r.URL,
			Time:   time.Unix(r.CreatedAt, 0),
		}
	}
	return articles, nil
}
