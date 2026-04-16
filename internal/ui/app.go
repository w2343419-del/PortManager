package ui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"portmanager/internal/core"
	"portmanager/internal/util"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type App struct {
	monitor *core.PortMonitor

	mu           sync.Mutex
	scanning     bool
	wasMinimized bool
	fixingBounds bool

	mw           *walk.MainWindow
	notifyIcon   *walk.NotifyIcon
	appIcon      *walk.Icon
	titleLabel   *walk.Label
	statusLabel  *walk.Label
	countLabel   *walk.Label
	results      *walk.ListBox
	startupCheck *walk.CheckBox
	portItems    []core.PortInfo
	exiting      bool
}

func NewApp() *App {
	return &App{
		monitor: core.NewPortMonitor(),
	}
}

func (a *App) Run() error {
	const cardWidth = 460
	const cardHeight = 560
	bodyFontFamily := chooseBodyFontFamily()
	headingFontFamily := chooseHeadingFontFamily()

	pageBg := walk.RGB(241, 243, 246)
	cardBg := walk.RGB(250, 251, 253)
	headingColor := walk.RGB(22, 27, 34)
	mutedColor := walk.RGB(108, 117, 125)
	accentColor := walk.RGB(0, 122, 204)

	window := MainWindow{
		AssignTo: &a.mw,
		Title:    "PortManager",
		Icon:     "PortManager.ico",
		Font:     Font{Family: bodyFontFamily, PointSize: 10},
		OnSizeChanged: func() {
			a.handleWindowSizeChanged(cardWidth, cardHeight)
		},
		Size:       Size{Width: cardWidth, Height: cardHeight},
		MinSize:    Size{Width: cardWidth, Height: cardHeight},
		MaxSize:    Size{Width: cardWidth, Height: cardHeight},
		Visible:    false,
		Background: SolidColorBrush{Color: pageBg},
		Layout:     VBox{Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Composite{
				Background: SolidColorBrush{Color: cardBg},
				Layout:     HBox{Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 12}},
				Children: []Widget{
					Label{AssignTo: &a.titleLabel, Text: "PortManager", Font: Font{Family: headingFontFamily, PointSize: 18}, TextColor: headingColor},
					HSpacer{},
					Label{Text: "v1.1.0", Font: Font{Family: bodyFontFamily, PointSize: 9}, TextColor: mutedColor},
				},
			},
			Composite{
				Background: SolidColorBrush{Color: cardBg},
				Layout:     VBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 10}, Spacing: 8},
				Children: []Widget{
					Label{Text: "端口扫描", Font: Font{Family: headingFontFamily, PointSize: 12}, TextColor: headingColor},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							PushButton{
								Text:    "快速扫描",
								Font:    Font{Family: bodyFontFamily, PointSize: 10},
								MinSize: Size{Width: 126, Height: 36},
								OnClicked: func() {
									go a.scanPorts(core.ScanModeQuick)
								},
							},
							PushButton{
								Text:    "全面扫描",
								Font:    Font{Family: bodyFontFamily, PointSize: 10},
								MinSize: Size{Width: 126, Height: 36},
								OnClicked: func() {
									go a.scanPorts(core.ScanModeFull)
								},
							},
							PushButton{
								Text:    "关闭端口",
								Font:    Font{Family: bodyFontFamily, PointSize: 10},
								MinSize: Size{Width: 126, Height: 36},
								OnClicked: func() {
									a.closeSelectedPort()
								},
							},
						},
					},
					Label{AssignTo: &a.statusLabel, Text: "准备就绪", Font: Font{Family: bodyFontFamily, PointSize: 10}, TextColor: accentColor},
				},
			},
			Composite{
				Background:    SolidColorBrush{Color: cardBg},
				Layout:        VBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 10}, Spacing: 8},
				StretchFactor: 1,
				Children: []Widget{
					Composite{
						Layout: HBox{},
						Children: []Widget{
							Label{Text: "扫描结果", Font: Font{Family: headingFontFamily, PointSize: 12}, TextColor: headingColor},
							HSpacer{},
							Label{AssignTo: &a.countLabel, Text: "0", Font: Font{Family: headingFontFamily, PointSize: 16}, TextColor: headingColor},
						},
					},
					ListBox{
						AssignTo:      &a.results,
						Font:          Font{Family: bodyFontFamily, PointSize: 10},
						MinSize:       Size{Height: 200},
						Model:         []string{},
						StretchFactor: 1,
						ContextMenuItems: []MenuItem{
							Action{Text: "查看端口信息与建议", OnTriggered: func() { a.showSelectedPortInsight() }},
						},
					},
					Label{Text: "提示: 先选中条目，再执行关闭或右键分析", Font: Font{Family: bodyFontFamily, PointSize: 9}, TextColor: mutedColor},
				},
			},
			Composite{
				Background: SolidColorBrush{Color: cardBg},
				Layout:     VBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 10}, Spacing: 8},
				Children: []Widget{
					Label{Text: "设置", Font: Font{Family: headingFontFamily, PointSize: 12}, TextColor: headingColor},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							CheckBox{
								AssignTo: &a.startupCheck,
								Text:     "开机自启动",
								Font:     Font{Family: bodyFontFamily, PointSize: 10},
								Checked:  util.IsAutoStartupEnabled(),
								OnCheckedChanged: func() {
									go a.toggleStartup(a.startupCheck.Checked())
								},
							},
							HSpacer{},
							PushButton{Text: "关于", Font: Font{Family: bodyFontFamily, PointSize: 10}, MinSize: Size{Width: 92, Height: 32}, OnClicked: func() { a.showAbout() }},
							PushButton{
								Text:    "退出",
								Font:    Font{Family: bodyFontFamily, PointSize: 10},
								MinSize: Size{Width: 92, Height: 32},
								OnClicked: func() {
									a.exitApp()
								},
							},
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							PushButton{Text: "刷新", Font: Font{Family: bodyFontFamily, PointSize: 10}, MinSize: Size{Width: 92, Height: 32}, OnClicked: func() { go a.scanPorts(core.ScanModeQuick) }},
							HSpacer{},
						},
					},
				},
			},
		},
	}

	if err := window.Create(); err != nil {
		return err
	}

	// Ensure both window icon and tray icon use the same app icon.
	a.loadAppIcon()
	if a.appIcon != nil {
		_ = a.mw.SetIcon(a.appIcon)
	}

	if err := a.initTray(); err != nil {
		return err
	}

	a.mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if a.exiting {
			return
		}

		*canceled = true
		a.minimizeToTray()
	})

	if err := a.positionBottomRight(); err != nil {
		return err
	}

	a.mw.Show()
	// Re-align once after show so final non-client size and DPI scaling are reflected.
	_ = a.positionBottomRight()
	a.mw.Run()
	a.disposeTray()
	if a.appIcon != nil {
		a.appIcon.Dispose()
		a.appIcon = nil
	}

	return nil
}

