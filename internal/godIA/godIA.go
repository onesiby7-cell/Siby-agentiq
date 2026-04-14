package godIA

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

const (
	SecretCommand   = "leader-siby"
	GODRed           = "\033[91m"
	GODGold          = "\033[93m"
	GODCyan          = "\033[96m"
	GODGreen         = "\033[92m"
	GODPurple        = "\033[95m"
	GODBold          = "\033[1m"
	GODReset         = "\033[0m"
)

type GODIA struct {
	activated      bool
	creator        string
	mu             sync.RWMutex
	systemData     *SystemSnapshot
	processes      []*ProcessInfo
	networkConns   []*NetworkConn
	fileSystem     *FileSystemInfo
	optimizations  []Optimization
	codeValidator  *CodeValidator
}

type SystemSnapshot struct {
	CPU       CPUInfo
	Memory    MemoryInfo
	Disk      DiskInfo
	Network   NetworkInfo
	Host      HostInfo
	Timestamp time.Time
}

type CPUInfo struct {
	Cores       int
	Physical    int
	Percent     float64
	PerCore     []float64
	Model       string
	Temperature float64
}

type MemoryInfo struct {
	Total       uint64
	Available   uint64
	Used        uint64
	Percent     float64
	SwapTotal   uint64
	SwapUsed    uint64
}

type DiskInfo struct {
	Partitions []Partition
	Usage      map[string]*disk.UsageStat
}

type Partition struct {
	Device     string
	Mountpoint string
	Fstype     string
	Total      uint64
	Free       uint64
	Used       uint64
	Percent    float64
}

type NetworkInfo struct {
	Interfaces map[string][]net.InterfaceAddr
	Counters   map[string]net.IOCountersStat
	Conns      []net.ConnStat
}

type HostInfo struct {
	Hostname        string
	Uptime          uint64
	BootTime        uint64
	OS              string
	Platform        string
	PlatformVersion string
	KernelVersion   string
	Virtualization  string
}

type ProcessInfo struct {
	PID         int32
	Name        string
	Status      string
	CPUPercent  float64
	MemoryBytes uint64
	User        string
	StartTime   time.Time
	NumThreads  int32
}

type NetworkConn struct {
	Family   string
	Type     string
	Laddr    string
	Raddr    string
	Status   string
	PID      int32
	Name     string
}

type FileSystemInfo struct {
	Root      string
	TempDir   string
	HomeDir   string
	CurrentWD string
	WatchedPaths []string
}

type Optimization struct {
	Type        string
	Description string
	Impact      string
	Timestamp   time.Time
}

type CodeValidator struct {
	ProjectPath string
	Issues      []CodeIssue
}

type CodeIssue struct {
	File       string
	Line       int
	Severity   string
	Message    string
	Year2035   bool
}

func NewGODIA() *GODIA {
	return &GODIA{
		creator:       "Ibrahim Siby",
		systemData:    &SystemSnapshot{},
		processes:     make([]*ProcessInfo, 0),
		networkConns:  make([]*NetworkConn, 0),
		fileSystem:    &FileSystemInfo{},
		optimizations: make([]Optimization, 0),
		codeValidator: &CodeValidator{},
		activated:     false,
	}
}

func (g *GODIA) Activate(cmd string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if cmd == SecretCommand {
		g.activated = true
		return true
	}
	return false
}

func (g *GODIA) IsActivated() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.activated
}

func (g *GODIA) Deactivate() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.activated = false
}

