package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	tuikit "github.com/moneycaringcoder/tuikit-go"
	"github.com/moneycaringcoder/tuikit-go/tuitest"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// testTickers returns synthetic ticker data for testing.
func testTickers() []ticker.Ticker {
	return []ticker.Ticker{
		{Symbol: "BTCUSDT", LastPrice: 67432.10, PriceChangePercent: 2.41, QuoteVolume: 4_200_000_000},
		{Symbol: "ETHUSDT", LastPrice: 3512.50, PriceChangePercent: -1.20, QuoteVolume: 2_100_000_000},
		{Symbol: "SOLUSDT", LastPrice: 142.30, PriceChangePercent: 5.67, QuoteVolume: 800_000_000},
		{Symbol: "DOGEUSDT", LastPrice: 0.1234, PriceChangePercent: -3.45, QuoteVolume: 500_000_000},
		{Symbol: "ADAUSDT", LastPrice: 0.45, PriceChangePercent: 0.12, QuoteVolume: 300_000_000},
	}
}

// testCryptoApp builds a tuikit.App wrapping a CryptoView with test data.
func testCryptoApp(t testing.TB) (*tuitest.TestModel, *CryptoView) {
	t.Helper()
	cfg := config.Default()
	initial := testTickers()
	cv := NewCryptoView(initial, &cfg)

	app := tuikit.NewApp(
		tuikit.WithLayout(&tuikit.DualPane{
			Main:         cv,
			Side:         cv.Panel,
			SideWidth:    30,
			MinMainWidth: 70,
			SideRight:    true,
			ToggleKey:    "p",
		}),
		tuikit.WithStatusBar(
			func() string { return " test left" },
			func() string { return "test right " },
		),
	)

	tm := tuitest.NewTestModel(t, app.Model(), 120, 40)

	// Mark connected so header renders correctly
	tm.SendMsg(connMsg{connected: true})

	return tm, cv
}

func TestCryptoRendersTable(t *testing.T) {
	tm, _ := testCryptoApp(t)
	s := tm.Screen()

	tuitest.AssertContains(t, s, "cryptstream")
	tuitest.AssertContains(t, s, "BTC")
	tuitest.AssertContains(t, s, "ETH")
	tuitest.AssertContains(t, s, "SOL")
}

func TestCryptoRendersHeader(t *testing.T) {
	tm, _ := testCryptoApp(t)
	s := tm.Screen()

	tuitest.AssertContains(t, s, "cryptstream")
	tuitest.AssertContains(t, s, "5 pairs")
}

func TestCryptoCursorNavigation(t *testing.T) {
	tm, cv := testCryptoApp(t)

	// Initial cursor at 0
	if cv.CursorPos() != 0 {
		t.Errorf("expected cursor at 0, got %d", cv.CursorPos())
	}

	tm.SendKey("down")
	if cv.CursorPos() != 1 {
		t.Errorf("expected cursor at 1 after down, got %d", cv.CursorPos())
	}

	tm.SendKey("down")
	tm.SendKey("down")
	if cv.CursorPos() != 3 {
		t.Errorf("expected cursor at 3, got %d", cv.CursorPos())
	}

	tm.SendKey("up")
	if cv.CursorPos() != 2 {
		t.Errorf("expected cursor at 2 after up, got %d", cv.CursorPos())
	}
}

func TestCryptoJumpToTopBottom(t *testing.T) {
	tm, cv := testCryptoApp(t)

	// Jump to bottom
	tm.SendKey("G")
	if cv.CursorPos() != cv.VisibleCount()-1 {
		t.Errorf("expected cursor at bottom (%d), got %d", cv.VisibleCount()-1, cv.CursorPos())
	}

	// Jump to top
	tm.SendKey("g")
	if cv.CursorPos() != 0 {
		t.Errorf("expected cursor at 0, got %d", cv.CursorPos())
	}
}

func TestCryptoSortCycle(t *testing.T) {
	tm, _ := testCryptoApp(t)

	// Tab cycles sort column
	tm.SendKey("tab")
	s := tm.Screen()
	tuitest.AssertNotEmpty(t, s)

	tm.SendKey("tab")
	s = tm.Screen()
	tuitest.AssertNotEmpty(t, s)
}

