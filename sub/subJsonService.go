package sub

import (
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/disp911/spacex-ui/v2/database/model"
	"github.com/disp911/spacex-ui/v2/logger"
	"github.com/disp911/spacex-ui/v2/util/json_util"
	"github.com/disp911/spacex-ui/v2/util/random"
	"github.com/disp911/spacex-ui/v2/web/service"
	"github.com/disp911/spacex-ui/v2/xray"
)

//go:embed default.json
var defaultJson string

// SubJsonService handles JSON subscription configuration generation and management.
type SubJsonService struct {
	configJson       map[string]any
	defaultOutbounds []json_util.RawMessage
	tcpMask          map[string]any
	udpMask          map[string]any
	mux              string

	inboundService service.InboundService
	SubService     *SubService
}

// NewSubJsonService creates a new JSON subscription service with the given configuration.
func NewSubJsonService(fragment string, noises string, mux string, rules string, subService *SubService) *SubJsonService {
	var configJson map[string]any
	var defaultOutbounds []json_util.RawMessage
	json.Unmarshal([]byte(defaultJson), &configJson)
	if outboundSlices, ok := configJson["outbounds"].([]any); ok {
		for _, defaultOutbound := range outboundSlices {
			jsonBytes, _ := json.Marshal(defaultOutbound)
			defaultOutbounds = append(defaultOutbounds, jsonBytes)
		}
	}

	if rules != "" {
		var newRules []any
		routing, _ := configJson["routing"].(map[string]any)
		defaultRules, _ := routing["rules"].([]any)
		json.Unmarshal([]byte(rules), &newRules)
		defaultRules = append(newRules, defaultRules...)
		routing["rules"] = defaultRules
		configJson["routing"] = routing
	}

	return &SubJsonService{
		configJson:       configJson,
		defaultOutbounds: defaultOutbounds,
		tcpMask:          fragmentMask(fragment),
		udpMask:          noiseMask(noises),
		mux:              mux,
		SubService:       subService,
	}
}

// GetJson generates a JSON subscription configuration for the given subscription ID and host.
func (s *SubJsonService) GetJson(subId string, host string) (string, string, error) {
	inbounds, err := s.SubService.getInboundsBySubId(subId)
	if err != nil || len(inbounds) == 0 {
		return "", "", err
	}

	var header string
	var traffic xray.ClientTraffic
	var clientTraffics []xray.ClientTraffic
	var configArray []json_util.RawMessage

	// Prepare Inbounds
	for _, inbound := range inbounds {
		clients, err := s.inboundService.GetClients(inbound)
		if err != nil {
			logger.Error("SubJsonService - GetClients: Unable to get clients from inbound")
		}
		if clients == nil {
			continue
		}
		if len(inbound.Listen) > 0 && inbound.Listen[0] == '@' {
			listen, port, streamSettings, err := s.SubService.getFallbackMaster(inbound.Listen, inbound.StreamSettings)
			if err == nil {
				inbound.Listen = listen
				inbound.Port = port
				inbound.StreamSettings = streamSettings
			}
		}

		for _, client := range clients {
			if client.SubID == subId {
				clientTraffics = append(clientTraffics, s.SubService.getClientTraffics(inbound.ClientStats, client.Email))
				newConfigs := s.getConfig(inbound, client, host)
				configArray = append(configArray, newConfigs...)
			}
		}
	}

	if len(configArray) == 0 {
		return "", "", nil
	}

	// Prepare statistics
	for index, clientTraffic := range clientTraffics {
		if index == 0 {
			traffic.Up = clientTraffic.Up
			traffic.Down = clientTraffic.Down
			traffic.Total = clientTraffic.Total
			if clientTraffic.ExpiryTime > 0 {
				traffic.ExpiryTime = clientTraffic.ExpiryTime
			}
		} else {
			traffic.Up += clientTraffic.Up
			traffic.Down += clientTraffic.Down
			if traffic.Total == 0 || clientTraffic.Total == 0 {
				traffic.Total = 0
			} else {
				traffic.Total += clientTraffic.Total
			}
			if clientTraffic.ExpiryTime != traffic.ExpiryTime {
				traffic.ExpiryTime = 0
			}
		}
	}

	// Combile outbounds
	var finalJson []byte
	if len(configArray) == 1 {
		finalJson, _ = json.MarshalIndent(configArray[0], "", "  ")
	} else {
		finalJson, _ = json.MarshalIndent(configArray, "", "  ")
	}

	header = fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d", traffic.Up, traffic.Down, traffic.Total, traffic.ExpiryTime/1000)
	return string(finalJson), header, nil
}

