package metrics

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	netutil "github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"fmt"
	"runtime"
)

type ProcessInfo struct {
	PID    int32   `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	Memory float32 `json:"memory"`
}

type InterfaceStats struct {
	Name    string `json:"name"`
	RX      uint64 `json:"rx"`
	TX      uint64 `json:"tx"`
	RXSpeed uint64 `json:"rx_speed"` // bytes/sec
	TXSpeed uint64 `json:"tx_speed"` // bytes/sec
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
	Hostname     string           `json:"hostname"`
	OS           string           `json:"os"`
	Uptime       uint64           `json:"uptime"`
	CPUUsage     float64          `json:"cpu_usage"`
	CPUCount     int              `json:"cpu_count"`
	RAMUsage     float64          `json:"ram_usage"`
	TotalRAM     uint64           `json:"total_ram"`
	DiskUsage    float64          `json:"disk_usage"`
	TotalDisk    uint64           `json:"total_disk"`
	NetRX        uint64           `json:"net_rx"`
	NetTX        uint64           `json:"net_tx"`
	Interfaces   []InterfaceStats `json:"interfaces"`
	Connections  []NetConn        `json:"connections"`
	Docker       []DockerInfo     `json:"docker"`
	SSHLogins    []SSHLogin       `json:"ssh_logins"`
	DiskIO       DiskIOStats      `json:"disk_io"`
	LocalIP      string           `json:"local_ip"`
	PublicIP     string           `json:"public_ip"`
	MachineUUID  string           `json:"machine_uuid"`
	Processes    []ProcessInfo    `json:"processes"`
	Timestamp    int64            `json:"timestamp"`
}

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
		Docker:      getDockerStats(),
		SSHLogins:   getSSHLogins(),
		DiskIO:      getDiskIO(),
		LocalIP:     getPrimaryLocalIP(),
		PublicIP:    getPublicIP(),
		MachineUUID: machineID,
		Processes:   getTopProcesses(),
		Timestamp:   time.Now().Unix(),
	}

	return stats, nil
}

func getTopProcesses() []ProcessInfo {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	var infos []ProcessInfo
	for _, p := range procs {
		name, _ := p.Name()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		
		if cpu > 0.1 || mem > 1.0 { // Only collect active/heavy processes
			infos = append(infos, ProcessInfo{
				PID:    p.Pid,
				Name:   name,
				CPU:    cpu,
				Memory: mem,
			})
		}
		
		if len(infos) > 10 { // Limit to top 10
			break
		}
	}
	return infos
}

func getInterfaceStats() []InterfaceStats {
	ioCounters, err := netutil.IOCounters(true)
	if err != nil {
		return nil
	}

	var stats []InterfaceStats
	for _, io := range ioCounters {
		if io.BytesRecv == 0 && io.BytesSent == 0 {
			continue // Skip inactive
		}
		stats = append(stats, InterfaceStats{
			Name: io.Name,
			RX:   io.BytesRecv,
			TX:   io.BytesSent,
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
		if len(result) > 10 {
			break
		}
	}
	return result
}

func getDiskIO() DiskIOStats {
	io, err := disk.IOCounters()
	if err != nil || len(io) == 0 {
		return DiskIOStats{}
	}
	// Aggregate all disks for simplicity or pick the primary one
	var totalRead, totalWrite uint64
	for _, stats := range io {
		totalRead += stats.ReadBytes
		totalWrite += stats.WriteBytes
	}
	return DiskIOStats{ReadBytes: totalRead, WriteBytes: totalWrite}
}

func getSSHLogins() []SSHLogin {
	// Standard Linux 'last' command
	out, err := exec.Command("last", "-n", "5", "-i").Output()
	if err != nil {
		return nil
	}

	var logins []SSHLogin
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 10 {
			logins = append(logins, SSHLogin{
				User: fields[0],
				IP:   fields[2],
				Date: strings.Join(fields[3:7], " "),
			})
		}
	}
	return logins
}

func getDockerStats() []DockerInfo {
	// We can check if docker is running by looking at the socket
	if _, err := os.Stat("/var/run/docker.sock"); os.IsNotExist(err) {
		return nil
	}

	// For production-grade we would use the Docker SDK, 
	// but to keep dependencies minimal as requested, we can use a quick curl to the socket or a shell command.
	// Using 'docker ps' is safer if the user has docker installed.
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}|{{.Status}}|{{.Image}}", "--limit", "10").Output()
	if err != nil {
		return nil
	}

	var containers []DockerInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) == 3 {
			containers = append(containers, DockerInfo{
				Name:   parts[0],
				Status: parts[1],
				Image:  parts[2],
			})
		}
	}
	return containers
}

func getPublicIP() string {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "unknown"
	}
	return string(ip)
}

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
