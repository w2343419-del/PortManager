package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Config 全局配置
type Config struct {
	ScanInterval      int    `json:"scan_interval"`      // 扫描间隔(秒)
	AlertThreshold    int    `json:"alert_threshold"`    // 告警阈值(MB/s)
	CriticalThreshold int    `json:"critical_threshold"` // 严重告警阈值(MB/s)
	AutoKill          bool   `json:"auto_kill"`          // 自动关闭
	AutoStartup       bool   `json:"auto_startup"`       // 开机自启动
	WhitelistPorts    []int  `json:"whitelist_ports"`    // 白名单端口
	Theme             string `json:"theme"`              // 主题
}

var (
	cfg  *Config
	mu   sync.RWMutex
	path string
)

func DefaultConfig() *Config {
	return &Config{
		ScanInterval:      2,
		AlertThreshold:    50,
		CriticalThreshold: 200,
		AutoKill:          false,
		AutoStartup:       false,
		WhitelistPorts:    []int{},
		Theme:             "light",
	}
}

func Init() error {
	configDir := os.ExpandEnv("${APPDATA}\\PortManager")
	path = filepath.Join(configDir, "config.json")
	os.MkdirAll(configDir, 0755)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg = DefaultConfig()
		return Save()
	}

	return Load()
}

func Load() error {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		cfg = DefaultConfig()
		return nil
	}

	cfg = DefaultConfig()
	return json.Unmarshal(data, cfg)
}

func Save() error {
	mu.RLock()
	defer mu.RUnlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()

	c := *cfg
	if cfg.WhitelistPorts != nil {
		c.WhitelistPorts = make([]int, len(cfg.WhitelistPorts))
		copy(c.WhitelistPorts, cfg.WhitelistPorts)
	}
	return &c
}

func Update(newCfg *Config) error {
	mu.Lock()
	cfg = newCfg
	mu.Unlock()

	return Save()
}
