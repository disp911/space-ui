package service

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"

	"github.com/op/go-logging"
)

// xrayLogTestDB opens a log database of its own with entries at the moments given.
func xrayLogTestDB(t *testing.T, at ...time.Time) {
	t.Helper()
	logger.InitLogger(logging.CRITICAL)
	t.Setenv("SPACE_UI_LOG_FOLDER", t.TempDir())
	if err := database.InitLogDB(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := database.GetLogDB().DB(); err == nil {
			sqlDB.Close()
		}
	})
	for i, m := range at {
		// The day column as the ingest job fills it, by the server's clock: reading ignores it.
		row := model.XrayLogEntry{Day: m.Local().Format("2006-01-02"), Timestamp: m.UnixMicro(), FromAddress: "a", ToAddress: strings.Repeat("x", i+1), Inbound: "in", Outbound: "direct"}
		if err := database.GetLogDB().Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
}

// TestXrayLogDaysFollowTheZone checks that the log's days, their entries and
// the times in an exported file follow the time zone they are read in.
func TestXrayLogDaysFollowTheZone(t *testing.T) {
	// 21 November 2026, 22:30 and 23:30 UTC, and 22 November, 12:00 UTC.
	late := time.Date(2026, 11, 21, 22, 30, 0, 0, time.UTC)
	later := time.Date(2026, 11, 21, 23, 30, 0, 0, time.UTC)
	noon := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	xrayLogTestDB(t, late, later, noon)
	db := database.GetLogDB()
	free := []string{"direct"}

	for tz, c := range map[string]struct {
		days   []string
		counts []int64
	}{
		// Moscow, UTC+3: all three on the 22nd.
		"Europe/Moscow": {[]string{"2026-11-22"}, []int64{3}},
		// New York, UTC-5: the night ones still on the 21st.
		"America/New_York": {[]string{"2026-11-22", "2026-11-21"}, []int64{1, 2}},
		"UTC":              {[]string{"2026-11-22", "2026-11-21"}, []int64{1, 2}},
	} {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			t.Fatal(err)
		}
		if days := xrayLogDates(db, loc); !slices.Equal(days, c.days) {
			t.Errorf("%s: days %v, want %v", tz, days, c.days)
			continue
		}
		for i, day := range c.days {
			query, ok := xrayLogQuery(db, loc, day, "", "", "", "", "", free, nil)
			var n int64
			if !ok || query.Count(&n).Error != nil || n != c.counts[i] {
				t.Errorf("%s %s: %d entries, want %d", tz, day, n, c.counts[i])
			}
		}
	}

	ny, _ := time.LoadLocation("America/New_York")
	var out bytes.Buffer
	if err := exportXrayLogs(&out, db, ny, "2026-11-21", "", "", "", "", "", free, nil); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "2026/11/21 17:30:00") || !strings.HasPrefix(lines[1], "2026/11/21 18:30:00") {
		t.Errorf("export in New York time:\n%s", out.String())
	}
}

// TestXrayLogDayRangeKeepsClockChanges checks a day's span across a clock change.
func TestXrayLogDayRangeKeepsClockChanges(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	from, to, ok := xrayLogDayRange("2026-11-01", ny)
	if !ok || time.Duration(to-from)*time.Microsecond != 25*time.Hour {
		t.Errorf("1 November 2026 in New York: %v", time.Duration(to-from)*time.Microsecond)
	}
	if _, _, ok := xrayLogDayRange("not a day", ny); ok {
		t.Error("a word passes for a day")
	}
}
