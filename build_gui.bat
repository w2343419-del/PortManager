@echo off
setlocal
cd /d "%~dp0"
go build -ldflags "-H=windowsgui" -o PortManager.exe .
if errorlevel 1 (
    echo.
    echo 构建失败。
    pause
    exit /b 1
)
echo.
echo 构建完成：PortManager.exe (windowsgui)
