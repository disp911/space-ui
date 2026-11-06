// Package site is the website the panel shows on port 443 to everyone who is
// not one of its clients: the files of the site, the certificates it is
// served with and the check that it is up. It answers the way an ordinary web
// server does, so nothing about it tells a visitor or a scanner what else
// runs behind the domain.
package site

import (
	"bytes"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/gzip"
)

// Files serves a website from fsys as Caddy's file_server with encode gzip
// and 404.html as its error page serves one, answer for answer:
//   - a file at its path, a folder by its index.html; a folder asked for
//     without its slash, or a file with one, is redirected with 308;
//   - anything missing, a folder without an index or a hidden file is
//     404.html with status 404, or an empty 404 when there is none;
//   - a file is given to GET and HEAD only, others get an empty 405;
//   - every file carries an ETag and its modification time, so revisits
//     get 304, and Vary: Accept-Encoding;
//   - text over 512 bytes is compressed for clients that take gzip.
func Files(fsys fs.FS) http.Handler {
	return &files{fsys: fsys}
}

type files struct {
	fsys fs.FS
}

func (h *files) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name, ok := fileName(r.URL.Path)
	if !ok {
		h.notFound(w, r)
		return
	}
	info, err := fs.Stat(h.fsys, name)
	if err != nil {
		h.notFound(w, r)
		return
	}
	index := false
	if info.IsDir() {
		page := path.Join(name, "index.html")
		if pi, err := fs.Stat(h.fsys, page); err == nil && !pi.IsDir() {
			name, info, index = page, pi, true
		}
	}
	if !info.Mode().IsRegular() {
		h.notFound(w, r)
		return
	}
	switch slash := strings.HasSuffix(r.URL.Path, "/"); {
	case index && !slash:
		redirect(w, r, r.URL.Path+"/")
		return
	case !index && slash:
		redirect(w, r, strings.TrimSuffix(r.URL.Path, "/"))
		return
	}
	// Set before anything can go wrong, as Caddy's file server sets it.
	w.Header().Add("Vary", "Accept-Encoding")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	h.send(w, r, name, info, 0)
}

// fileName turns a request path into a name in the site's files: "." for
// the root, nothing for a path that reaches for a hidden file.
func fileName(urlPath string) (string, bool) {
	clean := path.Clean("/" + urlPath)
	for _, part := range strings.Split(clean, "/") {
		if strings.HasPrefix(part, ".") {
			return "", false
		}
	}
	if clean == "/" {
		return ".", true
	}
	return strings.TrimPrefix(clean, "/"), true
}

// redirect sends the visitor on to the canonical path, its query kept, as
// Caddy's file server does.
func redirect(w http.ResponseWriter, r *http.Request, to string) {
	for strings.HasPrefix(to, "//") {
		to = strings.TrimPrefix(to, "/")
	}
	if r.URL.RawQuery != "" {
		to += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, to, http.StatusPermanentRedirect)
}

