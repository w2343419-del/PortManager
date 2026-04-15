package core

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type PortInfo struct {
	Port        int
	Process     string
	PID         int32
	Path        string
	Protocol    string
	State       string // OPEN, CLOSED
	TrafficRate float64
	RiskLevel   int // 0: normal, 1: medium, 2: critical
}

type PortMonitor struct{}

type ScanMode string

const (
	ScanModeQuick ScanMode = "quick"
	ScanModeFull  ScanMode = "full"
)

var quickScanPortSet = map[int]struct{}{
	20: {}, 21: {}, 22: {}, 23: {}, 25: {}, 53: {}, 67: {}, 68: {}, 69: {},
	80: {}, 110: {}, 123: {}, 135: {}, 137: {}, 138: {}, 139: {}, 143: {},
	161: {}, 389: {}, 443: {}, 445: {}, 465: {}, 587: {}, 993: {}, 995: {},
	1433: {}, 1521: {}, 3306: {}, 3389: {}, 5432: {}, 5900: {}, 6379: {},
	7001: {}, 8000: {}, 8080: {}, 8443: {}, 8888: {}, 9200: {}, 27017: {},
}

type listeningEntry struct {
	Protocol string
	Local    string
	PID      int32
}

func NewPortMonitor() *PortMonitor {
	return &PortMonitor{}
}

// ScanPorts 扫描系统当前监听端口
func (pm *PortMonitor) ScanPorts() ([]PortInfo, error) {
	return pm.ScanPortsWithMode(ScanModeFull)
}

func (pm *PortMonitor) ScanPortsWithMode(mode ScanMode) ([]PortInfo, error) {
	entries, err := parseListeningEntries()
	if err != nil {
		return nil, err
	}

	results := make([]PortInfo, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	for _, entry := range entries {
		port, ok := extractPort(entry.Local)
		if !ok {
			continue
		}

		if mode == ScanModeQuick {
			if _, exists := quickScanPortSet[port]; !exists {
				continue
			}
		}

		key := fmt.Sprintf("%s-%d-%d", entry.Protocol, port, entry.PID)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		name, path := pm.getProcessInfo(entry.PID)
		results = append(results, PortInfo{
			Port:     port,
			Process:  name,
			PID:      entry.PID,
			Path:     path,
			Protocol: entry.Protocol,
			State:    "OPEN",
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Port == results[j].Port {
			if results[i].Protocol == results[j].Protocol {
				return results[i].PID < results[j].PID
			}
			return results[i].Protocol < results[j].Protocol
		}
		return results[i].Port < results[j].Port
	})

	return results, nil
}

func (pm *PortMonitor) getProcessInfo(pid int32) (string, string) {
	name := fmt.Sprintf("PID_%d", pid)
	path := ""

	// tasklist 返回 CSV，解析比按空格切割更稳定
	taskCmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	if output, err := taskCmd.Output(); err == nil {
		r := csv.NewReader(strings.NewReader(strings.TrimSpace(string(output))))
		rec, recErr := r.Read()
		if recErr == nil && len(rec) > 0 && !strings.Contains(strings.ToLower(rec[0]), "no tasks") {
			name = strings.TrimSpace(rec[0])
		}
	}

	wmicCmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "ExecutablePath", "/value")
	if output, err := wmicCmd.Output(); err == nil {
		for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r", ""), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "ExecutablePath=") {
				path = strings.TrimSpace(strings.TrimPrefix(line, "ExecutablePath="))
				break
			}
		}
	}

	return name, path
}

func parseListeningEntries() ([]listeningEntry, error) {
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.ReplaceAll(string(output), "\r", ""), "\n")
	entries := make([]listeningEntry, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToUpper(fields[0])
		switch proto {
		case "TCP":
			if len(fields) < 5 {
				continue
			}
			if strings.ToUpper(fields[3]) != "LISTENING" {
				continue
			}
			pid, pidErr := strconv.ParseInt(fields[4], 10, 32)
			if pidErr != nil {
				continue
			}
			entries = append(entries, listeningEntry{
				Protocol: "TCP",
				Local:    fields[1],
				PID:      int32(pid),
			})

		case "UDP":
			// UDP 没有 LISTENING 状态，按已绑定端口处理。
			pid, pidErr := strconv.ParseInt(fields[len(fields)-1], 10, 32)
			if pidErr != nil {
				continue
			}
			entries = append(entries, listeningEntry{
				Protocol: "UDP",
				Local:    fields[1],
				PID:      int32(pid),
			})
		}
	}

	return entries, nil
}

func extractPort(localAddr string) (int, bool) {
	localAddr = strings.TrimSpace(localAddr)
	idx := strings.LastIndex(localAddr, ":")
	if idx == -1 || idx >= len(localAddr)-1 {
		return 0, false
	}

	portStr := localAddr[idx+1:]
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return 0, false
	}

	return port, true
}

// ClosePort 关闭端口
func ClosePort(port int) error {
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	portStr := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, portStr) && strings.Contains(line, "LISTENING") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				pidStr := fields[len(fields)-1]
				killCmd := exec.Command("taskkill", "/PID", pidStr, "/F")
				if err := killCmd.Run(); err == nil {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("failed to close port %d", port)
}
