package metrics

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	netutil "github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

// --- Data Structs ---

type ProcessInfo struct {
	PID    int32   `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	Memory float32 `json:"memory"`
}

type InterfaceStats struct {
	Name string `json:"name"`
	RX   uint64 `json:"rx"`
	TX   uint64 `json:"tx"`
}

type NetConn struct {
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
	Status     string `json:"status"`
}

type DockerInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Image  string `json:"image"`
}

type SSHLogin struct {
	User string `json:"user"`
	IP   string `json:"ip"`
	Date string `json:"date"`
}

type DiskIOStats struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
}

type Stats struct {
	Hostname    string           `json:"hostname"`
	OS          string           `json:"os"`
	Uptime      uint64           `json:"uptime"`
	CPUUsage    float64          `json:"cpu_usage"`
	CPUCount    int              `json:"cpu_count"`
	RAMUsage    float64          `json:"ram_usage"`
	TotalRAM    uint64           `json:"total_ram"`
	DiskUsage   float64          `json:"disk_usage"`
	TotalDisk   uint64           `json:"total_disk"`
	NetRX       uint64           `json:"net_rx"`
	NetTX       uint64           `json:"net_tx"`
	Interfaces  []InterfaceStats `json:"interfaces"`
	Connections []NetConn        `json:"connections"`
	Docker      []DockerInfo     `json:"docker"`
	SSHLogins   []SSHLogin       `json:"ssh_logins"`
	DiskIO      DiskIOStats      `json:"disk_io"`
	LocalIP     string           `json:"local_ip"`
	PublicIP    string           `json:"public_ip"`
	MachineUUID string           `json:"machine_uuid"`
	Processes   []ProcessInfo    `json:"processes"`
	Timestamp   int64            `json:"timestamp"`
}

// --- Public IP Cache ---
// Prevents hammering api.ipify.org every heartbeat

var (
	cachedPublicIP   string
	publicIPExpiry   time.Time
	publicIPMu       sync.Mutex
	publicIPCacheTTL = 5 * time.Minute
)

// --- Main Collection ---

func Collect(ctx context.Context, machineID string) (*Stats, error) {
	hInfo, _ := host.InfoWithContext(ctx)
	vMem, _ := mem.VirtualMemoryWithContext(ctx)
	cpuPerc, _ := cpu.PercentWithContext(ctx, 0, false)
	dUsage, _ := disk.UsageWithContext(ctx, "/")
	netIO, _ := netutil.IOCountersWithContext(ctx, false)

	var cpuVal float64
	if len(cpuPerc) > 0 {
		cpuVal = cpuPerc[0]
	}

	var rx, tx uint64
	if len(netIO) > 0 {
		rx = netIO[0].BytesRecv
		tx = netIO[0].BytesSent
	}

	stats := &Stats{
		Hostname:    hInfo.Hostname,
		OS:          hInfo.OS,
		Uptime:      hInfo.Uptime,
		CPUUsage:    cpuVal,
		CPUCount:    runtime.NumCPU(),
		RAMUsage:    vMem.UsedPercent,
		TotalRAM:    vMem.Total,
		DiskUsage:   dUsage.UsedPercent,
		TotalDisk:   dUsage.Total,
		NetRX:       rx,
		NetTX:       tx,
		Interfaces:  getInterfaceStats(),
		Connections: getTopConnections(),
		Docker:      getDockerContainers(),
		SSHLogins:   getSSHLogins(),
		DiskIO:      getDiskIO(),
		LocalIP:     getPrimaryLocalIP(),
		PublicIP:    getCachedPublicIP(),
		MachineUUID: machineID,
		Processes:   getTopProcesses(),
		Timestamp:   time.Now().Unix(),
	}

	return stats, nil
}

// --- Process Collection ---

func getTopProcesses() []ProcessInfo {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	var infos []ProcessInfo
	for _, p := range procs {
		name, _ := p.Name()
		cpuPct, _ := p.CPUPercent()
		memPct, _ := p.MemoryPercent()

		if cpuPct > 0.1 || memPct > 1.0 {
			infos = append(infos, ProcessInfo{
				PID:    p.Pid,
				Name:   name,
				CPU:    cpuPct,
				Memory: memPct,
			})
		}

		if len(infos) >= 10 {
			break
		}
	}
	return infos
}

// --- Network Collection ---

func getInterfaceStats() []InterfaceStats {
	ioCounters, err := netutil.IOCounters(true)
	if err != nil {
		return nil
	}

	var stats []InterfaceStats
	for _, ioc := range ioCounters {
		if ioc.BytesRecv == 0 && ioc.BytesSent == 0 {
			continue
		}
		stats = append(stats, InterfaceStats{
			Name: ioc.Name,
			RX:   ioc.BytesRecv,
			TX:   ioc.BytesSent,
		})
	}
	return stats
}

func getTopConnections() []NetConn {
	conns, err := netutil.Connections("tcp")
	if err != nil {
		return nil
	}

	var result []NetConn
	for _, c := range conns {
		if c.Status == "ESTABLISHED" {
			result = append(result, NetConn{
				LocalAddr:  fmt.Sprintf("%s:%d", c.Laddr.IP, c.Laddr.Port),
				RemoteAddr: fmt.Sprintf("%s:%d", c.Raddr.IP, c.Raddr.Port),
				Status:     c.Status,
			})
		}
		if len(result) >= 15 {
			break
		}
	}
	return result
}

// --- Disk I/O ---

func getDiskIO() DiskIOStats {
	counters, err := disk.IOCounters()
	if err != nil || len(counters) == 0 {
		return DiskIOStats{}
	}

	var totalRead, totalWrite uint64
	for _, s := range counters {
		totalRead += s.ReadBytes
		totalWrite += s.WriteBytes
	}
	return DiskIOStats{ReadBytes: totalRead, WriteBytes: totalWrite}
}

// --- SSH Login Audit (NO os/exec — reads /var/log/auth.log directly) ---

func getSSHLogins() []SSHLogin {
	// Try common auth log paths
	logPaths := []string{
		"/var/log/auth.log",     // Ubuntu/Debian
		"/var/log/secure",       // RHEL/CentOS
	}

	for _, logPath := range logPaths {
		logins := parseAuthLog(logPath)
		if logins != nil {
			return logins
		}
	}
	return nil
}

func parseAuthLog(path string) []SSHLogin {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var logins []SSHLogin
	scanner := bufio.NewScanner(file)

	// Read all lines and keep only last N "Accepted" SSH entries
	var allAccepted []SSHLogin
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Accepted") && strings.Contains(line, "sshd") {
			login := parseSSHLine(line)
			if login != nil {
				allAccepted = append(allAccepted, *login)
			}
		}
	}

	// Return last 5 entries
	start := 0
	if len(allAccepted) > 5 {
		start = len(allAccepted) - 5
	}
	logins = allAccepted[start:]
	return logins
}

func parseSSHLine(line string) *SSHLogin {
	// Format: "May  8 03:00:01 hostname sshd[12345]: Accepted publickey for user from 1.2.3.4 port 22 ssh2"
	parts := strings.Fields(line)
	if len(parts) < 11 {
		return nil
	}

	// Find "for" keyword to get user and "from" for IP
	var user, ip, date string
	date = strings.Join(parts[0:3], " ")

	for i, p := range parts {
		if p == "for" && i+1 < len(parts) {
			user = parts[i+1]
		}
		if p == "from" && i+1 < len(parts) {
			ip = parts[i+1]
		}
	}

	if user == "" || ip == "" {
		return nil
	}

	return &SSHLogin{User: user, IP: ip, Date: date}
}

// --- Docker (NO os/exec — reads Docker socket directly via HTTP) ---

func getDockerContainers() []DockerInfo {
	// Check if Docker socket exists
	if _, err := os.Stat("/var/run/docker.sock"); os.IsNotExist(err) {
		return nil
	}

	// Connect to Docker daemon via Unix socket (no exec needed)
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}

	resp, err := client.Get("http://localhost/containers/json?limit=10")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// Limit response size to 64KB
	limitedReader := io.LimitReader(resp.Body, 65536)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil
	}

	var containers []struct {
		Names  []string `json:"Names"`
		State  string   `json:"State"`
		Image  string   `json:"Image"`
		Status string   `json:"Status"`
	}
	if err := json.Unmarshal(body, &containers); err != nil {
		return nil
	}

	var result []DockerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		result = append(result, DockerInfo{
			Name:   name,
			Status: c.Status,
			Image:  c.Image,
		})
	}
	return result
}

// --- Public IP (Cached, rate-limited) ---

func getCachedPublicIP() string {
	publicIPMu.Lock()
	defer publicIPMu.Unlock()

	if cachedPublicIP != "" && time.Now().Before(publicIPExpiry) {
		return cachedPublicIP
	}

	ip := fetchPublicIP()
	cachedPublicIP = ip
	publicIPExpiry = time.Now().Add(publicIPCacheTTL)
	return ip
}

func fetchPublicIP() string {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()

	// Limit to 64 bytes — an IP address is at most 45 chars
	ip, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(ip))
}

// --- Local IP ---

func getPrimaryLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}
