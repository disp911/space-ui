package service

import (
	"encoding/json"
	"net"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/disp911/spacex-ui/v2/database"
	"github.com/disp911/spacex-ui/v2/database/model"
	"github.com/disp911/spacex-ui/v2/logger"
	"github.com/disp911/spacex-ui/v2/util/common"
	"github.com/disp911/spacex-ui/v2/xray"
)

// An XHTTP inbound on a loopback address without TLS is served by the
// website: a CDN, or a client, reaches it over HTTPS on port 443 at the
// inbound's path, and the website passes the requests on to Xray.

// xhttpSiteRoute is one such inbound: requests whose path starts with path
// go to 127.0.0.1:port.
type xhttpSiteRoute struct {
	path string
	port int
}

// xhttpSiteRoutes holds the routes of the running Xray, longest path first.
var xhttpSiteRoutes atomic.Pointer[[]xhttpSiteRoute]

// XhttpSiteRoute returns the loopback port of the XHTTP inbound the website
// passes a request for p to. The longest matching path wins.
func XhttpSiteRoute(p string) (int, bool) {
	routes := xhttpSiteRoutes.Load()
	if routes == nil {
		return 0, false
	}
	for _, r := range *routes {
		if strings.HasPrefix(p, r.path) {
			return r.port, true
		}
	}
	return 0, false
}

// siteXhttpPath returns the path the website serves an inbound at, as Xray
// matches it: XHTTP over plain HTTP on a loopback address only. The root
// path is never served, it would take the whole website.
func siteXhttpPath(listen string, stream map[string]any) (string, bool) {
	if ip := net.ParseIP(listen); listen != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return "", false
	}
	if network, _ := stream["network"].(string); network != "xhttp" {
		return "", false
	}
	if security, _ := stream["security"].(string); security != "" && security != "none" {
		return "", false
	}
	xs, _ := stream["xhttpSettings"].(map[string]any)
	path, _ := xs["path"].(string)
	path, _, _ = strings.Cut(path, "?")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// Xray adds the slash when the session ID or the sequence number is a
	// path segment, as both are unless placed elsewhere.
	sessionAt, _ := xs["sessionIDPlacement"].(string)
	seqAt, _ := xs["seqPlacement"].(string)
	if (sessionAt == "" || sessionAt == "path" || seqAt == "" || seqAt == "path") && !strings.HasSuffix(path, "/") {
		path += "/"
	}
	if path == "/" {
		return "", false
	}
	return path, true
}

// inboundSiteXhttpPath is siteXhttpPath for a saved inbound, which must be
// enabled.
func inboundSiteXhttpPath(ib *model.Inbound) (string, bool) {
	if !ib.Enable || !model.IsXrayProtocol(ib.Protocol) {
		return "", false
	}
	var stream map[string]any
	if json.Unmarshal([]byte(ib.StreamSettings), &stream) != nil {
		return "", false
	}
	return siteXhttpPath(ib.Listen, stream)
}

// pathsOverlap reports whether one path starts with the other: the website
// could not tell their requests apart.
func pathsOverlap(a, b string) bool {
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

// siteReservedPaths lists the paths the website keeps for itself: the
// panel's, the subscriptions' and those of the Telegram WEB proxies.
func siteReservedPaths(exceptID int) []string {
	settings := SettingService{}
	var paths []string
	if p, err := settings.GetBasePath(); err == nil && p != "/" {
		paths = append(paths, p)
	}
	for _, get := range []func() (string, error){settings.GetSubPath, settings.GetSubJsonPath, settings.GetSubClashPath} {
		if p, err := get(); err == nil && p != "" && p != "/" {
			paths = append(paths, p)
		}
	}
	if web, err := webPathsOfOtherInbounds(exceptID); err == nil {
		for p := range web {
			paths = append(paths, "/"+p+"/")
		}
	}
	return paths
}

// guardSiteXhttpPath refuses an inbound the website would serve at a path it
// cannot tell apart from the panel's, a subscription's, a WEB proxy's or
// another such inbound's.
func (s *InboundService) guardSiteXhttpPath(next *model.Inbound) error {
	path, ok := inboundSiteXhttpPath(next)
	if !ok {
		return nil
	}
	for _, p := range siteReservedPaths(next.Id) {
		if pathsOverlap(path, p) {
			return common.NewErrorf("XHTTP path %s overlaps the path %s the panel already serves on its website.", path, p)
		}
	}
	var others []*model.Inbound
	if err := database.GetDB().Model(model.Inbound{}).Where("id != ?", next.Id).Find(&others).Error; err != nil {
		return err
	}
	for _, other := range others {
		if p, ok := inboundSiteXhttpPath(other); ok && pathsOverlap(path, p) {
			return common.NewErrorf("XHTTP path %s overlaps the path %s of inbound %q, both served on the website.", path, p, other.Remark)
		}
	}
	return nil
}

// publishXhttpSiteRoutes records the routes of the Xray config that now
// runs, or none for nil. A route whose path the website keeps for itself is
// left out, so the panel stays reachable whatever the template holds.
func publishXhttpSiteRoutes(cfg *xray.Config) {
	if cfg == nil {
		xhttpSiteRoutes.Store(nil)
		return
	}
	reserved := siteReservedPaths(0)
	var routes []xhttpSiteRoute
next:
	for _, ib := range cfg.InboundConfigs {
		var listen string
		json.Unmarshal(ib.Listen, &listen)
		var stream map[string]any
		if json.Unmarshal(ib.StreamSettings, &stream) != nil {
			continue
		}
		path, ok := siteXhttpPath(listen, stream)
		if !ok {
			continue
		}
		for _, p := range reserved {
			if pathsOverlap(path, p) {
				logger.Warningf("Website: XHTTP inbound %s is not served, its path %s overlaps %s", ib.Tag, path, p)
				continue next
			}
		}
		routes = append(routes, xhttpSiteRoute{path: path, port: ib.Port})
	}
	sort.SliceStable(routes, func(i, j int) bool { return len(routes[i].path) > len(routes[j].path) })
	xhttpSiteRoutes.Store(&routes)
}

// trustSiteForwardedFor lets Xray take the client's address from the
// X-Forwarded-For header the website sets on the requests it passes to an
// inbound it serves. Only the website reaches a loopback inbound.
func trustSiteForwardedFor(listen string, stream map[string]any) {
	if _, ok := siteXhttpPath(listen, stream); !ok {
		return
	}
	sockopt, _ := stream["sockopt"].(map[string]any)
	if sockopt == nil {
		sockopt = map[string]any{}
		stream["sockopt"] = sockopt
	}
	if trusted, _ := sockopt["trustedXForwardedFor"].([]any); len(trusted) == 0 {
		sockopt["trustedXForwardedFor"] = []any{"X-Forwarded-For"}
	}
}
