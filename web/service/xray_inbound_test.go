package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/op/go-logging"

	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"
)

// TestXrayInboundIsTheSameForTheAPI checks the inbound Xray gets, by a
// restart or through the API alike: only the clients that are on, with the
// fields Xray reads, and a transport without the panel's own fields that
// trusts X-Forwarded-For where the website serves it.
func TestXrayInboundIsTheSameForTheAPI(t *testing.T) {
	t.Setenv("SPACE_UI_LOG_FOLDER", t.TempDir())
	logger.InitLogger(logging.CRITICAL)
	t.Cleanup(logger.CloseLogger)

	ib := &model.Inbound{
		Listen:   "127.0.0.1",
		Port:     8443,
		Protocol: model.VLESS,
		Settings: `{"clients":[
			{"id":"a","email":"on","flow":"xtls-rprx-vision-udp443","limitIp":2,"subId":"s1","enable":true},
			{"id":"b","email":"used-up","enable":true},
			{"id":"c","email":"off","enable":false}],"decryption":"none"}`,
		StreamSettings: `{"network":"xhttp","security":"none","xhttpSettings":{"path":"/media/"},
			"externalProxy":[{"forceTls":"tls","dest":"cdn.example.net","port":443}]}`,
	}
	got, err := xrayInbound(ib, map[string]bool{"on": true, "used-up": false})
	if err != nil {
		t.Fatal(err)
	}

	var settings struct {
		Clients []map[string]any `json:"clients"`
	}
	if err := json.Unmarshal([]byte(got.Settings), &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.Clients) != 1 || settings.Clients[0]["email"] != "on" {
		t.Fatalf("clients %v, want only the one that is on", settings.Clients)
	}
	c := settings.Clients[0]
	if c["flow"] != "xtls-rprx-vision" || c["limitIp"] != nil || c["subId"] != nil || c["enable"] != nil {
		t.Errorf("client %v, want Vision without the panel's fields", c)
	}

	var stream map[string]any
	if err := json.Unmarshal([]byte(got.StreamSettings), &stream); err != nil {
		t.Fatal(err)
	}
	if _, ok := stream["externalProxy"]; ok {
		t.Error("External Proxy reached Xray")
	}
	trusted, _ := stream["sockopt"].(map[string]any)["trustedXForwardedFor"].([]any)
	if len(trusted) != 1 || trusted[0] != "X-Forwarded-For" {
		t.Errorf("trustedXForwardedFor %v, want X-Forwarded-For for an inbound the website serves", trusted)
	}
	if !strings.Contains(ib.StreamSettings, "externalProxy") || !strings.Contains(ib.Settings, "used-up") {
		t.Error("the inbound given was changed")
	}
}