// notFound is the site's 404.html served as a file with status 404, to any
// method and never compressed, as Caddy's handle_errors does with it, or
// an empty 404 when the site has no such page.
func (h *files) notFound(w http.ResponseWriter, r *http.Request) {
	info, err := fs.Stat(h.fsys, "404.html")
	if err != nil || !info.Mode().IsRegular() {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Add("Vary", "Accept-Encoding")
	h.send(w, r, "404.html", info, http.StatusNotFound)
}

// send serves a file with its ETag, type and time through http.ServeContent,
// compressed when the client takes gzip, or with status in place of any
// ServeContent would answer.
func (h *files) send(w http.ResponseWriter, r *http.Request, name string, info fs.FileInfo, status int) {
	f, err := h.fsys.Open(name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer f.Close()
	content, ok := f.(io.ReadSeeker)
	if !ok {
		data, err := io.ReadAll(f)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		content = bytes.NewReader(data)
	}

	header := w.Header()
	modTime := info.ModTime()
	if useful := modTime.Unix() != 0 && modTime.Unix() != 1; useful {
		// The ETag Caddy's file server makes: modification time in
		// nanoseconds and size, in base 36.
		header.Set("Etag", `"`+strconv.FormatInt(modTime.UnixNano(), 36)+"-"+strconv.FormatInt(info.Size(), 36)+`"`)
	} else {
		modTime = time.Time{}
	}
	if t := mime.TypeByExtension(path.Ext(name)); t != "" {
		header.Set("Content-Type", t)
	} else {
		// No type, and none sniffed by ServeContent either.
		header["Content-Type"] = nil
	}

	if status != 0 {
		w = statusWriter{w, status}
	} else if acceptsGzip(r) {
		// The ETag a client sends back names the encoding the file was
		// sent with; the file itself has the plain one.
		if etags := r.Header["If-None-Match"]; len(etags) > 0 {
			joined := strings.Join(etags, ", ")
			if stripped := strings.ReplaceAll(joined, `-gzip"`, `"`); stripped != joined || len(etags) > 1 {
				r.Header["If-None-Match"] = []string{stripped}
			}
		}
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.Close()
		w = gw
	}
	http.ServeContent(w, r, info.Name(), modTime, content)
}

// statusWriter answers with code whatever status it is given.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w statusWriter) WriteHeader(int) { w.ResponseWriter.WriteHeader(w.code) }

// acceptsGzip reports whether the client takes gzip: named in
// Accept-Encoding with a weight above zero, and not opening a WebSocket.
func acceptsGzip(r *http.Request) bool {
	if r.Header.Get("Sec-Websocket-Key") != "" {
		return false
	}
	for _, value := range r.Header["Accept-Encoding"] {
		for _, part := range strings.Split(value, ",") {
			coding, params, found := strings.Cut(part, ";")
			if !strings.EqualFold(strings.TrimSpace(coding), "gzip") {
				continue
			}
			q := 1.0
			if found {
				p, _, _ := strings.Cut(params, ";")
				p = strings.TrimSpace(p)
				if len(p) >= 2 && strings.EqualFold(p[:2], "q=") {
					if v, err := strconv.ParseFloat(p[2:], 32); err == nil && v >= 0 && v <= 1 {
						q = v
					}
				}
			}
			if q >= 0.00001 {
				return true
			}
		}
	}
	return false
}

// minCompress is the size a body must exceed to be compressed.
const minCompress = 512

// compressedTypes are the types Caddy's encoder compresses, a trailing "*"
// matching any rest.
var compressedTypes = []string{
	"application/atom+xml*", "application/eot*", "application/font*", "application/geo+json*",
	"application/graphql+json*", "application/graphql-response+json*", "application/javascript*",
	"application/json*", "application/ld+json*", "application/manifest+json*", "application/opentype*",
	"application/otf*", "application/rss+xml*", "application/truetype*", "application/ttf*",
	"application/vnd.api+json*", "application/vnd.ms-fontobject*", "application/wasm*",
	"application/x-httpd-cgi*", "application/x-javascript*", "application/x-opentype*",
	"application/x-otf*", "application/x-perl*", "application/x-protobuf*", "application/x-ttf*",
	"application/xhtml+xml*", "application/xml*", "font/ttf*", "font/otf*", "image/svg+xml*",
	"image/vnd.microsoft.icon*", "image/x-icon*", "multipart/bag*", "multipart/mixed*", "text/*",
}

func compressedType(t string) bool {
	for _, pattern := range compressedTypes {
		if prefix, wild := strings.CutSuffix(pattern, "*"); wild && strings.HasPrefix(t, prefix) || t == pattern {
			return true
		}
	}
	return false
}

// gzipPool keeps writers at Caddy's gzip level, 5.
var gzipPool = sync.Pool{New: func() any {
	w, _ := gzip.NewWriterLevel(nil, 5)
	return w
}}

// gzipWriter compresses a response as Caddy's encoder does: it decides at
// the first byte of the body, or at the end for a body never written as
// HEAD has, and compresses a body of a listed type over 512 bytes unless it
// is a part of the file.
type gzipWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	partial     bool
	gz          *gzip.Writer
	wroteBody   bool
}

func (w *gzipWriter) WriteHeader(code int) {
	w.status = code
	if code == http.StatusPartialContent {
		w.partial = true
	}
}

func (w *gzipWriter) start() {
	if w.partial {
		return
	}
	header := w.Header()
	if header.Get("Content-Encoding") != "" || !compressedType(header.Get("Content-Type")) {
		return
	}
	w.gz = gzipPool.Get().(*gzip.Writer)
	w.gz.Reset(w.ResponseWriter)
	header.Del("Content-Length")
	header.Set("Content-Encoding", "gzip")
	header.Del("Accept-Ranges")
	if etag := header.Get("Etag"); etag != "" && !strings.HasPrefix(etag, "W/") {
		header.Set("Etag", strings.TrimSuffix(etag, `"`)+`-gzip"`)
	}
}

func (w *gzipWriter) over(n int) bool {
	if n > minCompress {
		return true
	}
	cl, err := strconv.Atoi(w.Header().Get("Content-Length"))
	return err == nil && cl > minCompress
}

func (w *gzipWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		if w.over(len(p)) {
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", http.DetectContentType(p))
			}
			w.start()
		}
		if w.status != 0 {
			w.ResponseWriter.WriteHeader(w.status)
		}
		w.wroteHeader = true
	}
	if w.gz == nil {
		return w.ResponseWriter.Write(p)
	}
	w.wroteBody = true
	return w.gz.Write(p)
}

func (w *gzipWriter) Close() {
	if !w.wroteHeader {
		if w.over(0) {
			w.start()
		}
		if w.status != 0 {
			w.ResponseWriter.WriteHeader(w.status)
		}
		w.wroteHeader = true
	}
	if w.gz != nil {
		if w.wroteBody {
			w.gz.Close()
		}
		w.gz.Reset(nil)
		gzipPool.Put(w.gz)
		w.gz = nil
	}
}

// Source returns the site to show: the administrator's own in dir when it
// has an index.html, made otherwise, and whether it is their own.
func Source(dir string, made fs.FS) (fs.FS, bool) {
	own := os.DirFS(dir)
	if info, err := fs.Stat(own, "index.html"); err == nil && info.Mode().IsRegular() {
		return own, true
	}
	return made, false
}

// Feedback answers a POST to path, where the made site's pages send «Was
// this page helpful?», with 204 and keeps nothing; everything else goes to
// next.
func Feedback(path string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == path && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
