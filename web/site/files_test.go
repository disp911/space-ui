package site

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

var modTime = time.Date(2025, 3, 14, 9, 26, 53, 0, time.UTC)

func testSite() fstest.MapFS {
	long := strings.Repeat("<p>Lorem ipsum dolor sit amet.</p>\n", 40)
	return fstest.MapFS{
		"index.html":        {Data: []byte("<h1>Home</h1>" + long), ModTime: modTime},
		"docs/index.html":   {Data: []byte("<h1>Docs</h1>"), ModTime: modTime},
		"docs/intro.html":   {Data: []byte("<h1>Intro</h1>"), ModTime: modTime},
		"logo.png":          {Data: bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 300), ModTime: modTime},
		"404.html":          {Data: []byte("<h1>Lost</h1>"), ModTime: modTime},
		".git/config":       {Data: []byte("secret"), ModTime: modTime},
		"empty/placeholder": {Data: []byte("x"), ModTime: modTime},
	}
}

func get(t *testing.T, h http.Handler, method, target string, header map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFilesServesPagesAndFolders(t *testing.T) {
	h := Files(testSite())
	cases := []struct {
		path, want string
	}{
		{"/", "<h1>Home</h1>"},
		{"/index.html", "<h1>Home</h1>"},
		{"/docs/", "<h1>Docs</h1>"},
		{"/docs/intro.html", "<h1>Intro</h1>"},
	}
	for _, c := range cases {
		resp := get(t, h, http.MethodGet, c.path, nil)
		if resp.StatusCode != http.StatusOK || !strings.HasPrefix(body(t, resp), c.want) {
			t.Errorf("%s: got %d", c.path, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("%s: Content-Type %q", c.path, ct)
		}
	}
}

func TestFilesRedirectsToTheCanonicalPath(t *testing.T) {
	// A folder takes its slash and a file loses one, as with Caddy: 308,
	// the query kept, to any method, and no Vary.
	for _, c := range []struct{ method, path, to string }{
		{http.MethodGet, "/docs?v=2", "/docs/?v=2"},
		{http.MethodPost, "/docs", "/docs/"},
		{http.MethodGet, "/docs/intro.html/", "/docs/intro.html"},
	} {
		resp := get(t, Files(testSite()), c.method, c.path, nil)
		if resp.StatusCode != http.StatusPermanentRedirect || resp.Header.Get("Location") != c.to || resp.Header.Get("Vary") != "" {
			t.Errorf("%s %s: got %d to %q", c.method, c.path, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
}

func TestFilesAnswersMissingWithTheSitesOwnPage(t *testing.T) {
	h := Files(testSite())
	// A folder without an index is as missing as a file that is not there,
	// and hidden files are never served.
	for _, p := range []string{"/nope", "/docs/nope.html", "/empty/", "/.git/config", "/docs/../.git/config"} {
		resp := get(t, h, http.MethodGet, p, nil)
		if resp.StatusCode != http.StatusNotFound || body(t, resp) != "<h1>Lost</h1>" {
			t.Errorf("%s: got %d", p, resp.StatusCode)
		}
	}
	// The page is served as a file, to any method, and never compressed.
	for _, m := range []string{http.MethodGet, http.MethodPut, http.MethodPost} {
		resp := get(t, h, m, "/nope", map[string]string{"Accept-Encoding": "gzip"})
		hd := resp.Header
		if resp.StatusCode != http.StatusNotFound || hd.Get("Etag") == "" || hd.Get("Last-Modified") == "" || hd.Get("Accept-Ranges") != "bytes" ||
			hd.Get("Vary") != "Accept-Encoding" || hd.Get("Content-Encoding") != "" || body(t, resp) != "<h1>Lost</h1>" {
			t.Errorf("%s: got %d %v", m, resp.StatusCode, hd)
		}
	}
}

func TestFilesAnswersMissingPlainlyWithoutAPage(t *testing.T) {
	fsys := testSite()
	delete(fsys, "404.html")
	resp := get(t, Files(fsys), http.MethodGet, "/nope", nil)
	// Caddy's file server without an error page: the status and nothing more.
	if b := body(t, resp); resp.StatusCode != http.StatusNotFound || b != "" || len(resp.Header) != 0 {
		t.Fatalf("got %d %q %v", resp.StatusCode, b, resp.Header)
	}
}

func TestFilesRefusesOtherMethods(t *testing.T) {
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions} {
		resp := get(t, Files(testSite()), m, "/", nil)
		if resp.StatusCode != http.StatusMethodNotAllowed || resp.Header.Get("Allow") != "GET, HEAD" {
			t.Errorf("%s: got %d, Allow %q", m, resp.StatusCode, resp.Header.Get("Allow"))
		}
		// Caddy's file server says no more than the status.
		if b := body(t, resp); b != "" || resp.Header.Get("Content-Type") != "" || resp.Header.Get("Content-Length") != "" || resp.Header.Get("Vary") != "Accept-Encoding" {
			t.Errorf("%s: body %q, headers %v", m, b, resp.Header)
		}
	}
	// What is not there is missing to every method.
	if resp := get(t, Files(testSite()), http.MethodPut, "/x", nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("PUT /x: got %d", resp.StatusCode)
	}
}

func TestFilesHeadHasNoBody(t *testing.T) {
	resp := get(t, Files(testSite()), http.MethodHead, "/docs/intro.html", nil)
	if resp.StatusCode != http.StatusOK || body(t, resp) != "" || resp.Header.Get("Content-Length") != "14" {
		t.Fatalf("got %d, length %q", resp.StatusCode, resp.Header.Get("Content-Length"))
	}
	// A body that would be compressed is announced so, without a length.
	resp = get(t, Files(testSite()), http.MethodHead, "/", map[string]string{"Accept-Encoding": "gzip"})
	if resp.StatusCode != http.StatusOK || body(t, resp) != "" || resp.Header.Get("Content-Encoding") != "gzip" || resp.Header.Get("Content-Length") != "" {
		t.Fatalf("gzip: got %d %v", resp.StatusCode, resp.Header)
	}
}

func TestFilesRevisitsGetNotModified(t *testing.T) {
	h := Files(testSite())
	first := get(t, h, http.MethodGet, "/logo.png", nil)
	etag := first.Header.Get("ETag")
	if etag != `"d8fvrbx09k3k-xc"` {
		t.Fatalf("ETag %q", etag)
	}
	if lm := first.Header.Get("Last-Modified"); lm != "Fri, 14 Mar 2025 09:26:53 GMT" {
		t.Fatalf("Last-Modified %q", lm)
	}
	if resp := get(t, h, http.MethodGet, "/logo.png", map[string]string{"If-None-Match": etag}); resp.StatusCode != http.StatusNotModified {
		t.Errorf("If-None-Match: got %d", resp.StatusCode)
	}
	if resp := get(t, h, http.MethodGet, "/logo.png", map[string]string{"If-Modified-Since": "Fri, 14 Mar 2025 09:26:53 GMT"}); resp.StatusCode != http.StatusNotModified {
		t.Errorf("If-Modified-Since: got %d", resp.StatusCode)
	}
}

func TestFilesServesRanges(t *testing.T) {
	resp := get(t, Files(testSite()), http.MethodGet, "/logo.png", map[string]string{"Range": "bytes=4-7", "Accept-Encoding": "gzip"})
	if resp.StatusCode != http.StatusPartialContent || body(t, resp) != "\x89PNG" || resp.Header.Get("Content-Encoding") != "" {
		t.Fatalf("got %d", resp.StatusCode)
	}
}

func TestFilesCompressesTextForGzipClients(t *testing.T) {
	h := Files(testSite())
	resp := get(t, h, http.MethodGet, "/", map[string]string{"Accept-Encoding": "gzip, deflate, br"})
	if resp.Header.Get("Content-Encoding") != "gzip" || resp.Header.Get("Vary") != "Accept-Encoding" {
		t.Fatalf("headers %v", resp.Header)
	}
	if etag := resp.Header.Get("ETag"); !strings.HasSuffix(etag, `-gzip"`) {
		t.Errorf("ETag %q does not name the encoding", etag)
	}
	if resp.Header.Get("Accept-Ranges") != "" {
		t.Errorf("ranges offered of a compressed body")
	}
	if again := get(t, h, http.MethodGet, "/", map[string]string{"Accept-Encoding": "gzip", "If-None-Match": resp.Header.Get("ETag")}); again.StatusCode != http.StatusNotModified {
		t.Errorf("revisit: got %d", again.StatusCode)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	if !strings.HasPrefix(string(plain), "<h1>Home</h1>") {
		t.Errorf("unpacked %q", plain[:20])
	}

	// Pictures and small pages go as they are, and so does everything for
	// a client that takes no gzip or opens a WebSocket; Vary is there all
	// the same.
	for _, c := range []struct{ path, enc, ws string }{{"/logo.png", "gzip", ""}, {"/docs/intro.html", "gzip", ""}, {"/", "", ""}, {"/", "gzip;q=0", ""}, {"/", "gzip", "x"}} {
		resp := get(t, h, http.MethodGet, c.path, map[string]string{"Accept-Encoding": c.enc, "Sec-WebSocket-Key": c.ws})
		if resp.Header.Get("Content-Encoding") != "" || resp.Header.Get("Vary") != "Accept-Encoding" {
			t.Errorf("%s with %q: %v", c.path, c.enc, resp.Header)
		}
	}
	// A weight is read as Caddy reads it.
	if again := get(t, h, http.MethodGet, "/", map[string]string{"Accept-Encoding": "gzip;q=0.5", "If-None-Match": resp.Header.Get("ETag")}); again.StatusCode != http.StatusNotModified {
		t.Errorf("revisit with a weight: got %d", again.StatusCode)
	}
}

func TestFilesSendsNoServerHeader(t *testing.T) {
	resp := get(t, Files(testSite()), http.MethodGet, "/", nil)
	if s := resp.Header.Get("Server"); s != "" {
		t.Fatalf("Server %q", s)
	}
}

func TestSourcePrefersTheAdministratorsSite(t *testing.T) {
	made := fstest.MapFS{"index.html": {Data: []byte("made"), ModTime: modTime}}
	dir := t.TempDir()
	if got, own := Source(dir, made); own {
		t.Fatal("an empty folder replaced the made site")
	} else if _, ok := got.(fstest.MapFS); !ok {
		t.Fatalf("got %T", got)
	}
	writeFile(t, dir+"/index.html", "own")
	got, own := Source(dir, made)
	if b := body(t, get(t, Files(got), http.MethodGet, "/", nil)); !own || b != "own" {
		t.Fatalf("own %v, got %q", own, b)
	}
}

func TestFeedbackKeepsNothing(t *testing.T) {
	h := Feedback("/api/feedback", Files(testSite()))
	req := httptest.NewRequest(http.MethodPost, "/api/feedback", strings.NewReader(`{"page":"/","helpful":true}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("POST: %d %q", rec.Code, rec.Body)
	}
	// Anything else at the path is the site's, as a GET of a missing page.
	if resp := get(t, h, http.MethodGet, "/api/feedback", nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET: %d", resp.StatusCode)
	}
	if resp := get(t, h, http.MethodPost, "/", nil); resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /: %d", resp.StatusCode)
	}
}
