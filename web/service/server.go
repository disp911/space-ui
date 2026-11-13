package service

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disp911/space-ui/v3/config"
	"github.com/disp911/space-ui/v3/database"
	"github.com/disp911/space-ui/v3/database/model"
	"github.com/disp911/space-ui/v3/logger"
	"github.com/disp911/space-ui/v3/util/common"
	"github.com/disp911/space-ui/v3/util/sys"
	"github.com/disp911/space-ui/v3/xray"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"gorm.io/gorm"
)

// ProcessState represents the current state of a system process.
type ProcessState string

// Process state constants
const (
	Running ProcessState = "running" // Process is running normally
	Stop    ProcessState = "stop"    // Process is stopped
	Error   ProcessState = "error"   // Process is in error state
)

// Status represents comprehensive system and application status information.
// It includes CPU, memory, disk, network statistics, and Xray process status.
type Status struct {
	T           time.Time `json:"-"`
	Cpu         float64   `json:"cpu"`
	CpuCores    int       `json:"cpuCores"`
	LogicalPro  int       `json:"logicalPro"`
	CpuSpeedMhz float64   `json:"cpuSpeedMhz"`
	Mem         struct {
		Current uint64 `json:"current"`
		Total   uint64 `json:"total"`
	} `json:"mem"`
	Swap struct {
		Current uint64 `json:"current"`
		Total   uint64 `json:"total"`
	} `json:"swap"`
	Disk struct {
		Current uint64 `json:"current"`
		Total   uint64 `json:"total"`
	} `json:"disk"`
	Xray struct {
		State    ProcessState `json:"state"`
		ErrorMsg string       `json:"errorMsg"`
		Version  string       `json:"version"`
		// Mem is the resident memory of the Xray process, in bytes.
		Mem uint64 `json:"mem"`
		// Uptime is how long the running process has been up, in seconds.
		Uptime uint64 `json:"uptime"`
		// Online is the number of clients with a live connection.
		Online int `json:"online"`
	} `json:"xray"`
	// Telemt is the Telegram proxy behind the mtproto inbound; nil when
	// there is no mtproto inbound.
	Telemt *TelemtStatus `json:"telemt,omitempty"`

	Uptime   uint64    `json:"uptime"`
	Loads    []float64 `json:"loads"`
	TcpCount int       `json:"tcpCount"`
	UdpCount int       `json:"udpCount"`
	NetIO    struct {
		Up   uint64 `json:"up"`
		Down uint64 `json:"down"`
	} `json:"netIO"`
	NetTraffic struct {
		Sent uint64 `json:"sent"`
		Recv uint64 `json:"recv"`
	} `json:"netTraffic"`
	PublicIP struct {
		IPv4 string `json:"ipv4"`
		IPv6 string `json:"ipv6"`
	} `json:"publicIP"`
	AppStats struct {
		Threads uint32 `json:"threads"`
		Mem     uint64 `json:"mem"`
		Uptime  uint64 `json:"uptime"`
	} `json:"appStats"`
}

// panelStart is when this panel process started; its uptime counts from it.
var panelStart = time.Now()

// Release represents information about a software release from GitHub.
type Release struct {
	TagName string `json:"tag_name"` // The tag name of the release
}

// ServerService provides business logic for server monitoring and management.
// It handles system status collection, IP detection, and application statistics.
type ServerService struct {
	xrayService        XrayService
	inboundService     InboundService
	telemtService      TelemtService
	cachedIPv4         string
	cachedIPv6         string
	noIPv6             bool
	mu                 sync.Mutex
	lastCPUTimes       cpu.TimesStat
	hasLastCPUSample   bool
	hasNativeCPUSample bool
	emaCPU             float64
	cpuHistory         []CPUSample
	cachedCpuSpeedMhz  float64
	lastCpuInfoAttempt time.Time
}

// AggregateCpuHistory returns up to maxPoints averaged buckets of size bucketSeconds over recent data.
func (s *ServerService) AggregateCpuHistory(bucketSeconds int, maxPoints int) []map[string]any {
	if bucketSeconds <= 0 || maxPoints <= 0 {
		return nil
	}
	cutoff := time.Now().Add(-time.Duration(bucketSeconds*maxPoints) * time.Second).Unix()
	s.mu.Lock()
	// find start index (history sorted ascending)
	hist := s.cpuHistory
	// binary-ish scan (simple linear from end since size capped ~10800 is fine)
	startIdx := 0
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].T < cutoff {
			startIdx = i + 1
			break
		}
	}
	if startIdx >= len(hist) {
		s.mu.Unlock()
		return []map[string]any{}
	}
	slice := hist[startIdx:]
	// copy for unlock
	tmp := make([]CPUSample, len(slice))
	copy(tmp, slice)
	s.mu.Unlock()
	if len(tmp) == 0 {
		return []map[string]any{}
	}
	var out []map[string]any
	var acc []float64
	bSize := int64(bucketSeconds)
	curBucket := (tmp[0].T / bSize) * bSize
	flush := func(ts int64) {
		if len(acc) == 0 {
			return
		}
		sum := 0.0
		for _, v := range acc {
			sum += v
		}
		avg := sum / float64(len(acc))
		out = append(out, map[string]any{"t": ts, "cpu": avg})
		acc = acc[:0]
	}
	for _, p := range tmp {
		b := (p.T / bSize) * bSize
		if b != curBucket {
			flush(curBucket)
			curBucket = b
		}
		acc = append(acc, p.Cpu)
	}
	flush(curBucket)
	if len(out) > maxPoints {
		out = out[len(out)-maxPoints:]
	}
	return out
}

