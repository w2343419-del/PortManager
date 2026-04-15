package util

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// GetStartupPath 获取启动项注册表路径
func GetStartupPath() string {
	return `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
}

// IsAutoStartupEnabled 检查开机自启动是否启用
func IsAutoStartupEnabled() bool {
	cmd := exec.Command("reg", "query", GetStartupPath(), "/v", "PortManager")
	err := cmd.Run()
	return err == nil
}

// EnableAutoStartup 启用开机自启动
func EnableAutoStartup() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// 使用 reg add 命令添加到注册表
	cmd := exec.Command("reg", "add", GetStartupPath(), "/v", "PortManager", "/t", "REG_SZ", "/d", exePath, "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable auto-startup: %v", err)
	}

	return nil
}

// DisableAutoStartup 禁用开机自启动
func DisableAutoStartup() error {
	cmd := exec.Command("reg", "delete", GetStartupPath(), "/v", "PortManager", "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable auto-startup: %v", err)
	}

	return nil
}
