package service

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/op/go-logging"
	"github.com/xtls/xray-core/infra/conf"

	"github.com/disp911/space-ui/v3/config"
	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"
)

func TestSiteInboundIsWhatXrayTakes(t *testing.T) {
	ib, client, err := newSiteInbound("docs.example.com", "/run/space-ui/site.sock")
	if err != nil {
		t.Fatal(err)
	}
	var stream conf.StreamConfig
	if err := json.Unmarshal([]byte(ib.StreamSettings), &stream); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Build(); err != nil {
		t.Fatalf("Xray refuses the stream settings: %v", err)
	}
	r := stream.REALITYSettings
	if r.Type != "unix" || r.Xver != 1 || r.ServerNames[0] != "docs.example.com" || len(r.ShortIds) != 8 {
		t.Fatalf("target type %q, names %v, short ids %v", r.Type, r.ServerNames, r.ShortIds)
	}
	// XHTTP and the WEB proxy reach their inbounds through the website, at
	// full speed.
	if r.LimitFallbackDownload.BytesPerSec != 0 || r.LimitFallbackUpload.BytesPerSec != 0 {
		t.Fatalf("the website is slowed down: %+v %+v", r.LimitFallbackUpload, r.LimitFallbackDownload)
	}

	var settings conf.VLessInboundConfig
	if err := json.Unmarshal([]byte(ib.Settings), &settings); err != nil {
		t.Fatal(err)
	}
	if _, err := settings.Build(); err != nil {
		t.Fatalf("Xray refuses the settings: %v", err)
	}
	if client.Flow != "xtls-rprx-vision" || len(client.Email) != 8 || len(client.SubID) != 16 {
		t.Fatalf("client %+v", client)
	}
	if ib.Port != 443 || ib.Tag != "inbound-443" || !ib.Enable || ib.Protocol != model.VLESS {
		t.Fatalf("inbound %+v", ib)
	}
}

func TestSiteTarget(t *testing.T) {
	sock := config.GetSiteSocket()
	ib, _, err := newSiteInbound("docs.example.com", sock)
	if err != nil {
		t.Fatal(err)
	}
	if !siteTarget(ib) {
		t.Fatal("the site inbound must be told apart by its target")
	}
	foreign, _, err := newSiteInbound("docs.example.com", "www.example.org:443")
	if err != nil {
		t.Fatal(err)
	}
	if siteTarget(foreign) {
		t.Fatal("a REALITY inbound with another target is not the site's")
	}
	other := *ib
	other.Protocol = model.Trojan
	if siteTarget(&other) {
		t.Fatal("only VLESS carries the website")
	}
}

