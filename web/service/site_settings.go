package service

import (
	"crypto/tls"
	"net"
	"regexp"
	"strings"

	"github.com/disp911/space-ui/v3/config"
	"github.com/disp911/space-ui/v3/util/common"
	"github.com/disp911/space-ui/v3/web/site"
)

// SiteSettings is what the panel's settings page sets of the website. The
// mode is set on the server only (space-ui site, the menu): moving the panel
// from the browser could leave the administrator outside it.
type SiteSettings struct {
	Domain string `json:"domain"`
	// CertFile and KeyFile are the administrator's certificate on the
	// panel's own port; empty, the panel obtains one for Domain itself.
	// Behind REALITY they are kept as they are.
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

// SiteInfo is the website as the settings page shows it: the settings and
// what the running panel makes of them.
type SiteInfo struct {
	SiteSettings
	Mode string `json:"mode"`
	// Port is the panel's own port.
	Port int `json:"port"`
	// URL is where the website, the panel and the subscriptions are found,
	// such as https://example.com; empty while there is no website.
	URL string `json:"url"`
	// Socket is the website's address for a REALITY target, behind REALITY.
	Socket string `json:"socket"`
	// Cert is nil while the website has no certificate.
	Cert *SiteCert `json:"cert"`
	// Check is nil before the first look at the website from outside.
	Check *site.CheckResult `json:"check"`
}

// SiteCert is the website's certificate; Auto when the panel obtains it.
type SiteCert struct {
	Auto bool `json:"auto"`
	site.CertStatus
}

// hostName is a domain name such as example.com or xn--e1afmkfd.xn--p1ai.
var hostName = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9-]{2,63}$`)

// ApplySite stores the settings and fits the website's mode to them, as
// SwitchMode does: behind REALITY the site inbound takes the new domain.
// The panel takes them on its next start. Nothing is kept when they are
// refused.
func (s *SiteService) ApplySite(set SiteSettings) error {
	set.Domain = strings.ToLower(strings.TrimSpace(set.Domain))
	set.CertFile = strings.TrimSpace(set.CertFile)
	set.KeyFile = strings.TrimSpace(set.KeyFile)
	mode, err := s.settingService.GetSiteMode()
	if err != nil {
		return err
	}
	if set.Domain != "" && net.ParseIP(strings.Trim(set.Domain, "[]")) == nil && !hostName.MatchString(set.Domain) {
		return common.NewErrorf("%q is neither a domain name nor an IP address", set.Domain)
	}
	files := mode == SitePort
	if files && (set.CertFile != "" || set.KeyFile != "") {
		if _, err := tls.LoadX509KeyPair(set.CertFile, set.KeyFile); err != nil {
			return common.NewErrorf("the certificate files do not make a pair: %v", err)
		}
	}

	was, err := s.siteSettings()
	if err != nil {
		return err
	}
	if err := s.storeSite(set, files); err != nil {
		return err
	}
	if _, err := s.SwitchMode(mode); err != nil {
		if back := s.storeSite(was, true); back != nil {
			return common.Combine(err, back)
		}
		return err
	}
	return nil
}

func (s *SiteService) storeSite(set SiteSettings, files bool) error {
	if err := s.settingService.SetSiteDomain(set.Domain); err != nil {
		return err
	}
	if !files {
		return nil
	}
	if err := s.settingService.SetCertFile(set.CertFile); err != nil {
		return err
	}
	return s.settingService.SetKeyFile(set.KeyFile)
}

func (s *SiteService) siteSettings() (set SiteSettings, err error) {
	if set.Domain, err = s.settingService.GetSiteDomain(); err != nil {
		return
	}
	if set.CertFile, err = s.settingService.GetCertFile(); err != nil {
		return
	}
	set.KeyFile, err = s.settingService.GetKeyFile()
	return
}

// Site describes the website seen from a request to host.
func (s *SiteService) Site(host string) (*SiteInfo, error) {
	set, err := s.siteSettings()
	if err != nil {
		return nil, err
	}
	port, err := s.settingService.GetPort()
	if err != nil {
		return nil, err
	}
	mode, err := s.settingService.GetSiteMode()
	if err != nil {
		return nil, err
	}
	info := &SiteInfo{SiteSettings: set, Mode: mode, Port: port, URL: SiteBase(host)}
	if info.URL == "" {
		return info, nil
	}
	cert, check, auto := GetSiteStatus()
	switch {
	case auto:
		info.Cert = &SiteCert{Auto: true, CertStatus: cert}
		if !check.At.IsZero() {
			info.Check = &check
		}
	case !SiteBehindReality() && set.CertFile != "":
		if cert, err := site.FileStatus(set.CertFile); err == nil {
			info.Cert = &SiteCert{CertStatus: cert}
		}
	}
	if SiteBehindReality() {
		info.Socket = config.GetSiteSocket()
	}
	return info, nil
}
