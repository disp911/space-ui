package sub

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/disp911/space-ui/v3/database/model"
	"github.com/xtls/xray-core/infra/conf"
)

// An XHTTP inbound behind the website, on a loopback address without TLS,
// reached through a CDN: the JSON subscription gives the client the client's
// side of the XHTTP settings and, per External Proxy row, that row's TLS.
func TestJSONSubscriptionOfAnXhttpInboundBehindACDN(t *testing.T) {
	s := NewSubJsonService("", "", "", "", NewSubService(false, "-ieo"))
	ib := &model.Inbound{
		Protocol: model.VLESS, Listen: "127.0.0.1", Port: 10001, Remark: "cdn",
		Settings: `{"clients":[{"id":"00000000-0000-0000-0000-000000000000","email":"t"}],"decryption":"none","encryption":"none"}`,
		StreamSettings: `{"network":"xhttp","security":"none","xhttpSettings":` + cdnXhttp + `,
			"sockopt":{"trustedXForwardedFor":["X-Forwarded-For"]},
			"externalProxy":[
				{"forceTls":"tls","dest":"188.72.103.4","port":443,"remark":"cdn","sni":"cdn.example.com","fingerprint":"chrome","alpn":["h2","http/1.1"]},
				{"forceTls":"same","dest":"192.168.1.10","port":8443,"remark":"same"}]}`,
	}
	configs := s.getConfig(ib, model.Client{ID: "00000000-0000-0000-0000-000000000000", Email: "t"}, "panel.example.com")
	if len(configs) != 2 {
		t.Fatalf("%d configs", len(configs))
	}
	type outbound struct {
		Settings       map[string]any `json:"settings"`
		StreamSettings map[string]any `json:"streamSettings"`
	}
	streamOf := func(raw []byte) map[string]any {
		var cfg struct {
			Outbounds []json.RawMessage `json:"outbounds"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		// The client's Xray must take the outbound as it is.
		detour := new(conf.OutboundDetourConfig)
		if err := json.Unmarshal(cfg.Outbounds[0], detour); err != nil {
			t.Fatal(err)
		}
		if _, err := detour.Build(); err != nil {
			t.Fatalf("the client's Xray refuses the outbound: %v", err)
		}
		var out outbound
		json.Unmarshal(cfg.Outbounds[0], &out)
		return out.StreamSettings
	}

	cdn := streamOf(configs[0])
	if cdn["security"] != "tls" {
		t.Fatalf("security %v", cdn["security"])
	}
	tls, _ := cdn["tlsSettings"].(map[string]any)
	if tls["serverName"] != "cdn.example.com" || tls["fingerprint"] != "chrome" || !reflect.DeepEqual(tls["alpn"], []any{"h2", "http/1.1"}) {
		t.Errorf("tls %v", tls)
	}
	xhttp, _ := cdn["xhttpSettings"].(map[string]any)
	for _, k := range []string{"scMaxBufferedPosts", "scStreamUpServerSecs", "noSSEHeader", "serverMaxHeaderBytes", "h2Params"} {
		if _, ok := xhttp[k]; ok {
			t.Errorf("the server's own %s reached the client", k)
		}
	}
	for k, want := range map[string]any{"path": "/media/v2/chunks/upload", "host": "cdn.example.com", "mode": "packet-up",
		"uplinkHTTPMethod": "GET", "sessionIDTable": "base36", "uplinkDataPlacement": "header", "scMaxEachPostBytes": "32768"} {
		if xhttp[k] != want {
			t.Errorf("%s = %v, want %v", k, xhttp[k], want)
		}
	}
	if _, ok := xhttp["xmux"].(map[string]any); !ok {
		t.Error("XMUX did not reach the client")
	}
	if _, ok := cdn["sockopt"]; ok {
		t.Error("the inbound's sockopt reached the client")
	}

	// The second row keeps the inbound's security: the first row's TLS
	// stays with the first row. (Xray takes VLESS without TLS only to a
	// private address, hence the row's.)
	if same := streamOf(configs[1]); same["security"] != "none" || same["tlsSettings"] != nil {
		t.Errorf("second row %v / %v", same["security"], same["tlsSettings"])
	}
}