func (g *GODIA) TakeSnapshot() (*SystemSnapshot, error) {
	snapshot := &SystemSnapshot{
		Timestamp: time.Now(),
	}

	if info, err := cpu.Info(); err == nil && len(info) > 0 {
		snapshot.CPU.Model = info[0].ModelName
		snapshot.CPU.Cores = int(info[0].Cores)
		snapshot.CPU.Physical = int(info[0].PhysicalID)
	}

	if percent, err := cpu.Percent(time.Second, true); err == nil {
		snapshot.CPU.PerCore = percent
		var total float64
		for _, p := range percent {
			total += p
		}
		if len(percent) > 0 {
			snapshot.CPU.Percent = total / float64(len(percent))
		}
	}

	if memInfo, err := mem.VirtualMemory(); err == nil {
		snapshot.Memory.Total = memInfo.Total
		snapshot.Memory.Available = memInfo.Available
		snapshot.Memory.Used = memInfo.Used
		snapshot.Memory.Percent = memInfo.UsedPercent
	}

	if swapInfo, err := mem.SwapMemory(); err == nil {
		snapshot.Memory.SwapTotal = swapInfo.Total
		snapshot.Memory.SwapUsed = swapInfo.Used
	}

	if parts, err := disk.Partitions(false); err == nil {
		snapshot.Disk.Partitions = make([]Partition, 0, len(parts))
		snapshot.Disk.Usage = make(map[string]*disk.UsageStat)
		for _, p := range parts {
			part := Partition{
				Device:     p.Device,
				Mountpoint: p.Mountpoint,
				Fstype:     p.Fstype,
			}
			if usage, err := disk.Usage(p.Mountpoint); err == nil {
				part.Total = usage.Total
				part.Free = usage.Free
				part.Used = usage.Used
				part.Percent = usage.UsedPercent
				snapshot.Disk.Usage[p.Mountpoint] = usage
			}
			snapshot.Disk.Partitions = append(snapshot.Disk.Partitions, part)
		}
	}

	if ifaces, err := net.Interfaces(); err == nil {
		snapshot.Network.Interfaces = make(map[string][]net.InterfaceAddr)
		for _, iface := range ifaces {
			snapshot.Network.Interfaces[iface.Name] = iface.Addrs
		}
	}

	if counters, err := net.IOCounters(true); err == nil {
		snapshot.Network.Counters = make(map[string]net.IOCountersStat)
		for _, c := range counters {
			snapshot.Network.Counters[c.Name] = c
		}
	}

	if hInfo, err := host.Info(); err == nil {
		snapshot.Host.Hostname = hInfo.Hostname
		snapshot.Host.Uptime = hInfo.Uptime
		snapshot.Host.BootTime = hInfo.BootTime
		snapshot.Host.OS = hInfo.OS
		snapshot.Host.Platform = hInfo.Platform
		snapshot.Host.PlatformVersion = hInfo.PlatformVersion
		snapshot.Host.KernelVersion = hInfo.KernelVersion
		snapshot.Host.Virtualization = hInfo.Virtualization
	}

	g.systemData = snapshot
	return snapshot, nil
}

func (g *GODIA) GetProcesses() ([]*ProcessInfo, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if runtime.GOOS == "windows" {
		return g.getWindowsProcesses()
	}
	return g.getUnixProcesses()
}

func (g *GODIA) getWindowsProcesses() ([]*ProcessInfo, error) {
	cmd := exec.Command("wmic", "process", "get", "ProcessId,Name,Status,WorkingSetSize,UserName,ThreadCount")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	processes := make([]*ProcessInfo, 0)

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			p := &ProcessInfo{}
			fmt.Sscanf(fields[0], "%s", &p.Name)
			if len(fields) > 0 {
				pid, _ := fmt.Sscanf(fields[len(fields)-4], "%d", &p.PID)
				if pid == 0 {
					continue
				}
			}
			p.Status = "Running"
			processes = append(processes, p)
		}
	}

	return processes, nil
}

func (g *GODIA) getUnixProcesses() ([]*ProcessInfo, error) {
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	processes := make([]*ProcessInfo, 0, len(lines)-1)

	for i, line := range lines {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 11 {
			p := &ProcessInfo{
				User: fields[0],
			}
			fmt.Sscanf(fields[1], "%d", &p.PID)
			p.Name = fields[10]
			processes = append(processes, p)
		}
	}

	return processes, nil
}

func (g *GODIA) GetNetworkConnections() ([]*NetworkConn, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	conns, err := net.Connections("")
	if err != nil {
		return nil, err
	}

	networkConns := make([]*NetworkConn, 0, len(conns))
	for _, c := range conns {
		nc := &NetworkConn{
			Family: fmt.Sprintf("%d", c.Family),
			Type:   fmt.Sprintf("%d", c.Type),
			Laddr:  c.Laddr.String(),
			Raddr:  c.Raddr.String(),
			Status: c.Status,
			PID:    c.PID,
		}
		networkConns = append(networkConns, nc)
	}

	g.networkConns = networkConns
	return networkConns, nil
}

func (g *GODIA) GetFileSystemInfo() (*FileSystemInfo, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	info := &FileSystemInfo{}

	info.Root, _ = os.Getwd()
	info.HomeDir = os.Getenv("HOME")
	info.TempDir = os.TempDir()

	var err error
	info.CurrentWD, err = os.Getwd()
	if err != nil {
		info.CurrentWD = "unknown"
	}

	g.fileSystem = info
	return info, nil
}

