package site

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	"go.uber.org/zap"
)

// CaddyTLS gives t the TLS Caddy serves by default: TLS 1.2 and up, in
// TLS 1.2 only the ECDHE suites with AEAD ciphers, and only the curves
// X25519MLKEM768, X25519 and P-256. A visitor offering nothing else is
// refused as Caddy refuses it.
func CaddyTLS(t *tls.Config) *tls.Config {
	t.MinVersion = tls.VersionTLS12
	t.CipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}
	t.CurvePreferences = []tls.CurveID{tls.X25519MLKEM768, tls.X25519, tls.CurveP256}
	return t
}

// Certs obtains and renews the site's certificate on its own, with no
// script beside the panel, from Let's Encrypt. A renewed certificate is served from the next
// handshake on, with nothing restarted. The name is a domain or an IP
// address; Let's Encrypt gives an address a certificate of some six days,
// renewed every few. The authorities check the name over port 80 or over
// port 443 by TLS. Where the panel holds neither, the check opens the port
// for its few seconds; behind REALITY port 80 answers it (HTTPChallenge)
// and 443 reaches the site like any other visitor (TLSConfig).
type Certs struct {
	domain string
	folder string
	cache  *certmagic.Cache
	cfg    *certmagic.Config
	le     *certmagic.ACMEIssuer

	mu     sync.Mutex
	failed string
	at     time.Time

	stopgap     sync.Once
	stopgapCert *tls.Certificate
}

// CertStatus is what the panel shows of the certificate.
type CertStatus struct {
	Domain string `json:"domain"`
	// Issuer and NotAfter describe the certificate served, zero while
	// there is none.
	Issuer   string    `json:"issuer"`
	NotAfter time.Time `json:"notAfter"`
	// Error is the last failure to obtain or renew it, cleared by success.
	Error   string    `json:"error"`
	ErrorAt time.Time `json:"errorAt"`
}

// NewCerts manages the certificate of domain, keeping it and the account in
// folder.
func NewCerts(folder, domain string) *Certs {
	domain = strings.Trim(domain, "[]")
	c := &Certs{domain: domain, folder: folder}
	var cfg *certmagic.Config
	c.cache = certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(certmagic.Certificate) (*certmagic.Config, error) { return cfg, nil },
		Logger:           zap.NewNop(),
	})
	cfg = certmagic.New(c.cache, certmagic.Config{
		Storage: &certmagic.FileStorage{Path: folder},
		Logger:  zap.NewNop(),
		OnEvent: c.onEvent,
	})
	le := certmagic.ACMEIssuer{
		CA:     certmagic.LetsEncryptProductionCA,
		TestCA: certmagic.LetsEncryptStagingCA,
		Agreed: true,
		Logger: zap.NewNop(),
	}
	if net.ParseIP(domain) != nil {
		// Let's Encrypt certifies addresses only in its short-lived profile.
		le.Profile = "shortlived"
		// A visitor opening an address names no server and gets the
		// certificate all the same. A domain's is not handed out without
		// its name, as Caddy refuses such a visitor.
		cfg.DefaultServerName = domain
	}
	c.le = certmagic.NewACMEIssuer(cfg, le)
	cfg.Issuers = []certmagic.Issuer{c.le}
	c.cfg = cfg
	return c
}

// Start loads the certificate from storage, or obtains one in the
// background, and keeps it renewed until Stop.
func (c *Certs) Start(ctx context.Context) error {
	return c.cfg.ManageAsync(ctx, []string{c.domain})
}

// Copies are where Xray, or anything else, finds the certificate the panel
// keeps for its website: CopyFiles in folder, rewritten whenever a new one
// is obtained, so a reader that reloads its files (Xray does every hour)
// follows the renewals. They are never removed: a configuration that names
// them keeps working with the last certificate.
const (
	copyCert = "site.crt"
	copyKey  = "site.key"
)

// CopyFiles are the copy of the certificate and its key in folder, or ""
// while there is none.
func CopyFiles(folder string) (cert, key string) {
	cert, key = filepath.Join(folder, copyCert), filepath.Join(folder, copyKey)
	if _, err := os.Stat(cert); err != nil {
		return "", ""
	}
	if _, err := os.Stat(key); err != nil {
		return "", ""
	}
	return cert, key
}

