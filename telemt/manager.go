package telemt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Instance is the desired state of the telemt process serving one inbound.
type Instance struct {
	InboundID int
	Tag       string
	Listen    string
	Port      int
	TLSDomain string
	// Web, when set, also serves the users as a Telegram WEB proxy.
	Web *WebInstance
	// Users are all valid clients, enabled or not; Emails maps their telemt
	// user names to the panel client emails they stand for.
	Users  []User
	Emails map[string]string
}

// WebInstance is the WEB proxy of an inbound; the manager picks the port of
// its listener.
type WebInstance struct {
	Host      string
	BasePath  string
	PublicIP  string
	DecoyPort int
}

// ClientTraffic is the traffic of one client since the previous collection.
type ClientTraffic struct {
	Email    string
	Up, Down int64
	Online   bool
}

// InboundTraffic is the traffic of one inbound since the previous collection.
type InboundTraffic struct {
	Tag      string
	Up, Down int64
	Clients  []ClientTraffic
}

// Manager runs one telemt process per mtproto inbound and keeps the set of
// processes in line with the panel's inbounds. It is safe for concurrent use.
type Manager struct {
	binary  string
	baseDir func(inboundID int) string
	// OnLog, when set, receives every output line of every process.
	OnLog func(inboundID int, line string)

	mu        sync.Mutex
	instances map[int]*instance
	// stopped keeps every process down until Start; see Stop.
	stopped bool
	// webRoutes maps the base path of every running WEB proxy to the port of
	// its listener. It is replaced, never changed, so readers need no lock.
	webRoutes atomic.Pointer[map[string]int]

	versionMu   sync.Mutex
	version     string
	versionRead bool
}

// ProcessInfo describes the process serving an inbound.
type ProcessInfo struct {
	Running   bool
	PID       int
	StartedAt time.Time
}

type instance struct {
	proc        *process
	desired     Instance
	metricsPort int
	// apiPort and apiToken reach the read-only control API of the process.
	apiPort  int
	apiToken string
	// webPort is the loopback port of the WEB listener.
	webPort    int
	runningKey string
	config     []byte
	lastErr    string
	// seen holds the counters of the previous collection, per telemt user.
	seen map[string]UserCounters
}

// NewManager returns a manager for the bundled binary and default paths.
func NewManager() *Manager {
	return newManager(GetBinaryPath(), GetInstanceDir)
}

func newManager(binary string, baseDir func(int) string) *Manager {
	return &Manager{binary: binary, baseDir: baseDir, instances: map[int]*instance{}}
}

// Installed reports whether the telemt binary is present.
func (m *Manager) Installed() bool {
	return fileExists(m.binary)
}

// Sync makes the running processes match want. Inbounds missing from want
// are stopped. For each wanted inbound, a change in users is written to the
// config and hot-reloaded, so Telegram sessions stay up; a change to listen
// address, port or TLS domain restarts the process, as does a process that
// has exited. While the manager is stopped, nothing is started. Errors are
// recorded per inbound and the first one is returned.
func (m *Manager) Sync(want []Instance) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wanted := make(map[int]bool, len(want))
	var firstErr error
	for _, w := range want {
		wanted[w.InboundID] = true
		if err := m.syncLocked(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for id, inst := range m.instances {
		if !wanted[id] {
			if inst.proc != nil {
				inst.proc.stop()
			}
			delete(m.instances, id)
			_ = os.RemoveAll(m.baseDir(id))
		}
	}
	m.publishWebRoutesLocked()
	return firstErr
}

// publishWebRoutesLocked records the WEB proxies of the running processes.
func (m *Manager) publishWebRoutesLocked() {
	routes := map[string]int{}
	for _, inst := range m.instances {
		if inst.proc != nil && inst.proc.running() && inst.desired.Web != nil && inst.webPort != 0 {
			routes[inst.desired.Web.BasePath] = inst.webPort
		}
	}
	m.webRoutes.Store(&routes)
}

// WebRoute returns the loopback port of the WEB proxy whose base path is
// basePath, as of the last sync.
func (m *Manager) WebRoute(basePath string) (int, bool) {
	routes := m.webRoutes.Load()
	if routes == nil {
		return 0, false
	}
	port, ok := (*routes)[basePath]
	return port, ok
}

func (m *Manager) syncLocked(w Instance) error {
	inst := m.instances[w.InboundID]
	if inst == nil {
		inst = &instance{}
		m.instances[w.InboundID] = inst
	}
	inst.desired = w

	if m.stopped {
		if inst.proc != nil {
			inst.proc.stop()
		}
		return nil
	}

	if !anyEnabled(w.Users) {
		// With nobody allowed in, the inbound simply serves nobody; stopping
		// the process also ends the sessions of the clients just disabled.
		if inst.proc != nil {
			inst.proc.stop()
		}
		inst.lastErr = ""
		return nil
	}

	running := inst.proc != nil && inst.proc.running()
	if !running || inst.metricsPort == 0 {
		ports, err := FreeLoopbackPorts(3)
		var token string
		if err == nil {
			token, err = RandomHex(16)
		}
		if err != nil {
			inst.lastErr = err.Error()
			return err
		}
		inst.metricsPort, inst.apiPort, inst.webPort, inst.apiToken = ports[0], ports[1], ports[2], token
	}
	s := Settings{
		Listen: w.Listen, Port: w.Port, TLSDomain: w.TLSDomain,
		MetricsPort: inst.metricsPort, APIPort: inst.apiPort, APIToken: inst.apiToken,
	}
	if web := w.Web; web != nil {
		s.Web = &Web{
			Host: web.Host, BasePath: web.BasePath, PublicIP: web.PublicIP,
			ListenPort: inst.webPort, DecoyPort: web.DecoyPort,
			MaxProfiles: WebProfileCeiling(len(w.Users)),
		}
	}
	data, err := BuildConfig(s, w.Users)
	if err != nil {
		inst.lastErr = err.Error()
		return err
	}

	dir := m.baseDir(w.InboundID)
	configPath := filepath.Join(dir, "telemt.toml")
	if running && inst.runningKey == s.restartKey() {
		if string(data) == string(inst.config) {
			return nil
		}
		if err := writeConfig(configPath, data); err != nil {
			inst.lastErr = err.Error()
			return err
		}
		inst.config = data
		return inst.proc.reload()
	}

	if inst.proc != nil {
		inst.proc.stop()
	}
	if err := writeConfig(configPath, data); err != nil {
		inst.lastErr = err.Error()
		return err
	}
	var onLine func(string)
	if m.OnLog != nil {
		id, hook := w.InboundID, m.OnLog
		onLine = func(line string) { hook(id, line) }
	}
	proc, err := startProcess(m.binary, configPath, dir, onLine)
	if err != nil {
		inst.lastErr = err.Error()
		return err
	}
	inst.proc = proc
	inst.runningKey = s.restartKey()
	inst.config = data
	inst.seen = nil
	inst.lastErr = ""
	return nil
}

// StopAll stops every process, for panel shutdown.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, inst := range m.instances {
		if inst.proc != nil {
			inst.proc.stop()
		}
		delete(m.instances, id)
	}
	m.publishWebRoutesLocked()
}