func (s *SubJsonService) getConfig(inbound *model.Inbound, client model.Client, host string) []json_util.RawMessage {
	var newJsonArray []json_util.RawMessage
	stream := s.streamData(inbound.StreamSettings)

	externalProxies, ok := stream["externalProxy"].([]any)
	if !ok || len(externalProxies) == 0 {
		externalProxies = []any{
			map[string]any{
				"forceTls": "same",
				"dest":     host,
				"port":     float64(inbound.Port),
				"remark":   "",
			},
		}
	}

	delete(stream, "externalProxy")

	for _, ep := range externalProxies {
		extPrxy := ep.(map[string]any)
		inbound.Listen = extPrxy["dest"].(string)
		inbound.Port = int(extPrxy["port"].(float64))
		// Each address gets its own copy: the TLS one row sets must not
		// reach the rows after it.
		newStream := maps.Clone(stream)
		if tls, ok := stream["tlsSettings"].(map[string]any); ok {
			newStream["tlsSettings"] = maps.Clone(tls)
		}
		switch extPrxy["forceTls"].(string) {
		case "tls":
			if newStream["security"] != "tls" {
				newStream["security"] = "tls"
				newStream["tlsSettings"] = map[string]any{}
			}
			applyExternalProxyTLS(extPrxy, newStream["tlsSettings"].(map[string]any))
		case "none":
			if newStream["security"] != "none" {
				newStream["security"] = "none"
				delete(newStream, "tlsSettings")
			}
		}
		// Hysteria builds its stream anew and puts the masks in itself.
		if !model.IsHysteria(inbound.Protocol) {
			s.addMasks(newStream)
		}
		streamSettings, _ := json.MarshalIndent(newStream, "", "  ")

		var newOutbounds []json_util.RawMessage

		switch inbound.Protocol {
		case "vmess":
			newOutbounds = append(newOutbounds, s.genVnext(inbound, streamSettings, client))
		case "vless":
			newOutbounds = append(newOutbounds, s.genVless(inbound, streamSettings, client))
		case "trojan", "shadowsocks":
			newOutbounds = append(newOutbounds, s.genServer(inbound, streamSettings, client))
		case "hysteria", "hysteria2":
			newOutbounds = append(newOutbounds, s.genHy(inbound, newStream, client))
		}

		newOutbounds = append(newOutbounds, s.defaultOutbounds...)
		newConfigJson := make(map[string]any)
		maps.Copy(newConfigJson, s.configJson)

		newConfigJson["outbounds"] = newOutbounds
		newConfigJson["remarks"] = s.SubService.genRemark(inbound, client.Email, extPrxy["remark"].(string))

		newConfig, _ := json.MarshalIndent(newConfigJson, "", "  ")
		newJsonArray = append(newJsonArray, newConfig)
	}

	return newJsonArray
}

func (s *SubJsonService) streamData(stream string) map[string]any {
	var streamSettings map[string]any
	json.Unmarshal([]byte(stream), &streamSettings)
	security, _ := streamSettings["security"].(string)
	switch security {
	case "tls":
		streamSettings["tlsSettings"] = s.tlsData(streamSettings["tlsSettings"].(map[string]any))
	case "reality":
		streamSettings["realitySettings"] = s.realityData(streamSettings["realitySettings"].(map[string]any))
	}
	delete(streamSettings, "sockopt")

	// remove proxy protocol
	network, _ := streamSettings["network"].(string)
	switch network {
	case "xhttp":
		// The client's side only, as a link's extra has it: no settings of
		// the server's own, no Host header (Xray takes it from "host" alone)
		// and no header without a value.
		if xhttp, ok := streamSettings["xhttpSettings"].(map[string]any); ok {
			for k := range xhttp {
				if xhttpServerOnly[k] && k != "path" && k != "host" && k != "mode" {
					delete(xhttp, k)
				}
			}
			if headers, ok := xhttp["headers"].(map[string]any); ok {
				for k, v := range headers {
					if strings.EqualFold(k, "host") || v == "" || v == nil {
						delete(headers, k)
					}
				}
			}
		}
	case "tcp":
		streamSettings["tcpSettings"] = s.removeAcceptProxy(streamSettings["tcpSettings"])
	case "ws":
		streamSettings["wsSettings"] = s.removeAcceptProxy(streamSettings["wsSettings"])
	case "httpupgrade":
		streamSettings["httpupgradeSettings"] = s.removeAcceptProxy(streamSettings["httpupgradeSettings"])
	}
	return streamSettings
}

