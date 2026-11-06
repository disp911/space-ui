package service

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/disp911/spacex-ui/v2/config"
	"github.com/disp911/spacex-ui/v2/database"
	"github.com/disp911/spacex-ui/v2/database/model"
	"github.com/disp911/spacex-ui/v2/util/common"
	"github.com/disp911/spacex-ui/v2/util/random"
)

// SiteService switches where the panel and its website answer. Behind
// REALITY an inbound on 443 whose target is the website's socket carries
// them, the site inbound: whoever is not a client gets the website itself
// ("selfsteal"). It is an ordinary inbound the administrator may change or
// delete; the panel finds it by its target. On the panel's own port the
// socket is shut, so the inbounds aimed at it are switched off, and their
// clients keep their keys for when REALITY comes back.
type SiteService struct {
	settingService SettingService
	inboundService InboundService
}

// SwitchMode puts the website in the mode, with the site inbound made,
// fitted or switched off to suit. It changes only the database: the panel
// takes the mode on its next start. The site inbound is returned while the
// mode is SiteReality.
func (s *SiteService) SwitchMode(mode string) (*model.Inbound, error) {
	if mode != SitePort && mode != SiteReality {
		return nil, common.NewErrorf("unknown website mode %q", mode)
	}
	domain, err := s.settingService.GetSiteDomain()
	if err != nil {
		return nil, err
	}
	port, err := s.settingService.GetPort()
	if err != nil {
		return nil, err
	}
	if mode == SiteReality {
		if domain == "" || net.ParseIP(domain) != nil {
			return nil, common.NewError("REALITY needs a domain name for the website, not an IP address")
		}
		if p, _ := s.settingService.GetBasePath(); p == "/" {
			return nil, common.NewError("the panel needs a secret path to share port 443 with the website")
		}
	}

	ib, err := s.SiteInbound()
	if err != nil {
		return nil, err
	}
	if mode == SiteReality {
		var skip []int
		if ib != nil {
			skip = []int{ib.Id}
		}
		if err := s.tcpFree(443, skip, false); err != nil {
			return nil, err
		}
		if ib == nil {
			ib, err = s.makeInbound(domain)
		} else {
			err = s.fitInbound(ib, domain)
		}
		if err != nil {
			return nil, err
		}
	} else {
		aimed, err := s.socketInbounds()
		if err != nil {
			return nil, err
		}
		if err := s.tcpFree(port, aimed, true); err != nil {
			return nil, err
		}
		if len(aimed) > 0 {
			if err := database.GetDB().Model(model.Inbound{}).Where("id IN ?", aimed).Update("enable", false).Error; err != nil {
				return nil, err
			}
		}
		ib = nil
	}
	if err := s.settingService.SetSiteMode(mode); err != nil {
		return nil, err
	}
	return ib, nil
}

// SiteInbound is the inbound on port 443 whose target is the website's
// socket, a switched-on one first; nil when there is none.
func (s *SiteService) SiteInbound() (*model.Inbound, error) {
	var list []*model.Inbound
	if err := database.GetDB().Model(model.Inbound{}).Where("port = ?", 443).Order("enable DESC, id").Find(&list).Error; err != nil {
		return nil, err
	}
	for _, ib := range list {
		if tcp, _ := inboundNets(ib); tcp && siteTarget(ib) {
			return ib, nil
		}
	}
	return nil, nil
}

// socketInbounds are the switched-on inbounds whose target is the website's
// socket, on any port.
func (s *SiteService) socketInbounds() ([]int, error) {
	var list []*model.Inbound
	if err := database.GetDB().Model(model.Inbound{}).Where("enable = ?", true).Find(&list).Error; err != nil {
		return nil, err
	}
	var ids []int
	for _, ib := range list {
		if siteTarget(ib) {
			ids = append(ids, ib.Id)
		}
	}
	return ids, nil
}

// tcpFree makes sure no other inbound holds the TCP port: a switched-on one
// would keep the website from it, and with REALITY in front a switched-off
// one could not be edited or switched on without breaking it. Inbounds over
// UDP, such as Hysteria on 443, stay beside the website.
func (s *SiteService) tcpFree(port int, skip []int, enabledOnly bool) error {
	q := database.GetDB().Model(model.Inbound{}).Where("port = ?", port)
	if len(skip) > 0 {
		q = q.Where("id NOT IN ?", skip)
	}
	if enabledOnly {
		q = q.Where("enable = ?", true)
	}
	var others []*model.Inbound
	if err := q.Find(&others).Error; err != nil {
		return err
	}
	var names []string
	for _, o := range others {
		if tcp, _ := inboundNets(o); tcp {
			names = append(names, fmt.Sprintf("%q (id %d)", o.Remark, o.Id))
		}
	}
	if len(names) == 0 {
		return nil
	}
	return common.NewErrorf("TCP port %d is taken by inbound %s: move it to another port first", port, strings.Join(names, ", "))
}

