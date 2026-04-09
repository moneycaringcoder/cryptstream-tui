package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/moneycaringcoder/tuikit-go/tuitest"
)

// Session tests: record a scripted flow, save it to testdata/sessions,
// and assert the file round-trips via LoadSession. These act as a
// committed regression baseline — any breakage in the UI event pipeline
// that changes the step output trips the test on CI.
//
// Set CRYPTSTREAM_UPDATE_SESSIONS=1 to re-record the golden files.

const sessionsDir = "../../testdata/sessions"

func sessionPath(name string) string {
	return filepath.Join(sessionsDir, name+".tuisess")
}

func recordAndVerify(t *testing.T, name string, play func(r *tuitest.SessionRecorder)) {
	t.Helper()
	path := sessionPath(name)
	force := os.Getenv("CRYPTSTREAM_UPDATE_SESSIONS") == "1"

	if _, err := os.Stat(path); force || os.IsNotExist(err) {
		tm, _ := testCryptoApp(t)
		rec := tuitest.NewSessionRecorder(tm)
		play(rec)
		if err := rec.Save(path); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}

	sess, err := tuitest.LoadSession(path)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	if len(sess.Steps) == 0 {
		t.Errorf("session %s has no steps", name)
	}
	if sess.Cols == 0 || sess.Lines == 0 {
		t.Errorf("session %s has zero viewport: %dx%d", name, sess.Cols, sess.Lines)
	}
}

// TestSession_SymbolNavigation covers navigating through the ticker
// table: move cursor down a few rows then back up.
func TestSession_SymbolNavigation(t *testing.T) {
	recordAndVerify(t, "symbol_navigation", func(r *tuitest.SessionRecorder) {
		r.Key("down").Key("down").Key("down").Key("up")
	})
}

// TestSession_SettingsOpen covers the settings overlay flow: open the
// config editor via "c", navigate down, then close with esc.
func TestSession_SettingsOpen(t *testing.T) {
	recordAndVerify(t, "settings_open", func(r *tuitest.SessionRecorder) {
		r.Key("c").Key("down").Key("down").Key("esc")
	})
}
