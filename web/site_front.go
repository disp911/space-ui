package web

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/disp911/space-ui/v3/config"
	"github.com/disp911/space-ui/v3/logger"
	"github.com/disp911/space-ui/v3/web/network"
	"github.com/disp911/space-ui/v3/web/service"
	"github.com/disp911/space-ui/v3/web/site"
	"github.com/disp911/space-ui/v3/web/site/docsite"

	"github.com/pires/go-proxyproto"
)

// siteFront answers on the website's listeners: the panel's secret path is the
// panel's, the subscription paths are the subscription server's, and every
// other path is the website's. A path is matched as whole segments, so
// "/sub/" does not take "/subscribe". The website answers only by its own
// names: to any other host, or to none, it gives the empty 200 Caddy gives
// a request no site of it is for.
type siteFront struct {
	basePath string
	panel    http.Handler
	site     http.Handler
	hosts    []string
}

// ownHost reports whether the request names one of the website's hosts,
// with any port; with no hosts known every name is the website's.
func (f siteFront) ownHost(r *http.Request) bool {
	if len(f.hosts) == 0 {
		return true
	}
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	for _, h := range f.hosts {
		if strings.EqualFold(host, h) {
			return true
		}
	}
	return false
}

// siteHosts are the names the website answers by: its own, and the WEB
// proxy's, whose visitors without credentials it is shown to.
func (s *Server) siteHosts(domain string) []string {
	if domain == "" {
		return nil
	}
	hosts := []string{strings.TrimSuffix(strings.TrimPrefix(domain, "["), "]")}
	if web, _ := s.settingService.GetWebDomain(); web != "" && !strings.EqualFold(web, domain) {
		hosts = append(hosts, web)
	}
	return hosts
}

func (f siteFront) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if underPath(p, f.basePath) {
		f.panel.ServeHTTP(w, r)
		return
	}
	if sub := subFront.Load(); sub != nil {
		for _, prefix := range sub.paths {
			if underPath(p, prefix) {
				sub.handler.ServeHTTP(w, r)
				return
			}
		}
	}
	if !f.ownHost(r) {
		return
	}
	f.site.ServeHTTP(w, r)
}

// underPath reports whether p lies under prefix, a folder path ending in
// "/". The folder asked for without its slash is the website's, as any
// unknown path is: answering it otherwise would show that something is
// there.
func underPath(p, prefix string) bool {
	if !strings.HasSuffix(prefix, "/") || prefix == "/" {
		return false
	}
	return strings.HasPrefix(p, prefix)
}

type subRoute struct {
	handler http.Handler
	paths   []string
}

var subFront atomic.Pointer[subRoute]

// SetSubFront hands the website the subscription server's handler and the
// paths it answers, while the site serves the subscriptions on port 443 in
// place of the server's own port; nil takes them back.
func SetSubFront(handler http.Handler, paths []string) {
	if handler == nil {
		subFront.Store(nil)
		return
	}
	subFront.Store(&subRoute{handler: handler, paths: paths})
}

var siteRunning atomic.Bool

// SiteRunning reports whether the website serves port 443, and with it the
// subscriptions. The web server starts first, so the subscription server
// asks this once the website is up or has failed to come up.
func SiteRunning() bool {
	return siteRunning.Load()
}

// siteRedirect sends a plain HTTP visitor to the same address over HTTPS;
// one naming no host is sent to none, as Caddy sends it.
func siteRedirect() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		// As Caddy's automatic HTTPS: 308, no body, the connection closed.
		w.Header().Set("Location", "https://"+host+r.URL.RequestURI())
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusPermanentRedirect)
	})
}

// asCaddy names the server in every answer as Caddy does: the website's TLS
// and HTTP/2 are those of Caddy, the Go server it is built on.
func asCaddy(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Caddy")
		h.ServeHTTP(w, r)
	})
}

// quietLog drops what net/http would print about every visitor whose TLS
// handshake fails, which a public port sees all day.
var quietLog = log.New(io.Discard, "", 0)

