package sub

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/disp911/space-ui/v3/database/model"
	"github.com/xtls/xray-core/infra/conf"
)

// The fragmentation and noises of the Formats tab, as the tab saves them.
const (
	tabFragment = `{"packets":"tlshello","length":"100-200","interval":"10-20","maxSplit":"300-400"}`
	tabNoises   = `[{"type":"rand","packet":"10-20","delay":"10-16","applyTo":"ip"},
		{"type":"base64","packet":"aGk-","delay":"5","applyTo":"ipv4"},
		{"type":"str","packet":"hello","delay":"","applyTo":"ip"}]`
)

// clientConfig builds what a JSON subscription gives a client and has the
// client's Xray take all of it.
func clientConfig(t *testing.T, s *SubJsonService, ib *model.Inbound, c model.Client) map[string]any {
	t.Helper()
	configs := s.getConfig(ib, c, "docs.example.com")
	if len(configs) == 0 {
		t.Fatal("no config")
	}
	var x conf.Config
	if err := json.Unmarshal(configs[0], &x); err != nil {
		t.Fatal(err)
	}
	if _, err := x.Build(); err != nil {
		t.Fatalf("the client's Xray refuses the config: %v", err)
	}
	var cfg map[string]any
	json.Unmarshal(configs[0], &cfg)
	return cfg
}

func TestFragmentAndNoisesGoIntoTheProxysFinalmask(t *testing.T) {
	s := NewSubJsonService(tabFragment, tabNoises, "", "", NewSubService(false, "-ieo"))
	ib := &model.Inbound{
		Protocol: model.VLESS, Port: 443, Remark: "site",
		Settings: `{"clients":[{"id":"00000000-0000-0000-0000-000000000000","email":"t","flow":"xtls-rprx-vision"}],"decryption":"none","encryption":"none"}`,
		StreamSettings: `{"network":"tcp","security":"reality","tcpSettings":{"header":{"type":"none"}},
			"realitySettings":{"serverNames":["docs.example.com"],"shortIds":["ab"],"settings":{"publicKey":"jNXHt1yRo0vDuchQlIP6Z0ZvjT3KtzVI-T4E7RoLJS0","fingerprint":"firefox"}},
			"sockopt":{"acceptProxyProtocol":false},
			"finalmask":{"tcp":[{"type":"header-custom","settings":{"clients":[[{"packet":"AQ=="}]],"servers":[[{"packet":"Ag=="}]]}}]}}`,
	}
	cfg := clientConfig(t, s, ib, model.Client{ID: "00000000-0000-0000-0000-000000000000", Email: "t", Flow: "xtls-rprx-vision"})

	outbounds, _ := cfg["outbounds"].([]any)
	for _, o := range outbounds {
		if tag := o.(map[string]any)["tag"]; tag == "direct_out" {
			t.Fatal("the freedom outbound Xray 26 deprecates is still there")
		}
	}
	stream := outbounds[0].(map[string]any)["streamSettings"].(map[string]any)
	if _, ok := stream["sockopt"]; ok {
		t.Errorf("sockopt %v: the proxy dials straight out", stream["sockopt"])
	}
	fm := stream["finalmask"].(map[string]any)

	tcp := fm["tcp"].([]any)
	if len(tcp) != 2 || tcp[0].(map[string]any)["type"] != "header-custom" {
		t.Fatalf("the inbound's own mask comes first, then ours next to the socket: %v", tcp)
	}
	frag := tcp[1].(map[string]any)
	want := map[string]any{"packets": "tlshello", "length": "100-200", "delay": "10-20", "maxSplit": "300-400"}
	if frag["type"] != "fragment" || !reflect.DeepEqual(frag["settings"], want) {
		t.Errorf("fragment %v", frag)
	}

	udp := fm["udp"].([]any)
	noise := udp[0].(map[string]any)
	items := noise["settings"].(map[string]any)["noise"]
	wantItems := []any{
		map[string]any{"rand": "10-20", "delay": "10-16"},
		map[string]any{"type": "hex", "packet": "68693e", "delay": "5"},
		map[string]any{"type": "str", "packet": "hello"},
	}
	if len(udp) != 1 || noise["type"] != "noise" || !reflect.DeepEqual(items, wantItems) {
		t.Errorf("noise %v", udp)
	}
}

