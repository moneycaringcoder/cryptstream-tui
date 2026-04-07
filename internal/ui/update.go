package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
	"github.com/moneycaringcoder/cryptstream-tui/internal/defiyields"
	"github.com/moneycaringcoder/cryptstream-tui/internal/feargreed"
	"github.com/moneycaringcoder/cryptstream-tui/internal/news"
	"github.com/moneycaringcoder/cryptstream-tui/internal/funding"
	"github.com/moneycaringcoder/cryptstream-tui/internal/liquidation"
	"github.com/moneycaringcoder/cryptstream-tui/internal/ticker"
)

// Update handles all incoming messages and returns the updated model + next cmd.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.termW = msg.Width
		m.termH = msg.Height
		m.visibleRows = m.tableVisibleRows()
		m.clampCursor()
		return m, nil

	case tickMsg:
		// Flash countdowns
		if m.notifyTicks > 0 {
			m.notifyTicks--
			if m.notifyTicks == 0 {
				m.notifyMsg = ""
			}
		}
		if m.newsFlash > 0 {
			m.newsFlash--
		}
		return m, tickCmd()

	case tickerMsg:
		t := ticker.Ticker(msg)
		t.FlashUntil = time.Now().Add(m.cfg.FlashDuration.Unwrap())
		thresh := m.cfg.FlashThreshold
		if prev, ok := m.tickers[t.Symbol]; ok {
			diff := t.LastPrice - prev.LastPrice
			t.PriceDelta = diff
			switch {
			case diff > thresh:
				t.Flash = ticker.FlashPositive
			case diff < -thresh:
				t.Flash = ticker.FlashNegative
			default:
				t.Flash = ticker.FlashNeutral
			}
		}

		// Track price history for sparklines.
		maxHist := m.cfg.SparklineLength
		h := m.priceHistory[t.Symbol]
		h = append(h, t.LastPrice)
		if len(h) > maxHist {
			h = h[len(h)-maxHist:]
		}
		m.priceHistory[t.Symbol] = h

		// Track volume history for spike detection.
		volWindow := m.cfg.VolumeWindow
		if volWindow < 2 {
			volWindow = 2
		}
		vh := m.volumeHistory[t.Symbol]
		vh = append(vh, t.QuoteVolume)
		if len(vh) > volWindow {
			vh = vh[len(vh)-volWindow:]
		}
		m.volumeHistory[t.Symbol] = vh

		// Compute volume spike.
		if len(vh) >= 2 {
			var sum float64
			for _, v := range vh[:len(vh)-1] {
				sum += v
			}
			avg := sum / float64(len(vh)-1)
			if avg > 0 {
				ratio := t.QuoteVolume / avg
				mult := m.cfg.VolumeSpikeMultiplier
				if mult <= 0 {
					mult = 2.0
				}
				t.VolumeSpiking = ratio >= mult
				t.VolumeSpikeRatio = ratio
			}
		}
		m.tickers[t.Symbol] = t

		m.rebuildSorted()
		return m, nil

	case fundingMsg:
		if msg != nil {
			m.fundingRates = map[string]funding.Info(msg)
			m.notifyMsg = fmt.Sprintf("↻ funding rates updated (%d)", len(msg))
			m.notifyTicks = 20
		}
		return m, fundingTickCmd()

	case fundingTickMsg:
		return m, fetchFundingCmd()

	case fngMsg:
		idx := feargreed.Index(msg)
		if idx.Value > 0 {
			m.fearGreed = idx
		}
		return m, fngTickCmd()

	case fngTickMsg:
		return m, fetchFngCmd()

	case defiMsg:
		if msg != nil {
			m.defiPools = []defiyields.Pool(msg)
		}
		return m, defiTickCmd()

	case defiTickMsg:
		return m, fetchDefiCmd()

	case newsMsg:
		if msg != nil {
			newArticles := []news.Article(msg)
			// Detect if top article changed
			if len(m.newsArticles) == 0 || (len(newArticles) > 0 && newArticles[0].Title != m.newsArticles[0].Title) {
				m.newsFlash = 20 // ~2s flash
				if len(m.newsArticles) > 0 {
					// Count how many new articles
					newCount := 0
					for _, a := range newArticles {
						if a.Title == m.newsArticles[0].Title {
							break
						}
						newCount++
					}
					if newCount > 0 {
						m.notifyMsg = fmt.Sprintf("📰 %d new articles", newCount)
						m.notifyTicks = 25
					}
				}
			}
			m.newsArticles = newArticles
			m.visibleRows = m.tableVisibleRows()
			m.clampCursor()
		}
		return m, newsTickCmd()

	case newsTickMsg:
		return m, fetchNewsCmd()

	case liqMsg:
		l := liquidation.Liq(msg)
		// Add to recent liqs (newest first, max 10)
		m.recentLiqs = append([]liquidation.Liq{l}, m.recentLiqs...)
		if len(m.recentLiqs) > 10 {
			m.recentLiqs = m.recentLiqs[:10]
		}
		// Flash the coin's symbol in the table for 2 seconds
		m.liqFlash[l.Symbol] = time.Now().Add(2 * time.Second)
		return m, nil

	case connMsg:
		m.connected = msg.connected
		if msg.connected {
			m.notifyMsg = "✓ connected to Binance"
			m.notifyTicks = 30
		} else {
			m.notifyMsg = "⚠ reconnecting..."
			m.notifyTicks = 50
		}
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		// Help screen
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q":
				m.showHelp = false
			}
			return m, nil
		}

		// Article reader overlay
		if m.showArticle {
			switch msg.String() {
			case "esc", "q":
				m.showArticle = false
			case "enter":
				if m.newsCursor >= 0 && m.newsCursor < len(m.newsArticles) {
					url := m.newsArticles[m.newsCursor].URL
					if url != "" {
						return m, openURL(url)
					}
				}
			}
			return m, nil
		}

		// Coin detail overlay
		if m.showDetail {
			switch msg.String() {
			case "esc", "q":
				m.showDetail = false
			case "s":
				if m.cursor >= 0 && m.cursor < len(m.sorted) {
					sym := m.sorted[m.cursor].Symbol
					wasStarred := m.watchlist.IsStarred(sym)
					m.watchlist.Toggle(sym)
					if wasStarred {
						m.notifyMsg = "★ unstarred " + strings.TrimSuffix(sym, "USDT")
					} else {
						m.notifyMsg = "★ starred " + strings.TrimSuffix(sym, "USDT")
					}
					m.notifyTicks = 20
					if !m.panelOn {
						m.panelOn = true
						m.cfg.PanelLayout = "right"
						m.visibleRows = m.tableVisibleRows()
					}
					m.rebuildSorted()
					m.clampCursor()
				}
			}
			return m, nil
		}

		// DeFi yields view
		if m.showDefi {
			switch msg.String() {
			case "d", "esc", "q":
				m.showDefi = false
			case "j", "down":
				m.defiCursor++
				m.clampDefiCursor()
			case "k", "up":
				m.defiCursor--
				m.clampDefiCursor()
			case "g", "home":
				m.defiCursor = 0
				m.clampDefiCursor()
			case "G", "end":
				m.defiCursor = len(m.defiPools) - 1
				m.clampDefiCursor()
			}
			return m, nil
		}

		// Config editor handles its own keys when open.
		if m.configUI.active {
			return m.updateConfig(msg)
		}

		// Command bar mode
		if m.commandMode {
			switch msg.String() {
			case "esc":
				m.commandMode = false
				m.commandBuf = ""
			case "enter":
				m.commandMode = false
				cmd := m.commandBuf
				m.commandBuf = ""
				return m.execCommand(cmd)
			case "backspace":
				if len(m.commandBuf) > 0 {
					m.commandBuf = m.commandBuf[:len(m.commandBuf)-1]
				}
			default:
				k := msg.String()
				if len(k) == 1 && k[0] >= 32 && k[0] <= 126 {
					m.commandBuf += k
				}
			}
			return m, nil
		}

		// Search mode handles its own keys.
		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.searchQuery = ""
				m.rebuildSorted()
				m.cursor = 0
				m.clampCursor()
			case "enter":
				m.searching = false
				// keep the filter active
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.rebuildSorted()
					m.cursor = 0
					m.clampCursor()
				}
			default:
				k := msg.String()
				if len(k) == 1 && k[0] >= 32 && k[0] <= 126 {
					m.searchQuery += k
					m.rebuildSorted()
					m.cursor = 0
					m.clampCursor()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.newsCursor >= 0 {
				m.newsCursor++
				if m.newsCursor >= len(m.newsArticles) {
					m.newsCursor = len(m.newsArticles) - 1
				}
				if m.newsCursor >= 5 {
					m.newsCursor = 4 // max visible in band
				}
			} else {
				m.cursor++
				m.clampCursor()
			}
		case "k", "up":
			if m.newsCursor >= 0 {
				m.newsCursor--
				if m.newsCursor < 0 {
					m.newsCursor = 0
				}
			} else {
				m.cursor--
				m.clampCursor()
			}
		case "g", "home":
			m.cursor = 0
			m.clampCursor()
		case "G", "end":
			m.cursor = len(m.sorted) - 1
			m.clampCursor()
		case "ctrl+d":
			half := m.visibleRows / 2
			if half < 1 {
				half = 1
			}
			m.cursor += half
			m.clampCursor()
		case "ctrl+u":
			half := m.visibleRows / 2
			if half < 1 {
				half = 1
			}
			m.cursor -= half
			m.clampCursor()
		case "tab":
			m.sortCol = (m.sortCol + 1) % sortColCount
			m.rebuildSorted()
			m.clampCursor()
		case "shift+tab":
			m.sortCol = (m.sortCol - 1 + sortColCount) % sortColCount
			m.rebuildSorted()
			m.clampCursor()
		case "s":
			if m.cursor >= 0 && m.cursor < len(m.sorted) {
				sym := m.sorted[m.cursor].Symbol
				wasStarred := m.watchlist.IsStarred(sym)
				m.watchlist.Toggle(sym)
				if wasStarred {
					m.notifyMsg = "★ unstarred " + strings.TrimSuffix(sym, "USDT")
				} else {
					m.notifyMsg = "★ starred " + strings.TrimSuffix(sym, "USDT")
				}
				m.notifyTicks = 20 // ~2s
				// Auto-show sidebar so user sees the starred coin appear
				if !m.panelOn {
					m.panelOn = true
					m.cfg.PanelLayout = "right"
					m.visibleRows = m.tableVisibleRows()
				}
				m.rebuildSorted()
				m.clampCursor()
			}
		case "f":
			m.filterMode = (m.filterMode + 1) % filterModeCount
			m.rebuildSorted()
			m.cursor = 0
			m.clampCursor()
		case "/":
			m.searching = true
			m.searchQuery = ""
		case "c":
			m.configUI = configState{active: true}
		case "p":
			m.panelOn = !m.panelOn
			if m.panelOn {
				m.cfg.PanelLayout = "right"
			} else {
				m.cfg.PanelLayout = "off"
			}
			m.visibleRows = m.tableVisibleRows()
			m.clampCursor()
		case "d":
			m.showDefi = true
			m.defiCursor = 0
			m.defiScroll = 0
		case "n":
			if m.newsOn && len(m.newsArticles) > 0 {
				if m.newsCursor >= 0 {
					m.newsCursor = -1 // unfocus
				} else {
					m.newsCursor = 0 // focus on first headline
				}
			}
		case "N":
			m.newsOn = !m.newsOn
			m.newsCursor = -1
			m.visibleRows = m.tableVisibleRows()
			m.clampCursor()
		case "?":
			m.showHelp = true
		case ":":
			m.commandMode = true
			m.commandBuf = ""
		case "enter":
			// If news cursor is active, open article reader
			if m.newsCursor >= 0 && m.newsCursor < len(m.newsArticles) {
				m.showArticle = true
			} else if m.cursor >= 0 && m.cursor < len(m.sorted) {
				// Open coin detail overlay
				m.showDetail = true
			}
		case "esc":
			// Clear news cursor first, then search filter
			if m.newsCursor >= 0 {
				m.newsCursor = -1
			} else if m.searchQuery != "" {
				m.searchQuery = ""
				m.rebuildSorted()
				m.cursor = 0
				m.clampCursor()
			}
		}
	}

	return m, nil
}

