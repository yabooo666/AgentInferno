package metrics

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	netutil "github.com/shirou/gopsutil/net"
)

type Stats struct {
	Hostname     string  `json:"hostname"`
	OS           string  `json:"os"`
	Uptime       uint64  `json:"uptime"`
	CPUUsage     float64 `json:"cpu_usage"`
	RAMUsage     float64 `json:"ram_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	NetRX        uint64  `json:"net_rx"`
	NetTX        uint64  `json:"net_tx"`
	LocalIP      string  `json:"local_ip"`
	PublicIP     string  `json:"public_ip"`
	MachineUUID  string  `json:"machine_uuid"`
	Timestamp    int64   `json:"timestamp"`
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
		RAMUsage:    vMem.UsedPercent,
		DiskUsage:   dUsage.UsedPercent,
		NetRX:       rx,
		NetTX:       tx,
		LocalIP:     getPrimaryLocalIP(),
		PublicIP:    getPublicIP(),
		MachineUUID: machineID,
		Timestamp:   time.Now().Unix(),
	}

	return stats, nil
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
