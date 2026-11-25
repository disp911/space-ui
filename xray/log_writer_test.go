package xray

import (
	"strings"
	"testing"
	"time"

	"github.com/op/go-logging"

	"github.com/disp911/space-ui/v3/logger"
)

// TestLogWriterKeepsXrayLevels checks that a line Xray gave a level is
// logged at it, whatever it says, and that a line without one is an error
// only when it says something failed.
func TestLogWriterKeepsXrayLevels(t *testing.T) {
	t.Setenv("SPACE_UI_LOG_FOLDER", t.TempDir())
	logger.InitLogger(logging.CRITICAL)
	t.Cleanup(logger.CloseLogger)

	w := NewLogWriter()
	for _, line := range []string{
		"2026/11/25 07:10:16.000001 [Info] transport/internet/tcp: REALITY: processed invalid connection from 1.2.3.4:5: authentication failed or validation criteria not met",
		"2026/11/25 07:10:16.000002 [Info] app/proxyman/inbound: connection ends > io: read/write on closed pipe",
		"2026/11/25 07:10:16.000003 [Warning] core: Xray 26.11.9 started",
		"2026/11/25 07:10:16.000004 [Error] app/dispatcher: something broke",
		"Failed to start: main: failed to load config files",
		"An unlevelled line",
	} {
		w.Write([]byte(line))
	}

	want := map[string]string{
		"REALITY: processed invalid connection": "INFO",
		"connection ends":                       "INFO",
		"Xray 26.11.9 started":                  "WARNING",
		"something broke":                       "ERROR",
		"Failed to start":                       "ERROR",
		"An unlevelled line":                    "DEBUG",
	}
	lines := logger.GetLogs(100, "debug", time.UTC)
	for part, level := range want {
		found := false
		for _, l := range lines {
			if strings.Contains(l, part) {
				found = true
				if !strings.Contains(l, " "+level+" - XRAY: ") {
					t.Errorf("%q logged as %q, want %s", part, l, level)
				}
			}
		}
		if !found {
			t.Errorf("%q was not logged", part)
		}
	}
}