func TestHysteriaTakesTheMasksToo(t *testing.T) {
	s := NewSubJsonService(tabFragment, tabNoises, "", "", NewSubService(false, "-ieo"))
	ib := &model.Inbound{
		Protocol: model.Hysteria, Port: 443, Remark: "hy",
		Settings: `{"version":2,"clients":[{"auth":"secret","email":"t"}]}`,
		StreamSettings: `{"network":"hysteria","security":"tls","tlsSettings":{"serverName":"docs.example.com","alpn":["h3"],"settings":{"fingerprint":"chrome"}},
			"hysteriaSettings":{"version":2,"finalmask":{"udp":[{"type":"salamander","settings":{"password":"p"}}]}}}`,
	}
	cfg := clientConfig(t, s, ib, model.Client{Auth: "secret", Email: "t"})
	stream := cfg["outbounds"].([]any)[0].(map[string]any)["streamSettings"].(map[string]any)
	udp := stream["finalmask"].(map[string]any)["udp"].([]any)
	if len(udp) != 2 || udp[0].(map[string]any)["type"] != "salamander" || udp[1].(map[string]any)["type"] != "noise" {
		t.Fatalf("udp masks %v", udp)
	}
}

// Without fragmentation or noises nothing is added, and the inbound's masks
// reach the client as they are.
func TestNoMasksOfOurOwnWhenTheyAreOff(t *testing.T) {
	s := NewSubJsonService("", "", "", "", NewSubService(false, "-ieo"))
	stream := map[string]any{"finalmask": map[string]any{"udp": []any{map[string]any{"type": "salamander"}}}}
	s.addMasks(stream)
	if got := stream["finalmask"].(map[string]any)["udp"].([]any); len(got) != 1 {
		t.Fatalf("udp %v", got)
	}
	if _, ok := stream["finalmask"].(map[string]any)["tcp"]; ok {
		t.Fatal("a tcp mask out of nowhere")
	}
}

func TestOurMaskSitsAboveTheMasksThatMakeTheSocket(t *testing.T) {
	own := []any{map[string]any{"type": "mkcp-aes128gcm"}, map[string]any{"type": "udphop"}}
	got := withMask(own, map[string]any{"type": "noise"})
	var types []string
	for _, m := range got {
		types = append(types, m.(map[string]any)["type"].(string))
	}
	if strings.Join(types, ",") != "mkcp-aes128gcm,noise,udphop" {
		t.Fatalf("masks %v", types)
	}
	if len(own) != 2 {
		t.Fatal("the inbound's own list was changed")
	}
}

// Every address of an inbound gets the masks once, and the inbound's stream
// is left as it was.
func TestMasksAreAddedOncePerAddress(t *testing.T) {
	s := NewSubJsonService(tabFragment, "", "", "", NewSubService(false, "-ieo"))
	ib := &model.Inbound{
		Protocol: model.VLESS, Port: 443, Remark: "ws",
		Settings: `{"clients":[{"id":"00000000-0000-0000-0000-000000000000","email":"t"}],"decryption":"none","encryption":"none"}`,
		StreamSettings: `{"network":"ws","security":"none","wsSettings":{"path":"/"},
			"finalmask":{"tcp":[{"type":"header-custom","settings":{"clients":[[{"packet":"AQ=="}]],"servers":[[{"packet":"Ag=="}]]}}]},
			"externalProxy":[{"forceTls":"same","dest":"a.example.com","port":443,"remark":"a"},{"forceTls":"same","dest":"b.example.com","port":443,"remark":"b"}]}`,
	}
	for i, raw := range s.getConfig(ib, model.Client{ID: "00000000-0000-0000-0000-000000000000", Email: "t"}, "docs.example.com") {
		var cfg struct {
			Outbounds []struct {
				StreamSettings struct {
					Finalmask struct {
						Tcp []map[string]any `json:"tcp"`
					} `json:"finalmask"`
				} `json:"streamSettings"`
			} `json:"outbounds"`
		}
		json.Unmarshal(raw, &cfg)
		if n := len(cfg.Outbounds[0].StreamSettings.Finalmask.Tcp); n != 2 {
			t.Errorf("address %d: %d tcp masks, want 2", i, n)
		}
	}
}
