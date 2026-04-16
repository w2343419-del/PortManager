# Contributing

感谢你为 PortManager 做贡献。

## 开发环境

- Windows 10/11
- Go 1.25.5+

## 本地运行

```powershell
go mod tidy
go test ./...
go build -ldflags "-H=windowsgui" -o PortManager.exe .
```

可直接运行：

```powershell
.\run.bat
```

## 分支与提交建议

- 新功能：`feat/...`
- 修复：`fix/...`
- 文档：`docs/...`
- 清理/重构：`chore/...` 或 `refactor/...`

提交信息建议：

- `feat: ...`
- `fix: ...`
- `docs: ...`
- `chore: ...`

## PR 前检查

请确保以下项通过：

1. `go test ./...`
2. `go build -ldflags "-H=windowsgui" -o PortManager.exe .`
3. 手工验证核心流程：
   - 快速扫描与全面扫描
   - 托盘最小化与恢复
   - 单实例行为（重复启动不产生第二窗口）

## 代码风格

- 保持现有命名和目录结构（`internal/core`、`internal/ui`、`internal/util`）。
- 只改与任务相关的代码，避免无关重排。
- Windows 相关命令建议保持隐藏命令窗口行为。

## 文档更新要求

如果改动涉及行为变化，请同步更新：

- `README.md`
- `RELEASE.md`
