package ui

import (
	"fmt"
	"log"
	"portmanager/internal/core"
	"portmanager/internal/util"
	"strings"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type App struct {
	monitor *core.PortMonitor

	mu       sync.Mutex
	scanning bool

	mw           *walk.MainWindow
	statusLabel  *walk.Label
	countLabel   *walk.Label
	results      *walk.ListBox
	startupCheck *walk.CheckBox
	portItems    []core.PortInfo
}

func NewApp() *App {
	return &App{
		monitor: core.NewPortMonitor(),
	}
}

func (a *App) Run() error {
	window := MainWindow{
		AssignTo: &a.mw,
		Title:    "PortManager",
		MinSize:  Size{Width: 1220, Height: 780},
		Layout:   VBox{MarginsZero: false, SpacingZero: false},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 1},
				Children: []Widget{
					Label{Text: "PortManager", Font: Font{Bold: true, PointSize: 18}},
					Label{Text: "端口检测管理工具"},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					GroupBox{
						Title:  "🔍 端口扫描",
						Layout: VBox{},
						Children: []Widget{
							Composite{
								Layout: HBox{},
								Children: []Widget{
									PushButton{
										Text: "快速扫描",
										OnClicked: func() {
											go a.scanPorts(core.ScanModeQuick)
										},
									},
									PushButton{
										Text: "全面扫描",
										OnClicked: func() {
											go a.scanPorts(core.ScanModeFull)
										},
									},
								},
							},
							Label{AssignTo: &a.statusLabel, Text: "准备就绪"},
							Label{Text: "快速扫描: 常用/高风险端口 | 全面扫描: 全部监听端口"},
						},
					},
					GroupBox{
						Title:  "📊 扫描结果",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "发现端口数", Font: Font{Bold: true}},
							Label{AssignTo: &a.countLabel, Text: "0", Font: Font{Bold: true, PointSize: 18}},
							ListBox{AssignTo: &a.results, Model: []string{}, StretchFactor: 1},
							PushButton{
								Text: "关闭选中端口",
								OnClicked: func() {
									a.closeSelectedPort()
								},
							},
						},
					},
					GroupBox{
						Title:  "⚙️ 设置",
						Layout: VBox{},
						Children: []Widget{
							CheckBox{
								AssignTo: &a.startupCheck,
								Text:     "开机自启动",
								Checked:  util.IsAutoStartupEnabled(),
								OnCheckedChanged: func() {
									go a.toggleStartup(a.startupCheck.Checked())
								},
							},
							PushButton{
								Text: "关于",
								OnClicked: func() {
									a.showAbout()
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := window.Run(); err != nil {
		return err
	}

	return nil
}

func (a *App) scanPorts(mode core.ScanMode) {
	a.mu.Lock()
	if a.scanning {
		a.mu.Unlock()
		a.runOnUI(func() {
			a.statusLabel.SetText("扫描正在进行中")
		})
		return
	}
	a.scanning = true
	a.mu.Unlock()

	modeLabel := "全面扫描"
	if mode == core.ScanModeQuick {
		modeLabel = "快速扫描"
	}

	a.runOnUI(func() {
		a.statusLabel.SetText(fmt.Sprintf("正在%s，请稍候...", modeLabel))
	})

	ports, err := a.monitor.ScanPortsWithMode(mode)
	if err != nil {
		a.runOnUI(func() {
			a.statusLabel.SetText(fmt.Sprintf("扫描失败：%v", err))
			walk.MsgBox(a.mw, "扫描失败", err.Error(), walk.MsgBoxIconError)
		})
		a.finishScan()
		return
	}

	a.runOnUI(func() {
		a.refreshResults(ports)
		a.statusLabel.SetText(fmt.Sprintf("%s完成，发现 %d 个开放端口", modeLabel, len(ports)))
	})

	a.finishScan()
}

func (a *App) refreshResults(ports []core.PortInfo) {
	a.portItems = append(a.portItems[:0], ports...)
	a.countLabel.SetText(fmt.Sprintf("%d", len(ports)))

	items := make([]string, 0, len(ports))
	for _, port := range ports {
		items = append(items, buildPortSummary(port))
	}
	if a.results != nil {
		if err := a.results.SetModel(items); err != nil {
			log.Printf("set list model: %v", err)
		}
		if len(items) > 0 {
			_ = a.results.SetCurrentIndex(0)
		} else {
			_ = a.results.SetCurrentIndex(-1)
		}
	}
}

func (a *App) closeSelectedPort() {
	if a.results == nil {
		return
	}

	index := a.results.CurrentIndex()
	if index < 0 || index >= len(a.portItems) {
		walk.MsgBox(a.mw, "提示", "请先选中一个端口", walk.MsgBoxIconInformation)
		return
	}

	port := a.portItems[index].Port
	go a.closePort(port)
}

func (a *App) closePort(port int) {
	a.runOnUI(func() {
		a.statusLabel.SetText(fmt.Sprintf("正在关闭端口 %d...", port))
	})

	if err := core.ClosePort(port); err != nil {
		a.runOnUI(func() {
			a.statusLabel.SetText(fmt.Sprintf("关闭失败：%v", err))
			walk.MsgBox(a.mw, "关闭失败", err.Error(), walk.MsgBoxIconError)
		})
		return
	}

	a.runOnUI(func() {
		a.statusLabel.SetText(fmt.Sprintf("端口 %d 已关闭", port))
	})

	go a.scanPorts(core.ScanModeFull)
}

func (a *App) toggleStartup(enabled bool) {
	var err error
	if enabled {
		err = util.EnableAutoStartup()
	} else {
		err = util.DisableAutoStartup()
	}

	if err != nil {
		a.runOnUI(func() {
			a.statusLabel.SetText(fmt.Sprintf("更新自启动失败：%v", err))
			if a.startupCheck != nil {
				a.startupCheck.SetChecked(!enabled)
			}
			walk.MsgBox(a.mw, "更新失败", err.Error(), walk.MsgBoxIconError)
		})
		return
	}

	cfg := util.Get()
	cfg.AutoStartup = enabled
	if saveErr := util.Update(cfg); saveErr != nil {
		log.Printf("update config: %v", saveErr)
	}

	a.runOnUI(func() {
		if enabled {
			a.statusLabel.SetText("已启用开机自启动")
		} else {
			a.statusLabel.SetText("已关闭开机自启动")
		}
	})
}

func (a *App) showAbout() {
	walk.MsgBox(a.mw, "关于 PortManager", strings.Join([]string{
		"PortManager 是一个 Windows 端口检测与管理工具。",
		"",
		"特性：",
		"- 扫描常用端口",
		"- 查看占用进程",
		"- 一键关闭端口",
		"- 开机自启动",
		"",
		"当前版本：v1.0.0",
	}, "\n"), walk.MsgBoxIconInformation)
}

func (a *App) finishScan() {
	a.mu.Lock()
	a.scanning = false
	a.mu.Unlock()
}

func (a *App) runOnUI(fn func()) {
	if a.mw == nil {
		fn()
		return
	}

	a.mw.Synchronize(fn)
}

func buildPortSummary(port core.PortInfo) string {
	parts := []string{fmt.Sprintf("端口 %d", port.Port)}
	if port.Process != "" {
		parts = append(parts, port.Process)
	}
	if port.PID != 0 {
		parts = append(parts, fmt.Sprintf("PID %d", port.PID))
	}
	return strings.Join(parts, " | ")
}