// updateConfig handles key events in the config editor.
func (m Model) updateConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.configUI.savedNotice > 0 {
		m.configUI.savedNotice--
	}

	if m.configUI.editing {
		switch msg.String() {
		case "esc":
			m.configUI.editing = false
			m.configUI.editBuf = ""
			m.configUI.editErr = ""
		case "enter":
			f := configFields[m.configUI.cursor]
			if err := f.set(&m.cfg, m.configUI.editBuf); err != nil {
				m.configUI.editErr = err.Error()
			} else {
				m.configUI.editing = false
				m.configUI.editBuf = ""
				m.configUI.editErr = ""
				m.configUI.dirty = true
				// Re-derive all state from config
				m.styles = NewStyles(m.cfg)
				m.panelOn = parsePanelOn(m.cfg.PanelLayout)
				m.sortCol = parseSortCol(m.cfg.DefaultSort)
				m.sortAsc = m.cfg.SortAscending
				m.filterMode = parseFilterMode(m.cfg.DefaultFilter)
				m.visibleRows = m.tableVisibleRows()
				m.rebuildSorted()
				m.clampCursor()
			}
		case "backspace":
			if len(m.configUI.editBuf) > 0 {
				m.configUI.editBuf = m.configUI.editBuf[:len(m.configUI.editBuf)-1]
			}
		default:
			k := msg.String()
			if len(k) == 1 && k[0] >= 32 && k[0] <= 126 {
				m.configUI.editBuf += k
				m.configUI.editErr = ""
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "esc", "c":
		if m.configUI.dirty {
			config.Save(m.cfg)
			m.configUI.savedNotice = 0
		}
		m.configUI.active = false
	case "j", "down":
		if m.configUI.cursor < len(configFields)-1 {
			m.configUI.cursor++
		}
	case "k", "up":
		if m.configUI.cursor > 0 {
			m.configUI.cursor--
		}
	case "enter":
		f := configFields[m.configUI.cursor]
		m.configUI.editing = true
		m.configUI.editBuf = f.get(m.cfg)
		m.configUI.editErr = ""
	case "r":
		// Reset current field to default
		f := configFields[m.configUI.cursor]
		def := config.Default()
		defVal := f.get(def)
		if err := f.set(&m.cfg, defVal); err == nil {
			m.configUI.dirty = true
			m.styles = NewStyles(m.cfg)
			m.notifyMsg = "↻ reset " + f.label
			m.notifyTicks = 20
		}
	case "ctrl+s":
		config.Save(m.cfg)
		m.configUI.dirty = false
		m.configUI.savedNotice = 20 // ~2s at 100ms ticks
	}

	return m, nil
}

// handleMouse processes mouse events for table clicks and scroll wheel.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.showHelp || m.configUI.active || m.showArticle || m.showDetail {
		return m, nil
	}

	// Only handle press events, ignore release to prevent double-firing
	if msg.Action == tea.MouseActionRelease {
		return m, nil
	}

	tableW := m.tableWidth()

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if msg.X < tableW {
			if m.showDefi {
				m.defiCursor--
				m.clampDefiCursor()
			} else {
				m.cursor--
				m.clampCursor()
			}
		}
	case tea.MouseButtonWheelDown:
		if msg.X < tableW {
			if m.showDefi {
				m.defiCursor++
				m.clampDefiCursor()
			} else {
				m.cursor++
				m.clampCursor()
			}
		}
	case tea.MouseButtonLeft:
		x := msg.X
		y := msg.Y

		// Only handle clicks within the table/news area (not the sidebar)
		if x >= tableW {
			break
		}

		newsH := m.newsHeight()
		tableEnd := 2 + m.visibleRows
		newsStart := m.termH - 2 - newsH
		if y >= 2 && y < tableEnd {
			if m.showDefi {
				row := m.defiScroll + (y - 3) // -3: title + sep + col header
				if row >= 0 && row < len(m.defiPools) {
					m.defiCursor = row
					m.clampDefiCursor()
				}
			} else {
				row := m.offset + (y - 2)
				if row < len(m.sorted) {
					now := time.Now()
					// Double-click detection: same row within 400ms → star
					if row == m.lastClickRow && now.Sub(m.lastClickAt) < 400*time.Millisecond {
						sym := m.sorted[row].Symbol
						wasStarred := m.watchlist.IsStarred(sym)
						m.watchlist.Toggle(sym)
						if wasStarred {
							m.notifyMsg = "★ unstarred " + strings.TrimSuffix(sym, "USDT")
						} else {
							m.notifyMsg = "★ starred " + strings.TrimSuffix(sym, "USDT")
						}
						m.notifyTicks = 20
						if !m.panelOn {
							m.panelOn = true
							m.cfg.PanelLayout = "right"
							m.visibleRows = m.tableVisibleRows()
						}
						m.rebuildSorted()
						m.clampCursor()
						m.lastClickRow = -1 // reset to avoid triple-click
					} else {
						m.cursor = row
						m.clampCursor()
						m.lastClickRow = row
						m.lastClickAt = now
					}
				}
			}
		} else if newsH > 0 && y >= newsStart && y < newsStart+newsH {
			lineIdx := y - newsStart - 1 // -1 for separator line
			if lineIdx >= 0 && lineIdx < 5 && lineIdx < len(m.newsArticles) {
				m.newsCursor = lineIdx
				m.showArticle = true
			}
		}
	}
	return m, nil
}

