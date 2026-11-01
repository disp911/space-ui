package web

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// h2cBackend stands in for an XHTTP inbound: plain HTTP/1.1 and h2c, as
// Xray's XHTTP server listens.
func h2cBackend(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(h)
	srv.Config.Protocols = new(http.Protocols)
	srv.Config.Protocols.SetHTTP1(true)
	srv.Config.Protocols.SetUnencryptedHTTP2(true)
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func xhttpTo(port int) func(string) (int, bool) {
	return func(p string) (int, bool) { return port, strings.HasPrefix(p, "/media/v2/") }
}

func TestWebProxyFrontSendsXhttpPathsToXrayOverH2C(t *testing.T) {
	var seen *http.Request
	var body string
	xray := h2cBackend(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		fmt.Fprint(w, "from xray")
	}))
	front := webProxyFront{panel: panelStub, route: routeTo(0), xhttp: xhttpTo(backendPort(t, xray))}

	// A GET with a body, as CDNs that pass only GET carry the uplink.
	req := httptest.NewRequest(http.MethodGet, "https://cdn.example.com/media/v2/abc/0?x=1", strings.NewReader("packet"))
	req.RemoteAddr = "198.51.100.7:5555"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("X-Session", "s1")
	rec := httptest.NewRecorder()
	front.ServeHTTP(rec, req)

	if rec.Body.String() != "from xray" {
		t.Fatalf("answer = %q, want Xray's", rec.Body.String())
	}
	if seen.ProtoMajor != 2 {
		t.Errorf("Xray was reached over %s, want h2c", seen.Proto)
	}
	if seen.URL.Path != "/media/v2/abc/0" || seen.URL.RawQuery != "x=1" || body != "packet" {
		t.Errorf("Xray got %s?%s with %q", seen.URL.Path, seen.URL.RawQuery, body)
	}
	if seen.Host != "cdn.example.com" || seen.Header.Get("X-Session") != "s1" {
		t.Errorf("host %q, X-Session %q: the request must reach Xray unchanged", seen.Host, seen.Header.Get("X-Session"))
	}
	if got := seen.Header.Get("X-Forwarded-For"); got != "198.51.100.7" {
		t.Errorf("X-Forwarded-For = %q, want the real client only", got)
	}
}

func TestWebProxyFrontStreamsXhttpAnswers(t *testing.T) {
	release := make(chan struct{})
	xray := h2cBackend(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "first")
		w.(http.Flusher).Flush()
		<-release
		fmt.Fprint(w, "second")
	}))
	front := httptest.NewServer(webProxyFront{panel: panelStub, route: routeTo(0), xhttp: xhttpTo(backendPort(t, xray))})
	defer front.Close()

	resp, err := http.Get(front.URL + "/media/v2/abc")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 5)
	done := make(chan error, 1)
	go func() { _, err := io.ReadFull(resp.Body, buf); done <- err }()
	select {
	case err := <-done:
		if err != nil || string(buf) != "first" {
			t.Fatalf("read %q, %v", buf, err)
		}
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("the first part of a streamed answer was held back")
	}
	close(release)
	rest, _ := io.ReadAll(resp.Body)
	if string(rest) != "second" {
		t.Errorf("rest = %q", rest)
	}
}

func TestWebProxyFrontPutsTelemtBeforeXhttpAndThePanelAfter(t *testing.T) {
	telemt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "telemt") }))
	defer telemt.Close()
	xray := h2cBackend(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "xray") }))
	front := webProxyFront{
		panel: panelStub,
		route: routeTo(backendPort(t, telemt)),
		xhttp: func(string) (int, bool) { return backendPort(t, xray), true },
	}
	for path, want := range map[string]string{"/c0ffee/x": "telemt", "/media/v2/a": "xray"} {
		rec := httptest.NewRecorder()
		front.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com"+path, nil))
		if rec.Body.String() != want {
			t.Errorf("%s answered by %q, want %s", path, rec.Body.String(), want)
		}
	}
	front.xhttp = xhttpTo(backendPort(t, xray))
	rec := httptest.NewRecorder()
	front.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/media/v3/a", nil))
	if !strings.HasPrefix(rec.Body.String(), "panel ") {
		t.Errorf("an unknown path answered by %q, want the panel", rec.Body.String())
	}
}

func TestWebProxyFrontShowsThePanelWhenXrayIsDown(t *testing.T) {
	xray := h2cBackend(t, http.NotFoundHandler())
	port := backendPort(t, xray)
	xray.Close()
	front := webProxyFront{panel: panelStub, route: routeTo(0), xhttp: xhttpTo(port)}
	rec := httptest.NewRecorder()
	front.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "https://example.com/media/v2/a", nil))
	if !strings.HasPrefix(rec.Body.String(), "panel ") {
		t.Errorf("answer = %q, want the panel's", rec.Body.String())
	}
}