func (a *App) initTray() error {
	if a.mw == nil {
		return nil
	}

	ni, err := walk.NewNotifyIcon(a.mw)
	if err != nil {
		return err
	}

	a.notifyIcon = ni
	_ = ni.SetToolTip("PortManager")

	if a.appIcon != nil {
		_ = ni.SetIcon(a.appIcon)
	}

	openAction := walk.NewAction()
	openAction.SetText("打开主界面")
	openAction.Triggered().Attach(func() {
		a.showFromTray()
	})

	exitAction := walk.NewAction()
	exitAction.SetText("退出")
	exitAction.Triggered().Attach(func() {
		a.exitApp()
	})

	if err := ni.ContextMenu().Actions().Add(openAction); err != nil {
		return err
	}
	if err := ni.ContextMenu().Actions().Add(walk.NewSeparatorAction()); err != nil {
		return err
	}
	if err := ni.ContextMenu().Actions().Add(exitAction); err != nil {
		return err
	}

	ni.MouseUp().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			a.showFromTray()
		}
	})

	return ni.SetVisible(true)
}

func (a *App) minimizeToTray() {
	if a.mw == nil {
		return
	}

	a.mw.SetVisible(false)
	if a.notifyIcon != nil {
		_ = a.notifyIcon.SetVisible(true)
	}
}

func (a *App) showFromTray() {
	if a.mw == nil {
		return
	}

	a.mw.Show()
	_ = a.positionBottomRight()
	if a.statusLabel != nil {
		a.statusLabel.SetText("已从托盘恢复")
	}
}

func (a *App) disposeTray() {
	if a.notifyIcon != nil {
		_ = a.notifyIcon.Dispose()
		a.notifyIcon = nil
	}
}

func (a *App) loadAppIcon() {
	if a.appIcon != nil {
		return
	}

	var candidates []string
	if exePath, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "PortManager.ico"))
		candidates = append(candidates, exePath)
	}
	candidates = append(candidates, "PortManager.ico")

	for _, path := range candidates {
		icon, err := walk.NewIconFromFile(path)
		if err == nil {
			a.appIcon = icon
			return
		}
	}
}

func (a *App) exitApp() {
	if !a.confirmExit() {
		return
	}

	a.exiting = true
	a.disposeTray()
	if a.mw != nil {
		a.mw.Close()
	}
}

func (a *App) handleWindowSizeChanged(targetWidth, targetHeight int) {
	if a.mw == nil || a.fixingBounds {
		return
	}

	if win.IsIconic(a.mw.Handle()) {
		a.wasMinimized = true
		return
	}

	if !a.wasMinimized {
		return
	}

	a.wasMinimized = false
	a.fixingBounds = true
	defer func() {
		a.fixingBounds = false
	}()

	b := a.mw.BoundsPixels()
	if b.Width != targetWidth || b.Height != targetHeight {
		_ = a.mw.SetBoundsPixels(walk.Rectangle{X: b.X, Y: b.Y, Width: targetWidth, Height: targetHeight})
	}

	_ = a.positionBottomRight()
}

