package service

import (
	"encoding/json"
	"errors"
	"runtime"
	"sync"

	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"
	"github.com/disp911/space-ui/v3/xray"

	"go.uber.org/atomic"
)

var (
	p                 *xray.Process
	lock              sync.Mutex
	isNeedXrayRestart atomic.Bool // Indicates that restart was requested for Xray
	isManuallyStopped atomic.Bool // Indicates that Xray was stopped manually from the panel
	result            string
)

// XrayService provides business logic for Xray process management.
// It handles starting, stopping, restarting Xray, and managing its configuration.
type XrayService struct {
	inboundService InboundService
	settingService SettingService
	xrayAPI        xray.XrayAPI
}

// IsXrayRunning checks if the Xray process is currently running.
func (s *XrayService) IsXrayRunning() bool {
	return p != nil && p.IsRunning()
}

// GetXrayErr returns the error from the Xray process, if any.
func (s *XrayService) GetXrayErr() error {
	if p == nil {
		return nil
	}

	err := p.GetErr()
	if err == nil {
		return nil
	}

	if runtime.GOOS == "windows" && err.Error() == "exit status 1" {
		// exit status 1 on Windows means that Xray process was killed
		// as we kill process to stop in on Windows, this is not an error
		return nil
	}

	return err
}

// GetXrayResult returns the result string from the Xray process.
func (s *XrayService) GetXrayResult() string {
	if result != "" {
		return result
	}
	if s.IsXrayRunning() {
		return ""
	}
	if p == nil {
		return ""
	}

	result = p.GetResult()

	if runtime.GOOS == "windows" && result == "exit status 1" {
		// exit status 1 on Windows means that Xray process was killed
		// as we kill process to stop in on Windows, this is not an error
		return ""
	}

	return result
}

// GetXrayVersion returns the version of the running Xray process.
func (s *XrayService) GetXrayVersion() string {
	if p == nil {
		return "Unknown"
	}
	return p.GetVersion()
}

// RemoveIndex removes an element at the specified index from a slice.
// Returns a new slice with the element removed.
func RemoveIndex(s []any, index int) []any {
	return append(s[:index], s[index+1:]...)
}

// GetXrayConfig retrieves and builds the Xray configuration from settings and inbounds.
func (s *XrayService) GetXrayConfig() (*xray.Config, error) {
	templateConfig, err := s.settingService.GetXrayConfigTemplate()
	if err != nil {
		return nil, err
	}

	xrayConfig := &xray.Config{}
	err = json.Unmarshal([]byte(templateConfig), xrayConfig)
	if err != nil {
		return nil, err
	}

	_, _, _ = s.inboundService.AddTraffic(nil, nil)

	inbounds, err := s.inboundService.GetAllInbounds()
	if err != nil {
		return nil, err
	}
	for _, inbound := range inbounds {
		if !inbound.Enable || !model.IsXrayProtocol(inbound.Protocol) {
			continue
		}
		enabled := make(map[string]bool, len(inbound.ClientStats))
		for _, clientTraffic := range inbound.ClientStats {
			enabled[clientTraffic.Email] = clientTraffic.Enable
		}
		runtimeInbound, err := xrayInbound(inbound, enabled)
		if err != nil {
			return nil, err
		}
		xrayConfig.InboundConfigs = append(xrayConfig.InboundConfigs, *runtimeInbound.GenXrayInboundConfig())
	}
	return xrayConfig, nil
}