func siteTestDB(t *testing.T) {
	t.Helper()
	logger.InitLogger(logging.CRITICAL)
	if err := database.InitDB(filepath.Join(t.TempDir(), "space-ui.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.CloseDB() })
	s := SettingService{}
	for k, v := range map[string]string{"siteDomain": "docs.example.com", "webBasePath": "/secret/", "webPort": "2053", "subEnable": "true"} {
		if err := s.saveSetting(k, v); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSwitchModeKeepsTheSiteInbound(t *testing.T) {
	siteTestDB(t)
	sites := SiteService{}
	inbounds := InboundService{}

	ib, err := sites.SwitchMode(SiteReality)
	if err != nil {
		t.Fatal(err)
	}
	if ib == nil || !ib.Enable || ib.Port != 443 || !siteTarget(ib) {
		t.Fatalf("site inbound %+v", ib)
	}
	link, sub, err := sites.FirstClientLinks(ib)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "vless" || u.Host != "docs.example.com:443" || u.Query().Get("sni") != "docs.example.com" ||
		u.Query().Get("flow") != "xtls-rprx-vision" || u.Query().Get("security") != "reality" || u.Query().Get("pbk") == "" {
		t.Fatalf("link %s", link)
	}
	if !strings.HasPrefix(sub, "https://docs.example.com/sub/") {
		t.Fatalf("subscription %s", sub)
	}

	// It is an ordinary inbound: changed, switched off and on as any other.
	renamed := *ib
	renamed.Remark = "mine"
	renamed.ExpiryTime = 1
	if err := inbounds.guardPanelPort(&renamed); err != nil {
		t.Fatalf("the site inbound takes any change: %v", err)
	}
	if err := inbounds.guardPanelPort(&model.Inbound{Port: 2053, Enable: true, Protocol: model.Trojan}); err != nil {
		t.Fatalf("behind REALITY the panel opens no port of its own: %v", err)
	}

	// On its own port the socket is shut: the inbound is switched off, and
	// found again by its target.
	if got, err := sites.SwitchMode(SitePort); err != nil || got != nil {
		t.Fatalf("own port: %v %v", got, err)
	}
	kept, err := sites.SiteInbound()
	if err != nil || kept == nil || kept.Id != ib.Id || kept.Enable {
		t.Fatalf("kept %+v %v", kept, err)
	}
	if err := inbounds.guardPanelPort(&model.Inbound{Port: 2053, Enable: true, Protocol: model.Trojan}); err == nil {
		t.Fatal("on its own port the panel keeps that port")
	}
	if err := inbounds.guardPanelPort(&model.Inbound{Port: 2053, Enable: true, Protocol: model.Hysteria}); err != nil {
		t.Fatalf("UDP shares the panel's port: %v", err)
	}

	// Back behind REALITY the same inbound comes back on.
	back, err := sites.SwitchMode(SiteReality)
	if err != nil || back == nil || back.Id != ib.Id || !back.Enable {
		t.Fatalf("back %+v %v", back, err)
	}

	// Deleted, it is made anew on the next switch. (Deleting through the
	// service would ask the running Xray to drop it, and tests run none.)
	if err := database.GetDB().Delete(&model.Inbound{}, ib.Id).Error; err != nil {
		t.Fatal(err)
	}
	again, err := sites.SwitchMode(SiteReality)
	if err != nil || again == nil || again.Id == ib.Id {
		t.Fatalf("again %+v %v", again, err)
	}
}

func TestSwitchModeRefusesATakenPort(t *testing.T) {
	siteTestDB(t)
	db := database.GetDB()
	other := &model.Inbound{Tag: "inbound-443", Port: 443, Protocol: model.Trojan, Remark: "old", Enable: true, Settings: `{"clients":[]}`}
	hysteria := &model.Inbound{Tag: "inbound-443-udp", Port: 443, Protocol: model.Hysteria, Remark: "h2", Enable: true, Settings: `{"clients":[]}`}
	for _, ib := range []*model.Inbound{other, hysteria} {
		if err := db.Create(ib).Error; err != nil {
			t.Fatal(err)
		}
	}
	sites := SiteService{}
	if _, err := sites.SwitchMode(SiteReality); err == nil || !strings.Contains(err.Error(), `"old"`) || strings.Contains(err.Error(), `"h2"`) {
		t.Fatalf("REALITY must not start beside another inbound on TCP 443, and only that one is named: %v", err)
	}
	if mode, _ := (&SettingService{}).GetSiteMode(); mode != SitePort {
		t.Fatalf("a refused switch must leave the mode, got %q", mode)
	}
	if err := db.Delete(other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := sites.SwitchMode(SiteReality); err != nil {
		t.Fatalf("Hysteria on UDP 443 does not stand in REALITY's way: %v", err)
	}

	panelPort := &model.Inbound{Tag: "inbound-2053", Port: 2053, Protocol: model.VLESS, Remark: "on the panel", Enable: true}
	if err := db.Create(panelPort).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := sites.SwitchMode(SitePort); err == nil || !strings.Contains(err.Error(), "2053") {
		t.Fatalf("the panel must not move onto a port an inbound holds: %v", err)
	}
}

func TestPortsAreSharedByTCPAndUDP(t *testing.T) {
	siteTestDB(t)
	inbounds := InboundService{}
	reality := &model.Inbound{Tag: "inbound-8443", Port: 8443, Protocol: model.VLESS, Enable: true,
		StreamSettings: `{"network":"tcp","security":"reality"}`}
	if err := database.GetDB().Create(reality).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		ib    *model.Inbound
		taken bool
	}{
		{&model.Inbound{Port: 8443, Protocol: model.Hysteria}, false},
		{&model.Inbound{Port: 8443, Protocol: model.VMESS, StreamSettings: `{"network":"kcp"}`}, false},
		{&model.Inbound{Port: 8443, Protocol: model.Trojan}, true},
		{&model.Inbound{Port: 8443, Protocol: model.Shadowsocks, Settings: `{"network":"tcp,udp"}`}, true},
		{&model.Inbound{Port: 8443, Protocol: model.Shadowsocks, Settings: `{"network":"udp"}`}, false},
		{&model.Inbound{Port: 8443, Listen: "127.0.0.1", Protocol: model.Trojan}, true},
		{&model.Inbound{Port: 8444, Protocol: model.Trojan}, false},
	} {
		taken, err := inbounds.checkPortExist(tc.ib, 0)
		if err != nil || taken != tc.taken {
			t.Errorf("%s %s on %d: taken %v, want %v (%v)", tc.ib.Protocol, tc.ib.Settings+tc.ib.StreamSettings, tc.ib.Port, taken, tc.taken, err)
		}
	}
}

func TestInboundTag(t *testing.T) {
	for _, tc := range []struct {
		ib   model.Inbound
		want string
	}{
		{model.Inbound{Port: 443, Protocol: model.VLESS}, "inbound-443"},
		{model.Inbound{Port: 443, Protocol: model.Hysteria}, "inbound-443-udp"},
		{model.Inbound{Port: 443, Listen: "0.0.0.0", Protocol: model.WireGuard}, "inbound-443-udp"},
		{model.Inbound{Port: 8443, Listen: "10.0.0.1", Protocol: model.Trojan}, "inbound-10.0.0.1:8443"},
		{model.Inbound{Port: 1080, Protocol: model.Mixed, Settings: `{"udp":true}`}, "inbound-1080"},
	} {
		if got := InboundTag(&tc.ib); got != tc.want {
			t.Errorf("%s on %s:%d: %s, want %s", tc.ib.Protocol, tc.ib.Listen, tc.ib.Port, got, tc.want)
		}
	}
}
