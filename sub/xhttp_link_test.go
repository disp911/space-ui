package sub

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// cdnXhttp has the shape the inbound window saves for an XHTTP inbound
// behind a CDN: every client setting set, the server's own ones too.
const cdnXhttp = `{"path":"/media/v2/chunks/upload","host":"cdn.example.com","mode":"packet-up",
	"headers":{"Host":"cdn.example.com","Content-Type":"video/mp2t","X-Empty":""},
	"scMaxBufferedPosts":512,"scMaxEachPostBytes":"32768","scStreamUpServerSecs":"20-80","noSSEHeader":true,
	"serverMaxHeaderBytes":65536,"h2Params":{"MaxConcurrentStreams":250},
	"xPaddingBytes":"15-27","xPaddingObfsMode":true,"xPaddingKey":"segment","xPaddingHeader":"X-Content-Revision",
	"xPaddingPlacement":"header","xPaddingMethod":"tokenish",
	"uplinkHTTPMethod":"GET","sessionIDPlacement":"header","sessionIDKey":"X-Delivery-Session",
	"sessionIDTable":"base36","sessionIDLength":"6-9","seqPlacement":"header","seqKey":"X-Chunk-Index",
	"uplinkDataPlacement":"header","uplinkDataKey":"X-Content-Data","uplinkChunkSize":8192,
	"noGRPCHeader":false,"scMinPostsIntervalMs":"10",
	"xmux":{"maxConcurrency":"1","maxConnections":0,"cMaxReuseTimes":1000,"hMaxRequestTimes":"40-60","hMaxReusableSecs":"30-40","hKeepAlivePeriod":5}}`

func TestXhttpExtraCarriesTheClientSide(t *testing.T) {
	var xhttp map[string]any
	if err := json.Unmarshal([]byte(cdnXhttp), &xhttp); err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	json.Unmarshal([]byte(`{
		"headers":{"Content-Type":"video/mp2t"},"scMaxEachPostBytes":"32768",
		"xPaddingBytes":"15-27","xPaddingObfsMode":true,"xPaddingKey":"segment","xPaddingHeader":"X-Content-Revision",
		"xPaddingPlacement":"header","xPaddingMethod":"tokenish",
		"uplinkHTTPMethod":"GET","sessionIDPlacement":"header","sessionIDKey":"X-Delivery-Session",
		"sessionIDTable":"base36","sessionIDLength":"6-9","seqPlacement":"header","seqKey":"X-Chunk-Index",
		"uplinkDataPlacement":"header","uplinkDataKey":"X-Content-Data","uplinkChunkSize":8192,"scMinPostsIntervalMs":"10",
		"xmux":{"maxConcurrency":"1","maxConnections":0,"cMaxReuseTimes":1000,"hMaxRequestTimes":"40-60","hMaxReusableSecs":"30-40","hKeepAlivePeriod":5}}`), &want)
	if got := xhttpExtra(xhttp); !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.Marshal(got)
		t.Fatalf("extra\n got %s", gotJSON)
	}

	params := map[string]string{}
	applyShareNetworkParams(map[string]any{"network": "xhttp", "xhttpSettings": xhttp}, "xhttp", params)
	if params["path"] != "/media/v2/chunks/upload" || params["host"] != "cdn.example.com" || params["mode"] != "packet-up" || params["x_padding_bytes"] != "15-27" {
		t.Errorf("params %v", params)
	}
	var extra map[string]any
	if err := json.Unmarshal([]byte(params["extra"]), &extra); err != nil || !reflect.DeepEqual(extra, want) {
		t.Errorf("the link's extra is not xhttpExtra: %s", params["extra"])
	}

	// Defaults stay out: a plain inbound's link is what it was.
	plain := map[string]any{"path": "/", "mode": "auto", "xPaddingBytes": "100-1000", "scMaxEachPostBytes": "1000000",
		"sessionIDPlacement": "", "uplinkChunkSize": float64(0), "noGRPCHeader": false, "headers": map[string]any{}}
	if got := xhttpExtra(plain); !reflect.DeepEqual(got, map[string]any{"xPaddingBytes": "100-1000"}) {
		t.Errorf("plain extra %v", got)
	}
}

func TestExternalProxyRowsSetTheirTLS(t *testing.T) {
	params := map[string]string{"type": "xhttp", "security": "none", "path": "/x"}
	var rows []any
	json.Unmarshal([]byte(`[
		{"forceTls":"tls","dest":"188.72.103.4","port":443,"remark":"cdn","sni":"cdn.example.com","fingerprint":"chrome","alpn":["h2","http/1.1"]},
		{"forceTls":"tls","dest":"cdn2.example.com","port":443,"remark":"bare"},
		{"forceTls":"same","dest":"a.example.com","port":8443,"remark":"same","sni":"ignored.example.com"}]`), &rows)
	s := &SubService{}
	out := s.buildExternalProxyURLLinks(rows, params, "none",
		func(dest string, port int) string { return "vless://id@" + dest + ":" + strconv.Itoa(port) },
		func(ep map[string]any) string { return ep["remark"].(string) })
	links := strings.Split(out, "\n")
	if len(links) != 3 {
		t.Fatalf("links %q", out)
	}
	q := func(link string) url.Values {
		u, err := url.Parse(link)
		if err != nil {
			t.Fatal(err)
		}
		return u.Query()
	}
	cdn := q(links[0])
	if cdn.Get("security") != "tls" || cdn.Get("sni") != "cdn.example.com" || cdn.Get("fp") != "chrome" || cdn.Get("alpn") != "h2,http/1.1" {
		t.Errorf("cdn row %s", links[0])
	}
	if bare := q(links[1]); bare.Get("security") != "tls" || bare.Has("sni") || bare.Has("fp") {
		t.Errorf("a row without its own TLS %s", links[1])
	}
	if same := q(links[2]); same.Get("security") != "none" || same.Has("sni") {
		t.Errorf("a row without Force TLS takes no SNI of its own: %s", links[2])
	}
	if _, ok := params["sni"]; ok {
		t.Error("one row's TLS leaked into the shared params")
	}
}