// CPUSample single CPU utilization sample
type CPUSample struct {
	T   int64   `json:"t"`   // unix seconds
	Cpu float64 `json:"cpu"` // percent 0..100
}

type LogEntry struct {
	DateTime    time.Time
	FromAddress string
	ToAddress   string
	Inbound     string
	Outbound    string
	Email       string
	Event       int
}

func getPublicIP(url string) string {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "N/A"
	}
	defer resp.Body.Close()

	// Don't retry if access is blocked or region-restricted
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnavailableForLegalReasons {
		return "N/A"
	}
	if resp.StatusCode != http.StatusOK {
		return "N/A"
	}

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "N/A"
	}

	ipString := strings.TrimSpace(string(ip))
	if ipString == "" {
		return "N/A"
	}

	return ipString
}

func (s *ServerService) GetStatus(lastStatus *Status) *Status {
	now := time.Now()
	status := &Status{
		T: now,
	}

	// CPU stats
	util, err := s.sampleCPUUtilization()
	if err != nil {
		logger.Warning("get cpu percent failed:", err)
	} else {
		status.Cpu = util
	}

	status.CpuCores, err = cpu.Counts(false)
	if err != nil {
		logger.Warning("get cpu cores count failed:", err)
	}

	status.LogicalPro = runtime.NumCPU()

	if status.CpuSpeedMhz = s.cachedCpuSpeedMhz; s.cachedCpuSpeedMhz == 0 && time.Since(s.lastCpuInfoAttempt) > 5*time.Minute {
		s.lastCpuInfoAttempt = time.Now()
		done := make(chan struct{})
		go func() {
			defer close(done)
			cpuInfos, err := cpu.Info()
			if err != nil {
				logger.Warning("get cpu info failed:", err)
				return
			}
			if len(cpuInfos) > 0 {
				s.cachedCpuSpeedMhz = cpuInfos[0].Mhz
				status.CpuSpeedMhz = s.cachedCpuSpeedMhz
			} else {
				logger.Warning("could not find cpu info")
			}
		}()
		select {
		case <-done:
		case <-time.After(1500 * time.Millisecond):
			logger.Warning("cpu info query timed out; will retry later")
		}
	} else if s.cachedCpuSpeedMhz != 0 {
		status.CpuSpeedMhz = s.cachedCpuSpeedMhz
	}

	// Uptime
	upTime, err := host.Uptime()
	if err != nil {
		logger.Warning("get uptime failed:", err)
	} else {
		status.Uptime = upTime
	}

	// Memory stats
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		logger.Warning("get virtual memory failed:", err)
	} else {
		status.Mem.Current = memInfo.Used
		status.Mem.Total = memInfo.Total
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		logger.Warning("get swap memory failed:", err)
	} else {
		status.Swap.Current = swapInfo.Used
		status.Swap.Total = swapInfo.Total
	}

	// Disk stats
	diskInfo, err := disk.Usage("/")
	if err != nil {
		logger.Warning("get disk usage failed:", err)
	} else {
		status.Disk.Current = diskInfo.Used
		status.Disk.Total = diskInfo.Total
	}

	// Load averages
	avgState, err := load.Avg()
	if err != nil {
		logger.Warning("get load avg failed:", err)
	} else {
		status.Loads = []float64{avgState.Load1, avgState.Load5, avgState.Load15}
	}

	// Network stats
	ioStats, err := net.IOCounters(false)
	if err != nil {
		logger.Warning("get io counters failed:", err)
	} else if len(ioStats) > 0 {
		ioStat := ioStats[0]
		status.NetTraffic.Sent = ioStat.BytesSent
		status.NetTraffic.Recv = ioStat.BytesRecv

		if lastStatus != nil {
			duration := now.Sub(lastStatus.T)
			seconds := float64(duration) / float64(time.Second)
			up := uint64(float64(status.NetTraffic.Sent-lastStatus.NetTraffic.Sent) / seconds)
			down := uint64(float64(status.NetTraffic.Recv-lastStatus.NetTraffic.Recv) / seconds)
			status.NetIO.Up = up
			status.NetIO.Down = down
		}
	} else {
		logger.Warning("can not find io counters")
	}

	// TCP/UDP connections
	status.TcpCount, err = sys.GetTCPCount()
	if err != nil {
		logger.Warning("get tcp connections failed:", err)
	}

	status.UdpCount, err = sys.GetUDPCount()
	if err != nil {
		logger.Warning("get udp connections failed:", err)
	}

	// IP fetching with caching
	showIp4ServiceLists := []string{
		"https://api4.ipify.org",
		"https://ipv4.icanhazip.com",
		"https://v4.api.ipinfo.io/ip",
		"https://ipv4.myexternalip.com/raw",
		"https://4.ident.me",
		"https://check-host.net/ip",
	}
	showIp6ServiceLists := []string{
		"https://api6.ipify.org",
		"https://ipv6.icanhazip.com",
		"https://v6.api.ipinfo.io/ip",
		"https://ipv6.myexternalip.com/raw",
		"https://6.ident.me",
	}

	if s.cachedIPv4 == "" {
		for _, ip4Service := range showIp4ServiceLists {
			s.cachedIPv4 = getPublicIP(ip4Service)
			if s.cachedIPv4 != "N/A" {
				break
			}
		}
	}

	if s.cachedIPv6 == "" && !s.noIPv6 {
		for _, ip6Service := range showIp6ServiceLists {
			s.cachedIPv6 = getPublicIP(ip6Service)
			if s.cachedIPv6 != "N/A" {
				break
			}
		}
	}

	if s.cachedIPv6 == "N/A" {
		s.noIPv6 = true
	}

	status.PublicIP.IPv4 = s.cachedIPv4
	status.PublicIP.IPv6 = s.cachedIPv6

	// Xray status
	if s.xrayService.IsXrayRunning() {
		status.Xray.State = Running
		status.Xray.ErrorMsg = ""
	} else {
		err := s.xrayService.GetXrayErr()
		if err != nil {
			status.Xray.State = Error
		} else {
			status.Xray.State = Stop
		}
		status.Xray.ErrorMsg = s.xrayService.GetXrayResult()
	}
	status.Xray.Version = s.xrayService.GetXrayVersion()
	if p != nil {
		status.Xray.Mem = processMemory(p.GetPID())
		if status.Xray.State == Running {
			status.Xray.Uptime = p.GetUptime()
			status.Xray.Online = len(p.GetOnlineClients())
		}
	}

	status.Telemt = s.telemtService.Status()

	// Application stats
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	status.AppStats.Mem = rtm.Sys
	status.AppStats.Threads = uint32(runtime.NumGoroutine())
	status.AppStats.Uptime = uint64(time.Since(panelStart).Seconds())

	return status
}

