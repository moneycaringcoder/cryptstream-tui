# Contributing

## Setup

```bash
git clone https://github.com/moneycaringcoder/cryptstream-tui
cd cryptstream-tui
go mod download
```

## Build & test

```bash
go build ./cmd/cryptstream
go test ./...
```

## Pull requests

- Fork the repo, create a feature branch
- Keep changes focused — one feature or fix per PR
- Make sure `go build ./...` and `go test ./...` pass
- Open a PR against `main`
