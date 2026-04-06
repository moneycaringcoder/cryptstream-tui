package liquidation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const URL = "wss://fstream.binance.com/ws/!forceOrder@arr"

// Liq represents a single liquidation event.
type Liq struct {
	Symbol   string
	Side     string  // "LONG" or "SHORT" (the side that got liquidated)
	Notional float64 // price * quantity in USDT
	Time     time.Time
}

// FormatNotional returns a human-readable notional value.
func (l Liq) FormatNotional() string {
	v := l.Notional
	if v >= 1e6 {
		return fmt.Sprintf("$%.1fM", v/1e6)
	}
	if v >= 1e3 {
		return fmt.Sprintf("$%.0fK", v/1e3)
	}
	return fmt.Sprintf("$%.0f", v)
}

// DisplaySymbol strips the USDT suffix.
func (l Liq) DisplaySymbol() string {
	return strings.TrimSuffix(l.Symbol, "USDT")
}

// rawForceOrder is the Binance WebSocket forceOrder event shape.
type rawForceOrder struct {
	Order struct {
		Symbol string `json:"s"`
		Side   string `json:"S"` // "BUY" means short got liquidated, "SELL" means long got liquidated
		Price  string `json:"p"`
		Qty    string `json:"q"`
		Time   int64  `json:"T"`
	} `json:"o"`
}

// Stream connects to the Binance liquidation WebSocket and sends
// large liquidation events (>= minNotional) to ch.
func Stream(ch chan<- Liq, done <-chan struct{}, minNotional float64) {
	for {
		select {
		case <-done:
			return
		default:
		}

		if err := streamOnce(ch, done, minNotional); err != nil {
			select {
			case <-done:
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func streamOnce(ch chan<- Liq, done <-chan struct{}, minNotional float64) error {
	conn, _, err := websocket.DefaultDialer.Dial(URL, nil)
	if err != nil {
		return fmt.Errorf("liq ws dial: %w", err)
	}
	defer conn.Close()

	go func() {
		<-done
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-done:
				return nil
			default:
				return fmt.Errorf("liq ws read: %w", err)
			}
		}

		var raw rawForceOrder
		if err := json.Unmarshal(msg, &raw); err != nil {
			continue
		}

		o := raw.Order
		if !strings.HasSuffix(o.Symbol, "USDT") {
			continue
		}

		price, _ := strconv.ParseFloat(o.Price, 64)
		qty, _ := strconv.ParseFloat(o.Qty, 64)
		notional := price * qty

		if notional < minNotional {
			continue
		}

		// BUY order = short got liquidated, SELL = long got liquidated
		side := "LONG"
		if o.Side == "BUY" {
			side = "SHORT"
		}

		liq := Liq{
			Symbol:   o.Symbol,
			Side:     side,
			Notional: notional,
			Time:     time.UnixMilli(o.Time),
		}

		select {
		case ch <- liq:
		case <-done:
			return nil
		}
	}
}
