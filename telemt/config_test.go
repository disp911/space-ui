package telemt

import (
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

func testSettings() Settings {
	return Settings{Port: 8443, TLSDomain: "www.example.com", MetricsPort: 19090}
}

func lookup(m map[string]any, path string) any {
	var cur any = m
	for _, key := range strings.Split(path, "/") {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[key]
	}
	return cur
}

func TestBuildConfigRendersInbound(t *testing.T) {
	users := []User{
		{Name: "alice", Secret: "00112233445566778899aabbccddeeff", MaxUniqueIPs: 3},
		{Name: "bob.pc", Secret: "0123456789abcdef0123456789abcdef"},
	}
	s := testSettings()
	s.Listen = "10.0.0.5"
	data, err := BuildConfig(s, users)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal(data, &got); err != nil {
		t.Fatalf("generated config is not valid TOML: %v\n%s", err, data)
	}
	want := map[string]any{
		"general/modes/tls":                    true,
		"general/modes/classic":                false,
		"general/modes/secure":                 false,
		"general/disable_colors":               true,
		"general/quota_state_path":             "telemt.limit.json",
		"general/use_middle_proxy":             false,
		"server/port":                          int64(8443),
		"server/metrics_listen":                "127.0.0.1:19090",
		"server/api/enabled":                   false,
		"censorship/tls_domain":                "www.example.com",
		"censorship/mask":                      true,
		"censorship/tls_emulation":             true,
		"access/users/alice":                   "00112233445566778899aabbccddeeff",
		"access/users/bob.pc":                  "0123456789abcdef0123456789abcdef",
		"access/user_max_unique_ips/alice":     int64(3),
		"access/user_max_unique_ips/bob.pc":    nil,
		"general/links":                        nil,
		"access/user_enabled":                  nil,
		"access/users/does-not-exist-sanity":   nil,
		"server/metrics_whitelist":             []any{"127.0.0.1/32"},
		"server/listeners":                     []any{map[string]any{"ip": "10.0.0.5"}},
		"censorship/tls_front_dir":             "tlsfront",
		"general/log_level":                    "normal",
		"access/user_max_unique_ips/unlimited": nil,
	}
	for path, value := range want {
		if g := lookup(got, path); !equalValue(g, value) {
			t.Errorf("%s = %#v, want %#v", path, g, value)
		}
	}
}

func equalValue(a, b any) bool {
	as, aok := a.([]any)
	bs, bok := b.([]any)
	if aok || bok {
		if len(as) != len(bs) {
			return false
		}
		for i := range as {
			if !equalValue(as[i], bs[i]) {
				return false
			}
		}
		return true
	}
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if aok || bok {
		if len(am) != len(bm) {
			return false
		}
		for k := range am {
			if !equalValue(am[k], bm[k]) {
				return false
			}
		}
		return true
	}
	return a == b
}

func TestBuildConfigRendersLimitsAndAPI(t *testing.T) {
	expires := time.Date(2026, 12, 31, 23, 59, 59, 0, time.FixedZone("MSK", 3*3600))
	users := []User{
		{Name: "alice", Secret: strings.Repeat("a", 32), MaxTCPConns: 8, RateUpBps: 5_000_000, RateDownBps: 20_000_000, Expires: expires},
		{Name: "bob", Secret: strings.Repeat("b", 32), Disabled: true, RateDownBps: 1_000_000},
		{Name: "carol", Secret: strings.Repeat("c", 32)},
	}
	s := testSettings()
	s.APIPort, s.APIToken = 19091, strings.Repeat("f", 32)
	data, err := BuildConfig(s, users)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal(data, &got); err != nil {
		t.Fatalf("generated config is not valid TOML: %v\n%s", err, data)
	}
	want := map[string]any{
		"access/users/bob":                       strings.Repeat("b", 32),
		"access/user_enabled/bob":                false,
		"access/user_enabled/alice":              nil,
		"access/user_max_tcp_conns/alice":        int64(8),
		"access/user_max_tcp_conns/carol":        nil,
		"access/user_expirations/alice":          "2026-12-31T20:59:59Z",
		"access/user_expirations/bob":            nil,
		"access/user_rate_limits/alice/up_bps":   int64(5_000_000),
		"access/user_rate_limits/alice/down_bps": int64(20_000_000),
		"access/user_rate_limits/bob/up_bps":     int64(0),
		"access/user_rate_limits/bob/down_bps":   int64(1_000_000),
		"access/user_rate_limits/carol":          nil,
		"server/api/enabled":                     true,
		"server/api/listen":                      "127.0.0.1:19091",
		"server/api/whitelist":                   []any{"127.0.0.1/32"},
		"server/api/auth_header":                 strings.Repeat("f", 32),
		"server/api/read_only":                   true,
		// Telegram is always reached directly.
		"upstreams": nil,
	}
	for path, value := range want {
		if g := lookup(got, path); !equalValue(g, value) {
			t.Errorf("%s = %#v, want %#v", path, g, value)
		}
	}
}

func TestBuildConfigWithoutListenBindsEverywhere(t *testing.T) {
	data, err := BuildConfig(testSettings(), []User{{Name: "a", Secret: strings.Repeat("a", 32)}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "listeners") {
		t.Fatalf("empty listen must not write listeners:\n%s", data)
	}
}

func TestBuildConfigRejectsInvalidInput(t *testing.T) {
	ok := User{Name: "a", Secret: strings.Repeat("a", 32)}
	cases := map[string]struct {
		settings Settings
		users    []User
	}{
		"no users":           {testSettings(), nil},
		"bad port":           {Settings{Port: 0, TLSDomain: "example.com", MetricsPort: 9090}, []User{ok}},
		"metrics port clash": {Settings{Port: 9090, TLSDomain: "example.com", MetricsPort: 9090}, []User{ok}},
		"no metrics port":    {Settings{Port: 8443, TLSDomain: "example.com"}, []User{ok}},
		"empty domain":       {Settings{Port: 8443, MetricsPort: 9090}, []User{ok}},
		"domain with path":   {Settings{Port: 8443, TLSDomain: "example.com/x", MetricsPort: 9090}, []User{ok}},
		"domain with quote":  {Settings{Port: 8443, TLSDomain: `exa"mple.com`, MetricsPort: 9090}, []User{ok}},
		"listen not an IP":   {Settings{Listen: "example.com", Port: 8443, TLSDomain: "example.com", MetricsPort: 9090}, []User{ok}},
		"bad user name":      {testSettings(), []User{{Name: "bad name", Secret: ok.Secret}}},
		"quote in user name": {testSettings(), []User{{Name: `a"b`, Secret: ok.Secret}}},
		"short secret":       {testSettings(), []User{{Name: "a", Secret: "abc"}}},
		"uppercase secret":   {testSettings(), []User{{Name: "a", Secret: strings.Repeat("A", 32)}}},
		"duplicate user":     {testSettings(), []User{ok, ok}},
		"rate above max":     {testSettings(), []User{{Name: "a", Secret: ok.Secret, RateDownBps: MaxRateBps + 1}}},
		"API port clash":     {Settings{Port: 8443, TLSDomain: "example.com", MetricsPort: 9090, APIPort: 9090, APIToken: strings.Repeat("f", 32)}, []User{ok}},
		"API without token":  {Settings{Port: 8443, TLSDomain: "example.com", MetricsPort: 9090, APIPort: 9091}, []User{ok}},
		"API token quote":    {Settings{Port: 8443, TLSDomain: "example.com", MetricsPort: 9090, APIPort: 9091, APIToken: strings.Repeat("f", 31) + `'`}, []User{ok}},
	}
	for name, tc := range cases {
		if _, err := BuildConfig(tc.settings, tc.users); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestUserNameMapsEmailsToValidUniqueNames(t *testing.T) {
	if got := UserName("iphone-my"); got != "iphone-my" {
		t.Fatalf("valid emails must be kept, got %q", got)
	}
	seen := map[string]string{}
	for _, email := range []string{"user@mail.ru", "user_mail.ru", "Телефон Степана", "a b", "a_b", strings.Repeat("x", 80) + "@y", ""} {
		name := UserName(email)
		if !userNamePattern.MatchString(name) {
			t.Errorf("UserName(%q) = %q is not a valid telemt user name", email, name)
		}
		if other, dup := seen[name]; dup {
			t.Errorf("UserName(%q) and UserName(%q) collide on %q", email, other, name)
		}
		seen[name] = email
		if UserName(email) != name {
			t.Errorf("UserName(%q) is not stable", email)
		}
	}
}

func TestRestartKeyIgnoresUsers(t *testing.T) {
	a := testSettings()
	if a.restartKey() != testSettings().restartKey() {
		t.Fatal("identical settings must share a restart key")
	}
	for name, mutate := range map[string]func(*Settings){
		"listen":       func(s *Settings) { s.Listen = "10.0.0.1" },
		"port":         func(s *Settings) { s.Port++ },
		"domain":       func(s *Settings) { s.TLSDomain = "other.example.com" },
		"metrics port": func(s *Settings) { s.MetricsPort++ },
		"API port":     func(s *Settings) { s.APIPort = 19091 },
	} {
		c := testSettings()
		mutate(&c)
		if c.restartKey() == a.restartKey() {
			t.Errorf("changing %s must change the restart key", name)
		}
	}
}

func testWeb() *Web {
	return &Web{Host: "proxy.example.com", BasePath: "c0ffee42", PublicIP: "203.0.113.10", ListenPort: 19092, DecoyPort: 19093, MaxProfiles: 32}
}

func TestBuildConfigRendersWebProxy(t *testing.T) {
	s := testSettings()
	s.Web = testWeb()
	data, err := BuildConfig(s, []User{alice, {Name: "bob", Secret: bob.Secret, Disabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal(data, &got); err != nil {
		t.Fatalf("generated config is not valid TOML: %v\n%s", err, data)
	}
	// Listing the WEB listener stops telemt from adding its own, so the
	// Fake-TLS ones are listed too.
	listeners, _ := lookup(got, "server/listeners").([]any)
	if len(listeners) != 3 {
		t.Fatalf("want the two Fake-TLS listeners and the WEB one, got %#v\n%s", listeners, data)
	}
	for i, ip := range []string{"0.0.0.0", "::"} {
		l := listeners[i].(map[string]any)
		if l["ip"] != ip || l["transport"] != nil || l["port"] != nil {
			t.Errorf("listener %d = %#v, want a plain listener on %s", i, l, ip)
		}
	}
	webListener := map[string]any{
		"ip": "127.0.0.1", "port": int64(19092), "transport": "web", "proxy_protocol": false,
		"web_client_ip_source": "x_forwarded_for", "web_trusted_proxy_cidrs": []any{"127.0.0.1/32"},
	}
	for key, want := range webListener {
		if g := listeners[2].(map[string]any)[key]; !equalValue(g, want) {
			t.Errorf("WEB listener %s = %#v, want %#v", key, g, want)
		}
	}
	for path, want := range map[string]any{
		"web/enabled":             true,
		"web/carrier":             "https",
		"web/carriers":            []any{"websocket"},
		"web/limits/max_profiles": int64(32),
	} {
		if g := lookup(got, path); !equalValue(g, want) {
			t.Errorf("%s = %#v, want %#v", path, g, want)
		}
	}
	vhosts, _ := lookup(got, "web/vhosts").([]any)
	if len(vhosts) != 1 {
		t.Fatalf("want one vhost, got %#v", vhosts)
	}
	vhost := vhosts[0].(map[string]any)
	for path, want := range map[string]any{
		"host":           "proxy.example.com",
		"base_path":      "c0ffee42",
		"public_addr":    "203.0.113.10:443",
		"decoy/mode":     "http_upstream",
		"decoy/upstream": "http://127.0.0.1:19093",
	} {
		if g := lookup(vhost, path); !equalValue(g, want) {
			t.Errorf("vhost %s = %#v, want %#v", path, g, want)
		}
	}
	// Every user gets a profile; telemt keeps the disabled ones out itself.
	wantProfiles := []any{
		map[string]any{"user": "alice", "secret_mode": "dd"},
		map[string]any{"user": "bob", "secret_mode": "dd"},
	}
	if g := vhost["profiles"]; !equalValue(g, wantProfiles) {
		t.Errorf("profiles = %#v, want %#v", g, wantProfiles)
	}
}

func TestBuildConfigWebProxyKeepsTheListenAddressAndIPv6PublicAddress(t *testing.T) {
	s := testSettings()
	s.Listen = "10.0.0.5"
	s.Web = testWeb()
	s.Web.PublicIP = "2001:db8::1"
	data, err := BuildConfig(s, []User{alice})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	listeners, _ := lookup(got, "server/listeners").([]any)
	if len(listeners) != 2 || listeners[0].(map[string]any)["ip"] != "10.0.0.5" || listeners[1].(map[string]any)["transport"] != "web" {
		t.Fatalf("listeners = %#v", listeners)
	}
	vhost := lookup(got, "web/vhosts").([]any)[0].(map[string]any)
	if vhost["public_addr"] != "[2001:db8::1]:443" {
		t.Fatalf("public_addr = %#v", vhost["public_addr"])
	}
}

func TestBuildConfigRejectsInvalidWebProxy(t *testing.T) {
	for name, mutate := range map[string]func(*Web){
		"IP as host":            func(w *Web) { w.Host = "203.0.113.10" },
		"host without a dot":    func(w *Web) { w.Host = "localhost" },
		"numeric last label":    func(w *Web) { w.Host = "proxy.123" },
		"nested path":           func(w *Web) { w.BasePath = "a/b" },
		"empty path":            func(w *Web) { w.BasePath = "" },
		"no public IP":          func(w *Web) { w.PublicIP = "" },
		"unspecified IP":        func(w *Web) { w.PublicIP = "0.0.0.0" },
		"listener on metrics":   func(w *Web) { w.ListenPort = 19090 },
		"decoy on listener":     func(w *Web) { w.DecoyPort = w.ListenPort },
		"no decoy":              func(w *Web) { w.DecoyPort = 0 },
		"more users than seats": func(w *Web) { w.MaxProfiles = 1 },
	} {
		s := testSettings()
		s.Web = testWeb()
		mutate(s.Web)
		if _, err := BuildConfig(s, []User{alice, bob}); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestWebProfileCeiling(t *testing.T) {
	for users, want := range map[int]int{0: 32, 1: 32, 32: 32, 33: 64, 64: 64, 65: 128, 1000: 1024} {
		if got := WebProfileCeiling(users); got != want {
			t.Errorf("WebProfileCeiling(%d) = %d, want %d", users, got, want)
		}
	}
}

func TestRestartKeyReloadsWebAddressesButRestartsForItsListener(t *testing.T) {
	plain := testSettings()
	web := testSettings()
	web.Web = testWeb()
	if plain.restartKey() == web.restartKey() {
		t.Fatal("turning the WEB proxy on must restart telemt")
	}
	for name, mutate := range map[string]func(*Web){
		"host":       func(w *Web) { w.Host = "other.example.com" },
		"path":       func(w *Web) { w.BasePath = "beef" },
		"public IP":  func(w *Web) { w.PublicIP = "203.0.113.11" },
		"decoy port": func(w *Web) { w.DecoyPort++ },
	} {
		c := testSettings()
		c.Web = testWeb()
		mutate(c.Web)
		if c.restartKey() != web.restartKey() {
			t.Errorf("changing the WEB %s must only reload", name)
		}
	}
	for name, mutate := range map[string]func(*Web){
		"listener port": func(w *Web) { w.ListenPort++ },
		"profile seats": func(w *Web) { w.MaxProfiles = 64 },
	} {
		c := testSettings()
		c.Web = testWeb()
		mutate(c.Web)
		if c.restartKey() == web.restartKey() {
			t.Errorf("changing the WEB %s must restart", name)
		}
	}
}

func TestNormalizeSourceDeny(t *testing.T) {
	got, err := NormalizeSourceDeny([]string{" 203.0.113.7 ", "10.20.30.40/16", "2001:db8::1", "2001:db8:abcd::/32", "10.20.0.0/16"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"203.0.113.7/32", "10.20.0.0/16", "2001:db8::1/128", "2001:db8::/32"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, bad := range []string{"example.com", "10.0.0.0/33", "300.1.1.1", "fe80::1%eth0", "1.2.3.4/", "::/129"} {
		if _, err := NormalizeSourceDeny([]string{bad}); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if _, err := NormalizeSourceDeny(make([]string, MaxSourceDeny+1)); err == nil {
		t.Error("more than MaxSourceDeny accepted")
	}
}

func TestBuildConfigRendersSourceDenyAndUniqueIPMode(t *testing.T) {
	s := testSettings()
	s.IPLimitMode, s.IPLimitWindow = "combined", 120
	users := []User{
		{Name: "alice", Secret: "00112233445566778899aabbccddeeff", SourceDeny: []string{"203.0.113.7", "10.0.0.0/8"}},
		{Name: "bob", Secret: "0123456789abcdef0123456789abcdef"},
	}
	data, err := BuildConfig(s, users)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := toml.Unmarshal(data, &got); err != nil {
		t.Fatalf("generated config is not valid TOML: %v\n%s", err, data)
	}
	if v := lookup(got, "access/user_max_unique_ips_mode"); v != "combined" {
		t.Errorf("mode = %v", v)
	}
	if v := lookup(got, "access/user_max_unique_ips_window_secs"); v != int64(120) {
		t.Errorf("window = %v", v)
	}
	deny, _ := lookup(got, "access/user_source_deny/alice").([]any)
	if len(deny) != 2 || deny[0] != "203.0.113.7/32" || deny[1] != "10.0.0.0/8" {
		t.Errorf("alice deny = %v", deny)
	}
	if v := lookup(got, "access/user_source_deny/bob"); v != nil {
		t.Errorf("bob deny = %v", v)
	}

	// Defaults stay out of the file, telemt keeps its own.
	data, _ = BuildConfig(testSettings(), users[1:])
	for _, key := range []string{"user_source_deny", "user_max_unique_ips_mode", "user_max_unique_ips_window_secs"} {
		if strings.Contains(string(data), key) {
			t.Errorf("%s written without a value", key)
		}
	}
}

func TestBuildConfigRejectsBadSourceDenyAndUniqueIPMode(t *testing.T) {
	user := User{Name: "alice", Secret: "00112233445566778899aabbccddeeff"}
	bad := user
	bad.SourceDeny = []string{"not-an-ip"}
	if _, err := BuildConfig(testSettings(), []User{bad}); err == nil {
		t.Error("bad denied source accepted")
	}
	for _, mutate := range []func(*Settings){
		func(s *Settings) { s.IPLimitMode = "sometimes" },
		func(s *Settings) { s.IPLimitWindow = -1 },
		func(s *Settings) { s.IPLimitWindow = MaxIPLimitWindow + 1 },
	} {
		s := testSettings()
		mutate(&s)
		if _, err := BuildConfig(s, []User{user}); err == nil {
			t.Errorf("settings %+v accepted", s)
		}
	}
}

func TestSourceDenyChangeRestarts(t *testing.T) {
	a := []User{{Name: "alice", SourceDeny: []string{"203.0.113.7/32"}}}
	b := []User{{Name: "alice", SourceDeny: []string{"203.0.113.8/32"}}}
	if sourceDenyKey(a) == sourceDenyKey(b) || sourceDenyKey(a) == sourceDenyKey(nil) {
		t.Error("a change of denied sources must change the restart key")
	}
	if sourceDenyKey([]User{{Name: "alice", MaxUniqueIPs: 3}}) != sourceDenyKey(nil) {
		t.Error("other user fields must not restart")
	}
}