func (a *App) positionBottomRight() error {
	if a.mw == nil {
		return nil
	}

	bounds := a.mw.BoundsPixels()
	width := bounds.Width
	height := bounds.Height
	if width <= 0 || height <= 0 {
		s := a.mw.SizePixels()
		width = s.Width
		height = s.Height
	}

	workArea := win.RECT{
		Left:   0,
		Top:    0,
		Right:  win.GetSystemMetrics(win.SM_CXSCREEN),
		Bottom: win.GetSystemMetrics(win.SM_CYSCREEN),
	}

	monitor := win.MonitorFromWindow(a.mw.Handle(), win.MONITOR_DEFAULTTONEAREST)
	if monitor != 0 {
		mi := win.MONITORINFO{CbSize: uint32(unsafe.Sizeof(win.MONITORINFO{}))}
		if win.GetMonitorInfo(monitor, &mi) {
			workArea = mi.RcWork
		}
	}

	margin := 14
	x := int(workArea.Right) - width - margin
	y := int(workArea.Bottom) - height - margin
	if x < int(workArea.Left) {
		x = int(workArea.Left)
	}
	if y < int(workArea.Top) {
		y = int(workArea.Top)
	}

	return a.mw.SetBoundsPixels(walk.Rectangle{X: x, Y: y, Width: width, Height: height})
}

func chooseBodyFontFamily() string {
	preferred := []string{
		"Microsoft YaHei UI",
		"Segoe UI",
		"Segoe UI Variable Text",
		"Inter",
	}

	installed := readInstalledFontNames()
	for _, family := range preferred {
		if fontNameExists(installed, family) {
			return family
		}
	}

	return "Segoe UI"
}

func chooseHeadingFontFamily() string {
	preferred := []string{
		"Anthropic Serif",
		"Georgia",
		"Times New Roman",
	}

	installed := readInstalledFontNames()
	for _, family := range preferred {
		if fontNameExists(installed, family) {
			return family
		}
	}

	return "Georgia"
}

func readInstalledFontNames() []string {
	keys := []string{
		`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`,
		`HKCU\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`,
	}
	result := make([]string, 0, 512)

	for _, key := range keys {
		cmd := exec.Command("reg", "query", key)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r", ""), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(strings.ToUpper(line), "HKEY_") {
				continue
			}

			parts := strings.SplitN(line, "REG_", 2)
			if len(parts) == 0 {
				continue
			}

			name := strings.TrimSpace(parts[0])
			if name != "" {
				result = append(result, name)
			}
		}
	}

	return result
}

func fontNameExists(installed []string, family string) bool {
	f := strings.ToLower(family)
	for _, n := range installed {
		if strings.Contains(strings.ToLower(n), f) {
			return true
		}
	}

	return false
}

func (a *App) scanPorts(mode core.ScanMode) {
	startAt := time.Now()

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
		a.statusLabel.SetText(fmt.Sprintf("%s完成，发现 %d 个开放端口，耗时 %d ms", modeLabel, len(ports), time.Since(startAt).Milliseconds()))
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

func (a *App) showSelectedPortInsight() {
	if a.results == nil {
		return
	}

	index := a.results.CurrentIndex()
	if index < 0 || index >= len(a.portItems) {
		walk.MsgBox(a.mw, "提示", "请先选中一个端口", walk.MsgBoxIconInformation)
		return
	}

	port := a.portItems[index]
	insight, err := core.BuildPortInsight(port)
	if err != nil {
		walk.MsgBox(a.mw, "端口分析失败", err.Error(), walk.MsgBoxIconError)
		return
	}

	lines := []string{
		fmt.Sprintf("端口: %d", insight.Port),
		fmt.Sprintf("协议: %s", insight.Protocol),
		fmt.Sprintf("进程: %s", insight.Process),
		fmt.Sprintf("PID: %d", insight.PID),
		"",
		fmt.Sprintf("用途: %s", insight.Usage),
		fmt.Sprintf("流量评估: %s（活跃连接 %d）", insight.TrafficLevel, insight.ActiveConnections),
		"",
		fmt.Sprintf("推荐: %s", insight.Recommendation),
		fmt.Sprintf("依据: %s", insight.Reason),
	}

	icon := walk.MsgBoxIconInformation
	if insight.Recommendation == "建议关闭" {
		icon = walk.MsgBoxIconWarning
	}

	walk.MsgBox(a.mw, fmt.Sprintf("端口 %d 分析", insight.Port), strings.Join(lines, "\n"), icon)
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
		"- 快速扫描 / 全面扫描",
		"- 查看占用进程与 PID",
		"- 一键关闭端口",
		"- 右键查看端口信息与建议",
		"- 开机自启动",
		"",
		"当前版本：v1.1.0",
	}, "\n"), walk.MsgBoxIconInformation)
}

func (a *App) confirmExit() bool {
	return walk.MsgBox(a.mw, "退出确认", "确定要退出 PortManager 吗？", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes
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
	parts := []string{}
	if port.Protocol != "" {
		parts = append(parts, port.Protocol)
	}
	parts = append(parts, fmt.Sprintf("端口 %d", port.Port))
	if port.Process != "" {
		parts = append(parts, port.Process)
	}
	if port.PID != 0 {
		parts = append(parts, fmt.Sprintf("PID %d", port.PID))
	}
	return strings.Join(parts, " | ")
}
