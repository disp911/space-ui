package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/telemt"
)

func TestValidateMTProtoClients(t *testing.T) {
	good := model.Client{Email: "a", ID: strings.Repeat("0f", 16)}
	if err := validateMTProtoClients(model.MTProto, []model.Client{good}); err != nil {
		t.Fatalf("valid secret rejected: %v", err)
	}
	for _, id := range []string{"", "0f0f", strings.Repeat("0F", 16), strings.Repeat("zz", 16), "8e3a3c8e-0f57-4a57-9a6e-4b3c7f0f5e11"} {
		bad := model.Client{Email: "b", ID: id}
		if err := validateMTProtoClients(model.MTProto, []model.Client{good, bad}); err == nil {
			t.Errorf("secret %q must be rejected", id)
		}
	}
	// Other protocols keep their own ID rules.
	uuidClient := model.Client{Email: "c", ID: "8e3a3c8e-0f57-4a57-9a6e-4b3c7f0f5e11"}
	if err := validateMTProtoClients(model.VLESS, []model.Client{uuidClient}); err != nil {
		t.Fatalf("vless clients must not be checked as mtproto: %v", err)
	}
}

func TestValidateMTProtoInboundIgnoresOtherProtocols(t *testing.T) {
	inbound := &model.Inbound{Protocol: model.VLESS, Settings: `not json`}
	if err := validateMTProtoInbound(inbound, nil); err != nil {
		t.Fatalf("non-mtproto inbounds must pass unchanged: %v", err)
	}
}

func TestIsXrayProtocol(t *testing.T) {
	if model.IsXrayProtocol(model.MTProto) {
		t.Fatal("mtproto must not be treated as an Xray protocol")
	}
	for _, p := range []model.Protocol{model.VMESS, model.VLESS, model.Trojan, model.Shadowsocks, model.Hysteria, model.WireGuard} {
		if !model.IsXrayProtocol(p) {
			t.Errorf("%s must be an Xray protocol", p)
		}
	}
}

func TestMTProtoUserCarriesLimits(t *testing.T) {
	expiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	c := model.Client{ID: strings.Repeat("0f", 16), LimitIP: 2, MaxConns: 5, RateUp: 3, RateDown: 25, ExpiryTime: expiry.UnixMilli()}
	u := mtprotoUser("alice", c, true)
	if !u.Disabled || u.MaxUniqueIPs != 2 || u.MaxTCPConns != 5 || u.RateUpBps != 3_000_000 || u.RateDownBps != 25_000_000 || !u.Expires.Equal(expiry) {
		t.Fatalf("user = %+v", u)
	}
	// An expiry counted from the first connection is negative and stays with
	// the panel; negative limits mean nothing.
	c.RateDown = 1 << 40
	if u := mtprotoUser("phone", c, false); u.RateDownBps != telemt.MaxRateBps {
		t.Fatalf("an oversized rate must be capped at telemt's maximum, got %d", u.RateDownBps)
	}
	c.ExpiryTime, c.RateUp, c.MaxConns = -86400000, -1, -1
	u = mtprotoUser("alice", c, false)
	if !u.Expires.IsZero() || u.RateUpBps != 0 || u.MaxTCPConns != 0 {
		t.Fatalf("user = %+v", u)
	}
}

func webSettingsOf(t *testing.T, inbound *model.Inbound) mtprotoWeb {
	t.Helper()
	var s mtprotoSettings
	if err := json.Unmarshal([]byte(inbound.Settings), &s); err != nil || s.Web == nil {
		t.Fatalf("settings %s: %v", inbound.Settings, err)
	}
	return *s.Web
}

func TestPrepareMTProtoWebFillsPathAndAddress(t *testing.T) {
	lookups := 0
	lookup := func(host string) (string, error) {
		lookups++
		if host != "se.gateway.example.org" {
			t.Errorf("looked up %q", host)
		}
		return "203.0.113.10", nil
	}
	inbound := &model.Inbound{Protocol: model.MTProto, Settings: `{"tlsDomain":"www.example.com","web":{"enable":true,"host":" SE.Gateway.Example.org. ","path":"","publicIp":""},"clients":[]}`}
	if err := prepareMTProtoWeb(inbound, nil, lookup); err != nil {
		t.Fatal(err)
	}
	web := webSettingsOf(t, inbound)
	if web.Host != "se.gateway.example.org" || web.PublicIP != "203.0.113.10" || !telemt.ValidWebBasePath(web.Path) || len(web.Path) != 24 {
		t.Fatalf("web = %+v", web)
	}

	// A kept path and an entered address stay; the DNS is not asked.
	again := &model.Inbound{Protocol: model.MTProto, Settings: inbound.Settings}
	if err := prepareMTProtoWeb(again, map[string]bool{"elsewhere": true}, lookup); err != nil {
		t.Fatal(err)
	}
	if got := webSettingsOf(t, again); got != web || lookups != 1 {
		t.Fatalf("a valid WEB proxy must be kept: %+v (lookups %d)", got, lookups)
	}

	// A path another inbound uses, as a clone's would be, is replaced.
	clone := &model.Inbound{Protocol: model.MTProto, Settings: inbound.Settings}
	if err := prepareMTProtoWeb(clone, map[string]bool{web.Path: true}, lookup); err != nil {
		t.Fatal(err)
	}
	if got := webSettingsOf(t, clone); got.Path == web.Path || !telemt.ValidWebBasePath(got.Path) {
		t.Fatalf("a taken path must be replaced, got %q", got.Path)
	}
}

