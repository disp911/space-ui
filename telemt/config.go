// Package telemt runs the bundled telemt MTProto proxy for Telegram. Every
// panel inbound with the mtproto protocol gets its own telemt process; the
// panel renders the inbound into telemt's TOML config and reads per-client
// traffic back from telemt's loopback metrics endpoint.
package telemt

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Settings are the per-inbound options.
type Settings struct {
	// Listen is the IP to bind; empty binds all interfaces.
	Listen string
	// Port is the public TCP port Telegram clients connect to.
	Port int
	// TLSDomain is the real website the Fake-TLS handshake imitates. It is
	// encoded into every ee-link, so changing it invalidates issued links.
	TLSDomain string
	// MetricsPort is the loopback port of telemt's metrics endpoint.
	MetricsPort int
	// APIPort is the loopback port of telemt's control API, which the panel
	// only reads from; 0 keeps the API off.
	APIPort int
	// APIToken is the exact Authorization header the API demands.
	APIToken string
	// Web, when set, also serves the users as a Telegram WEB proxy.
	Web *Web
	// IPLimitMode is how telemt counts a user's unique IPs against its
	// limit: one of IPLimitModes, empty for telemt's default (active_window).
	IPLimitMode string
	// IPLimitWindow is the window in seconds of the time_window and combined
	// modes; 0 leaves telemt's default of 30.
	IPLimitWindow int
}

// IPLimitModes are the ways telemt counts unique IPs: live connections only,
// addresses seen within the window, or both.
var IPLimitModes = []string{"active_window", "time_window", "combined"}

// MaxIPLimitWindow caps the unique-IP window at a week.
const MaxIPLimitWindow = 7 * 24 * 3600

// Web is the WEB proxy of an inbound: Telegram reaches it over HTTPS at
// https://Host/BasePath/, through a TLS front that forwards that subtree to
// the plain-HTTP listener on ListenPort. Requests without valid WEB
// credentials go on to the decoy site on DecoyPort.
type Web struct {
	Host     string
	BasePath string
	// PublicIP is the address Host resolves to; the front listens on its
	// port 443.
	PublicIP   string
	ListenPort int
	DecoyPort  int
	// MaxProfiles is telemt's ceiling on WEB users. Raising it needs a
	// restart, so it is kept above the user count; see WebProfileCeiling.
	MaxProfiles int
}

// WebProfileCeiling returns the WEB profile ceiling for a number of users:
// telemt's default of 32, then whole steps of 64 so that adding a client
// rarely restarts the process.
func WebProfileCeiling(users int) int {
	if users <= 32 {
		return 32
	}
	return (users + 63) / 64 * 64
}

// User is one proxy account.
type User struct {
	// Name is the telemt user name; see UserName.
	Name   string
	Secret string
	// Disabled users stay in the config marked as such: telemt cancels the
	// live sessions of a user it sees disabled, while a user that simply
	// disappears from the config keeps the sessions already open.
	Disabled bool
	// MaxUniqueIPs caps concurrent source IPs; 0 means unlimited.
	MaxUniqueIPs int
	// MaxTCPConns caps concurrent connections; 0 means unlimited.
	MaxTCPConns int
	// RateUpBps and RateDownBps cap the client's upload and download in
	// bits per second, up to MaxRateBps; 0 leaves that direction unlimited.
	RateUpBps, RateDownBps uint64
	// Expires is when telemt stops accepting the user; zero never expires.
	Expires time.Time
	// SourceDeny are the networks telemt refuses the user from, as CIDRs;
	// see NormalizeSourceDeny.
	SourceDeny []string
}

// MaxSourceDeny caps the networks denied to one user.
const MaxSourceDeny = 256