// processMemory returns the resident memory of the process with pid, in
// bytes, or 0 when there is no such process.
func processMemory(pid int) uint64 {
	if pid <= 0 {
		return 0
	}
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return 0
	}
	info, err := proc.MemoryInfo()
	if err != nil || info == nil {
		return 0
	}
	return info.RSS
}

func (s *ServerService) AppendCpuSample(t time.Time, v float64) {
	const capacity = 9300 // a little over 5 hours @ 2s, the longest range the dashboard shows
	s.mu.Lock()
	defer s.mu.Unlock()
	p := CPUSample{T: t.Unix(), Cpu: v}
	if n := len(s.cpuHistory); n > 0 && s.cpuHistory[n-1].T == p.T {
		s.cpuHistory[n-1] = p
	} else {
		s.cpuHistory = append(s.cpuHistory, p)
	}
	if len(s.cpuHistory) > capacity {
		s.cpuHistory = s.cpuHistory[len(s.cpuHistory)-capacity:]
	}
}

func (s *ServerService) sampleCPUUtilization() (float64, error) {
	// Try native platform-specific CPU implementation first (Windows, Linux, macOS)
	if pct, err := sys.CPUPercentRaw(); err == nil {
		s.mu.Lock()
		// First call to native method returns 0 (initializes baseline)
		if !s.hasNativeCPUSample {
			s.hasNativeCPUSample = true
			s.mu.Unlock()
			return 0, nil
		}
		// Smooth with EMA
		const alpha = 0.3
		if s.emaCPU == 0 {
			s.emaCPU = pct
		} else {
			s.emaCPU = alpha*pct + (1-alpha)*s.emaCPU
		}
		val := s.emaCPU
		s.mu.Unlock()
		return val, nil
	}
	// If native call fails, fall back to gopsutil times
	// Read aggregate CPU times (all CPUs combined)
	times, err := cpu.Times(false)
	if err != nil {
		return 0, err
	}
	if len(times) == 0 {
		return 0, fmt.Errorf("no cpu times available")
	}

	cur := times[0]

	s.mu.Lock()
	defer s.mu.Unlock()

	// If this is the first sample, initialize and return current EMA (0 by default)
	if !s.hasLastCPUSample {
		s.lastCPUTimes = cur
		s.hasLastCPUSample = true
		return s.emaCPU, nil
	}

	// Compute busy and total deltas
	// Note: Guest and GuestNice times are already included in User and Nice respectively,
	// so we exclude them to avoid double-counting (Linux kernel accounting)
	idleDelta := cur.Idle - s.lastCPUTimes.Idle
	busyDelta := (cur.User - s.lastCPUTimes.User) +
		(cur.System - s.lastCPUTimes.System) +
		(cur.Nice - s.lastCPUTimes.Nice) +
		(cur.Iowait - s.lastCPUTimes.Iowait) +
		(cur.Irq - s.lastCPUTimes.Irq) +
		(cur.Softirq - s.lastCPUTimes.Softirq) +
		(cur.Steal - s.lastCPUTimes.Steal)

	totalDelta := busyDelta + idleDelta

	// Update last sample for next time
	s.lastCPUTimes = cur

	// Guard against division by zero or negative deltas (e.g., counter resets)
	if totalDelta <= 0 {
		return s.emaCPU, nil
	}

	raw := 100.0 * (busyDelta / totalDelta)
	if raw < 0 {
		raw = 0
	}
	if raw > 100 {
		raw = 100
	}

	// Exponential moving average to smooth spikes
	const alpha = 0.3 // smoothing factor (0<alpha<=1). Higher = more responsive, lower = smoother
	if s.emaCPU == 0 {
		// Initialize EMA with the first real reading to avoid long warm-up from zero
		s.emaCPU = raw
	} else {
		s.emaCPU = alpha*raw + (1-alpha)*s.emaCPU
	}

	return s.emaCPU, nil
}

