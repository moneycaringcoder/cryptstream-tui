package ui_test

import (
	"strings"
	"testing"
	"time"

	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ui"
)

var testStyles = ui.NewStyles(config.Default())

func TestRenderRowContainsSymbol(t *testing.T) {
	tk := ticker.Ticker{
		Symbol:             "BTCUSDT",
		LastPrice:          67432.10,
		PriceChangePercent: 2.41,
		QuoteVolume:        4_200_000_000,
		FlashUntil:         time.Time{},
	}
	row := ui.RenderRow(testStyles, 1, tk, 120, false, nil)
	if !strings.Contains(row, "BTC") {
		t.Errorf("row should contain BTC symbol, got: %s", row)
	}
}

func TestRenderRowFlash(t *testing.T) {
	base := ticker.Ticker{
		Symbol:    "BTCUSDT",
		LastPrice: 67432.10,
		Flash:     ticker.FlashPositive,
	}

	base.FlashUntil = time.Time{}
	noFlash := ui.RenderRow(testStyles, 1, base, 120, false, nil)

	base.FlashUntil = time.Now().Add(1 * time.Second)
	withFlash := ui.RenderRow(testStyles, 1, base, 120, false, nil)

	if !strings.Contains(withFlash, "BTC") {
		t.Errorf("flash row should contain BTC symbol, got: %s", withFlash)
	}
	if noFlash == withFlash {
		t.Error("flash row and non-flash row should differ (flash style not applied)")
	}
}

func TestRenderRowCursorHighlight(t *testing.T) {
	tk := ticker.Ticker{Symbol: "ETHUSDT", LastPrice: 3500}
	normal := ui.RenderRow(testStyles, 1, tk, 120, false, nil)
	cursor := ui.RenderRow(testStyles, 1, tk, 120, true, nil)
	if normal == cursor {
		t.Error("cursor row should differ from normal row")
	}
}

func TestRenderSparklineInRow(t *testing.T) {
	tk := ticker.Ticker{Symbol: "BTCUSDT", LastPrice: 67500, QuoteVolume: 1e9}
	history := []float64{67000, 67100, 67200, 67300, 67400, 67500}
	row := ui.RenderRow(testStyles, 1, tk, 120, false, history)
	if !strings.ContainsAny(row, "▁▂▃▄▅▆▇█") {
		t.Errorf("expected sparkline characters in row, got: %s", row)
	}
}
