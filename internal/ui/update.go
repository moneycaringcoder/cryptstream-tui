package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/moneycaringcoder/cryptstream-tui/internal/config"
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
		m.tickers[t.Symbol] = t

		// Track price history for sparklines.
		maxHist := m.cfg.SparklineLength
		h := m.priceHistory[t.Symbol]
		h = append(h, t.LastPrice)
		if len(h) > maxHist {
			h = h[len(h)-maxHist:]
		}
		m.priceHistory[t.Symbol] = h

		m.rebuildSorted()
		return m, nil

	case connMsg:
		m.connected = msg.connected
		return m, nil

	case tea.KeyMsg:
		// Help screen
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q":
				m.showHelp = false
			}
			return m, nil
		}

		// Config editor handles its own keys when open.
		if m.configUI.active {
			return m.updateConfig(msg)
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
			m.cursor++
			m.clampCursor()
		case "k", "up":
			m.cursor--
			m.clampCursor()
		case "g", "home":
			m.cursor = 0
			m.clampCursor()
		case "G", "end":
			m.cursor = len(m.sorted) - 1
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
				m.watchlist.Toggle(m.sorted[m.cursor].Symbol)
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
		case "?":
			m.showHelp = true
		case "esc":
			// Clear active search filter
			if m.searchQuery != "" {
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
	case "ctrl+s":
		config.Save(m.cfg)
		m.configUI.dirty = false
		m.configUI.savedNotice = 20 // ~2s at 100ms ticks
	}

	return m, nil
}