// applyExternalProxyTLS puts the SNI, fingerprint and ALPN an External Proxy
// row with Force TLS sets into the client's TLS, over the inbound's, as the
// row's links have them (externalProxyTLS).
func applyExternalProxyTLS(ep map[string]any, tlsSettings map[string]any) {
	own := externalProxyTLS(ep)
	if sni, ok := own["sni"]; ok {
		tlsSettings["serverName"] = sni
	}
	if fp, ok := own["fp"]; ok {
		tlsSettings["fingerprint"] = fp
	}
	if alpn, ok := own["alpn"]; ok {
		tlsSettings["alpn"] = strings.Split(alpn, ",")
	}
}

func (s *SubJsonService) removeAcceptProxy(setting any) map[string]any {
	netSettings, ok := setting.(map[string]any)
	if ok {
		delete(netSettings, "acceptProxyProtocol")
	}
	return netSettings
}

func (s *SubJsonService) tlsData(tData map[string]any) map[string]any {
	tlsData := make(map[string]any, 1)
	tlsClientSettings, _ := tData["settings"].(map[string]any)

	tlsData["serverName"] = tData["serverName"]
	tlsData["alpn"] = tData["alpn"]
	if fingerprint, ok := tlsClientSettings["fingerprint"].(string); ok {
		tlsData["fingerprint"] = fingerprint
	}
	return tlsData
}

func (s *SubJsonService) realityData(rData map[string]any) map[string]any {
	rltyData := make(map[string]any, 1)
	rltyClientSettings, _ := rData["settings"].(map[string]any)

	rltyData["show"] = false
	rltyData["publicKey"] = rltyClientSettings["publicKey"]
	rltyData["fingerprint"] = rltyClientSettings["fingerprint"]
	rltyData["mldsa65Verify"] = rltyClientSettings["mldsa65Verify"]

	// Set random data
	rltyData["spiderX"] = "/" + random.Seq(15)
	shortIds, ok := rData["shortIds"].([]any)
	if ok && len(shortIds) > 0 {
		rltyData["shortId"] = shortIds[random.Num(len(shortIds))].(string)
	} else {
		rltyData["shortId"] = ""
	}
	serverNames, ok := rData["serverNames"].([]any)
	if ok && len(serverNames) > 0 {
		rltyData["serverName"] = serverNames[random.Num(len(serverNames))].(string)
	} else {
		rltyData["serverName"] = ""
	}

	return rltyData
}

func (s *SubJsonService) genVnext(inbound *model.Inbound, streamSettings json_util.RawMessage, client model.Client) json_util.RawMessage {
	outbound := Outbound{}
	usersData := make([]UserVnext, 1)

	usersData[0].ID = client.ID
	usersData[0].Email = client.Email
	usersData[0].Security = client.Security
	vnextData := make([]VnextSetting, 1)
	vnextData[0] = VnextSetting{
		Address: inbound.Listen,
		Port:    inbound.Port,
		Users:   usersData,
	}

	outbound.Protocol = string(inbound.Protocol)
	outbound.Tag = "proxy"
	if s.mux != "" {
		outbound.Mux = json_util.RawMessage(s.mux)
	}
	outbound.StreamSettings = streamSettings
	outbound.Settings = map[string]any{
		"vnext": vnextData,
	}

	result, _ := json.MarshalIndent(outbound, "", "  ")
	return result
}

