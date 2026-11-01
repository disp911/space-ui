package web

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"strings"
	"time"
)

// webProxyTransport carries WEB proxy requests to telemt on loopback. It
// ignores proxy settings from the environment and never compresses, so the
// bytes telemt sends reach Telegram unchanged.
var webProxyTransport = &http.Transport{
	Proxy:               nil,
	DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
	MaxIdleConns:        256,
	MaxIdleConnsPerHost: 256,
	IdleConnTimeout:     90 * time.Second,
	DisableCompression:  true,
}

// xhttpTransport carries requests to the XHTTP inbounds the website serves,
// over h2c as Xray's XHTTP server takes them, so that the many requests of
// one connection share a few streams instead of a socket each.
var xhttpTransport = func() *http.Transport {
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	return &http.Transport{
		Proxy:              nil,
		DialContext:        (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		Protocols:          protocols,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: true,
	}
}()

// webProxyBase returns the first segment of a path that has more after it:
// "abc" for "/abc/" and "/abc/api/v1/up", nothing for "/abc".
func webProxyBase(path string) (string, bool) {
	rest, ok := strings.CutPrefix(path, "/")
	if !ok {
		return "", false
	}
	base, _, ok := strings.Cut(rest, "/")
	if !ok || base == "" {
		return "", false
	}
	return base, true
}

// cloudflareNets are the addresses Cloudflare reaches origins from, as
// published at https://www.cloudflare.com/ips/.
var cloudflareNets = func() []netip.Prefix {
	var nets []netip.Prefix
	for _, s := range []string{
		"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
		"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
		"197.234.240.0/22", "198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
		"104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",
		"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32",
		"2405:8100::/32", "2a06:98c0::/29", "2c0f:f248::/32",
	} {
		nets = append(nets, netip.MustParsePrefix(s))
	}
	return nets
}()

// webClientIP returns the address of the client behind a WEB proxy request.
// Behind Cloudflare's proxy the connection comes from Cloudflare, which names
// the client in CF-Connecting-IP; the header is believed only from
// Cloudflare's own addresses, so nobody else can choose the address telemt
// counts.
func webClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	for _, n := range cloudflareNets {
		if n.Contains(peer) {
			if client, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))); err == nil {
				return client.Unmap().String()
			}
			break
		}
	}
	return host
}

// webProxyFront is the panel's HTTPS handler. A request under the secret
// path of a running WEB proxy goes to that proxy's telemt listener, and one
// under the path of an XHTTP inbound the website serves goes to Xray; both
// are passed on before the panel's middleware, whose compression and
// sessions would break the long polls and WebSockets Telegram uses and
// XHTTP's streamed answers. Everything else, the bare secret path included,
// is the panel's.
type webProxyFront struct {
	panel http.Handler
	// route returns the loopback port of the WEB proxy at a base path.
	route func(basePath string) (int, bool)
	// xhttp returns the loopback port of the XHTTP inbound a path is under.
	xhttp func(path string) (int, bool)
}

func (f webProxyFront) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if base, ok := webProxyBase(r.URL.EscapedPath()); ok {
		if port, ok := f.route(base); ok {
			f.passWeb(w, r, port)
			return
		}
	}
	if f.xhttp != nil {
		if port, ok := f.xhttp(r.URL.Path); ok {
			f.passXhttp(w, r, port)
			return
		}
	}
	f.panel.ServeHTTP(w, r)
}

// passXhttp hands a request to the XHTTP inbound on a loopback port. Xray
// believes the client address in X-Forwarded-For (see
// trustSiteForwardedFor), which replaces whatever the client sent.
func (f webProxyFront) passXhttp(w http.ResponseWriter, r *http.Request, port int) {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "http"
			pr.Out.URL.Host = fmt.Sprintf("127.0.0.1:%d", port)
			// The inbound may check the host name it is reached at.
			pr.Out.Host = pr.In.Host
			if ip := webClientIP(pr.In); ip != "" {
				pr.Out.Header.Set("X-Forwarded-For", ip)
			}
		},
		Transport:     xhttpTransport,
		FlushInterval: -1,
		// Should Xray be down, the visitor sees the site, not an error.
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, _ error) {
			f.panel.ServeHTTP(w, r)
		},
	}
	proxy.ServeHTTP(w, r)
}

// passWeb hands a request to the telemt WEB proxy on a loopback port.
func (f webProxyFront) passWeb(w http.ResponseWriter, r *http.Request, port int) {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "http"
			pr.Out.URL.Host = fmt.Sprintf("127.0.0.1:%d", port)
			// telemt matches the public host name and trusts exactly one
			// client address, which replaces whatever the client sent.
			pr.Out.Host = pr.In.Host
			if ip := webClientIP(pr.In); ip != "" {
				pr.Out.Header.Set("X-Forwarded-For", ip)
			}
		},
		Transport:     webProxyTransport,
		FlushInterval: -1,
		// Should telemt be down, the visitor sees the site, not an error.
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, _ error) {
			f.panel.ServeHTTP(w, r)
		},
	}
	proxy.ServeHTTP(w, r)
}

// webDecoyHandler is the site telemt shows a WEB request that carries no
// valid credentials. It is the panel's own handler, so such a request gets
// the same decoy page as any other unknown path. telemt replaces the host
// name with this listener's address; the panel's domain is put back, or the
// panel's domain check would answer differently. Only the WEB paths are
// served: this listener is plain HTTP and must not be a second way into the
// panel.
type webDecoyHandler struct {
	panel  http.Handler
	domain string
	route  func(basePath string) (int, bool)
}

func (h webDecoyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.EscapedPath()
	base, ok := webProxyBase(path)
	if !ok {
		base = strings.TrimPrefix(path, "/")
	}
	if _, ok := h.route(base); !ok {
		http.NotFound(w, r)
		return
	}
	if h.domain != "" {
		r.Host = h.domain
	}
	h.panel.ServeHTTP(w, r)
}