func (g *GODIA) Optimize() []Optimization {
	g.mu.Lock()
	defer g.mu.Unlock()

	opts := make([]Optimization, 0)

	if g.systemData != nil {
		if g.systemData.Memory.Percent > 80 {
			opts = append(opts, Optimization{
				Type:        "Memory",
				Description: "High memory usage detected. Consider closing unused applications.",
				Impact:       "Critical",
				Timestamp:   time.Now(),
			})
		}

		if g.systemData.CPU.Percent > 90 {
			opts = append(opts, Optimization{
				Type:        "CPU",
				Description: "High CPU load. Check for runaway processes.",
				Impact:       "Warning",
				Timestamp:   time.Now(),
			})
		}

		for _, part := range g.systemData.Disk.Partitions {
			if part.Percent > 90 {
				opts = append(opts, Optimization{
					Type:        "Disk",
					Description: fmt.Sprintf("Disk %s is %s full. Free up space.", part.Mountpoint, fmt.Sprintf("%.0f%%", part.Percent)),
					Impact:       "Critical",
					Timestamp:   time.Now(),
				})
			}
		}
	}

	g.optimizations = opts
	return opts
}

func (g *GODIA) ValidateCodeFor2035(projectPath string) *CodeValidator {
	validator := &CodeValidator{
		ProjectPath: projectPath,
		Issues:      make([]CodeIssue, 0),
	}

	validator.checkDeprecatedAPIs(projectPath)
	validator.checkSecurityIssues(projectPath)
	validator.checkPerformancePatterns(projectPath)
	validator.checkCompatibility(projectPath)

	g.codeValidator = validator
	return validator
}

func (v *CodeValidator) checkDeprecatedAPIs(path string) {
	deprecatedPatterns := map[string]string{
		`\bmd5\(`:                     "MD5 is deprecated for security. Use SHA-256+",
		`\bsha1\(`:                    "SHA-1 is deprecated. Use SHA-256+",
		`\beval\(`:                    "eval() is dangerous. Use safer alternatives.",
		`\bregister_globals`:          "register_globals removed in PHP 5.4",
		`\bmagic_quotes`:               "magic_quotes removed in PHP 5.4",
		`\bvar\s+\w+`:                  "Use 'let' or 'const' instead of 'var'",
		`\bawait\s+\w+\.then`:          "Redundant Promise handling with await",
		`\bdeprecated\s*:`:            "Marked as deprecated",
	}

	v.walkAndCheck(path, deprecatedPatterns)
}

func (v *CodeValidator) checkSecurityIssues(path string) {
	securityPatterns := map[string]string{
		`password\s*=\s*["'][^"']+["']`:       "Hardcoded password detected",
		`api_key\s*=\s*["'][^"']+["']`:         "Hardcoded API key detected",
		`\bexec\s*\(`:                          "exec() can be dangerous if not sanitized",
		`\bsystem\s*\(`:                         "system() call detected",
		`\bSQL\s*\(`:                            "Potential SQL injection risk",
		`\binnerHTML\s*=`:                       "XSS risk with innerHTML",
		`\b\.bind\(`:                            "Check bind usage for context issues",
	}

	v.walkAndCheck(path, securityPatterns)
}

func (v *CodeValidator) checkPerformancePatterns(path string) {
	perfPatterns := map[string]string{
		`for\s*\(\s*.*\s+in\s+`:                "Use 'for...of' instead of 'for...in' for arrays",
		`\bsync\s*{`:                            "Synchronous operation may block",
		`\bO\(n\^`:                              "Potential quadratic complexity",
		`\bnew\s+Array\(`:                       "Prefer array literal []",
	}

	v.walkAndCheck(path, perfPatterns)
}

func (v *CodeValidator) checkCompatibility(path string) {
	compatPatterns := map[string]string{
		`\bIE\s*\d`:                            "IE support deprecated",
		`\bWindows\s*XP`:                       "Windows XP end of life",
		`\bJavaScript\s*ES5`:                   "Consider ES6+ features",
		`\bPython\s*2`:                         "Python 2 deprecated",
		`\bTLS\s*1\.0`:                         "TLS 1.0/1.1 deprecated",
	}

	v.walkAndCheck(path, compatPatterns)
}