func (s *SubJsonService) genVless(inbound *model.Inbound, streamSettings json_util.RawMessage, client model.Client) json_util.RawMessage {
	outbound := Outbound{}
	outbound.Protocol = string(inbound.Protocol)
	outbound.Tag = "proxy"
	if s.mux != "" {
		outbound.Mux = json_util.RawMessage(s.mux)
		if strings.HasPrefix(client.Flow, "xtls-rprx-vision") {
			outbound.Mux = muxForVision(s.mux)
		}
	}
	outbound.StreamSettings = streamSettings

	// Add encryption for VLESS outbound from inbound settings
	var inboundSettings map[string]any
	json.Unmarshal([]byte(inbound.Settings), &inboundSettings)
	encryption, _ := inboundSettings["encryption"].(string)

	user := map[string]any{
		"id":         client.ID,
		"level":      8,
		"encryption": encryption,
	}
	if client.Flow != "" {
		user["flow"] = client.Flow
	}

	vnext := map[string]any{
		"address": inbound.Listen,
		"port":    inbound.Port,
		"users":   []any{user},
	}

	outbound.Settings = map[string]any{
		"vnext": []any{vnext},
	}
	result, _ := json.MarshalIndent(outbound, "", "  ")
	return result
}

func (s *SubJsonService) genServer(inbound *model.Inbound, streamSettings json_util.RawMessage, client model.Client) json_util.RawMessage {
	outbound := Outbound{}

	serverData := make([]ServerSetting, 1)
	serverData[0] = ServerSetting{
		Address:  inbound.Listen,
		Port:     inbound.Port,
		Level:    8,
		Password: client.Password,
	}

	if inbound.Protocol == model.Shadowsocks {
		var inboundSettings map[string]any
		json.Unmarshal([]byte(inbound.Settings), &inboundSettings)
		method, _ := inboundSettings["method"].(string)
		serverData[0].Method = method

		// server password in multi-user 2022 protocols
		if strings.HasPrefix(method, "2022") {
			if serverPassword, ok := inboundSettings["password"].(string); ok {
				serverData[0].Password = fmt.Sprintf("%s:%s", serverPassword, client.Password)
			}
		}
	}

	outbound.Protocol = string(inbound.Protocol)
	outbound.Tag = "proxy"
	if s.mux != "" {
		outbound.Mux = json_util.RawMessage(s.mux)
	}
	outbound.StreamSettings = streamSettings
	outbound.Settings = map[string]any{
		"servers": serverData,
	}

	result, _ := json.MarshalIndent(outbound, "", "  ")
	return result
}

func (s *SubJsonService) genHy(inbound *model.Inbound, newStream map[string]any, client model.Client) json_util.RawMessage {
	outbound := Outbound{}

	outbound.Protocol = string(inbound.Protocol)
	outbound.Tag = "proxy"

	if s.mux != "" {
		outbound.Mux = json_util.RawMessage(s.mux)
	}

	var settings, stream map[string]any
	json.Unmarshal([]byte(inbound.Settings), &settings)
	version, _ := settings["version"].(float64)
	outbound.Settings = map[string]any{
		"version": int(version),
		"address": inbound.Listen,
		"port":    inbound.Port,
	}

	json.Unmarshal([]byte(inbound.StreamSettings), &stream)
	hyStream := stream["hysteriaSettings"].(map[string]any)
	outHyStream := map[string]any{
		"version": int(version),
		"auth":    client.Auth,
	}
	if udpIdleTimeout, ok := hyStream["udpIdleTimeout"].(float64); ok {
		outHyStream["udpIdleTimeout"] = int(udpIdleTimeout)
	}
	newStream["hysteriaSettings"] = outHyStream

	if finalmask, ok := hyStream["finalmask"].(map[string]any); ok {
		newStream["finalmask"] = finalmask
	}
	s.addMasks(newStream)

	newStream["network"] = "hysteria"
	newStream["security"] = "tls"

	outbound.StreamSettings, _ = json.MarshalIndent(newStream, "", "  ")

	result, _ := json.MarshalIndent(outbound, "", "  ")
	return result
}

type Outbound struct {
	Protocol       string               `json:"protocol"`
	Tag            string               `json:"tag"`
	StreamSettings json_util.RawMessage `json:"streamSettings"`
	Mux            json_util.RawMessage `json:"mux,omitempty"`
	Settings       map[string]any       `json:"settings,omitempty"`
}

type VnextSetting struct {
	Address string      `json:"address"`
	Port    int         `json:"port"`
	Users   []UserVnext `json:"users"`
}

type UserVnext struct {
	ID       string `json:"id"`
	Email    string `json:"email,omitempty"`
	Security string `json:"security,omitempty"`
}