func (s *SiteService) makeInbound(domain string) (*model.Inbound, error) {
	db := database.GetDB()
	var userId int
	if err := db.Model(model.User{}).Select("id").Order("id").Limit(1).Scan(&userId).Error; err != nil {
		return nil, err
	}
	ib, client, err := newSiteInbound(domain, config.GetSiteSocket())
	if err != nil {
		return nil, err
	}
	ib.UserId = userId
	for {
		taken, err := s.inboundService.checkEmailsExistForClients([]model.Client{*client})
		if err != nil {
			return nil, err
		}
		if taken == "" {
			break
		}
		client.Email = lowerNum(8)
		if err := setClients(ib, client); err != nil {
			return nil, err
		}
	}

	tx := db.Begin()
	if err := tx.Save(ib).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := s.inboundService.AddClientStat(tx, ib.Id, client); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return ib, nil
}

// fitInbound switches the site inbound back on, on port 443 and for the
// domain set now.
func (s *SiteService) fitInbound(ib *model.Inbound, domain string) error {
	var stream map[string]any
	if err := json.Unmarshal([]byte(ib.StreamSettings), &stream); err != nil {
		return err
	}
	reality, _ := stream["realitySettings"].(map[string]any)
	changed := false
	if xver, _ := reality["xver"].(float64); xver == 0 {
		// Made before the website took the visitor's address from Xray.
		reality["xver"] = 1
		changed = true
	}
	if names, _ := reality["serverNames"].([]any); !slices.Contains(names, any(domain)) {
		reality["serverNames"] = []string{domain}
		changed = true
	}
	if changed {
		out, err := json.MarshalIndent(stream, "", "  ")
		if err != nil {
			return err
		}
		ib.StreamSettings = string(out)
	}
	ib.Enable = true
	ib.Port = 443
	ib.Tag = InboundTag(ib)
	return database.GetDB().Save(ib).Error
}