func (s *ServerService) GetXrayVersions() ([]string, error) {
	const (
		XrayURL    = "https://api.github.com/repos/XTLS/Xray-core/releases"
		bufferSize = 8192
	)

	resp, err := http.Get(XrayURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check HTTP status code - GitHub API returns object instead of array on error
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var errorResponse struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(bodyBytes, &errorResponse) == nil && errorResponse.Message != "" {
			return nil, fmt.Errorf("GitHub API error: %s", errorResponse.Message)
		}
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	buffer := bytes.NewBuffer(make([]byte, bufferSize))
	buffer.Reset()
	if _, err := buffer.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	var releases []Release
	if err := json.Unmarshal(buffer.Bytes(), &releases); err != nil {
		return nil, err
	}

	var versions []string
	for _, release := range releases {
		tagVersion := strings.TrimPrefix(release.TagName, "v")
		tagParts := strings.Split(tagVersion, ".")
		if len(tagParts) != 3 {
			continue
		}

		major, err1 := strconv.Atoi(tagParts[0])
		minor, err2 := strconv.Atoi(tagParts[1])
		patch, err3 := strconv.Atoi(tagParts[2])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}

		if major > 26 || (major == 26 && minor > 3) || (major == 26 && minor == 3 && patch >= 10) {
			versions = append(versions, release.TagName)
		}
	}
	return versions, nil
}

func (s *ServerService) StopXrayService() error {
	err := s.xrayService.StopXray()
	if err != nil {
		logger.Error("stop xray failed:", err)
		return err
	}
	return nil
}

// RestartXrayLater restarts Xray a moment from now, after the answer to the
// request that asked for it: behind REALITY that request runs through Xray,
// which a restart would drop. A failure is logged under what and shows in
// Xray's state.
func (s *ServerService) RestartXrayLater(what string) {
	go func() {
		time.Sleep(time.Second)
		if err := s.RestartXrayService(); err != nil {
			logger.Warning(what+": restart xray:", err)
		}
	}()
}

func (s *ServerService) RestartXrayService() error {
	err := s.xrayService.RestartXray(true)
	if err != nil {
		logger.Error("start xray failed:", err)
		return err
	}
	return nil
}

func (s *ServerService) downloadXRay(version string) (string, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	switch osName {
	case "darwin":
		osName = "macos"
	case "windows":
		osName = "windows"
	}

	switch arch {
	case "amd64":
		arch = "64"
	case "arm64":
		arch = "arm64-v8a"
	case "armv7":
		arch = "arm32-v7a"
	case "armv6":
		arch = "arm32-v6"
	case "armv5":
		arch = "arm32-v5"
	case "386":
		arch = "32"
	case "s390x":
		arch = "s390x"
	}

	fileName := fmt.Sprintf("Xray-%s-%s.zip", osName, arch)
	url := fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/%s/%s", version, fileName)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	os.Remove(fileName)
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

// DownloadXray fetches the release archive of the version while Xray keeps
// running, and returns where it is.
func (s *ServerService) DownloadXray(version string) (string, error) {
	return s.downloadXRay(version)
}

// InstallXray puts the Xray binary of a downloaded release archive in place
// of the running one and restarts Xray; the archive is removed.
func (s *ServerService) InstallXray(zipFileName string) error {
	defer os.Remove(zipFileName)

	zipFile, err := os.Open(zipFileName)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	stat, err := zipFile.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(zipFile, stat.Size())
	if err != nil {
		return err
	}

	// Helper to extract files
	copyZipFile := func(zipName string, fileName string) error {
		zipFile, err := reader.Open(zipName)
		if err != nil {
			return err
		}
		defer zipFile.Close()
		os.MkdirAll(filepath.Dir(fileName), 0755)
		os.Remove(fileName)
		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR|os.O_TRUNC, fs.ModePerm)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, zipFile)
		return err
	}

	// Xray stops only now, with the new binary at hand.
	if err := s.StopXrayService(); err != nil {
		logger.Warning("failed to stop xray before update:", err)
	}
	if runtime.GOOS == "windows" {
		targetBinary := filepath.Join("bin", "xray-windows-amd64.exe")
		err = copyZipFile("xray.exe", targetBinary)
	} else {
		err = copyZipFile("xray", xray.GetBinaryPath())
	}
	if err != nil {
		// Back on with whatever binary is there, rather than left stopped.
		s.xrayService.RestartXray(true)
		return err
	}

	if err := s.xrayService.RestartXray(true); err != nil {
		logger.Error("start xray failed:", err)
		return err
	}

	return nil
}

