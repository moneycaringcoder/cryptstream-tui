# cryptstream-tui

Live cryptocurrency price stream in your terminal — powered by Binance WebSocket.

## Features

- Real-time price updates via Binance WebSocket
- Top USDT pairs sorted by 24hr volume
- Responsive — resizes to your terminal
- Flash highlight on price update
- Bloomberg-style dark aesthetic

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
| `q` / `ctrl+c` | Quit |
