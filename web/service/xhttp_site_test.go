package service

import (
	"encoding/json"
	"testing"

	"github.com/disp911/spacex-ui/v2/database"
	"github.com/disp911/spacex-ui/v2/database/model"
)

func streamOf(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestSiteXhttpPath(t *testing.T) {
	const headers = `{"network":"xhttp","security":"none","xhttpSettings":{"path":"/media/v2/chunks/upload","sessionIDPlacement":"header","seqPlacement":"header"}}`
	for _, c := range []struct {
		name, listen, stream, want string
	}{
		{"header placements keep the path as it is", "127.0.0.1", headers, "/media/v2/chunks/upload"},
		{"localhost", "localhost", headers, "/media/v2/chunks/upload"},
		{"IPv6 loopback", "::1", headers, "/media/v2/chunks/upload"},
		{"a session in the path gets the slash", "127.0.0.1", `{"network":"xhttp","xhttpSettings":{"path":"/x"}}`, "/x/"},
		{"no leading slash, query dropped", "127.0.0.1", `{"network":"xhttp","xhttpSettings":{"path":"x/?ed=1"}}`, "/x/"},
		{"all addresses", "", headers, ""},
		{"public address", "0.0.0.0", headers, ""},
		{"unix socket", "/run/x.sock", headers, ""},
		{"TLS", "127.0.0.1", `{"network":"xhttp","security":"tls","xhttpSettings":{"path":"/x"}}`, ""},
		{"not XHTTP", "127.0.0.1", `{"network":"ws","wsSettings":{"path":"/x"}}`, ""},
		{"root path", "127.0.0.1", `{"network":"xhttp","xhttpSettings":{"path":"/"}}`, ""},
	} {
		got, ok := siteXhttpPath(c.listen, streamOf(t, c.stream))
		if got != c.want || ok != (c.want != "") {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, ok, c.want)
		}
	}
}

func TestPathsOverlap(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"/secret/", "/secret/", true},
		{"/sec", "/secret/", true},
		{"/secret/xhttp/", "/secret/", true},
		{"/media/", "/secret/", false},
		{"/subscribe/", "/sub/", false},
	} {
		if got := pathsOverlap(c.a, c.b); got != c.want {
			t.Errorf("pathsOverlap(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

func TestXhttpSiteRouteTakesTheLongestPath(t *testing.T) {
	routes := []xhttpSiteRoute{{"/media/v2/", 2}, {"/media/", 1}}
	xhttpSiteRoutes.Store(&routes)
	t.Cleanup(func() { xhttpSiteRoutes.Store(nil) })
	for p, want := range map[string]int{"/media/v2/up": 2, "/media/x": 1, "/other": 0} {
		if got, _ := XhttpSiteRoute(p); got != want {
			t.Errorf("XhttpSiteRoute(%q) = %d, want %d", p, got, want)
		}
	}
}

func TestTrustSiteForwardedFor(t *testing.T) {
	served := streamOf(t, `{"network":"xhttp","xhttpSettings":{"path":"/x"}}`)
	trustSiteForwardedFor("127.0.0.1", served)
	if got, _ := served["sockopt"].(map[string]any)["trustedXForwardedFor"].([]any); len(got) != 1 || got[0] != "X-Forwarded-For" {
		t.Errorf("trustedXForwardedFor = %v, want X-Forwarded-For", got)
	}
	own := streamOf(t, `{"network":"xhttp","xhttpSettings":{"path":"/x"},"sockopt":{"trustedXForwardedFor":["CF-Connecting-IP"]}}`)
	trustSiteForwardedFor("127.0.0.1", own)
	if got := own["sockopt"].(map[string]any)["trustedXForwardedFor"].([]any); got[0] != "CF-Connecting-IP" {
		t.Errorf("an inbound's own list was replaced: %v", got)
	}
	public := streamOf(t, `{"network":"xhttp","xhttpSettings":{"path":"/x"}}`)
	trustSiteForwardedFor("0.0.0.0", public)
	if _, ok := public["sockopt"]; ok {
		t.Error("an inbound the website does not serve must not trust X-Forwarded-For")
	}
}

func TestGuardSiteXhttpPath(t *testing.T) {
	siteTestDB(t)
	inbounds := InboundService{}
	xhttp := func(id int, path string) *model.Inbound {
		return &model.Inbound{Id: id, Enable: true, Protocol: model.VLESS, Listen: "127.0.0.1", Port: 10000 + id, Remark: "cdn",
			StreamSettings: `{"network":"xhttp","security":"none","xhttpSettings":{"path":"` + path + `"}}`}
	}
	if err := inbounds.guardSiteXhttpPath(xhttp(0, "/media/v2/upload")); err != nil {
		t.Errorf("a free path refused: %v", err)
	}
	for _, path := range []string{"/secret/x", "/sec", "/sub/x"} {
		if err := inbounds.guardSiteXhttpPath(xhttp(0, path)); err == nil {
			t.Errorf("%s overlaps a path the panel keeps, yet was taken", path)
		}
	}
	if err := database.GetDB().Create(xhttp(1, "/media/")).Error; err != nil {
		t.Fatal(err)
	}
	if err := inbounds.guardSiteXhttpPath(xhttp(0, "/media/v2/upload")); err == nil {
		t.Error("a path under another inbound's was taken")
	}
	if err := inbounds.guardSiteXhttpPath(xhttp(1, "/media/v3/")); err != nil {
		t.Errorf("an inbound clashed with itself: %v", err)
	}
	public := xhttp(0, "/secret/x")
	public.Listen = ""
	if err := inbounds.guardSiteXhttpPath(public); err != nil {
		t.Errorf("an inbound the website does not serve was checked: %v", err)
	}
}
