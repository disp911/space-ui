package service

import (
	"context"
	"sync"

	"github.com/disp911/space-ui/v3/util/common"
	"github.com/disp911/space-ui/v3/web/site"
)

// SiteState is what the running website reports of itself.
type SiteState struct {
	// Scheme and Host are where the website and the subscriptions are
	// found, such as https and example.com or 203.0.113.7:8443. Host is
	// empty when the panel knows no name of its own and a visitor's has to
	// do.
	Scheme, Host string
	// Reality is true while the website is behind REALITY, on its socket.
	Reality bool
	// Certs is nil when the panel does not obtain the certificate itself.
	Certs *site.Certs
	// Checker is nil when nothing looks at the site from outside.
	Checker *site.Checker
}

var siteStatus struct {
	sync.Mutex
	state *SiteState
}

// SetSiteStatus records the running website; nil when it stops.
func SetSiteStatus(state *SiteState) {
	siteStatus.Lock()
	siteStatus.state = state
	siteStatus.Unlock()
}

// SiteBase is where the website is found, such as https://example.com,
// seen from a request to host; "" while there is no website. The
// subscriptions are served there, at their paths, in place of their own
// port.
func SiteBase(host string) string {
	siteStatus.Lock()
	defer siteStatus.Unlock()
	state := siteStatus.state
	if state == nil {
		return ""
	}
	if state.Host != "" {
		host = state.Host
	}
	return state.Scheme + "://" + host
}

// GetSiteStatus describes the website's certificate and the last check of
// it; managed is false while the panel obtains no certificate itself.
func GetSiteStatus() (cert site.CertStatus, check site.CheckResult, managed bool) {
	siteStatus.Lock()
	state := siteStatus.state
	siteStatus.Unlock()
	if state == nil || state.Certs == nil {
		return cert, check, false
	}
	if state.Checker != nil {
		check = state.Checker.Last()
	}
	return state.Certs.Status(), check, true
}

// SiteBehindReality reports whether the running website is behind REALITY.
func SiteBehindReality() bool {
	siteStatus.Lock()
	defer siteStatus.Unlock()
	return siteStatus.state != nil && siteStatus.state.Reality
}

// RenewSiteCert obtains a new certificate for the website now and waits for
// the authority's answer.
func RenewSiteCert(ctx context.Context) error {
	siteStatus.Lock()
	state := siteStatus.state
	siteStatus.Unlock()
	if state == nil || state.Certs == nil {
		return common.NewError("the panel obtains no certificate for the website")
	}
	return state.Certs.Renew(ctx)
}
