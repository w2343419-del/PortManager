# PortManager

PortManager 是一个面向 Windows 的桌面端口管理工具（Go + Walk），用于快速查看监听端口、识别占用进程并执行安全处置。

当前版本：v1.1.0

## 功能概览

- 双模式扫描：快速扫描（速度优先）/ 全面扫描（信息完整）
- 端口列表：端口、协议、进程名、PID
- 端口处置：结束占用端口的进程
- 端口洞察：用途、活跃连接、风险等级、建议动作
- 托盘运行：关闭窗口后最小化到托盘
- 单实例：重复启动只激活已有窗口，不再创建第二个窗口
- 自启动管理：开机自启动启用/禁用

## 系统要求

- Windows 10/11
- 建议管理员权限运行
- Go 1.25.5（源码构建时）

## 快速启动

### 方式一：直接运行

1. 双击 run.bat
2. 允许管理员权限提升
3. 进入主界面后执行扫描和管理

### 方式二：源码构建

```powershell
go mod tidy
go build -ldflags "-H=windowsgui" -o PortManager.exe .
```

也可以双击 build_gui.bat。

## 使用流程

1. 点击 快速扫描 或 全面扫描。
2. 在列表中选择目标端口。
3. 根据需要执行：
   - 关闭端口
   - 右键 查看端口信息与建议

### 扫描模式说明

- 快速扫描：优先返回结果，聚焦常见高风险和常用端口。
- 全面扫描：扫描全部监听端口，并补全更完整的进程信息。
- 当你需要更完整的进程名/上下文时，建议使用全面扫描。

## 单实例与托盘行为

- 首次启动：正常打开主窗口。
- 已有实例时再次启动：仅激活已有窗口。
- 点击窗口关闭按钮：最小化到托盘，不退出进程。
- 托盘左键点击或右键菜单 打开主界面：恢复窗口。

## 构建与验证

```powershell
go test ./...
go build -ldflags "-H=windowsgui" -o PortManager.exe .
```

## 发布文件

发布包建议至少包含：

- PortManager.exe
- PortManager.exe.manifest
- PortManager.ico
- run.bat
- run.vbs
- README.md
- RELEASE.md
- LICENSE

## 参与贡献

提交规范、分支建议和本地验证流程见 CONTRIBUTING.md。

## 项目结构

```text
PortManager/
├─ main.go
├─ build_gui.bat
├─ run.bat
├─ run.vbs
├─ go.mod
├─ internal/
│  ├─ core/
│  ├─ ui/
│  └─ util/
└─ dist/
```

## 许可证

MIT，详见 LICENSE。
