// Package spa serves the panel's single-page frontend built by Vite.
package spa

import (
	"bytes"
	"encoding/json"
	"html"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// Config is handed to the frontend in index.html.
type Config struct {
	// Base is the panel's base path, e.g. "/" or "/secret/".
	Base string `json:"base"`
	// Mount is where the frontend is served, ending with a slash.
	Mount string `json:"mount"`
	// LoggedIn tells the frontend whether the request carried a session.
	LoggedIn bool   `json:"loggedIn"`
	Version  string `json:"version"`
	// Zone is the panel's time zone by its IANA name: the windows show their dates in it.
	Zone string `json:"zone,omitempty"`
}

// placeholder is the comment in the frontend's pages (index.html, sub.html)
// that the server replaces with the <base> element and the config.
const placeholder = "<!--app-config-->"

// Handler serves fsys: files under assets/ with a year-long cache, any other
// path with index.html, so that client-side routes survive a reload. The
// route must be registered with a *path wildcard.
func Handler(fsys fs.FS, config func(c *gin.Context) Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimPrefix(path.Clean("/"+c.Param("path")), "/")
		if name != "" && !strings.HasPrefix(path.Base(name), ".") && path.Ext(name) != "" {
			if ServeFile(c, fsys, name) {
				return
			}
			if strings.HasPrefix(name, "assets/") {
				c.Status(http.StatusNotFound)
				return
			}
		}
		cfg := config(c)
		ServePage(c, fsys, "index.html", cfg.Mount, cfg)
	}
}

// ServeFile answers with the file name of fsys, files under assets/ with a
// year-long cache, and tells whether it was there.
func ServeFile(c *gin.Context, fsys fs.FS, name string) bool {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return false
	}
	if strings.HasPrefix(name, "assets/") {
		// Vite puts a content hash in every asset name.
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	}
	c.Data(http.StatusOK, contentType(name), data)
	return true
}

// ServePage answers with the built page name of fsys, its relative URLs
// resolved against mount and config put in for the frontend to read, e.g.
// the panel's index.html or the subscription page.
func ServePage(c *gin.Context, fsys fs.FS, name, mount string, config any) {
	page, err := fs.ReadFile(fsys, name)
	if err != nil {
		c.String(http.StatusServiceUnavailable, "The panel frontend is not built: run npm --prefix frontend run build")
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", render(page, mount, config))
}

// render puts the <base> element and the config into a page.
func render(page []byte, mount string, config any) []byte {
	data, _ := json.Marshal(config)
	head := `<base href="` + html.EscapeString(mount) + `">` +
		`<meta name="app-config" content="` + html.EscapeString(string(data)) + `">`
	return bytes.Replace(page, []byte(placeholder), []byte(head), 1)
}

func contentType(name string) string {
	switch path.Ext(name) {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	case ".json", ".webmanifest":
		return "application/json"
	case ".png":
		return "image/png"
	case ".ico":
		return "image/x-icon"
	case ".html":
		return "text/html; charset=utf-8"
	}
	return "application/octet-stream"
}
