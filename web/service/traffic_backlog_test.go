package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/xray"
)

func TestTrafficBacklogAddsKeptTrafficToTheNextWrite(t *testing.T) {
	var b TrafficBacklog
	in := &xray.Traffic{IsInbound: true, Tag: "inbound-443", Up: 10, Down: 20}
	out := &xray.Traffic{IsOutbound: true, Tag: "direct", Up: 1, Down: 2}
	gleb := &xray.ClientTraffic{Email: "gleb", Up: 5, Down: 6}

	// Nothing kept: the traffic passes as it is.
	traffics, clients := b.Take([]*xray.Traffic{in}, []*xray.ClientTraffic{gleb})
	if len(traffics) != 1 || traffics[0] != in || len(clients) != 1 || clients[0] != gleb {
		t.Fatal("an empty backlog should hand the traffic back untouched")
	}

	// A failed write keeps it, twice over.
	b.Keep([]*xray.Traffic{in, out}, []*xray.ClientTraffic{gleb})
	b.Keep([]*xray.Traffic{in}, []*xray.ClientTraffic{{Email: "zero"}})

	nextIn := &xray.Traffic{IsInbound: true, Tag: "inbound-443", Up: 100, Down: 200}
	newClient := &xray.ClientTraffic{Email: "dasha", Up: 7, Down: 8}
	traffics, clients = b.Take([]*xray.Traffic{nextIn}, []*xray.ClientTraffic{newClient})

	got := map[string][2]int64{}
	for _, tr := range traffics {
		got[tr.Tag] = [2]int64{tr.Up, tr.Down}
	}
	if got["inbound-443"] != [2]int64{120, 240} || got["direct"] != [2]int64{1, 2} || len(got) != 2 {
		t.Fatalf("traffic after a kept write = %v", got)
	}
	gotClients := map[string][2]int64{}
	for _, c := range clients {
		gotClients[c.Email] = [2]int64{c.Up, c.Down}
	}
	if gotClients["gleb"] != [2]int64{5, 6} || gotClients["dasha"] != [2]int64{7, 8} || len(gotClients) != 2 {
		t.Fatalf("clients after a kept write = %v", gotClients)
	}
	// The values handed in are others' too: never changed.
	if nextIn.Up != 100 || nextIn.Down != 200 || in.Up != 10 {
		t.Fatal("Take changed the traffic it was given")
	}

	// Taken once, the backlog is empty again.
	if traffics, _ := b.Take(nil, nil); len(traffics) != 0 {
		t.Fatalf("the backlog was not emptied: %v", traffics)
	}
}

func TestTrafficIsRecordedOrReportedNotRecorded(t *testing.T) {
	listsTestDB(t)
	db := database.GetDB()
	if err := db.Create(&model.Inbound{Tag: "inbound-443", Port: 443, Protocol: model.VLESS}).Error; err != nil {
		t.Fatal(err)
	}
	var s InboundService
	traffic := []*xray.Traffic{{IsInbound: true, Tag: "inbound-443", Up: 3, Down: 4}}
	if _, _, err := s.AddTraffic(traffic, nil); err != nil {
		t.Fatal(err)
	}
	var ib model.Inbound
	if err := db.Where("tag = ?", "inbound-443").First(&ib).Error; err != nil || ib.Up != 3 || ib.Down != 4 {
		t.Fatalf("recorded traffic = %d/%d, %v", ib.Up, ib.Down, err)
	}

	// Without the database the write fails, and says nothing was recorded.
	database.CloseDB()
	_, _, err := s.AddTraffic(traffic, nil)
	if !errors.Is(err, ErrTrafficNotRecorded) {
		t.Fatalf("a failed write = %v, want ErrTrafficNotRecorded", err)
	}
}

func TestDatabaseIsInWALModeAndExportsItsRecentWrites(t *testing.T) {
	listsTestDB(t)
	db := database.GetDB()
	var mode string
	if err := db.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil || mode != "wal" {
		t.Fatalf("journal_mode = %q, %v", mode, err)
	}
	var timeout int
	if err := db.Raw("PRAGMA busy_timeout").Scan(&timeout).Error; err != nil || timeout != 30000 {
		t.Fatalf("busy_timeout = %d, %v", timeout, err)
	}
	if err := db.Create(&model.Inbound{Tag: "fresh", Port: 8443, Protocol: model.VLESS}).Error; err != nil {
		t.Fatal(err)
	}

	export := filepath.Join(t.TempDir(), "export.db")
	if err := database.ExportTo(export); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(export + "-wal"); err == nil {
		t.Fatal("the export left a WAL file beside it: it is not a single file")
	}
	if err := database.ValidateSQLiteDB(export); err != nil {
		t.Fatal(err)
	}
	// The write just made, still in the WAL, is in the copy.
	database.CloseDB()
	if err := database.InitDB(export); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := database.GetDB().Model(&model.Inbound{}).Where("tag = ?", "fresh").Count(&n).Error; err != nil || n != 1 {
		t.Fatalf("the exported copy has %d «fresh» inbounds, %v", n, err)
	}
}