// execCommand processes a vim-style command.
func (m Model) execCommand(cmd string) (tea.Model, tea.Cmd) {
	cmd = strings.TrimSpace(strings.ToLower(cmd))
	switch {
	case cmd == "q" || cmd == "quit":
		return m, tea.Quit
	case cmd == "w" || cmd == "save":
		config.Save(m.cfg)
		m.notifyMsg = "✓ config saved"
		m.notifyTicks = 20
	case cmd == "wq":
		config.Save(m.cfg)
		return m, tea.Quit
	case cmd == "help":
		m.showHelp = true
	case cmd == "config" || cmd == "settings":
		m.configUI = configState{active: true}
	case cmd == "defi":
		m.showDefi = true
		m.defiCursor = 0
	case cmd == "news":
		m.newsOn = !m.newsOn
		m.visibleRows = m.tableVisibleRows()
		m.clampCursor()
	case cmd == "panel":
		m.panelOn = !m.panelOn
		if m.panelOn {
			m.cfg.PanelLayout = "right"
		} else {
			m.cfg.PanelLayout = "off"
		}
		m.visibleRows = m.tableVisibleRows()
		m.clampCursor()
	case strings.HasPrefix(cmd, "sort "):
		arg := strings.TrimPrefix(cmd, "sort ")
		switch arg {
		case "vol", "volume":
			m.sortCol = SortVolume
		case "price":
			m.sortCol = SortPrice
		case "change":
			m.sortCol = SortChange
		case "symbol", "sym":
			m.sortCol = SortSymbol
		case "corr", "btc":
			m.sortCol = SortCorrelation
		default:
			m.notifyMsg = "unknown sort: " + arg
			m.notifyTicks = 20
			return m, nil
		}
		m.rebuildSorted()
		m.clampCursor()
		m.notifyMsg = "sorted by " + sortColName(m.sortCol)
		m.notifyTicks = 20
	case strings.HasPrefix(cmd, "filter "):
		arg := strings.TrimPrefix(cmd, "filter ")
		switch arg {
		case "all":
			m.filterMode = FilterAll
		case "gainers", "gain":
			m.filterMode = FilterGainers
		case "losers", "lose":
			m.filterMode = FilterLosers
		default:
			m.notifyMsg = "unknown filter: " + arg
			m.notifyTicks = 20
			return m, nil
		}
		m.rebuildSorted()
		m.cursor = 0
		m.clampCursor()
	case strings.HasPrefix(cmd, "go ") || strings.HasPrefix(cmd, "find "):
		parts := strings.Fields(cmd)
		if len(parts) >= 2 {
			q := strings.ToUpper(parts[1])
			for i, t := range m.sorted {
				if strings.Contains(t.Symbol, q) {
					m.cursor = i
					m.clampCursor()
					break
				}
			}
		}
	default:
		m.notifyMsg = "unknown command: " + cmd
		m.notifyTicks = 20
	}
	return m, nil
}

// openURL returns a Cmd that opens a URL in the default browser.
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		openBrowser(url)
		return nil
	}
}