func (s *ServerService) GetLogs(count string, level string, syslog string) []string {
	c, _ := strconv.Atoi(count)
	var lines []string

	if syslog == "true" {
		// Check if running on Windows - journalctl is not available
		if runtime.GOOS == "windows" {
			return []string{"Syslog is not supported on Windows. Please use application logs instead by unchecking the 'Syslog' option."}
		}

		// Validate and sanitize count parameter
		countInt, err := strconv.Atoi(count)
		if err != nil || countInt < 1 || countInt > 10000 {
			return []string{"Invalid count parameter - must be a number between 1 and 10000"}
		}

		// Validate level parameter - only allow valid syslog levels
		validLevels := map[string]bool{
			"0": true, "emerg": true,
			"1": true, "alert": true,
			"2": true, "crit": true,
			"3": true, "err": true,
			"4": true, "warning": true,
			"5": true, "notice": true,
			"6": true, "info": true,
			"7": true, "debug": true,
		}
		if !validLevels[level] {
			return []string{"Invalid level parameter - must be a valid syslog level"}
		}

		// Use hardcoded command with validated parameters
		cmd := exec.Command("journalctl", "-u", "space-ui", "--no-pager", "-n", strconv.Itoa(countInt), "-p", level)
		var out bytes.Buffer
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			return []string{"Failed to run journalctl command! Make sure systemd is available and space-ui service is registered."}
		}
		lines = strings.Split(out.String(), "\n")
	} else {
		lines = logger.GetLogs(c, level)
	}

	return lines
}

// xrayLogPageSize is how many access log rows one page of the viewer holds.
// The panel renders a page into a single HTML table, so this also bounds how
// much DOM the browser ever has to build.
const xrayLogPageSize = 200

// XrayLogPage is one page of the Xray access log plus what the panel needs to
// render the date picker, the client picker and the pager.
type XrayLogPage struct {
	Entries  []LogEntry `json:"entries"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
	Date     string     `json:"date"`
	Dates    []string   `json:"dates"`
	Email    string     `json:"email"`
	Clients  []string   `json:"clients"`
}

// Event values of a LogEntry: how the connection left the server.
const (
	xrayLogDirect = iota
	xrayLogBlocked
	xrayLogProxied
)

var xrayLogEventNames = [...]string{"DIRECT", "BLOCKED", "PROXY"}

// GetXrayLogs returns one page of stored access log entries for a single day.
// Entries are ingested into the log database by XrayLogIngestJob, so this is
// a plain indexed query rather than a scan of the log files.
func (s *ServerService) GetXrayLogs(
	date string,
	page int,
	email string,
	filter string,
	showDirect string,
	showBlocked string,
	showProxy string,
	freedoms []string,
	blackholes []string) XrayLogPage {

	result := XrayLogPage{
		Entries:  []LogEntry{},
		Page:     max(page, 1),
		PageSize: xrayLogPageSize,
		Email:    email,
	}

	db := database.GetLogDB()
	if db == nil {
		return result
	}

	result.Dates = xrayLogDates(db)
	result.Clients = xrayLogClients(db)
	result.Date = resolveXrayLogDay(result.Dates, date)
	if result.Date == "" {
		return result
	}

	query, ok := xrayLogQuery(db, result.Date, email, filter, showDirect, showBlocked, showProxy, freedoms, blackholes)
	if !ok {
		return result
	}

	if err := query.Count(&result.Total).Error; err != nil {
		logger.Warning("Failed to count Xray log entries:", err)
		return result
	}

	var rows []model.XrayLogEntry
	err := query.Order("timestamp desc").
		Limit(result.PageSize).
		Offset((result.Page - 1) * result.PageSize).
		Find(&rows).Error
	if err != nil {
		logger.Warning("Failed to read Xray log entries:", err)
		return result
	}

	for _, row := range rows {
		result.Entries = append(result.Entries, LogEntry{
			DateTime:    time.UnixMicro(row.Timestamp).UTC(),
			FromAddress: row.FromAddress,
			ToAddress:   row.ToAddress,
			Inbound:     row.Inbound,
			Outbound:    row.Outbound,
			Email:       row.Email,
			Event:       xrayLogEvent(row.Outbound, freedoms, blackholes),
		})
	}

	return result
}

// ResolveXrayLogDay maps a requested day onto the days that have stored
// entries, the same way the viewer does. It returns "" when nothing is stored.
func (s *ServerService) ResolveXrayLogDay(date string) string {
	db := database.GetLogDB()
	if db == nil {
		return ""
	}
	return resolveXrayLogDay(xrayLogDates(db), date)
}

// ExportXrayLogs writes every stored entry of day that matches the filters to
// w, oldest first, one line per entry. Rows are streamed from the database, so
// a busy day never has to fit in memory.
func (s *ServerService) ExportXrayLogs(
	w io.Writer,
	day string,
	email string,
	filter string,
	showDirect string,
	showBlocked string,
	showProxy string,
	freedoms []string,
	blackholes []string) error {

	db := database.GetLogDB()
	if db == nil || day == "" {
		return nil
	}

	query, ok := xrayLogQuery(db, day, email, filter, showDirect, showBlocked, showProxy, freedoms, blackholes)
	if !ok {
		return nil
	}

	rows, err := query.Order("timestamp asc").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	out := bufio.NewWriter(w)
	for rows.Next() {
		var row model.XrayLogEntry
		if err := db.ScanRows(rows, &row); err != nil {
			return err
		}
		if err := writeXrayLogLine(out, &row, xrayLogEvent(row.Outbound, freedoms, blackholes)); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return out.Flush()
}

// resolveXrayLogDay picks date if entries exist for it and the newest stored
// day otherwise; "" means nothing is stored at all.
func resolveXrayLogDay(days []string, date string) string {
	if slices.Contains(days, date) {
		return date
	}
	if len(days) == 0 {
		return ""
	}
	return days[0]
}

// xrayLogQuery builds the filtered query for one day that both the viewer and
// the export run. ok is false when the filters cannot match any row.
func xrayLogQuery(
	db *gorm.DB,
	day string,
	email string,
	filter string,
	showDirect string,
	showBlocked string,
	showProxy string,
	freedoms []string,
	blackholes []string) (*gorm.DB, bool) {

	query := db.Model(&model.XrayLogEntry{}).Where("day = ?", day)

	if email != "" {
		query = query.Where("email = ?", email)
	}

	if filter != "" {
		like := "%" + escapeSQLLike(filter) + "%"
		query = query.Where(
			`(from_address LIKE ? ESCAPE '\' OR to_address LIKE ? ESCAPE '\' OR inbound LIKE ? ESCAPE '\' OR outbound LIKE ? ESCAPE '\' OR email LIKE ? ESCAPE '\')`,
			like, like, like, like, like)
	}

	// The three checkboxes select on the outbound tag, which is what decides
	// whether a connection went out directly, was blackholed, or was proxied.
	if showDirect == "false" && len(freedoms) > 0 {
		query = query.Where("outbound NOT IN ?", freedoms)
	}
	if showBlocked == "false" && len(blackholes) > 0 {
		query = query.Where("outbound NOT IN ?", blackholes)
	}
	if showProxy == "false" {
		handled := append(append([]string{}, freedoms...), blackholes...)
		if len(handled) == 0 {
			return nil, false
		}
		query = query.Where("outbound IN ?", handled)
	}

	// The viewer runs Count and then the page read on these conditions. GORM
	// documents a chain as unsafe to reuse after a finisher, so freeze it.
	return query.Session(&gorm.Session{}), true
}