// caddyServer serves h with the limits Caddy's server has by default:
// a minute for a request's head, 16 KB of it at most, and five minutes
// for a connection left idle. As Caddy's, it also speaks HTTP/2 without
// TLS, and so waits for as many bytes as that protocol's preface before it
// answers a plain connection at all.
func caddyServer(h http.Handler) *http.Server {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: time.Minute,
		IdleTimeout:       5 * time.Minute,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          quietLog,
		Protocols:         protocols,
	}
}

// startReality opens the website on the socket the REALITY inbound on 443
// passes visitors to, with the panel at its secret path, plus port 80 for
// the authorities' checks and a redirect to HTTPS. The panel opens no port
// of its own then.
func (s *Server) startReality(panel http.Handler) error {
	domain, err := s.settingService.GetSiteDomain()
	if err != nil {
		return err
	}
	if domain == "" {
		return errSite("no domain is set for the website")
	}
	basePath, err := s.settingService.GetBasePath()
	if err != nil {
		return err
	}
	if basePath == "/" {
		return errSite("the panel needs a secret path to share port 443 with the website")
	}
	listen, _ := s.settingService.GetListen()

	certs := site.NewCerts(config.GetCertFolder(), domain)
	if err := certs.Start(s.ctx); err != nil {
		return err
	}
	ln, err := siteSocket(config.GetSiteSocket())
	if err != nil {
		certs.Stop()
		return err
	}
	pages, err := s.sitePages(domain)
	if err != nil {
		ln.Close()
		certs.Stop()
		return err
	}
	s.siteFront = siteFront{basePath: basePath, panel: panel, site: pages, hosts: s.siteHosts(domain)}
	s.siteHandler = asCaddy(webProxyFront{panel: s.siteFront, route: service.TelemtWebRoute, xhttp: service.XhttpSiteRoute})
	s.siteServer = caddyServer(s.siteHandler)
	go s.siteServer.Serve(tls.NewListener(ln, certs.TLSConfig()))
	logger.Info("Website running on", ln.Addr(), "for", domain)

	if l80, err := net.Listen("tcp", net.JoinHostPort(listen, "80")); err != nil {
		logger.Warning("Website: port 80 is not available, certificates are checked over 443 only:", err)
	} else {
		s.redirectServer = caddyServer(asCaddy(certs.HTTPChallenge(siteRedirect())))
		go s.redirectServer.Serve(l80)
	}

	s.siteCerts = certs
	s.siteTLS = certs.TLSConfig()
	s.siteCheck = &site.Checker{}
	s.siteName, s.siteAddr = domain, net.JoinHostPort(domain, "443")
	service.SetSiteStatus(&service.SiteState{Scheme: "https", Host: domain, Reality: true, Certs: certs, Checker: s.siteCheck})
	siteRunning.Store(true)

	return nil
}

