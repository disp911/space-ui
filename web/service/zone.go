package service

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// The panel's time zone: the logs show their times in it, the Xray log cuts
// its days in it, and the panel's windows show their dates in it. «Server
// time» (Local) is the server's own zone; the browser needs it by its IANA
// name, which Go does not keep, so the server works it out.

// PanelLocation is the panel's time zone, the server's own by default.
func PanelLocation() *time.Location {
	loc, err := new(SettingService).GetTimeLocation()
	if err != nil {
		return time.Local
	}
	return loc
}

// PanelZoneName is the panel's time zone by its IANA name, for the browser.
func PanelZoneName() string {
	if name, err := new(SettingService).getString("timeLocation"); err == nil && name != "" && name != "Local" {
		if _, err := time.LoadLocation(name); err == nil {
			return name
		}
	}
	etc, _ := os.ReadFile("/etc/timezone")
	link, _ := os.Readlink("/etc/localtime")
	return systemZoneName(os.Getenv("TZ"), string(etc), link, time.Now())
}

// systemZoneName is the server's zone by its IANA name: from TZ, from
// /etc/timezone, or from the file /etc/localtime links to; failing them, a
// whole-hour offset as Etc/GMT±N (which keeps no clock changes), else UTC.
func systemZoneName(tz, etcTimezone, localtimeLink string, now time.Time) string {
	known := func(name string) bool {
		if name == "" || name == "Local" {
			return false
		}
		_, err := time.LoadLocation(name)
		return err == nil
	}
	if name := strings.TrimPrefix(tz, ":"); known(name) {
		return name
	}
	if name := strings.TrimSpace(etcTimezone); known(name) {
		return name
	}
	if _, name, ok := strings.Cut(localtimeLink, "zoneinfo/"); ok && known(name) {
		return name
	}
	_, offset := now.Zone()
	if offset == 0 || offset%3600 != 0 || offset/3600 < -12 || offset/3600 > 14 {
		return "UTC"
	}
	// Etc/GMT names have the sign turned round: Etc/GMT-3 is UTC+3.
	hours := -offset / 3600
	if hours > 0 {
		return "Etc/GMT+" + strconv.Itoa(hours)
	}
	return "Etc/GMT" + strconv.Itoa(hours)
}