// xrayLogEvent classifies a connection by the outbound tag it left through.
func xrayLogEvent(outbound string, freedoms []string, blackholes []string) int {
	if slices.Contains(freedoms, outbound) {
		return xrayLogDirect
	}
	if slices.Contains(blackholes, outbound) {
		return xrayLogBlocked
	}
	return xrayLogProxied
}

// writeXrayLogLine writes one exported entry. The timestamp is in the server's
// local time and in Xray's own layout, so exported lines match the log files.
func writeXrayLogLine(w *bufio.Writer, row *model.XrayLogEntry, event int) error {
	line := time.UnixMicro(row.Timestamp).Format("2006/01/02 15:04:05.000000") +
		" FROM=" + row.FromAddress +
		" TO=" + row.ToAddress +
		" INBOUND=" + row.Inbound +
		" OUTBOUND=" + row.Outbound
	if row.Email != "" {
		line += " Email=" + row.Email
	}
	_, err := w.WriteString(line + " EVENT=" + xrayLogEventNames[event] + "\n")
	return err
}

// xrayLogDates lists the days that currently have stored entries, newest
// first. The day column is indexed, so this stays cheap as the table grows.
func xrayLogDates(db *gorm.DB) []string {
	var days []string
	err := db.Model(&model.XrayLogEntry{}).
		Distinct().
		Order("day desc").
		Pluck("day", &days).Error
	if err != nil {
		logger.Warning("Failed to list Xray log days:", err)
		return nil
	}
	return days
}

// xrayLogClients lists every client that appears anywhere in the stored logs.
// The list deliberately spans all retained days rather than just the selected
// one: a stable list means switching dates never silently drops the client
// the user picked, and a day where that client was idle is a meaningful
// answer in its own right.
func xrayLogClients(db *gorm.DB) []string {
	var emails []string
	err := db.Model(&model.XrayLogEntry{}).
		Where("email <> ''").
		Distinct().
		Order("email asc").
		Pluck("email", &emails).Error
	if err != nil {
		logger.Warning("Failed to list Xray log clients:", err)
		return nil
	}
	return emails
}

// escapeSQLLike neutralises the wildcards in a user supplied filter so that it
// matches literally.
func escapeSQLLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

func (s *ServerService) GetConfigJson() (any, error) {
	config, err := s.xrayService.GetXrayConfig()
	if err != nil {
		return nil, err
	}
	contents, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}

	var jsonData any
	err = json.Unmarshal(contents, &jsonData)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func (s *ServerService) GetDb() ([]byte, error) {
	// Update by manually trigger a checkpoint operation
	err := database.Checkpoint()
	if err != nil {
		return nil, err
	}
	// Open the file for reading
	file, err := os.Open(config.GetDBPath())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file contents
	fileContents, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return fileContents, nil
}

// PrepareDBImport checks the uploaded database and keeps it beside the
// current one, returning where; nothing is replaced yet.
func (s *ServerService) PrepareDBImport(file multipart.File) (string, error) {
	// Check if the file is a SQLite database
	isValidDb, err := database.IsSQLiteDB(file)
	if err != nil {
		return "", common.NewErrorf("Error checking db file format: %v", err)
	}
	if !isValidDb {
		return "", common.NewError("Invalid db file format")
	}

	// Reset the file reader to the beginning
	if _, err = file.Seek(0, 0); err != nil {
		return "", common.NewErrorf("Error resetting file reader: %v", err)
	}

	// Save the file as a temporary file
	tempPath := fmt.Sprintf("%s.temp", config.GetDBPath())

	// Remove the existing temporary file (if any)
	if _, err := os.Stat(tempPath); err == nil {
		if errRemove := os.Remove(tempPath); errRemove != nil {
			return "", common.NewErrorf("Error removing existing temporary db file: %v", errRemove)
		}
	}

	tempFile, err := os.Create(tempPath)
	if err != nil {
		return "", common.NewErrorf("Error creating temporary db file: %v", err)
	}
	_, err = io.Copy(tempFile, file)
	if cerr := tempFile.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		// Validate integrity (no migrations / side effects)
		if verr := database.ValidateSQLiteDB(tempPath); verr != nil {
			err = common.NewErrorf("Invalid or corrupt db file: %v", verr)
		}
	} else {
		err = common.NewErrorf("Error saving db: %v", err)
	}
	if err != nil {
		if rerr := os.Remove(tempPath); rerr != nil {
			logger.Warningf("Warning: failed to remove temp file: %v", rerr)
		}
		return "", err
	}
	return tempPath, nil
}