func TestPrepareMTProtoWebLeavesAnInboundWithoutWebAlone(t *testing.T) {
	fail := func(string) (string, error) { return "", errors.New("must not be called") }
	for _, settings := range []string{
		`{"tlsDomain":"www.example.com","clients":[]}`,
		`{"tlsDomain":"www.example.com","web":{"enable":false,"host":"bad host","path":""},"clients":[]}`,
	} {
		inbound := &model.Inbound{Protocol: model.MTProto, Settings: settings}
		if err := prepareMTProtoWeb(inbound, nil, fail); err != nil || inbound.Settings != settings {
			t.Errorf("settings changed to %s (%v)", inbound.Settings, err)
		}
	}
}

func TestPrepareMTProtoWebRejectsBadHostsAndUnresolvableOnes(t *testing.T) {
	ok := func(string) (string, error) { return "203.0.113.10", nil }
	for _, host := range []string{"", "203.0.113.10", "localhost", "a b.com"} {
		inbound := &model.Inbound{Protocol: model.MTProto, Settings: `{"web":{"enable":true,"host":"` + host + `"},"clients":[]}`}
		if err := prepareMTProtoWeb(inbound, nil, ok); err == nil {
			t.Errorf("host %q must be refused", host)
		}
	}
	unresolved := func(string) (string, error) { return "", errors.New("no such host") }
	inbound := &model.Inbound{Protocol: model.MTProto, Settings: `{"web":{"enable":true,"host":"proxy.example.com"},"clients":[]}`}
	if err := prepareMTProtoWeb(inbound, nil, unresolved); err == nil {
		t.Error("a host that does not resolve must ask for the IP")
	}
}

func TestTelemtWebInstanceNeedsThePanelAsFront(t *testing.T) {
	t.Cleanup(func() { SetTelemtWebFront(0, false) })
	web := &mtprotoWeb{Enable: true, Host: "proxy.example.com", Path: "c0ffee", PublicIP: "203.0.113.10"}
	SetTelemtWebFront(4321, false)
	if _, ok := telemtWebInstance(web); ok {
		t.Fatal("without HTTPS on 443 the WEB proxy must stay off")
	}
	SetTelemtWebFront(0, true)
	if _, ok := telemtWebInstance(web); ok {
		t.Fatal("without a decoy site the WEB proxy must stay off")
	}
	SetTelemtWebFront(4321, true)
	got, ok := telemtWebInstance(web)
	if !ok || *got != (telemt.WebInstance{Host: "proxy.example.com", BasePath: "c0ffee", PublicIP: "203.0.113.10", DecoyPort: 4321}) {
		t.Fatalf("got %+v %v", got, ok)
	}
}

func TestCheckSingleMTProtoInbound(t *testing.T) {
	dir := t.TempDir()
	if err := database.InitDB(filepath.Join(dir, "space-ui.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })
	db := database.GetDB()

	if err := checkSingleMTProtoInbound(0); err != nil {
		t.Fatalf("no mtproto inbound yet: %v", err)
	}
	if err := db.Create(&model.Inbound{Tag: "vless-1", Port: 1001, Protocol: model.VLESS}).Error; err != nil {
		t.Fatal(err)
	}
	if err := checkSingleMTProtoInbound(0); err != nil {
		t.Fatalf("other protocols do not count: %v", err)
	}
	mtproto := &model.Inbound{Tag: "mtproto-1", Port: 1002, Protocol: model.MTProto}
	if err := db.Create(mtproto).Error; err != nil {
		t.Fatal(err)
	}
	if err := checkSingleMTProtoInbound(0); !errors.Is(err, errSecondMTProtoInbound) {
		t.Fatalf("a second mtproto inbound must be refused, got %v", err)
	}
	if err := checkSingleMTProtoInbound(mtproto.Id); err != nil {
		t.Fatalf("the mtproto inbound itself must stay editable: %v", err)
	}
}

func TestTelemtStatusAndControls(t *testing.T) {
	dir := t.TempDir()
	if err := database.InitDB(filepath.Join(dir, "space-ui.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.CloseDB() })
	t.Cleanup(telemtManager().Start)
	db := database.GetDB()
	s := &TelemtService{}

	if st := s.Status(); st != nil {
		t.Fatalf("without an mtproto inbound there is no status, got %+v", st)
	}
	if err := s.StopTelemt(); err == nil {
		t.Fatal("stopping without an mtproto inbound must fail")
	}
	if err := s.RestartTelemt(); err == nil {
		t.Fatal("restarting without an mtproto inbound must fail")
	}

	inbound := &model.Inbound{Tag: "mtproto-1", Port: 1002, Protocol: model.MTProto}
	if err := db.Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	// Tests run without the telemt binary.
	st := s.Status()
	if st == nil || st.State != Error || st.ErrorMsg != errTelemtMissing {
		t.Fatalf("a missing binary must show as an error, got %+v", st)
	}
	if err := s.RestartTelemt(); err == nil || !strings.Contains(err.Error(), "turned off") {
		t.Fatalf("a disabled mtproto inbound must not restart, got %v", err)
	}
	if err := db.Model(inbound).Update("enable", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.RestartTelemt(); err == nil || err.Error() != errTelemtMissing {
		t.Fatalf("restarting without the binary must say so, got %v", err)
	}
	if err := s.StopTelemt(); err != nil {
		t.Fatal(err)
	}
	if !telemtManager().Stopped() {
		t.Fatal("StopTelemt must hold the proxy down")
	}
}

func TestProcessMemory(t *testing.T) {
	if processMemory(0) != 0 || processMemory(-1) != 0 {
		t.Fatal("no process has no memory")
	}
	if processMemory(os.Getpid()) == 0 {
		t.Fatal("the test process must report its memory")
	}
}