// copyCached writes the copy from the certificate in memory with the latest
// expiry.
func (c *Certs) copyCached() {
	var best certmagic.Certificate
	for _, cert := range c.cache.AllMatchingCertificates(c.domain) {
		if cert.Leaf != nil && (best.Leaf == nil || cert.Leaf.NotAfter.After(best.Leaf.NotAfter)) {
			best = cert
		}
	}
	if best.Leaf == nil {
		return
	}
	var chain []byte
	for _, der := range best.Certificate.Certificate {
		chain = append(chain, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	}
	der, err := x509.MarshalPKCS8PrivateKey(best.PrivateKey)
	if err != nil {
		return
	}
	c.writeCopy(chain, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// copyStored writes the copy from the files certmagic stored at the keys.
func (c *Certs) copyStored(certKey, keyKey string) {
	chain, err := os.ReadFile(filepath.Join(c.folder, filepath.FromSlash(certKey)))
	if err != nil {
		return
	}
	key, err := os.ReadFile(filepath.Join(c.folder, filepath.FromSlash(keyKey)))
	if err != nil {
		return
	}
	c.writeCopy(chain, key)
}

// writeCopy replaces the copy, the key first and each file whole, so a
// reader never takes a certificate with the key of another.
func (c *Certs) writeCopy(chain, key []byte) {
	if old, err := os.ReadFile(filepath.Join(c.folder, copyCert)); err == nil && bytes.Equal(old, chain) {
		return
	}
	if replaceFile(filepath.Join(c.folder, copyKey), key, 0o600) == nil {
		replaceFile(filepath.Join(c.folder, copyCert), chain, 0o644)
	}
}

func replaceFile(name string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(name), ".copy-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), name)
}

// Stop ends the renewals.
func (c *Certs) Stop() {
	c.cache.Stop()
}

// Renew obtains a new certificate now, even if the one served is fresh, and
// serves it from the next handshake on. It waits for the authority's answer,
// a first certificate included, and reports its failure.
func (c *Certs) Renew(ctx context.Context) error {
	old := c.cache.AllMatchingCertificates(c.domain)
	if err := c.cfg.RenewCertSync(ctx, c.domain, true); err != nil {
		c.mu.Lock()
		c.failed, c.at = err.Error(), time.Now()
		c.mu.Unlock()
		return err
	}
	// The renewal is in storage; the handshakes take certificates from memory.
	fresh, err := c.cfg.CacheManagedCertificate(ctx, c.domain)
	if err != nil {
		return err
	}
	var stale []string
	for _, cert := range old {
		if cert.Hash() != fresh.Hash() {
			stale = append(stale, cert.Hash())
		}
	}
	c.cache.Remove(stale)
	return nil
}

// TLSConfig serves the certificate over HTTP/2 and HTTP/1.1 and answers the
// authorities' TLS check. Until there is a certificate a self-signed one
// stands in, so the panel can be opened past the browser's warning.
func (c *Certs) TLSConfig() *tls.Config {
	t := CaddyTLS(c.cfg.TLSConfig())
	t.NextProtos = append([]string{"h2", "http/1.1"}, t.NextProtos...)
	get := t.GetCertificate
	t.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cert, err := get(hello)
		if err == nil || c.Status().Issuer != "" {
			return cert, err
		}
		return c.stopgapCertificate()
	}
	return t
}

// stopgapCertificate is the self-signed certificate served while there is
// no other.
func (c *Certs) stopgapCertificate() (*tls.Certificate, error) {
	var err error
	c.stopgap.Do(func() {
		c.stopgapCert, err = selfSigned(c.domain)
	})
	if c.stopgapCert == nil && err == nil {
		err = errors.New("no certificate yet")
	}
	return c.stopgapCert, err
}

func selfSigned(name string) (*tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(0, 0, 90),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(name); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{name}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return &tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

// HTTPChallenge answers the authorities' check over port 80 and passes
// every other request to next.
func (c *Certs) HTTPChallenge(next http.Handler) http.Handler {
	return c.le.HTTPChallengeHandler(next)
}

// Status describes the certificate served and the last failure.
func (c *Certs) Status() CertStatus {
	s := CertStatus{Domain: c.domain}
	for _, cert := range c.cache.AllMatchingCertificates(c.domain) {
		if cert.Leaf != nil && cert.Leaf.NotAfter.After(s.NotAfter) {
			s.NotAfter = cert.Leaf.NotAfter
			s.Issuer = issuerName(cert.Leaf)
		}
	}
	c.mu.Lock()
	s.Error, s.ErrorAt = c.failed, c.at
	c.mu.Unlock()
	return s
}

// FileStatus describes the certificate in an administrator's file: the name
// it is for, who issued it and until when.
func FileStatus(certFile string) (CertStatus, error) {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return CertStatus{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return CertStatus{}, fmt.Errorf("%s holds no PEM certificate", certFile)
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CertStatus{}, err
	}
	s := CertStatus{Domain: leaf.Subject.CommonName, Issuer: issuerName(leaf), NotAfter: leaf.NotAfter}
	switch {
	case len(leaf.DNSNames) > 0:
		s.Domain = leaf.DNSNames[0]
	case len(leaf.IPAddresses) > 0:
		s.Domain = leaf.IPAddresses[0].String()
	}
	return s, nil
}

func issuerName(leaf *x509.Certificate) string {
	if len(leaf.Issuer.Organization) > 0 {
		return leaf.Issuer.Organization[0]
	}
	return leaf.Issuer.CommonName
}

func (c *Certs) onEvent(_ context.Context, event string, data map[string]any) error {
	switch event {
	case "cert_failed":
		c.mu.Lock()
		c.failed, c.at = fmt.Sprint(data["error"]), time.Now()
		c.mu.Unlock()
	case "cached_managed_cert":
		// One kept from before, loaded at the start in the background.
		c.copyCached()
	case "cert_obtained":
		c.mu.Lock()
		c.failed, c.at = "", time.Time{}
		c.mu.Unlock()
		certKey, _ := data["certificate_path"].(string)
		keyKey, _ := data["private_key_path"].(string)
		c.copyStored(certKey, keyKey)
	}
	return nil
}