// ApplyDBImport puts a database PrepareDBImport kept in place of the
// current one, with Xray stopped meanwhile, migrates it and starts Xray.
// Should anything fail, the current database comes back.
func (s *ServerService) ApplyDBImport(tempPath string) error {
	defer func() {
		if _, err := os.Stat(tempPath); err == nil {
			if rerr := os.Remove(tempPath); rerr != nil {
				logger.Warningf("Warning: failed to remove temp file: %v", rerr)
			}
		}
	}()

	// Stop Xray (ignore error but log)
	if errStop := s.StopXrayService(); errStop != nil {
		logger.Warningf("Failed to stop Xray before DB import: %v", errStop)
	}

	// Close existing DB to release file locks (especially on Windows)
	if errClose := database.CloseDB(); errClose != nil {
		logger.Warningf("Failed to close existing DB before replacement: %v", errClose)
	}

	// Backup the current database for fallback
	fallbackPath := fmt.Sprintf("%s.backup", config.GetDBPath())

	// Remove the existing fallback file (if any)
	if _, err := os.Stat(fallbackPath); err == nil {
		if errRemove := os.Remove(fallbackPath); errRemove != nil {
			return common.NewErrorf("Error removing existing fallback db file: %v", errRemove)
		}
	}

	// Move the current database to the fallback location
	if err := os.Rename(config.GetDBPath(), fallbackPath); err != nil {
		return common.NewErrorf("Error backing up current db file: %v", err)
	}

	// Defer fallback cleanup ONLY if everything goes well
	defer func() {
		if _, err := os.Stat(fallbackPath); err == nil {
			if rerr := os.Remove(fallbackPath); rerr != nil {
				logger.Warningf("Warning: failed to remove fallback file: %v", rerr)
			}
		}
	}()

	// Move temp to DB path
	if err := os.Rename(tempPath, config.GetDBPath()); err != nil {
		// Restore from fallback
		if errRename := os.Rename(fallbackPath, config.GetDBPath()); errRename != nil {
			return common.NewErrorf("Error moving db file and restoring fallback: %v", errRename)
		}
		return common.NewErrorf("Error moving db file: %v", err)
	}

	// Open & migrate new DB
	if err := database.InitDB(config.GetDBPath()); err != nil {
		if errRename := os.Rename(fallbackPath, config.GetDBPath()); errRename != nil {
			return common.NewErrorf("Error migrating db and restoring fallback: %v", errRename)
		}
		return common.NewErrorf("Error migrating db: %v", err)
	}

	s.inboundService.MigrateDB()

	// Start Xray
	if err := s.RestartXrayService(); err != nil {
		return common.NewErrorf("Imported DB but failed to start Xray: %v", err)
	}

	return nil
}

