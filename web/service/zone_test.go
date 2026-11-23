package service

import (
	"testing"
	"time"
)

// TestSystemZoneName checks how the server's zone is named for the browser.
func TestSystemZoneName(t *testing.T) {
	moscow := time.FixedZone("MSK", 3*3600)
	india := time.FixedZone("IST", 5*3600+1800)
	newYork := time.FixedZone("EST", -5*3600)
	for _, c := range []struct {
		name, tz, etc, link string
		now                 time.Time
		want                string
	}{
		{"TZ", "Europe/Berlin", "", "", time.Now(), "Europe/Berlin"},
		{"TZ with a colon", ":Asia/Tokyo", "", "", time.Now(), "Asia/Tokyo"},
		{"/etc/timezone", "", "Europe/Moscow\n", "", time.Now(), "Europe/Moscow"},
		{"/etc/localtime", "", "", "/usr/share/zoneinfo/America/New_York", time.Now(), "America/New_York"},
		{"a wrong TZ is passed over", "Mars/Olympus", "Europe/Moscow", "", time.Now(), "Europe/Moscow"},
		{"an offset east", "", "", "", time.Date(2026, 11, 23, 12, 0, 0, 0, moscow), "Etc/GMT-3"},
		{"an offset west", "", "", "", time.Date(2026, 11, 23, 12, 0, 0, 0, newYork), "Etc/GMT+5"},
		{"a half hour", "", "", "", time.Date(2026, 11, 23, 12, 0, 0, 0, india), "UTC"},
		{"nothing", "", "", "", time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC), "UTC"},
	} {
		if got := systemZoneName(c.tz, c.etc, c.link, c.now); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
		if _, err := time.LoadLocation(systemZoneName(c.tz, c.etc, c.link, c.now)); err != nil {
			t.Errorf("%s: the name does not load: %v", c.name, err)
		}
	}
}
