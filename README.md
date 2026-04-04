# cryptstream-tui

Live cryptocurrency ticker in your terminal — real-time Binance WebSocket stream with a Bloomberg-style dark aesthetic.

## Features

- Real-time USDT pair prices via Binance WebSocket
- Per-tick heatmap sparklines (green/red per bar)
- Watchlist with persistent starred symbols pinned to top
- Gainers/losers filter
- Live search
- Price delta flash on updates (green up, red down)
- Sortable columns (volume, price, change, symbol)
- Responsive layout — columns adapt to terminal width

## Install

```bash
go install github.com/moneycaringcoder/cryptstream-tui/cmd/cryptstream@latest
```

## Run from source

```bash
git clone https://github.com/moneycaringcoder/cryptstream-tui
cd cryptstream-tui
go run ./cmd/cryptstream
```

## Keys

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll up/down |
| `g` / `G` | Jump to top/bottom |
| `tab` / `shift+tab` | Cycle sort column |
| `s` | Star/unstar symbol |
| `f` | Cycle filter (all/gainers/losers) |
| `/` | Search symbols |
| `esc` | Clear search |
| `q` / `ctrl+c` | Quit |

## Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Styling
- [gorilla/websocket](https://github.com/gorilla/websocket) — WebSocket client