// xrayInbound is an inbound as Xray runs it, whether a restart writes it into
// the config or the API adds it to the running Xray: only the clients that
// are on (enabled tells those the traffic stats switched off), each with
// just the fields Xray reads, and the transport without what only the panel
// keeps (External Proxy, the TLS and REALITY forms' settings), trusting
// X-Forwarded-For where the website serves the inbound. The inbound given
// is left as it is.
func xrayInbound(inbound *model.Inbound, enabled map[string]bool) (*model.Inbound, error) {
	runtimeInbound := *inbound

	settings := map[string]any{}
	json.Unmarshal([]byte(inbound.Settings), &settings)
	if clients, ok := settings["clients"].([]any); ok {
		var finalClients []any
		for _, client := range clients {
			c, ok := client.(map[string]any)
			if !ok {
				continue
			}
			email, _ := c["email"].(string)
			if enable, exists := enabled[email]; exists && !enable {
				logger.Infof("Remove Inbound User %s due to expiration or traffic limit", email)
				continue
			}
			if manualEnable, ok := c["enable"].(bool); ok && !manualEnable {
				continue
			}
			kept := map[string]any{}
			for _, key := range []string{"email", "id", "password", "flow", "method", "auth", "reverse"} {
				if v, ok := c[key]; ok {
					kept[key] = v
				}
			}
			if flow, ok := kept["flow"].(string); ok && flow == "xtls-rprx-vision-udp443" {
				kept["flow"] = "xtls-rprx-vision"
			}
			finalClients = append(finalClients, kept)
		}
		settings["clients"] = finalClients
		modifiedSettings, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return nil, err
		}
		runtimeInbound.Settings = string(modifiedSettings)
	}

	if len(inbound.StreamSettings) > 0 {
		var stream map[string]any
		json.Unmarshal([]byte(inbound.StreamSettings), &stream)
		if tlsSettings, ok := stream["tlsSettings"].(map[string]any); ok {
			delete(tlsSettings, "settings")
		} else if realitySettings, ok := stream["realitySettings"].(map[string]any); ok {
			delete(realitySettings, "settings")
		}
		delete(stream, "externalProxy")
		trustSiteForwardedFor(inbound.Listen, stream)
		newStream, err := json.MarshalIndent(stream, "", "  ")
		if err != nil {
			return nil, err
		}
		runtimeInbound.StreamSettings = string(newStream)
	}
	return &runtimeInbound, nil
}

// GetXrayTraffic fetches the current traffic statistics from the running Xray process.
func (s *XrayService) GetXrayTraffic() ([]*xray.Traffic, []*xray.ClientTraffic, error) {
	if !s.IsXrayRunning() {
		err := errors.New("xray is not running")
		logger.Debug("Attempted to fetch Xray traffic, but Xray is not running:", err)
		return nil, nil, err
	}
	apiPort := p.GetAPIPort()
	if err := s.xrayAPI.Init(apiPort); err != nil {
		logger.Debug("Failed to initialize Xray API:", err)
		return nil, nil, err
	}
	defer s.xrayAPI.Close()

	traffic, clientTraffic, err := s.xrayAPI.GetTraffic(true)
	if err != nil {
		logger.Debug("Failed to fetch Xray traffic:", err)
		return nil, nil, err
	}
	return traffic, clientTraffic, nil
}

// RestartXray restarts the Xray process, optionally forcing a restart even if config unchanged.
func (s *XrayService) RestartXray(isForce bool) error {
	lock.Lock()
	defer lock.Unlock()
	logger.Debug("restart Xray, force:", isForce)
	isManuallyStopped.Store(false)

	xrayConfig, err := s.GetXrayConfig()
	if err != nil {
		return err
	}

	if s.IsXrayRunning() {
		if !isForce && p.GetConfig().Equals(xrayConfig) && !isNeedXrayRestart.Load() {
			logger.Debug("It does not need to restart Xray")
			publishXhttpSiteRoutes(xrayConfig)
			return nil
		}
		p.Stop()
	}

	p = xray.NewProcess(xrayConfig)
	result = ""
	err = p.Start()
	if err != nil {
		publishXhttpSiteRoutes(nil)
		return err
	}
	publishXhttpSiteRoutes(xrayConfig)

	return nil
}

// StopXray stops the running Xray process.
func (s *XrayService) StopXray() error {
	lock.Lock()
	defer lock.Unlock()
	isManuallyStopped.Store(true)
	logger.Debug("Attempting to stop Xray...")
	publishXhttpSiteRoutes(nil)
	if s.IsXrayRunning() {
		return p.Stop()
	}
	return errors.New("xray is not running")
}

// SetToNeedRestart marks that Xray needs to be restarted.
func (s *XrayService) SetToNeedRestart() {
	isNeedXrayRestart.Store(true)
}

// IsNeedRestartAndSetFalse checks if restart is needed and resets the flag to false.
func (s *XrayService) IsNeedRestartAndSetFalse() bool {
	return isNeedXrayRestart.CompareAndSwap(true, false)
}

// DidXrayCrash checks if Xray crashed by verifying it's not running and wasn't manually stopped.
func (s *XrayService) DidXrayCrash() bool {
	return !s.IsXrayRunning() && !isManuallyStopped.Load()
}