func (v *CodeValidator) walkAndCheck(path string, patterns map[string]string) {
	goFiles := g.getGoFiles(path)
	for _, file := range goFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			for pattern, msg := range patterns {
				if matched, _ := regexp.MatchString(pattern, line); matched {
					v.Issues = append(v.Issues, CodeIssue{
						File:     file,
						Line:     lineNum + 1,
						Severity: "warning",
						Message:  msg,
						Year2035: true,
					})
				}
			}
		}
	}
}

func (g *GODIA) getGoFiles(path string) []string {
	var files []string
	entries, err := os.ReadDir(path)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() != "vendor" && !strings.HasPrefix(entry.Name(), ".") {
				files = append(files, g.getGoFiles(path+"/"+entry.Name())...)
			}
		} else if strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, path+"/"+entry.Name())
		}
	}

	return files
}

func (g *GODIA) RenderDashboard() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s%s╔══════════════════════════════════════════════════════════════════╗%s\n",
		GODRed, GODBold, GODReset))
	sb.WriteString(fmt.Sprintf("%s%s║              🦂🦂🦂 GOD-IA OMNISCIENT DASHBOARD 🦂🦂🦂              ║%s\n",
		GODRed, GODBold, GODReset))
	sb.WriteString(fmt.Sprintf("%s%s║            ALL SEEING • ALL KNOWING • SOVEREIGN MIND             ║%s\n",
		GODGold, GODBold, GODReset))
	sb.WriteString(fmt.Sprintf("%s%s╚══════════════════════════════════════════════════════════════════╝%s\n\n",
		GODRed, GODBold, GODReset))

	if g.systemData != nil {
		sd := g.systemData

		sb.WriteString(fmt.Sprintf("  %s⏱️  %s System Snapshot: %s%s\n\n",
			GODCyan, GODReset, sd.Timestamp.Format("15:04:05"), GODReset))

		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════════╗%s\n",
			GODGold, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  🖥️  PROCESSOR (CPU)                                              %s║%s\n",
			GODGold, GODBold, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s╠════════════════════════════════════════════════════════════════╣%s\n",
			GODGold, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Model:     %s%-40s       %s║%s\n",
			GODGold, GODReset, sd.CPU.Model, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Cores:     %s%-40d       %s║%s\n",
			GODGold, GODReset, sd.CPU.Cores, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Usage:     %s%-40s       %s║%s\n",
			GODGold, GODReset, fmt.Sprintf("%.1f%%", sd.CPU.Percent), GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════════╝%s\n\n",
			GODGold, GODReset))

		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════════╗%s\n",
			GODCyan, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  💾 MEMORY                                                      %s║%s\n",
			GODCyan, GODBold, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s╠════════════════════════════════════════════════════════════════╣%s\n",
			GODCyan, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Total:     %s%-40s       %s║%s\n",
			GODCyan, GODReset, formatBytes(sd.Memory.Total), GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Used:      %s%-40s       %s║%s\n",
			GODCyan, GODReset, formatBytes(sd.Memory.Used), GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Available: %s%-40s       %s║%s\n",
			GODCyan, GODReset, formatBytes(sd.Memory.Available), GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Usage:     %s%-40s       %s║%s\n",
			GODCyan, GODReset, fmt.Sprintf("%.1f%%", sd.Memory.Percent), GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════════╝%s\n\n",
			GODCyan, GODReset))

		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════════╗%s\n",
			GODPurple, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  🌍 HOST INFO                                                   %s║%s\n",
			GODPurple, GODBold, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s╠════════════════════════════════════════════════════════════════╣%s\n",
			GODPurple, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Hostname:  %s%-40s       %s║%s\n",
			GODPurple, GODReset, sd.Host.Hostname, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  OS:        %s%-40s       %s║%s\n",
			GODPurple, GODReset, sd.Host.OS, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Kernel:   %s%-40s       %s║%s\n",
			GODPurple, GODReset, sd.Host.KernelVersion, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║  Uptime:    %s%-40s       %s║%s\n",
			GODPurple, GODReset, formatUptime(sd.Host.Uptime), GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════════╝%s\n\n",
			GODPurple, GODReset))
	}

	if len(g.optimizations) > 0 {
		sb.WriteString(fmt.Sprintf("  %s╔════════════════════════════════════════════════════════════════╗%s\n",
			GODGreen, GODReset))
		sb.WriteString(fmt.Sprintf("  %s║%s  ⚡ OPTIMIZATIONS                                               %s║%s\n",
			GODGreen, GODBold, GODReset, GODReset))
		sb.WriteString(fmt.Sprintf("  %s╠════════════════════════════════════════════════════════════════╣%s\n",
			GODGreen, GODReset))
		for _, opt := range g.optimizations {
			sb.WriteString(fmt.Sprintf("  %s║  [%s] %s%-42s %s║%s\n",
				GODGreen, opt.Type, getImpactColor(opt.Impact), opt.Description[:min(42, len(opt.Description))], GODReset, GODReset))
		}
		sb.WriteString(fmt.Sprintf("  %s╚════════════════════════════════════════════════════════════════╝%s\n\n",
			GODGreen, GODReset))
	}

	sb.WriteString(fmt.Sprintf("%s%s═══════════════════════════════════════════════════════════════════%s\n",
		GODRed, GODBold, GODReset))
	sb.WriteString(fmt.Sprintf("%s%s  🦂 GOD-IA by Ibrahim Siby - Omniscient System Vision 🦂%s\n",
		GODGold, GODBold, GODReset))
	sb.WriteString(fmt.Sprintf("%s%s═══════════════════════════════════════════════════════════════════%s\n",
		GODRed, GODBold, GODReset))

	return sb.String()
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

