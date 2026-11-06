package web

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func named(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s %s", name, r.URL.Path)
	})
}

func TestSiteFrontRoutesByPath(t *testing.T) {
	SetSubFront(named("sub"), []string{"/sub/", "/json/"})
	defer SetSubFront(nil, nil)
	front := siteFront{basePath: "/s3cret/", panel: named("panel"), site: named("site")}
	for path, want := range map[string]string{
		"/":                    "site",
		"/docs/intro.html":     "site",
		"/s3cret/":             "panel",
		"/s3cret":              "site",
		"/s3cret/panel/api/x":  "panel",
		"/s3cretive":           "site",
		"/sub/abc":             "sub",
		"/sub/assets/app.js":   "sub",
		"/json/abc":            "sub",
		"/subscribe":           "site",
		"/clash/abc":           "site",
		"/.well-known/x":       "site",
		"/s3cret/../sub/abc":   "panel",
		"/favicon.ico":         "site",
		"/robots.txt":          "site",
		"/sub":                 "site",
		"/json":                "site",
		"/s3cret/assets/x.css": "panel",
	} {
		rec := httptest.NewRecorder()
		front.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if got, _, _ := strings.Cut(rec.Body.String(), " "); got != want {
			t.Errorf("%s: answered by %s, want %s", path, got, want)
		}
	}

	// Without the subscriptions on the website their paths are the site's.
	SetSubFront(nil, nil)
	rec := httptest.NewRecorder()
	front.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sub/abc", nil))
	if !strings.HasPrefix(rec.Body.String(), "site ") {
		t.Errorf("/sub/abc without subscriptions: %q", rec.Body.String())
	}
}

func TestSiteRedirect(t *testing.T) {
	for _, c := range []struct{ host, uri, want string }{
		{"example.com", "/docs/?q=1", "https://example.com/docs/?q=1"},
		{"example.com:80", "/", "https://example.com/"},
		{"", "/x", "https://fallback.org/x"},
	} {
		req := httptest.NewRequest(http.MethodGet, c.uri, nil)
		req.Host = c.host
		rec := httptest.NewRecorder()
		siteRedirect("fallback.org").ServeHTTP(rec, req)
		if rec.Code != http.StatusPermanentRedirect || rec.Header().Get("Location") != c.want {
			t.Errorf("%s%s: %d to %q", c.host, c.uri, rec.Code, rec.Header().Get("Location"))
		}
	}
}

// selfSigned makes a certificate for example.com and a pool that trusts it.
func selfSigned(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "example.com"},
		DNSNames:     []string{"example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, pool
}

// The website is served the way startSite serves it: http.Server.Serve on a
// TLS listener whose ALPN offers h2 first.
func TestSiteListenerSpeaksHTTP2(t *testing.T) {
	cert, pool := selfSigned(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12, NextProtos: []string{"h2", "http/1.1", "acme-tls/1"}}
	srv := &http.Server{Handler: named("site"), ErrorLog: quietLog}
	go srv.Serve(tls.NewListener(ln, cfg))
	defer srv.Close()
	addr := ln.Addr().String()

	client := func(h2 bool) *http.Client {
		tr := &http.Transport{
			TLSClientConfig:   &tls.Config{ServerName: "example.com", RootCAs: pool},
			ForceAttemptHTTP2: h2,
		}
		if !h2 {
			tr.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
		}
		return &http.Client{Transport: tr}
	}
	for _, h2 := range []bool{true, false} {
		resp, err := client(h2).Get("https://" + addr + "/docs/")
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		want := 1
		if h2 {
			want = 2
		}
		if resp.ProtoMajor != want || string(b) != "site /docs/" {
			t.Errorf("h2 %v: %s, %q", h2, resp.Proto, b)
		}
	}

	// Plain HTTP on the TLS port gets the standard library's refusal, as
	// from any server written in Go, not the panel's redirect.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, _ := bufio.NewReader(conn).ReadString('\n')
	if !strings.HasPrefix(line, "HTTP/1.0 400 Bad Request") {
		t.Errorf("plain HTTP: %q", line)
	}
}

func TestSiteSocketTakesTheAddressFromXray(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "site.sock")
	ln, err := siteSocket(sock)
	if err != nil {
		t.Skip("no unix sockets here:", err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.RemoteAddr)
	})}
	go srv.Serve(ln)
	defer srv.Close()

	ask := func(header string) string {
		c, err := net.Dial("unix", sock)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		io.WriteString(c, header+"GET / HTTP/1.1\r\nHost: docs.example.com\r\nConnection: close\r\n\r\n")
		res, err := http.ReadResponse(bufio.NewReader(c), nil)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		return string(body)
	}
	if got := ask("PROXY TCP4 203.0.113.7 198.51.100.1 51000 443\r\n"); got != "203.0.113.7:51000" {
		t.Fatalf("with a header the visitor is %q", got)
	}
	if got := ask(""); strings.HasPrefix(got, "203.0.113.7") {
		t.Fatalf("without a header the visitor is %q", got)
	}
}

func TestSiteAnswersAsCaddy(t *testing.T) {
	rec := httptest.NewRecorder()
	asCaddy(siteRedirect("example.com")).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.com/docs/?a=1", nil))
	h := rec.Header()
	if rec.Code != http.StatusPermanentRedirect || h.Get("Location") != "https://example.com/docs/?a=1" || h.Get("Server") != "Caddy" || rec.Body.Len() != 0 {
		t.Fatalf("%d %v %q", rec.Code, h, rec.Body)
	}
}

func TestSiteServerHasCaddysLimits(t *testing.T) {
	s := caddyServer(named("site"))
	if s.ReadHeaderTimeout != time.Minute || s.IdleTimeout != 5*time.Minute || s.MaxHeaderBytes != 16<<10 || s.ErrorLog != quietLog {
		t.Fatalf("%v %v %d", s.ReadHeaderTimeout, s.IdleTimeout, s.MaxHeaderBytes)
	}
}