// NormalizeSourceDeny checks a user's denied sources, addresses or networks
// in CIDR notation, and returns them as telemt reads them: a lone address as
// a /32 or /128 network, a network masked to its prefix, each once and in
// order.
func NormalizeSourceDeny(values []string) ([]string, error) {
	if len(values) > MaxSourceDeny {
		return nil, fmt.Errorf("at most %d denied sources per client", MaxSourceDeny)
	}
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		var prefix netip.Prefix
		if strings.Contains(v, "/") {
			p, err := netip.ParsePrefix(v)
			if err != nil || p.Addr().Zone() != "" {
				return nil, fmt.Errorf("invalid denied source %q: an address or a network such as 203.0.113.0/24", v)
			}
			prefix = p.Masked()
		} else {
			a, err := netip.ParseAddr(v)
			if err != nil || a.Zone() != "" {
				return nil, fmt.Errorf("invalid denied source %q: an address or a network such as 203.0.113.0/24", v)
			}
			prefix = netip.PrefixFrom(a, a.BitLen())
		}
		if s := prefix.String(); !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out, nil
}

// sourceDenyKey captures the users' denied sources: telemt applies them only
// on a new generation, which a reload does not start, so a change restarts
// the process.
func sourceDenyKey(users []User) string {
	var b strings.Builder
	for _, u := range users {
		if len(u.SourceDeny) > 0 {
			fmt.Fprintf(&b, "|deny|%s=%s", u.Name, strings.Join(u.SourceDeny, ","))
		}
	}
	return b.String()
}

// MaxRateBps is the highest per-direction rate telemt accepts.
const MaxRateBps = 100_000_000_000

