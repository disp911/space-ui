package telemt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/disp911/space-ui/v3/config"
)

// stopTimeout is how long Stop waits for a graceful exit before killing.
const stopTimeout = 5 * time.Second

// logLines is how many recent output lines a process keeps for the panel.
const logLines = 300

// maxPartialLine bounds an unterminated output line kept between writes.
const maxPartialLine = 64 << 10

// GetBinaryName returns the telemt binary filename for the current platform.
func GetBinaryName() string {
	return fmt.Sprintf("telemt-%s-%s", runtime.GOOS, runtime.GOARCH)
}

// GetBinaryPath returns the full path of the bundled telemt binary.
func GetBinaryPath() string {
	return filepath.Join(config.GetBinFolderPath(), GetBinaryName())
}

// GetInstanceDir returns the directory of the telemt instance serving one
// inbound. It holds the generated config and telemt's working files (its
// TLS-front cache and quota state).
func GetInstanceDir(inboundID int) string {
	return filepath.Join(config.GetBinFolderPath(), "telemt", fmt.Sprintf("inbound-%d", inboundID))
}

// IsInstalled reports whether a telemt binary is bundled for this platform.
// Releases only ship one for linux/amd64 and linux/arm64.
func IsInstalled() bool {
	return fileExists(GetBinaryPath())
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// writeConfig replaces the config file atomically, so telemt's file watcher
// never reads a half-written file.
func writeConfig(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// process is one running telemt instance.
type process struct {
	cmd       *exec.Cmd
	done      chan struct{}
	exitErr   error
	logs      *logBuffer
	startedAt time.Time
	// stopped is set when the process was stopped on purpose, so its exit
	// is not an error. The manager's lock guards it.
	stopped bool
}

// startProcess launches binary with the config at configPath.
func startProcess(binary, configPath, workDir string, onLine func(string)) (*process, error) {
	// The process runs inside workDir, so relative paths (the bin folder is
	// "bin" by default) would otherwise resolve against the wrong directory.
	var err error
	if binary, err = filepath.Abs(binary); err != nil {
		return nil, err
	}
	if configPath, err = filepath.Abs(configPath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, err
	}
	// telemt refuses a config whose path goes through a symlink, so it gets
	// the resolved one; writes through the original path land in the same
	// file.
	if configPath, err = filepath.EvalSymlinks(configPath); err != nil {
		return nil, err
	}
	if workDir, err = filepath.EvalSymlinks(workDir); err != nil {
		return nil, err
	}
	logs := newLogBuffer(logLines)
	logs.onLine = onLine
	cmd := exec.Command(binary, configPath)
	cmd.Dir = workDir
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &process{cmd: cmd, done: make(chan struct{}), logs: logs, startedAt: time.Now()}
	go func() {
		p.exitErr = cmd.Wait()
		close(p.done)
	}()
	return p, nil
}

func (p *process) running() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

// reload asks telemt to re-read its config. telemt also watches the file, so
// this is a belt-and-braces trigger for the change just written.
func (p *process) reload() error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return p.cmd.Process.Signal(syscall.SIGHUP)
}

// stop terminates the process gracefully, killing it if it does not exit.
func (p *process) stop() {
	if !p.running() {
		return
	}
	p.stopped = true
	if runtime.GOOS == "windows" {
		_ = p.cmd.Process.Kill()
	} else {
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
	}
	select {
	case <-p.done:
	case <-time.After(stopTimeout):
		_ = p.cmd.Process.Kill()
		<-p.done
	}
}

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// logBuffer keeps the last lines written by the process.
type logBuffer struct {
	mu      sync.Mutex
	max     int
	lines   []string
	partial string
	// onLine, when set, receives every complete line as it is written.
	onLine func(string)
}

func newLogBuffer(max int) *logBuffer {
	return &logBuffer{max: max}
}

func (b *logBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	text := b.partial + string(data)
	parts := strings.Split(text, "\n")
	b.partial = parts[len(parts)-1]
	if len(b.partial) > maxPartialLine {
		b.partial = b.partial[len(b.partial)-maxPartialLine:]
	}
	for _, line := range parts[:len(parts)-1] {
		line = strings.TrimRight(ansiEscape.ReplaceAllString(line, ""), "\r")
		if line == "" {
			continue
		}
		b.lines = append(b.lines, line)
		if b.onLine != nil {
			b.onLine(line)
		}
	}
	if extra := len(b.lines) - b.max; extra > 0 {
		b.lines = append([]string(nil), b.lines[extra:]...)
	}
	return len(data), nil
}

func (b *logBuffer) snapshot() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := append([]string(nil), b.lines...)
	if b.partial != "" {
		out = append(out, ansiEscape.ReplaceAllString(b.partial, ""))
	}
	if extra := len(out) - b.max; extra > 0 {
		out = out[extra:]
	}
	return out
}