func TestCryptoFilterCycle(t *testing.T) {
	tm, cv := testCryptoApp(t)

	// f cycles filter: all -> gainers -> losers -> all
	tm.SendKey("f")
	if cv.FilterMode() != FilterGainers {
		t.Errorf("expected FilterGainers, got %d", cv.FilterMode())
	}

	tm.SendKey("f")
	if cv.FilterMode() != FilterLosers {
		t.Errorf("expected FilterLosers, got %d", cv.FilterMode())
	}

	tm.SendKey("f")
	if cv.FilterMode() != FilterAll {
		t.Errorf("expected FilterAll, got %d", cv.FilterMode())
	}
}

func TestCryptoSearch(t *testing.T) {
	tm, cv := testCryptoApp(t)

	// Enter search mode
	tm.SendKey("/")
	if !cv.IsSearching() {
		t.Error("should be in search mode after /")
	}

	// Type search query
	tm.Type("btc")
	if cv.SearchQuery() != "btc" {
		t.Errorf("expected search query 'btc', got '%s'", cv.SearchQuery())
	}

	// Confirm search
	tm.SendKey("enter")
	if cv.IsSearching() {
		t.Error("should exit search mode after enter")
	}

	s := tm.Screen()
	tuitest.AssertContains(t, s, "BTC")
}

func TestCryptoSearchCancel(t *testing.T) {
	tm, cv := testCryptoApp(t)

	tm.SendKey("/")
	tm.Type("xyz")
	tm.SendKey("esc")

	if cv.IsSearching() {
		t.Error("should exit search mode after esc")
	}
	if cv.SearchQuery() != "" {
		t.Error("search query should be cleared after esc")
	}
}

func TestCryptoStarToggle(t *testing.T) {
	_, cv := testCryptoApp(t)

	sym := cv.sorted[0].Symbol

	// Ensure the symbol starts unstarred (real watchlist.json may have it starred)
	if cv.Watchlist.IsStarred(sym) {
		cv.Watchlist.Toggle(sym)
	}

	// Star via key handler
	cv.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if !cv.Watchlist.IsStarred(sym) {
		t.Errorf("expected %s to be starred", sym)
	}

	// Unstar
	cv.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if cv.Watchlist.IsStarred(sym) {
		t.Errorf("expected %s to be unstarred", sym)
	}
}

func TestCryptoResize(t *testing.T) {
	tm, _ := testCryptoApp(t)

	// Small size
	tm.SendResize(80, 20)
	s := tm.Screen()
	tuitest.AssertNotEmpty(t, s)
	tuitest.AssertContains(t, s, "cryptstream")

	// Very small
	tm.SendResize(50, 15)
	s = tm.Screen()
	tuitest.AssertNotEmpty(t, s)

	// Large
	tm.SendResize(200, 50)
	s = tm.Screen()
	tuitest.AssertContains(t, s, "BTC")
}

func TestCryptoEmptyState(t *testing.T) {
	cfg := config.Default()
	cv := NewCryptoView(nil, &cfg)

	app := tuikit.NewApp(
		tuikit.WithLayout(&tuikit.DualPane{
			Main:         cv,
			Side:         cv.Panel,
			SideWidth:    30,
			MinMainWidth: 70,
			SideRight:    true,
		}),
	)

	tm := tuitest.NewTestModel(t, app.Model(), 120, 40)
	s := tm.Screen()
	tuitest.AssertNotEmpty(t, s)
}

func TestCryptoTickerUpdate(t *testing.T) {
	tm, cv := testCryptoApp(t)

	// Send a price update
	updated := ticker.Ticker{
		Symbol:             "BTCUSDT",
		LastPrice:          70000.00,
		PriceChangePercent: 3.81,
		QuoteVolume:        5_000_000_000,
	}
	tm.SendMsg(tickerMsg(updated))

	// The ticker map should be updated
	btc := cv.tickers["BTCUSDT"]
	if btc.LastPrice != 70000.00 {
		t.Errorf("expected BTC price 70000, got %f", btc.LastPrice)
	}
}

func TestCryptoSelectedTicker(t *testing.T) {
	_, cv := testCryptoApp(t)

	tk, ok := cv.SelectedTicker()
	if !ok {
		t.Fatal("expected a selected ticker")
	}
	if tk.Symbol == "" {
		t.Error("selected ticker should have a symbol")
	}
}
