package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config holds all user-configurable settings.
type Config struct {
	// Display
	FlashDuration   Duration `json:"flash_duration"`
	SparklineLength int      `json:"sparkline_length"`
	DefaultSort     string   `json:"default_sort"`
	SortAscending   bool     `json:"sort_ascending"`

	// Behavior
	DefaultFilter       string  `json:"default_filter"`
	FilterCount         int     `json:"filter_count"`
	FlashThreshold      float64 `json:"flash_threshold"`

	// Panel
	PanelLayout string `json:"panel_layout"`

	// Connection
	WsURL           string   `json:"ws_url"`
	RestURL         string   `json:"rest_url"`
	MaxBackoff      Duration `json:"max_backoff"`

	// Theme
	Theme ThemeConfig `json:"theme"`
}

// ThemeConfig holds color overrides. Empty string = use default.
type ThemeConfig struct {
	Green      string `json:"green"`
	Red        string `json:"red"`
	Dim        string `json:"dim"`
	Separator  string `json:"separator"`
	Cursor     string `json:"cursor"`
	Footer     string `json:"footer"`
	FlashGreen string `json:"flash_green"`
	FlashRed   string `json:"flash_red"`
	Star       string `json:"star"`
}

// Duration wraps time.Duration for JSON marshal/unmarshal as a string like "300ms".
type Duration time.Duration

func (d Duration) Unwrap() time.Duration {
	return time.Duration(d)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		FlashDuration:   Duration(300 * time.Millisecond),
		SparklineLength: 20,
		DefaultSort:     "volume",
		SortAscending:   false,

		DefaultFilter:  "all",
		FilterCount:    20,
		FlashThreshold: 0.0001,

		PanelLayout: "off",

		WsURL:      "wss://stream.binance.com:9443/ws/!miniTicker@arr",
		RestURL:    "https://api.binance.com/api/v3/ticker/24hr",
		MaxBackoff: Duration(30 * time.Second),

		Theme: ThemeConfig{
			Green:      "#00ff88",
			Red:        "#ff4444",
			Dim:        "#555555",
			Separator:  "#333333",
			Cursor:     "#1a1a2e",
			Footer:     "#666666",
			FlashGreen: "#1a3a2a",
			FlashRed:   "#3a1a1a",
			Star:       "#ffaa00",
		},
	}
}

// Path returns the config file path.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "cryptstream", "config.json")
}

// Load reads config from disk, falling back to defaults for missing fields.
func Load() Config {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg Config) error {
	p := Path()
	os.MkdirAll(filepath.Dir(p), 0o755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// EnsureExists writes the default config file if it doesn't exist yet.
func EnsureExists() {
	p := Path()
	if _, err := os.Stat(p); err != nil {
		Save(Default())
	}
}
