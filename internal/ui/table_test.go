package ui_test

import (
	"strings"
	"testing"
	"time"

	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ui"
)

func TestColWidths(t *testing.T) {
	widths := ui.ColWidths(120)
	total := 0
	for _, w := range widths {
		total += w
	}
	if total > 120 {
		t.Errorf("total col widths %d exceed terminal width 120", total)
	}
	if len(widths) != 9 {
		t.Errorf("expected 9 column widths, got %d", len(widths))
	}
}

func TestRenderRowContainsSymbol(t *testing.T) {
	tk := ticker.Ticker{
		Symbol:             "BTCUSDT",
		LastPrice:          67432.10,
		PriceChangePercent: 2.41,
		QuoteVolume:        4_200_000_000,
		HighPrice:          68100,
		LowPrice:           65900,
		BidPrice:           67431,
		AskPrice:           67433,
		FlashUntil:         time.Time{}, // no flash
	}
	row := ui.RenderRow(1, tk, ui.ColWidths(120))
	if !strings.Contains(row, "BTC") {
		t.Errorf("row should contain BTC symbol, got: %s", row)
	}
}

func TestRenderRowFlash(t *testing.T) {
	base := ticker.Ticker{
		Symbol:    "BTCUSDT",
		LastPrice: 67432.10,
	}

	base.FlashUntil = time.Time{} // no flash
	noFlash := ui.RenderRow(1, base, ui.ColWidths(120))

	base.FlashUntil = time.Now().Add(1 * time.Second) // active flash
	withFlash := ui.RenderRow(1, base, ui.ColWidths(120))

	if !strings.Contains(withFlash, "BTC") {
		t.Errorf("flash row should contain BTC symbol, got: %s", withFlash)
	}
	if noFlash == withFlash {
		t.Error("flash row and non-flash row should differ (flash style not applied)")
	}
}
