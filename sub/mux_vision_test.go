package sub

import (
	"encoding/json"
	"testing"

	"github.com/disp911/space-ui/v3/database/model"
)

func TestVisionClientsKeepTCPOutOfMux(t *testing.T) {
	s := &SubJsonService{mux: `{"enabled":true,"concurrency":8,"xudpConcurrency":16,"xudpProxyUDP443":"reject"}`}
	ib := &model.Inbound{Protocol: model.VLESS, Port: 443, Settings: `{"encryption":"none"}`}
	muxOf := func(c model.Client) map[string]any {
		var out struct {
			Mux map[string]any `json:"mux"`
		}
		if err := json.Unmarshal(s.genVless(ib, nil, c), &out); err != nil {
			t.Fatal(err)
		}
		return out.Mux
	}

	vision := muxOf(model.Client{ID: "x", Flow: "xtls-rprx-vision"})
	if vision["concurrency"] != float64(-1) || vision["xudpConcurrency"] != float64(16) || vision["enabled"] != true {
		t.Fatalf("vision mux %v", vision)
	}
	plain := muxOf(model.Client{ID: "x"})
	if plain["concurrency"] != float64(8) {
		t.Fatalf("plain mux %v", plain)
	}

	s.mux = `{"enabled":false,"concurrency":8}`
	if off := muxOf(model.Client{ID: "x", Flow: "xtls-rprx-vision"}); off["concurrency"] != float64(8) || off["enabled"] != false {
		t.Fatalf("a mux that is off stays as it is: %v", off)
	}
}