// newSiteInbound makes the site inbound with its first client: VLESS with
// Vision over REALITY on port 443 with fresh keys, which sends whoever is
// not a client to the website's socket.
func newSiteInbound(domain, socket string) (*model.Inbound, *model.Client, error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UnixMilli()
	client := &model.Client{
		ID:        uuid.NewString(),
		Flow:      "xtls-rprx-vision",
		Email:     lowerNum(8),
		Enable:    true,
		SubID:     lowerNum(16),
		CreatedAt: now,
		UpdatedAt: now,
	}
	stream := map[string]any{
		"network":  "tcp",
		"security": "reality",
		// Links take the address the panel or the subscription is opened
		// at: behind REALITY that is the domain on 443.
		"realitySettings": map[string]any{
			"show": false,
			// The website learns each visitor's address from a PROXY
			// protocol header.
			"xver":         1,
			"target":       socket,
			"serverNames":  []string{domain},
			"privateKey":   base64.RawURLEncoding.EncodeToString(key.Bytes()),
			"minClientVer": "",
			"maxClientVer": "",
			"maxTimediff":  0,
			"shortIds":     shortIds(),
			"mldsa65Seed":  "",
			"settings": map[string]any{
				"publicKey":     base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()),
				"fingerprint":   "chrome",
				"serverName":    "",
				"spiderX":       "/",
				"mldsa65Verify": "",
			},
			// Nothing passed on to the website is slowed down: it is our own
			// site, not someone else's to be relayed to, and XHTTP and the
			// WEB proxy reach their inbounds through it. Caddy does not slow
			// an upload down either.
		},
		"tcpSettings": map[string]any{"acceptProxyProtocol": false, "header": map[string]any{"type": "none"}},
	}
	streamJSON, err := json.MarshalIndent(stream, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	sniffing, err := json.MarshalIndent(map[string]any{
		"enabled": false, "destOverride": []string{"http", "tls", "quic", "fakedns"}, "metadataOnly": false, "routeOnly": false,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	ib := &model.Inbound{
		Enable:         true,
		TrafficReset:   "never",
		Port:           443,
		Protocol:       model.VLESS,
		StreamSettings: string(streamJSON),
		Tag:            "inbound-443",
		Sniffing:       string(sniffing),
	}
	if err := setClients(ib, client); err != nil {
		return nil, nil, err
	}
	return ib, client, nil
}

func setClients(ib *model.Inbound, client *model.Client) error {
	settings, err := json.MarshalIndent(map[string]any{
		"clients":    []*model.Client{client},
		"decryption": "none",
		"encryption": "none",
		"fallbacks":  []any{},
		"testseed":   []int{900, 500, 900, 256},
	}, "", "  ")
	if err != nil {
		return err
	}
	ib.Settings = string(settings)
	return nil
}

// shortIds are of every even length from 2 to 16 hex digits in random
// order, as the inbound window makes them.
func shortIds() []string {
	ids := make([]string, 0, 8)
	for n := 2; n <= 16; n += 2 {
		b := make([]byte, n/2)
		rand.Read(b)
		ids = append(ids, hex.EncodeToString(b))
	}
	for i := len(ids) - 1; i > 0; i-- {
		j := random.Num(i + 1)
		ids[i], ids[j] = ids[j], ids[i]
	}
	return ids
}

func lowerNum(n int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[random.Num(len(chars))]
	}
	return string(b)
}

// FirstClientLinks are what the site inbound's first client connects with:
// its VLESS link and, while subscriptions are on, its subscription link,
// both through the website's domain on port 443.
func (s *SiteService) FirstClientLinks(ib *model.Inbound) (link, sub string, err error) {
	clients, err := s.inboundService.GetClients(ib)
	if err != nil || len(clients) == 0 {
		return "", "", err
	}
	c := clients[0]
	var stream struct {
		Reality struct {
			ServerNames []string `json:"serverNames"`
			ShortIds    []string `json:"shortIds"`
			Settings    struct {
				PublicKey   string `json:"publicKey"`
				Fingerprint string `json:"fingerprint"`
				SpiderX     string `json:"spiderX"`
			} `json:"settings"`
		} `json:"realitySettings"`
	}
	if err := json.Unmarshal([]byte(ib.StreamSettings), &stream); err != nil {
		return "", "", err
	}
	r := stream.Reality
	domain, err := s.settingService.GetSiteDomain()
	if err != nil {
		return "", "", err
	}
	q := url.Values{}
	q.Set("type", "tcp")
	q.Set("encryption", "none")
	q.Set("security", "reality")
	q.Set("pbk", r.Settings.PublicKey)
	q.Set("fp", r.Settings.Fingerprint)
	if len(r.ServerNames) > 0 {
		q.Set("sni", r.ServerNames[0])
	}
	if len(r.ShortIds) > 0 {
		q.Set("sid", r.ShortIds[0])
	}
	q.Set("spx", r.Settings.SpiderX)
	if c.Flow != "" {
		q.Set("flow", c.Flow)
	}
	name := c.Email
	if ib.Remark != "" {
		name = ib.Remark + "-" + c.Email
	}
	link = (&url.URL{
		Scheme:   "vless",
		User:     url.User(c.ID),
		Host:     net.JoinHostPort(domain, "443"),
		RawQuery: q.Encode(),
		Fragment: name,
	}).String()

	if on, _ := s.settingService.GetSubEnable(); on && c.SubID != "" {
		sub, _ = s.settingService.GetSubURI()
		if sub == "" {
			path, _ := s.settingService.GetSubPath()
			sub = "https://" + domain + path
		}
		if !strings.HasSuffix(sub, "/") {
			sub += "/"
		}
		sub += c.SubID
	}
	return link, sub, nil
}

// siteTarget reports whether an inbound sends whoever is not a client to the
// website's socket: VLESS over REALITY with the socket as its target.
func siteTarget(ib *model.Inbound) bool {
	if ib.Protocol != model.VLESS {
		return false
	}
	var stream struct {
		Security string `json:"security"`
		Reality  struct {
			Target string `json:"target"`
			Dest   string `json:"dest"`
		} `json:"realitySettings"`
	}
	if json.Unmarshal([]byte(ib.StreamSettings), &stream) != nil || stream.Security != "reality" {
		return false
	}
	target := stream.Reality.Target
	if target == "" {
		target = stream.Reality.Dest
	}
	return target == config.GetSiteSocket()
}

// guardPanelPort refuses an inbound that would take the port the panel
// listens on: its own port, unless it is behind REALITY and opens none.
// Inbounds over UDP share it.
func (s *InboundService) guardPanelPort(next *model.Inbound) error {
	if !next.Enable {
		return nil
	}
	if tcp, _ := inboundNets(next); !tcp {
		return nil
	}
	settings := SettingService{}
	if mode, err := settings.GetSiteMode(); err != nil || mode == SiteReality {
		return err
	}
	port, _ := settings.GetPort()
	listen, _ := settings.GetListen()
	if next.Port == port && listensOverlap(next.Listen, listen) {
		return common.NewErrorf("TCP port %d is the panel's own. Inbounds over UDP, such as Hysteria, may use it.", port)
	}
	return nil
}

// inboundNets reports whether an inbound listens over TCP, UDP or both.
func inboundNets(ib *model.Inbound) (tcp, udp bool) {
	switch ib.Protocol {
	case model.Hysteria, model.Hysteria2, model.WireGuard:
		return false, true
	case model.Shadowsocks, model.Tunnel, model.Mixed:
		var settings struct {
			Network string `json:"network"`
			UDP     bool   `json:"udp"`
		}
		json.Unmarshal([]byte(ib.Settings), &settings)
		if ib.Protocol == model.Mixed {
			return true, settings.UDP
		}
		if settings.Network == "" {
			return true, ib.Protocol == model.Shadowsocks
		}
		return strings.Contains(settings.Network, "tcp"), strings.Contains(settings.Network, "udp")
	}
	var stream struct {
		Network string `json:"network"`
	}
	json.Unmarshal([]byte(ib.StreamSettings), &stream)
	if stream.Network == "kcp" || stream.Network == "hysteria" {
		return false, true
	}
	return true, false
}

// listensOverlap reports whether two listen addresses share a socket: the
// same address, or either of them every address.
func listensOverlap(a, b string) bool {
	any := func(l string) bool { return l == "" || l == "0.0.0.0" || l == "::" || l == "::0" }
	return any(a) || any(b) || a == b
}