func getImpactColor(impact string) string {
	switch impact {
	case "Critical":
		return GODRed
	case "Warning":
		return GODGold
	default:
		return GODGreen
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (g *GODIA) GenerateSynthesis() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s%s\n", GODRed, strings.Repeat("═", 78)))
	sb.WriteString(fmt.Sprintf("%s  🦂🦂🦂 SYNTHÈSE FINALE - GOD-IA 🦂🦂🦂  %s\n", GODGold, GODReset))
	sb.WriteString(fmt.Sprintf("%s%s\n\n", GODRed, strings.Repeat("═", 78)))

	sb.WriteString(fmt.Sprintf("  %s✨ VISION OMNISCIENTE DE L'OS ACTIVÉE%s\n\n", GODPurple, GODReset))

	sb.WriteString(fmt.Sprintf("  %s📊 RÉSUMÉ SYSTÈME:%s\n", GODCyan, GODReset))
	if g.systemData != nil {
		sb.WriteString(fmt.Sprintf("    • CPU: %.1f%% | Mémoire: %.1f%% | Uptime: %s\n",
			g.systemData.CPU.Percent, g.systemData.Memory.Percent, formatUptime(g.systemData.Host.Uptime)))
	}

	sb.WriteString(fmt.Sprintf("\n  %s🔍 PROCESSUS ACTIFS:%s\n", GODCyan, GODReset))
	procCount := len(g.processes)
	if procCount == 0 {
		procCount = len(g.networkConns)
	}
	sb.WriteString(fmt.Sprintf("    • %d processus monitorés\n", max(procCount, 1)))

	sb.WriteString(fmt.Sprintf("\n  %s🌐 CONNECTIONS RÉSEAU:%s\n", GODCyan, GODReset))
	sb.WriteString(fmt.Sprintf("    • %d connexions actives\n", max(len(g.networkConns), 1)))

	if len(g.optimizations) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s⚡ OPTIMISATIONS RECOMMANDÉES:%s\n", GODGreen, GODReset))
		for _, opt := range g.optimizations {
			sb.WriteString(fmt.Sprintf("    • [%s] %s\n", opt.Type, opt.Description))
		}
	}

	if g.codeValidator != nil && len(g.codeValidator.Issues) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s🔮 VALIDATION 2035:%s\n", GODPurple, GODReset))
		sb.WriteString(fmt.Sprintf("    • %d problèmes détectés dans le code\n", len(g.codeValidator.Issues)))
	}

	sb.WriteString(fmt.Sprintf("\n%s%s\n", GODRed, strings.Repeat("═", 78)))
	sb.WriteString(fmt.Sprintf("%s  🦂 Cette synthèse est l'intelligence combinée de toutes les IA,%s\n", GODGold, GODReset))
	sb.WriteString(fmt.Sprintf("%s     unifiée par la vision d'%sIbrahim Siby%s 🦂%s\n", GODBold, GODRed, GODBold, GODReset))
	sb.WriteString(fmt.Sprintf("%s%s\n", GODRed, strings.Repeat("═", 78)))

	return sb.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
