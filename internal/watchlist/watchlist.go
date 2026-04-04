package watchlist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Watchlist tracks starred symbols and persists them to disk.
type Watchlist struct {
	mu      sync.Mutex
	symbols map[string]bool
	path    string
}

// New loads (or creates) a watchlist from the default config path.
func New() *Watchlist {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	p := filepath.Join(dir, "cryptstream", "watchlist.json")
	w := &Watchlist{
		symbols: make(map[string]bool),
		path:    p,
	}
	w.load()
	return w
}

// Toggle adds or removes a symbol. Returns true if now starred.
func (w *Watchlist) Toggle(symbol string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.symbols[symbol] {
		delete(w.symbols, symbol)
		w.save()
		return false
	}
	w.symbols[symbol] = true
	w.save()
	return true
}

// IsStarred returns true if the symbol is in the watchlist.
func (w *Watchlist) IsStarred(symbol string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.symbols[symbol]
}

func (w *Watchlist) load() {
	data, err := os.ReadFile(w.path)
	if err != nil {
		return
	}
	var list []string
	if json.Unmarshal(data, &list) == nil {
		for _, s := range list {
			w.symbols[s] = true
		}
	}
}

func (w *Watchlist) save() {
	list := make([]string, 0, len(w.symbols))
	for s := range w.symbols {
		list = append(list, s)
	}
	data, err := json.Marshal(list)
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(w.path), 0o755)
	os.WriteFile(w.path, data, 0o644)
}
