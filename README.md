# PortManager

PortManager 是一个面向 Windows 的桌面端口管理工具（Go + Walk），用于快速查看本机监听端口、识别占用进程并执行安全处置。

当前版本：v1.1.0

## 核心能力

- 双模式扫描：快速扫描（常见高风险与常用端口）/ 全面扫描（全部监听端口）
- 端口列表：端口号、协议、进程名、PID
- 一键关闭：关闭选中端口（本质是结束占用该端口的进程）
- 端口洞察：右键查看用途、活跃连接数、流量等级和处理建议
- 自启动管理：开机自启动启用/禁用

## 运行环境

- Windows 10/11
- 建议管理员权限运行（端口关闭与部分进程信息读取依赖权限）
- Go 1.25.5（源码构建时）

## 快速开始

### 方式 1：直接运行（推荐）

1. 双击 run.bat
2. 脚本会调用 run.vbs 并请求管理员权限
3. 自动启动 PortManager.exe

### 方式 2：源码构建

```powershell
go mod tidy
go build -ldflags "-H=windowsgui" -o PortManager.exe .
```

或双击 build_gui.bat。

### 单实例应用

PortManager 实现了单实例模式，确保同时只有一个应用实例在运行：

- **首次启动**：正常启动应用
- **重复启动**：自动激活现有窗口（如最小化则恢复）
- **收回托盘**：关闭应用窗口时自动最小化到系统托盘
- **从托盘恢复**：左键点击托盘图标或右键菜单 → 打开主界面

> 如果遇到多个PortManager窗口的情况，这是旧版本的行为。更新到最新版本即可解决。

## 使用说明

### 扫描端口

1. 点击 快速扫描 或 全面扫描
2. 等待状态提示完成
3. 在扫描结果列表查看条目

### 关闭端口

1. 在扫描结果中选中一条端口记录
2. 点击 关闭端口
3. 程序执行关闭后会自动刷新结果

### 查看端口信息与建议

1. 在扫描结果中选中一条端口记录
2. 右键点击 查看端口信息与建议
3. 弹窗中可查看：
   - 用途说明
   - 活跃连接数
   - 流量等级
   - 建议动作（建议关闭 / 无需操作 / 建议开启）及原因

## 发布产物建议

发布包至少包含以下文件：

- PortManager.exe（包含嵌入的图标和manifest）
- PortManager.exe.manifest
- PortManager.ico（应用图标）
- run.bat
- run.vbs
- README.md
- RELEASE.md
- LICENSE

> **注**：PortManager.exe 已通过 rsrc 工具嵌入 manifest 和图标资源，无需单独复制相应资源即可正常显示现代控件样式和应用图标。

## 项目结构

```text
PortManager/
├─ main.go
├─ run.bat
├─ run.vbs
├─ build_gui.bat
├─ go.mod
├─ internal/
│  ├─ core/
│  │  └─ port.go
│  ├─ ui/
│  │  └─ app.go
│  └─ util/
│     ├─ config.go
│     └─ startup.go
└─ vendor/
```

## 常见问题

### 程序无法启动

- 优先使用 run.bat 启动
- 确认 PortManager.exe 已构建
- 检查是否被安全软件拦截

### 扫描或关闭失败

- 大多数情况是权限不足，请以管理员身份运行

### 关闭端口导致应用不可用

- 关闭端口会结束对应进程，请确认该进程用途后再操作

### 最小化恢复后窗体位置异常

- v1.1.0 已增加恢复时尺寸与右下角位置自动修正

## 开发与验证

```powershell
go test ./...
go build -ldflags "-H=windowsgui" -o PortManager.exe .
```

## 许可证

MIT，见 LICENSE。