// Stop stops every process and keeps them down until Start: syncs in between
// only record the wanted state. Nothing is removed, so Start and the next
// sync bring the same inbounds back.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	for _, inst := range m.instances {
		if inst.proc != nil {
			inst.proc.stop()
		}
	}
	m.publishWebRoutesLocked()
}

// Start lets the next sync start the processes again after Stop.
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = false
}

// Stopped reports whether the processes are held down by Stop.
func (m *Manager) Stopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

// Running reports whether the process of an inbound is up.
func (m *Manager) Running(inboundID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst := m.instances[inboundID]
	return inst != nil && inst.proc != nil && inst.proc.running()
}

// Process describes the process of an inbound; it is zero when none runs.
func (m *Manager) Process(inboundID int) ProcessInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst := m.instances[inboundID]
	if inst == nil || inst.proc == nil || !inst.proc.running() {
		return ProcessInfo{}
	}
	return ProcessInfo{Running: true, PID: inst.proc.cmd.Process.Pid, StartedAt: inst.proc.startedAt}
}

// Version returns the version the binary reports, such as 3.5.16, or "" when
// it cannot be read. The binary is asked once.
func (m *Manager) Version() string {
	m.versionMu.Lock()
	defer m.versionMu.Unlock()
	if !m.versionRead && m.Installed() {
		m.version = readVersion(m.binary)
		m.versionRead = true
	}
	return m.version
}

// readVersion runs binary --version, which prints "telemt 3.5.16".
func readVersion(binary string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binary, "--version").Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(out))
	if len(fields) < 2 || fields[0] != "telemt" {
		return ""
	}
	return fields[1]
}

// LastError returns the last start or exit error of an inbound's process.
func (m *Manager) LastError(inboundID int) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst := m.instances[inboundID]
	if inst == nil {
		return ""
	}
	if inst.lastErr == "" && inst.proc != nil && !inst.proc.running() && !inst.proc.stopped && inst.proc.exitErr != nil {
		return inst.proc.exitErr.Error()
	}
	return inst.lastErr
}

// Logs returns the recent output of an inbound's current or last process.
func (m *Manager) Logs(inboundID int) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst := m.instances[inboundID]
	if inst == nil || inst.proc == nil {
		return nil
	}
	return inst.proc.logs.snapshot()
}

// CollectTraffic scrapes every running process and returns the traffic since
// the previous call. Counters that went backwards (a restarted process) are
// taken as new traffic from zero.
func (m *Manager) CollectTraffic(ctx context.Context) []InboundTraffic {
	type target struct {
		id     int
		port   int
		tag    string
		emails map[string]string
		proc   *process
	}
	m.mu.Lock()
	targets := make([]target, 0, len(m.instances))
	for id, inst := range m.instances {
		if inst.proc != nil && inst.proc.running() {
			targets = append(targets, target{id, inst.metricsPort, inst.desired.Tag, inst.desired.Emails, inst.proc})
		}
	}
	m.mu.Unlock()

	var result []InboundTraffic
	for _, t := range targets {
		scrapeCtx, cancel := context.WithTimeout(ctx, metricsTimeout)
		counters, err := FetchUserCounters(scrapeCtx, t.port)
		cancel()
		if err != nil {
			continue
		}

		m.mu.Lock()
		inst := m.instances[t.id]
		if inst == nil || inst.proc != t.proc {
			m.mu.Unlock()
			continue
		}
		prev := inst.seen
		inst.seen = counters
		m.mu.Unlock()

		it := InboundTraffic{Tag: t.tag}
		for user, c := range counters {
			email, ok := t.emails[user]
			if !ok {
				continue
			}
			old := prev[user]
			up, down := delta(c.Up, old.Up), delta(c.Down, old.Down)
			it.Up += up
			it.Down += down
			it.Clients = append(it.Clients, ClientTraffic{Email: email, Up: up, Down: down, Online: c.Connections > 0})
		}
		result = append(result, it)
	}
	return result
}

func delta(now, before uint64) int64 {
	if now < before {
		return int64(now)
	}
	return int64(now - before)
}
