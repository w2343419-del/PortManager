@echo off
setlocal
cd /d "%~dp0"

set "RSRC_EXE="
for /f "delims=" %%i in ('where rsrc.exe 2^>nul') do (
    set "RSRC_EXE=%%i"
    goto :found_rsrc
)

if not defined RSRC_EXE if exist "%GOPATH%\bin\rsrc.exe" set "RSRC_EXE=%GOPATH%\bin\rsrc.exe"

:found_rsrc
if defined RSRC_EXE (
    echo 正在嵌入图标和manifest...
    "%RSRC_EXE%" -manifest PortManager.exe.manifest -ico PortManager.ico -o rsrc_windows_amd64.syso
    if errorlevel 1 (
        echo 警告：资源文件生成失败，继续编译
    )
) else (
    echo 警告：未找到 rsrc.exe，图标资源可能不会更新
)

echo 正在编译应用...
go build -ldflags "-H=windowsgui" -o PortManager.exe .
if errorlevel 1 (
    echo.
    echo 构建失败。
    pause
    exit /b 1
)
echo.
echo 构建完成：PortManager.exe (windowsgui 含图标)
pause
