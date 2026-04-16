package core

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
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

type PortInsight struct {
	Port              int
	Protocol          string
	Process           string
	PID               int32
	Usage             string
	ActiveConnections int
	TrafficLevel      string
	Recommendation    string
	Reason            string
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

var commonPortUsage = map[int]string{
	20: "FTP 数据传输", 21: "FTP 控制通道", 22: "SSH 远程登录", 23: "Telnet 远程终端",
	25: "SMTP 邮件发送", 53: "DNS 解析服务", 67: "DHCP 服务端", 68: "DHCP 客户端",
	69: "TFTP 文件传输", 80: "HTTP Web 服务", 110: "POP3 邮件接收", 123: "NTP 时间同步",
	135: "Windows RPC", 137: "NetBIOS 名称服务", 138: "NetBIOS 数据报", 139: "NetBIOS 会话",
	143: "IMAP 邮件接收", 161: "SNMP 监控", 389: "LDAP 目录服务", 443: "HTTPS 安全 Web",
	445: "SMB 文件共享", 465: "SMTPS 加密邮件", 587: "SMTP 提交", 993: "IMAPS",
	995: "POP3S", 1433: "SQL Server", 1521: "Oracle", 3306: "MySQL", 3389: "RDP 远程桌面",
	5432: "PostgreSQL", 5900: "VNC 远程控制", 6379: "Redis", 7001: "WebLogic",
	8000: "常见开发服务", 8080: "HTTP 备用端口", 8443: "HTTPS 备用端口", 8888: "Jupyter/代理服务",
	9200: "Elasticsearch", 27017: "MongoDB",
}

var highRiskPorts = map[int]struct{}{
	21: {}, 23: {}, 135: {}, 137: {}, 138: {}, 139: {}, 445: {}, 3389: {},
	1433: {}, 1521: {}, 3306: {}, 5432: {}, 6379: {}, 9200: {}, 27017: {},
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

	pidSet := make(map[int32]struct{}, len(entries))
	for _, entry := range entries {
		pidSet[entry.PID] = struct{}{}
	}

	processNames := queryProcessNames(pidSet)

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

		name := processNames[entry.PID]
		if name == "" {
			name = fmt.Sprintf("PID_%d", entry.PID)
		}

		results = append(results, PortInfo{
			Port:     port,
			Process:  name,
			PID:      entry.PID,
			Path:     "",
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

func queryProcessNames(target map[int32]struct{}) map[int32]string {
	result := make(map[int32]string, len(target))
	if len(target) == 0 {
		return result
	}

	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	applyHiddenWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return result
	}

	decoded := strings.TrimSpace(decodeWindowsOutput(output))
	if decoded == "" {
		return result
	}

	r := csv.NewReader(strings.NewReader(decoded))
	for {
		rec, readErr := r.Read()
		if readErr != nil {
			break
		}
		if len(rec) < 2 {
			continue
		}

		pid64, pidErr := strconv.ParseInt(strings.TrimSpace(rec[1]), 10, 32)
		if pidErr != nil {
			continue
		}

		pid := int32(pid64)
		if _, needed := target[pid]; !needed {
			continue
		}

		name := strings.TrimSpace(rec[0])
		if name != "" {
			result[pid] = name
		}
	}

	return result
}

func parseListeningEntries() ([]listeningEntry, error) {
	cmd := exec.Command("netstat", "-ano")
	applyHiddenWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	decoded := decodeWindowsOutput(output)
	lines := strings.Split(strings.ReplaceAll(decoded, "\r", ""), "\n")
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
			if !isListeningState(fields[3]) {
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
	applyHiddenWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	decoded := decodeWindowsOutput(output)
	for _, line := range strings.Split(strings.ReplaceAll(decoded, "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 || strings.ToUpper(fields[0]) != "TCP" {
			continue
		}

		entryPort, ok := extractPort(fields[1])
		if !ok || entryPort != port || !isListeningState(fields[3]) {
			continue
		}

		pidStr := fields[4]
		killCmd := exec.Command("taskkill", "/PID", pidStr, "/F")
		applyHiddenWindow(killCmd)
		if err := killCmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to close port %d", port)
}

func BuildPortInsight(info PortInfo) (PortInsight, error) {
	activeConn, err := countActiveConnections(info.Port, info.PID, info.Protocol)
	if err != nil {
		return PortInsight{}, err
	}

	usage := commonPortUsage[info.Port]
	if usage == "" {
		usage = "自定义/未知服务端口"
	}

	trafficLevel := "低"
	switch {
	case activeConn == 0:
		trafficLevel = "空闲"
	case activeConn <= 5:
		trafficLevel = "低"
	case activeConn <= 20:
		trafficLevel = "中"
	default:
		trafficLevel = "高"
	}

	recommendation, reason := evaluateRecommendation(info, activeConn)

	return PortInsight{
		Port:              info.Port,
		Protocol:          info.Protocol,
		Process:           info.Process,
		PID:               info.PID,
		Usage:             usage,
		ActiveConnections: activeConn,
		TrafficLevel:      trafficLevel,
		Recommendation:    recommendation,
		Reason:            reason,
	}, nil
}

func countActiveConnections(port int, pid int32, protocol string) (int, error) {
	cmd := exec.Command("netstat", "-ano")
	applyHiddenWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	decoded := decodeWindowsOutput(output)
	active := 0
	protoExpected := strings.ToUpper(protocol)
	if protoExpected == "" {
		protoExpected = "TCP"
	}

	for _, line := range strings.Split(strings.ReplaceAll(decoded, "\r", ""), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToUpper(fields[0])
		if proto != protoExpected {
			continue
		}

		var local, pidField, state string
		if proto == "TCP" {
			if len(fields) < 5 {
				continue
			}
			local = fields[1]
			state = fields[3]
			pidField = fields[4]
		} else {
			local = fields[1]
			pidField = fields[len(fields)-1]
		}

		linePort, ok := extractPort(local)
		if !ok || linePort != port {
			continue
		}

		pidValue, pidErr := strconv.ParseInt(pidField, 10, 32)
		if pidErr != nil || int32(pidValue) != pid {
			continue
		}

		if proto == "TCP" {
			st := strings.ToUpper(strings.TrimSpace(state))
			if st == "LISTENING" || st == "LISTEN" || strings.Contains(st, "侦听") {
				continue
			}
		}

		active++
	}

	return active, nil
}

func evaluateRecommendation(info PortInfo, activeConn int) (string, string) {
	proc := strings.ToLower(info.Process)
	if proc == "system" || strings.Contains(proc, "svchost") || strings.Contains(proc, "lsass") {
		return "建议开启", "该端口由系统核心进程占用，关闭可能影响系统功能。"
	}

	if activeConn > 20 {
		return "建议开启", "端口当前活跃连接较多，正在承担业务流量。"
	}

	if _, risky := highRiskPorts[info.Port]; risky && activeConn == 0 {
		return "建议关闭", "该端口属于高风险服务端口且当前空闲，建议关闭以降低暴露面。"
	}

	if activeConn == 0 {
		return "建议关闭", "端口当前无活跃连接，若非明确需要可关闭。"
	}

	if activeConn <= 3 {
		return "无需操作", "端口存在少量连接，建议先观察后再处理。"
	}

	return "建议开启", "端口处于正常使用状态，建议保持开启。"
}

func decodeWindowsOutput(output []byte) string {
	if utf8.Valid(output) {
		return string(output)
	}

	decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(output)
	if err == nil {
		return string(decoded)
	}

	return string(output)
}

func isListeningState(state string) bool {
	state = strings.TrimSpace(strings.ToUpper(state))
	if state == "LISTENING" || state == "LISTEN" {
		return true
	}

	// 中文系统中 netstat 可能显示“侦听”。
	return strings.Contains(state, "侦听")
}

func applyHiddenWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