var (
	userNamePattern  = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)
	secretPattern    = regexp.MustCompile(`^[0-9a-f]{32}$`)
	tlsDomainPattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?)+$`)
	// The API token is generated by the panel; the pattern only keeps
	// anything else out of the config file.
	apiTokenPattern = regexp.MustCompile(`^[0-9a-f]{32,128}$`)
	// webBasePathPattern is a single path segment; telemt allows more, the
	// panel only generates one.
	webBasePathPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)
	numericLabelPattern = regexp.MustCompile(`\.[0-9]+$`)
)

// ValidWebHost reports whether host is a name Telegram accepts for a WEB
// proxy: a fully qualified host name, not an IP address.
func ValidWebHost(host string) bool {
	return ValidTLSDomain(host) && net.ParseIP(host) == nil && !numericLabelPattern.MatchString(host)
}

// ValidWebBasePath reports whether path can be the WEB proxy's base path.
func ValidWebBasePath(path string) bool {
	return webBasePathPattern.MatchString(path)
}

// ValidSecret reports whether secret is a 32-character lowercase hex secret.
func ValidSecret(secret string) bool {
	return secretPattern.MatchString(secret)
}

// ValidTLSDomain reports whether domain is a plain host name usable as the
// Fake-TLS domain.
func ValidTLSDomain(domain string) bool {
	return len(domain) <= 253 && tlsDomainPattern.MatchString(domain)
}

// Validate checks the settings before they are written to a config file.
func (s Settings) Validate() error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("invalid proxy port %d", s.Port)
	}
	if !ValidTLSDomain(s.TLSDomain) {
		return fmt.Errorf("invalid TLS domain %q", s.TLSDomain)
	}
	if s.Listen != "" && net.ParseIP(s.Listen) == nil {
		return fmt.Errorf("invalid listen address %q", s.Listen)
	}
	if s.MetricsPort < 1 || s.MetricsPort > 65535 || s.MetricsPort == s.Port {
		return fmt.Errorf("invalid metrics port %d", s.MetricsPort)
	}
	if s.IPLimitMode != "" && !slices.Contains(IPLimitModes, s.IPLimitMode) {
		return fmt.Errorf("invalid unique IP mode %q", s.IPLimitMode)
	}
	if s.IPLimitWindow < 0 || s.IPLimitWindow > MaxIPLimitWindow {
		return fmt.Errorf("invalid unique IP window %d s", s.IPLimitWindow)
	}
	if s.APIPort != 0 {
		if s.APIPort < 1 || s.APIPort > 65535 || s.APIPort == s.Port || s.APIPort == s.MetricsPort {
			return fmt.Errorf("invalid API port %d", s.APIPort)
		}
		if !apiTokenPattern.MatchString(s.APIToken) {
			return errors.New("invalid API token")
		}
	}
	if w := s.Web; w != nil {
		if !ValidWebHost(w.Host) {
			return fmt.Errorf("invalid WEB host %q", w.Host)
		}
		if !ValidWebBasePath(w.BasePath) {
			return fmt.Errorf("invalid WEB base path %q", w.BasePath)
		}
		if ip := net.ParseIP(w.PublicIP); ip == nil || ip.IsUnspecified() {
			return fmt.Errorf("invalid WEB public IP %q", w.PublicIP)
		}
		for _, p := range []int{w.ListenPort, w.DecoyPort} {
			if p < 1 || p > 65535 || p == s.Port || p == s.MetricsPort || p == s.APIPort {
				return fmt.Errorf("invalid WEB loopback port %d", p)
			}
		}
		if w.ListenPort == w.DecoyPort {
			return errors.New("the WEB listener and the decoy site share a port")
		}
		if w.MaxProfiles < 1 {
			return errors.New("invalid WEB profile ceiling")
		}
	}
	return nil
}

type fileConfig struct {
	General    fileGeneral    `toml:"general"`
	Server     fileServer     `toml:"server"`
	Censorship fileCensorship `toml:"censorship"`
	Access     fileAccess     `toml:"access"`
	Web        *fileWeb       `toml:"web,omitempty"`
}

type fileGeneral struct {
	LogLevel       string    `toml:"log_level"`
	DisableColors  bool      `toml:"disable_colors"`
	QuotaStatePath string    `toml:"quota_state_path"`
	UseMiddleProxy bool      `toml:"use_middle_proxy"`
	Modes          fileModes `toml:"modes"`
}

type fileModes struct {
	Classic bool `toml:"classic"`
	Secure  bool `toml:"secure"`
	TLS     bool `toml:"tls"`
}

type fileServer struct {
	Port             int            `toml:"port"`
	MetricsListen    string         `toml:"metrics_listen"`
	MetricsWhitelist []string       `toml:"metrics_whitelist"`
	API              fileAPI        `toml:"api"`
	Listeners        []fileListener `toml:"listeners,omitempty"`
}

type fileAPI struct {
	Enabled    bool     `toml:"enabled"`
	Listen     string   `toml:"listen,omitempty"`
	Whitelist  []string `toml:"whitelist,omitempty"`
	AuthHeader string   `toml:"auth_header,omitempty"`
	ReadOnly   bool     `toml:"read_only,omitempty"`
}

type fileListener struct {
	IP                   string   `toml:"ip"`
	Port                 int      `toml:"port,omitempty"`
	Transport            string   `toml:"transport,omitempty"`
	ProxyProtocol        *bool    `toml:"proxy_protocol,omitempty"`
	WebClientIPSource    string   `toml:"web_client_ip_source,omitempty"`
	WebTrustedProxyCIDRs []string `toml:"web_trusted_proxy_cidrs,omitempty"`
}

type fileWeb struct {
	Enabled bool   `toml:"enabled"`
	Carrier string `toml:"carrier"`
	// Carriers are tried first by clients that negotiate (Telegram
	// Desktop); the rest, iOS among them, use Carrier.
	Carriers []string       `toml:"carriers"`
	Limits   fileWebLimits  `toml:"limits"`
	Vhosts   []fileWebVhost `toml:"vhosts"`
}

type fileWebLimits struct {
	MaxProfiles int `toml:"max_profiles"`
}

type fileWebVhost struct {
	Host       string           `toml:"host"`
	BasePath   string           `toml:"base_path"`
	PublicAddr string           `toml:"public_addr"`
	Decoy      fileWebDecoy     `toml:"decoy"`
	Profiles   []fileWebProfile `toml:"profiles"`
}

type fileWebDecoy struct {
	Mode     string `toml:"mode"`
	Upstream string `toml:"upstream"`
}

type fileWebProfile struct {
	User       string `toml:"user"`
	SecretMode string `toml:"secret_mode"`
}

type fileCensorship struct {
	TLSDomain    string `toml:"tls_domain"`
	Mask         bool   `toml:"mask"`
	TLSEmulation bool   `toml:"tls_emulation"`
	TLSFrontDir  string `toml:"tls_front_dir"`
}

type fileAccess struct {
	Users            map[string]string        `toml:"users"`
	UserEnabled      map[string]bool          `toml:"user_enabled,omitempty"`
	UserMaxUniqueIPs map[string]int           `toml:"user_max_unique_ips,omitempty"`
	UserMaxTCPConns  map[string]int           `toml:"user_max_tcp_conns,omitempty"`
	UserExpirations  map[string]string        `toml:"user_expirations,omitempty"`
	UserRateLimits   map[string]fileRateLimit `toml:"user_rate_limits,omitempty"`
	UserSourceDeny   map[string][]string      `toml:"user_source_deny,omitempty"`
	// The unique-IP accounting of all users; empty and 0 keep telemt's defaults.
	UserMaxUniqueIPsMode       string `toml:"user_max_unique_ips_mode,omitempty"`
	UserMaxUniqueIPsWindowSecs int    `toml:"user_max_unique_ips_window_secs,omitempty"`
}

type fileRateLimit struct {
	UpBps   uint64 `toml:"up_bps"`
	DownBps uint64 `toml:"down_bps"`
}

// BuildConfig renders the telemt TOML config for one inbound. Only Fake-TLS
// (ee) mode is enabled: it is the only mode that looks like ordinary HTTPS to
// a censor. The metrics endpoint and, when asked for, a read-only control API
// are bound to loopback. Clients are relayed straight to the Telegram data
// centres: the Middle-End relays telemt tries by default only matter for a sponsored-channel ad tag, which the panel does
// not offer, and where they are unreachable they just log warnings on every
// start. telemt refuses to start without users, so at least one is required.
func BuildConfig(s Settings, users []User) ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, errors.New("telemt needs at least one user")
	}
	if s.Web != nil && len(users) > s.Web.MaxProfiles {
		return nil, fmt.Errorf("%d users exceed the WEB profile ceiling of %d", len(users), s.Web.MaxProfiles)
	}

	access := fileAccess{
		Users:            make(map[string]string, len(users)),
		UserEnabled:      map[string]bool{},
		UserMaxUniqueIPs: map[string]int{},
		UserMaxTCPConns:  map[string]int{},
		UserExpirations:  map[string]string{},
		UserRateLimits:   map[string]fileRateLimit{},
		UserSourceDeny:   map[string][]string{},

		UserMaxUniqueIPsMode:       s.IPLimitMode,
		UserMaxUniqueIPsWindowSecs: s.IPLimitWindow,
	}
	for _, u := range users {
		if !userNamePattern.MatchString(u.Name) {
			return nil, fmt.Errorf("invalid user name %q", u.Name)
		}
		if !ValidSecret(u.Secret) {
			return nil, fmt.Errorf("invalid secret for user %q", u.Name)
		}
		if _, dup := access.Users[u.Name]; dup {
			return nil, fmt.Errorf("duplicate user name %q", u.Name)
		}
		access.Users[u.Name] = u.Secret
		if u.Disabled {
			access.UserEnabled[u.Name] = false
		}
		if u.MaxUniqueIPs > 0 {
			access.UserMaxUniqueIPs[u.Name] = u.MaxUniqueIPs
		}
		if u.MaxTCPConns > 0 {
			access.UserMaxTCPConns[u.Name] = u.MaxTCPConns
		}
		if !u.Expires.IsZero() {
			access.UserExpirations[u.Name] = u.Expires.UTC().Format(time.RFC3339)
		}
		if u.RateUpBps > MaxRateBps || u.RateDownBps > MaxRateBps {
			return nil, fmt.Errorf("rate limit of user %q is above %d bit/s", u.Name, uint64(MaxRateBps))
		}
		if u.RateUpBps > 0 || u.RateDownBps > 0 {
			access.UserRateLimits[u.Name] = fileRateLimit{UpBps: u.RateUpBps, DownBps: u.RateDownBps}
		}
		deny, err := NormalizeSourceDeny(u.SourceDeny)
		if err != nil {
			return nil, fmt.Errorf("user %q: %w", u.Name, err)
		}
		if len(deny) > 0 {
			access.UserSourceDeny[u.Name] = deny
		}
	}

	cfg := fileConfig{
		General: fileGeneral{
			LogLevel:       "normal",
			DisableColors:  true,
			QuotaStatePath: "telemt.limit.json",
			UseMiddleProxy: false,
			Modes:          fileModes{TLS: true},
		},
		Server: fileServer{
			Port:             s.Port,
			MetricsListen:    fmt.Sprintf("127.0.0.1:%d", s.MetricsPort),
			MetricsWhitelist: []string{"127.0.0.1/32"},
		},
		Censorship: fileCensorship{
			TLSDomain:    s.TLSDomain,
			Mask:         true,
			TLSEmulation: true,
			TLSFrontDir:  "tlsfront",
		},
		Access: access,
	}
	if s.Listen != "" {
		cfg.Server.Listeners = []fileListener{{IP: s.Listen}}
	}
	if w := s.Web; w != nil {
		// Listing any listener stops telemt from adding its default ones,
		// so the Fake-TLS listeners it would bind are listed too.
		if s.Listen == "" {
			cfg.Server.Listeners = []fileListener{{IP: "0.0.0.0"}, {IP: "::"}}
		}
		noProxyProtocol := false
		cfg.Server.Listeners = append(cfg.Server.Listeners, fileListener{
			IP:                   "127.0.0.1",
			Port:                 w.ListenPort,
			Transport:            "web",
			ProxyProtocol:        &noProxyProtocol,
			WebClientIPSource:    "x_forwarded_for",
			WebTrustedProxyCIDRs: []string{"127.0.0.1/32"},
		})
		profiles := make([]fileWebProfile, 0, len(users))
		for _, u := range users {
			profiles = append(profiles, fileWebProfile{User: u.Name, SecretMode: "dd"})
		}
		cfg.Web = &fileWeb{
			Enabled: true,
			// iOS only speaks https; Desktop tries a WebSocket first.
			Carrier:  "https",
			Carriers: []string{"websocket"},
			Limits:   fileWebLimits{MaxProfiles: w.MaxProfiles},
			Vhosts: []fileWebVhost{{
				Host:       w.Host,
				BasePath:   w.BasePath,
				PublicAddr: net.JoinHostPort(w.PublicIP, "443"),
				Decoy:      fileWebDecoy{Mode: "http_upstream", Upstream: fmt.Sprintf("http://127.0.0.1:%d", w.DecoyPort)},
				Profiles:   profiles,
			}},
		}
	}
	if s.APIPort != 0 {
		cfg.Server.API = fileAPI{
			Enabled:    true,
			Listen:     fmt.Sprintf("127.0.0.1:%d", s.APIPort),
			Whitelist:  []string{"127.0.0.1/32"},
			AuthHeader: s.APIToken,
			ReadOnly:   true,
		}
	}
	return toml.Marshal(cfg)
}

// restartKey captures the settings telemt cannot hot-reload: a change in any
// of them needs a process restart, while user changes only need a reload.
func (s Settings) restartKey() string {
	key := fmt.Sprintf("%s|%d|%s|%d|%d|%s", s.Listen, s.Port, s.TLSDomain, s.MetricsPort, s.APIPort, s.APIToken)
	// The WEB host, path, address and decoy reload; its listener and
	// limits belong to the process.
	if w := s.Web; w != nil {
		key += fmt.Sprintf("|web|%d|%d", w.ListenPort, w.MaxProfiles)
	}
	return key
}

// UserName maps a panel client email to a telemt user name. telemt only
// accepts [A-Za-z0-9_.-]{1,64}; other emails get a stable name derived from
// the email, so traffic can still be attributed back to it.
func UserName(email string) string {
	if userNamePattern.MatchString(email) {
		return email
	}
	var b strings.Builder
	for _, r := range email {
		if r < 128 && (r == '_' || r == '.' || r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	name := b.String()
	sum := fnv32(email)
	if len(name) > 55 {
		name = name[:55]
	}
	return fmt.Sprintf("%s-%08x", name, sum)
}

func fnv32(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
