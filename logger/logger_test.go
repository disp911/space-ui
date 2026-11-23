package logger

import (
	"strings"
	"testing"
	"time"
)

// TestBufferedLogsShowTheGivenZone checks that a buffered line's time is
// written in the zone asked for, not in the one the line was logged in.
func TestBufferedLogsShowTheGivenZone(t *testing.T) {
	logBuffer = nil
	addToBuffer("INFO", "hello")
	at := logBuffer[0].time

	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	for _, loc := range []*time.Location{time.UTC, moscow} {
		lines := GetLogs(10, "debug", loc)
		want := at.In(loc).Format(timeFormat) + " INFO - hello"
		if len(lines) != 1 || lines[0] != want {
			t.Errorf("%s: %q, want %q", loc, lines, want)
		}
	}
	utc, msk := GetLogs(1, "debug", time.UTC)[0], GetLogs(1, "debug", moscow)[0]
	if strings.HasPrefix(utc, msk[:13]) {
		t.Errorf("the same hour in UTC and Moscow: %q, %q", utc, msk)
	}
}