type ServerSetting struct {
	Password string `json:"password"`
	Level    int    `json:"level"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Flow     string `json:"flow,omitempty"`
	Method   string `json:"method,omitempty"`
}

// muxForVision fits the mux of a client with Vision: its server takes only
// UDP (XUDP) inside mux and breaks the whole mux connection on a TCP
// stream, so TCP is left out of mux (concurrency -1) and goes by Vision.
func muxForVision(mux string) json_util.RawMessage {
	var m map[string]any
	if err := json.Unmarshal([]byte(mux), &m); err != nil {
		return json_util.RawMessage(mux)
	}
	if on, _ := m["enabled"].(bool); on {
		m["concurrency"] = -1
	}
	out, err := json.Marshal(m)
	if err != nil {
		return json_util.RawMessage(mux)
	}
	return out
}

// fragmentMask turns the fragmentation of the subscription's settings into
// the "fragment" mask of finalmask.tcp, which took the place of the freedom
// outbound's "fragment" in Xray 26. The settings are kept as they were
// saved: "interval" is what the mask calls "delay".
func fragmentMask(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(raw), &settings); err != nil || len(settings) == 0 {
		logger.Warning("SubJsonService - the fragmentation setting is not JSON:", err)
		return nil
	}
	if interval, ok := settings["interval"]; ok {
		if _, has := settings["delay"]; !has {
			settings["delay"] = interval
		}
		delete(settings, "interval")
	}
	return map[string]any{"type": "fragment", "settings": settings}
}

// noiseMask turns the noises of the subscription's settings, a list in the
// freedom outbound's shape ({type, packet, delay, applyTo}), into the "noise"
// mask of finalmask.udp, which took its place in Xray 26. A random packet
// gives its length as "rand"; base64 is sent as hex, since the freedom
// outbound read any base64 and the mask reads only the standard one. The
// mask has no "applyTo": it sends its noise whatever the server's address.
func noiseMask(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	var noises []map[string]any
	if err := json.Unmarshal([]byte(raw), &noises); err != nil {
		logger.Warning("SubJsonService - the noises setting is not JSON:", err)
		return nil
	}
	items := make([]any, 0, len(noises))
	for _, n := range noises {
		item := map[string]any{}
		if delay, ok := n["delay"]; ok && delay != "" {
			item["delay"] = delay
		}
		typ, _ := n["type"].(string)
		packet, _ := n["packet"].(string)
		packet = strings.TrimSpace(packet)
		switch strings.ToLower(typ) {
		case "rand":
			item["rand"] = packet
		case "base64":
			b, err := base64.RawURLEncoding.DecodeString(strings.NewReplacer("+", "-", "/", "_", "=", "").Replace(packet))
			if err != nil {
				logger.Warning("SubJsonService - a noise packet is not base64:", err)
				continue
			}
			item["type"], item["packet"] = "hex", hex.EncodeToString(b)
		default:
			item["type"], item["packet"] = typ, packet
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil
	}
	return map[string]any{"type": "noise", "settings": map[string]any{"noise": items}}
}

// addMasks puts the subscription's fragmentation and noise into the
// finalmask of an outbound's stream, beside the masks of the inbound's own,
// as a new map: the inbound's stays as it is for the next address.
func (s *SubJsonService) addMasks(stream map[string]any) {
	if s.tcpMask == nil && s.udpMask == nil {
		return
	}
	finalmask := map[string]any{}
	if own, ok := stream["finalmask"].(map[string]any); ok {
		maps.Copy(finalmask, own)
	}
	if s.tcpMask != nil {
		finalmask["tcp"] = withMask(finalmask["tcp"], s.tcpMask)
	}
	if s.udpMask != nil {
		finalmask["udp"] = withMask(finalmask["udp"], s.udpMask)
	}
	stream["finalmask"] = finalmask
}

// withMask adds a mask next to the socket, where the freedom outbound used
// to work: last in the list, since Xray wraps the last mask first. Only
// udphop and xicmp, which make the socket themselves, stay below it.
func withMask(list any, mask map[string]any) []any {
	masks, _ := list.([]any)
	masks = slices.Clone(masks)
	at := len(masks)
	for at > 0 {
		m, _ := masks[at-1].(map[string]any)
		typ, _ := m["type"].(string)
		if t := strings.ToLower(typ); t != "udphop" && t != "xicmp" {
			break
		}
		at--
	}
	return slices.Insert(masks, at, any(mask))
}