// IsValidGeofileName validates that the filename is safe for geofile operations.
// It checks for path traversal attempts and ensures the filename contains only safe characters.
func (s *ServerService) IsValidGeofileName(filename string) bool {
	if filename == "" {
		return false
	}

	// Check for path traversal attempts
	if strings.Contains(filename, "..") {
		return false
	}

	// Check for path separators (both forward and backward slash)
	if strings.ContainsAny(filename, `/\`) {
		return false
	}

	// Check for absolute path indicators
	if filepath.IsAbs(filename) {
		return false
	}

	// Additional security: only allow alphanumeric, dots, underscores, and hyphens
	// This is stricter than the general filename regex
	validGeofilePattern := `^[a-zA-Z0-9._-]+\.dat$`
	matched, _ := regexp.MatchString(validGeofilePattern, filename)
	return matched
}

func (s *ServerService) UpdateGeofile(fileName string) error {
	type geofileEntry struct {
		URL      string
		FileName string
	}
	geofileAllowlist := map[string]geofileEntry{
		"geoip.dat":      {"https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat", "geoip.dat"},
		"geosite.dat":    {"https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat", "geosite.dat"},
		"geoip_RU.dat":   {"https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat", "geoip_RU.dat"},
		"geosite_RU.dat": {"https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat", "geosite_RU.dat"},
	}

	// Strict allowlist check to avoid writing uncontrolled files
	if fileName != "" {
		if _, ok := geofileAllowlist[fileName]; !ok {
			return common.NewErrorf("Invalid geofile name: %q not in allowlist", fileName)
		}
	}

	downloadFile := func(url, destPath string) error {
		var req *http.Request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return common.NewErrorf("Failed to create HTTP request for %s: %v", url, err)
		}

		var localFileModTime time.Time
		if fileInfo, err := os.Stat(destPath); err == nil {
			localFileModTime = fileInfo.ModTime()
			if !localFileModTime.IsZero() {
				req.Header.Set("If-Modified-Since", localFileModTime.UTC().Format(http.TimeFormat))
			}
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return common.NewErrorf("Failed to download Geofile from %s: %v", url, err)
		}
		defer resp.Body.Close()

		// Parse Last-Modified header from server
		var serverModTime time.Time
		serverModTimeStr := resp.Header.Get("Last-Modified")
		if serverModTimeStr != "" {
			parsedTime, err := time.Parse(http.TimeFormat, serverModTimeStr)
			if err != nil {
				logger.Warningf("Failed to parse Last-Modified header for %s: %v", url, err)
			} else {
				serverModTime = parsedTime
			}
		}

		// Function to update local file's modification time
		updateFileModTime := func() {
			if !serverModTime.IsZero() {
				if err := os.Chtimes(destPath, serverModTime, serverModTime); err != nil {
					logger.Warningf("Failed to update modification time for %s: %v", destPath, err)
				}
			}
		}

		// Handle 304 Not Modified
		if resp.StatusCode == http.StatusNotModified {
			updateFileModTime()
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			return common.NewErrorf("Failed to download Geofile from %s: received status code %d", url, resp.StatusCode)
		}

		file, err := os.Create(destPath)
		if err != nil {
			return common.NewErrorf("Failed to create Geofile %s: %v", destPath, err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return common.NewErrorf("Failed to save Geofile %s: %v", destPath, err)
		}

		updateFileModTime()
		return nil
	}

	var errorMessages []string

	if fileName == "" {
		// Download all geofiles
		for _, entry := range geofileAllowlist {
			destPath := filepath.Join(config.GetBinFolderPath(), entry.FileName)
			if err := downloadFile(entry.URL, destPath); err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("Error downloading Geofile '%s': %v", entry.FileName, err))
			}
		}
	} else {
		entry := geofileAllowlist[fileName]
		destPath := filepath.Join(config.GetBinFolderPath(), entry.FileName)
		if err := downloadFile(entry.URL, destPath); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("Error downloading Geofile '%s': %v", entry.FileName, err))
		}
	}

	s.RestartXrayLater("geofile update")

	if len(errorMessages) > 0 {
		return common.NewErrorf("%s", strings.Join(errorMessages, "\r\n"))
	}

	return nil
}

func (s *ServerService) GetNewX25519Cert() (any, error) {
	// Run the command
	cmd := exec.Command(xray.GetBinaryPath(), "x25519")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")

	privateKeyLine := strings.Split(lines[0], ":")
	publicKeyLine := strings.Split(lines[1], ":")

	privateKey := strings.TrimSpace(privateKeyLine[1])
	publicKey := strings.TrimSpace(publicKeyLine[1])

	keyPair := map[string]any{
		"privateKey": privateKey,
		"publicKey":  publicKey,
	}

	return keyPair, nil
}

func (s *ServerService) GetNewmldsa65() (any, error) {
	// Run the command
	cmd := exec.Command(xray.GetBinaryPath(), "mldsa65")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")

	SeedLine := strings.Split(lines[0], ":")
	VerifyLine := strings.Split(lines[1], ":")

	seed := strings.TrimSpace(SeedLine[1])
	verify := strings.TrimSpace(VerifyLine[1])

	keyPair := map[string]any{
		"seed":   seed,
		"verify": verify,
	}

	return keyPair, nil
}

func (s *ServerService) GetNewEchCert(sni string) (any, error) {
	// Run the command
	cmd := exec.Command(xray.GetBinaryPath(), "tls", "ech", "--serverName", sni)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")
	if len(lines) < 4 {
		return nil, common.NewError("invalid ech cert")
	}

	configList := lines[1]
	serverKeys := lines[3]

	return map[string]any{
		"echServerKeys": serverKeys,
		"echConfigList": configList,
	}, nil
}

func (s *ServerService) GetNewVlessEnc() (any, error) {
	cmd := exec.Command(xray.GetBinaryPath(), "vlessenc")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")
	var auths []map[string]string
	var current map[string]string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Authentication:") {
			if current != nil {
				auths = append(auths, current)
			}
			current = map[string]string{
				"label": strings.TrimSpace(strings.TrimPrefix(line, "Authentication:")),
			}
		} else if strings.HasPrefix(line, `"decryption"`) || strings.HasPrefix(line, `"encryption"`) {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && current != nil {
				key := strings.Trim(parts[0], `" `)
				val := strings.Trim(parts[1], `" `)
				current[key] = val
			}
		}
	}

	if current != nil {
		auths = append(auths, current)
	}

	return map[string]any{
		"auths": auths,
	}, nil
}

func (s *ServerService) GetNewUUID() (map[string]string, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	return map[string]string{
		"uuid": newUUID.String(),
	}, nil
}

func (s *ServerService) GetNewmlkem768() (any, error) {
	// Run the command
	cmd := exec.Command(xray.GetBinaryPath(), "mlkem768")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")

	SeedLine := strings.Split(lines[0], ":")
	ClientLine := strings.Split(lines[1], ":")

	seed := strings.TrimSpace(SeedLine[1])
	client := strings.TrimSpace(ClientLine[1])

	keyPair := map[string]any{
		"seed":   seed,
		"client": client,
	}

	return keyPair, nil
}
