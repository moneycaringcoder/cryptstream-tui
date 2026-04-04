# cryptstream-tui

Live cryptocurrency ticker in your terminal — real-time Binance WebSocket stream with a Bloomberg-style dark aesthetic.

## Features

- Real-time USDT pair prices via Binance WebSocket
- Per-tick heatmap sparklines (green/red per bar)
- Watchlist with persistent starred symbols pinned to top
- Gainers/losers filter
- Live search by symbol
- Price delta flash on updates (green up, red down)
- Sortable columns (volume, price, change, symbol)
- Responsive layout — columns adapt to terminal width
- In-app settings editor with live theme preview
- Configurable colors, flash duration, sparkline length, and more
- Help screen with grouped keybind reference

## Install

```bash
go install github.com/moneycaringcoder/cryptstream-tui/cmd/cryptstream@latest
```

Or download a pre-built binary from [Releases](https://github.com/moneycaringcoder/cryptstream-tui/releases).

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
| `c` | Open settings editor |
| `?` | Toggle help screen |
| `esc` | Clear search / close overlay |
| `q` / `ctrl+c` | Quit |

## Configuration

Settings are stored in `%APPDATA%/cryptstream/config.json` (Windows) or `~/.config/cryptstream/config.json` (Linux/macOS). You can edit them directly or press `c` in the app.

Configurable options include flash duration, sparkline length, sort defaults, filter count, flash threshold, connection URLs, and a full 9-color theme.

## Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Styling
- [gorilla/websocket](https://github.com/gorilla/websocket) — WebSocket client

## License

[MIT](LICENSE)