// startOwnPort opens the panel on its own port with the website at the
// root: HTTPS with the certificate the panel obtains for its name, or with
// its certificate files, plain HTTP with neither. Without a secret path the
// panel keeps the root to itself and there is no website. It reports
// whether that is HTTPS on port 443.
func (s *Server) startOwnPort(panel http.Handler) (net.Listener, bool, error) {
	listen, err := s.settingService.GetListen()
	if err != nil {
		return nil, false, err
	}
	port, err := s.settingService.GetPort()
	if err != nil {
		return nil, false, err
	}
	basePath, err := s.settingService.GetBasePath()
	if err != nil {
		return nil, false, err
	}
	name, _ := s.settingService.GetSiteDomain()
	certFile, _ := s.settingService.GetCertFile()
	keyFile, _ := s.settingService.GetKeyFile()

	ln, err := net.Listen("tcp", net.JoinHostPort(listen, strconv.Itoa(port)))
	if err != nil {
		return nil, false, err
	}
	var tlsConf *tls.Config
	switch {
	case certFile != "" || keyFile != "":
		if cert, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
			logger.Error("Error loading certificates:", err)
		} else {
			tlsConf = site.CaddyTLS(&tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{"h2", "http/1.1"}})
		}
	case name != "":
		certs := site.NewCerts(config.GetCertFolder(), name)
		if err := certs.Start(s.ctx); err != nil {
			logger.Error("Certificate for", name+":", err)
		} else {
			s.siteCerts = certs
			s.siteCheck = &site.Checker{}
			s.siteName, s.siteAddr = name, net.JoinHostPort(name, strconv.Itoa(port))
			tlsConf = certs.TLSConfig()
		}
	}
	scheme := "http"
	if tlsConf != nil {
		scheme = "https"
		ln = tls.NewListener(network.NewAutoHttpsListener(ln), tlsConf)
	}
	logger.Info("Web server running", strings.ToUpper(scheme), "on", ln.Addr())

	if basePath == "/" {
		s.siteHandler = webProxyFront{panel: panel, route: service.TelemtWebRoute, xhttp: service.XhttpSiteRoute}
		s.siteFront = panel
		return ln, tlsConf != nil && port == 443, nil
	}

	// The name the site and the subscriptions go by, with the port unless
	// it is the scheme's own.
	host := name
	if host != "" && !(port == 443 && scheme == "https" || port == 80 && scheme == "http") {
		host = net.JoinHostPort(name, strconv.Itoa(port))
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	shown := host
	if shown == "" {
		shown = "localhost"
	}
	pages, err := s.sitePages(shown)
	if err != nil {
		ln.Close()
		s.stopSite()
		return nil, false, err
	}
	s.siteFront = siteFront{basePath: basePath, panel: panel, site: pages, hosts: s.siteHosts(name)}
	s.siteHandler = asCaddy(webProxyFront{panel: s.siteFront, route: service.TelemtWebRoute, xhttp: service.XhttpSiteRoute})
	service.SetSiteStatus(&service.SiteState{Scheme: scheme, Host: host, Certs: s.siteCerts, Checker: s.siteCheck})
	siteRunning.Store(true)
	return ln, tlsConf != nil && port == 443, nil
}

// siteSocket opens the socket a REALITY inbound passes visitors to. Only
// root, and so only Xray, may connect, which makes the PROXY protocol header
// Xray puts first (the inbound's xver) safe to believe: with it the website
// and the panel see each visitor's own address. A connection without one
// is taken as it comes.
func siteSocket(sock string) (net.Listener, error) {
	os.MkdirAll(filepath.Dir(sock), 0o755)
	os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return nil, err
	}
	os.Chmod(sock, 0o600)
	return &proxyproto.Listener{
		Listener:          ln,
		ReadHeaderTimeout: 10 * time.Second,
		// The library requires the header unless told otherwise.
		ConnPolicy: func(proxyproto.ConnPolicyOptions) (proxyproto.Policy, error) { return proxyproto.USE, nil },
	}, nil
}

// sitePages is the website itself: the administrator's own from the site
// folder, or else the documentation site the panel makes from the seed in
// the database.
func (s *Server) sitePages(domain string) (http.Handler, error) {
	seed, born, err := s.settingService.GetSiteSeed()
	if err != nil {
		return nil, err
	}
	made := docsite.Make(seed, domain, born, time.Now())
	fsys, own := site.Source(config.GetSiteFolder(), made)
	if own {
		return site.Files(fsys), nil
	}
	return site.Feedback(docsite.FeedbackPath, site.Files(fsys)), nil
}

// checkSite looks at the website from outside and logs when it goes down or
// comes back.
func (s *Server) checkSite() {
	if s.siteCheck == nil || s.ctx.Err() != nil {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	res := site.Check(ctx, s.siteName, s.siteAddr, nil)
	if !s.siteCheck.Record(res) {
		return
	}
	if res.OK {
		logger.Info("Website check: the site is up again")
	} else {
		logger.Warning("Website check failed:", res.Error)
	}
}

func (s *Server) stopSite() {
	siteRunning.Store(false)
	if s.siteServer != nil {
		s.siteServer.Close()
	}
	if s.redirectServer != nil {
		s.redirectServer.Close()
	}
	if s.siteCerts != nil {
		s.siteCerts.Stop()
	}
	service.SetSiteStatus(nil)
}

type errSite string

func (e errSite) Error() string { return string(e) }
