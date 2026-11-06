package site

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"slices"
	"testing"

	"github.com/caddyserver/certmagic"
)

func writeFile(t *testing.T, name, data string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCertsComeFromLetsEncryptOnly(t *testing.T) {
	c := NewCerts(t.TempDir(), "example.com")
	defer c.Stop()
	if n := len(c.cfg.Issuers); n != 1 || c.cfg.Issuers[0] != c.le || c.le.CA != certmagic.LetsEncryptProductionCA {
		t.Fatalf("%d issuers, CA %q", n, c.le.CA)
	}
}

func TestCertsTLSConfig(t *testing.T) {
	c := NewCerts(t.TempDir(), "example.com")
	defer c.Stop()
	cfg := c.TLSConfig()
	if !slices.Equal(cfg.NextProtos, []string{"h2", "http/1.1", "acme-tls/1"}) || cfg.MinVersion != tls.VersionTLS12 || cfg.GetCertificate == nil {
		t.Fatalf("got %v, min %x", cfg.NextProtos, cfg.MinVersion)
	}
	if s := c.Status(); s.Domain != "example.com" || !s.NotAfter.IsZero() || s.Error != "" {
		t.Fatalf("status %+v", s)
	}
}

// handshake connects a client with cfg to a server with CaddyTLS and the
// certificate Certs stands in with.
func handshake(t *testing.T, cfg *tls.Config) error {
	t.Helper()
	c := NewCerts(t.TempDir(), "example.com")
	defer c.Stop()
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	go tls.Server(a, c.TLSConfig()).Handshake()
	cfg.ServerName, cfg.InsecureSkipVerify = "example.com", true
	return tls.Client(b, cfg).Handshake()
}

func TestCaddyTLSRefusesWhatCaddyRefuses(t *testing.T) {
	for _, curve := range []tls.CurveID{tls.X25519MLKEM768, tls.X25519, tls.CurveP256, tls.CurveP384, tls.CurveP521} {
		for _, v := range []uint16{tls.VersionTLS12, tls.VersionTLS13} {
			err := handshake(t, &tls.Config{MinVersion: v, MaxVersion: v, CurvePreferences: []tls.CurveID{curve}})
			want := curve != tls.CurveP384 && curve != tls.CurveP521
			if v == tls.VersionTLS12 && curve == tls.X25519MLKEM768 {
				continue // a TLS 1.2 key exchange has no such group
			}
			if (err == nil) != want {
				t.Errorf("%s %s: %v", tls.VersionName(v), curve, err)
			}
		}
	}
	for _, s := range append(tls.CipherSuites(), tls.InsecureCipherSuites()...) {
		if !slices.Contains(s.SupportedVersions, tls.VersionTLS12) {
			continue
		}
		err := handshake(t, &tls.Config{MaxVersion: tls.VersionTLS12, CipherSuites: []uint16{s.ID}})
		want := s.ID == tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 || s.ID == tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384 || s.ID == tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
		if (err == nil) != want {
			t.Errorf("%s: %v", s.Name, err)
		}
	}
	if err := handshake(t, &tls.Config{MaxVersion: tls.VersionTLS11}); err == nil {
		t.Error("TLS 1.1 is taken")
	}
}

func TestCertsStandInUntilThereIsOne(t *testing.T) {
	for _, name := range []string{"docs.example.com", "203.0.113.7", "2001:db8::7"} {
		c := NewCerts(t.TempDir(), name)
		if ip := net.ParseIP(name); (ip != nil) != (c.le.Profile == "shortlived") {
			t.Errorf("%s: profile %q", name, c.le.Profile)
		}
		a, b := net.Pipe()
		cert, err := c.TLSConfig().GetCertificate(&tls.ClientHelloInfo{Conn: a})
		a.Close()
		b.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			t.Fatal(err)
		}
		if err := leaf.VerifyHostname(name); err != nil {
			t.Errorf("%s: the stand-in is for another name: %v", name, err)
		}
		c.Stop()
	}
}
