package util

import (
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	createMutexW     = kernel32.NewProc("CreateMutexW")
	openMutexW       = kernel32.NewProc("OpenMutexW")
	releaseMutex     = kernel32.NewProc("ReleaseMutex")
	closeSemaphore   = kernel32.NewProc("CloseHandle")
	findWindowW      = syscall.NewLazyDLL("user32.dll").NewProc("FindWindowW")
	setForeground    = syscall.NewLazyDLL("user32.dll").NewProc("SetForegroundWindow")
	isIconic         = syscall.NewLazyDLL("user32.dll").NewProc("IsIconic")
	showWindow       = syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow")
)

const SW_RESTORE = 9

var singleInstanceMutex uintptr

// CheckSingleInstance 检查是否已有实例在运行
// 如果是，激活现有窗口并返回true（应该退出当前实例）
// 如果否，获取互斥体并返回false（继续运行）
func CheckSingleInstance(mutexName string, windowClass string) bool {
	// 尝试创建互斥体
	mutexNameUTF16, _ := syscall.UTF16PtrFromString(mutexName)
	
	ret, _, _ := createMutexW.Call(
		0,  // bInitialOwner = false
		1,  // lpMutexAttributes = NULL (safe default)
		uintptr(unsafe.Pointer(mutexNameUTF16)),
	)
	
	if ret == 0 {
		return true // 创建失败，可能已有实例
	}
	
	singleInstanceMutex = ret
	
	// 检查互斥体的最后一个错误
	// ERROR_ALREADY_EXISTS 错误代码是 183
	lastErr := syscall.GetLastError()
	if lastErr == syscall.Errno(183) {
		// 互斥体已存在，说明有实例在运行
		// 尝试激活现有窗口
		if windowClass != "" {
			activateExistingWindow(windowClass)
		}
		
		// 关闭互斥体句柄
		closeSemaphore.Call(ret)
		return true
	}
	
	return false
}

// ReleaseSingleInstance 释放互斥体
func ReleaseSingleInstance() {
	if singleInstanceMutex != 0 {
		releaseMutex.Call(singleInstanceMutex)
		closeSemaphore.Call(singleInstanceMutex)
		singleInstanceMutex = 0
	}
}

func activateExistingWindow(windowClass string) {
	classNameUTF16, _ := syscall.UTF16PtrFromString(windowClass)
	
	// 查找窗口
	ret, _, _ := findWindowW.Call(
		uintptr(unsafe.Pointer(classNameUTF16)),
		0, // lpWindowName = NULL
	)
	
	if ret != 0 {
		hwnd := ret
		
		// 检查窗口是否最小化
		iconic, _, _ := isIconic.Call(hwnd)
		if iconic != 0 {
			// 如果最小化了，恢复窗口
			showWindow.Call(hwnd, SW_RESTORE)
		}
		
		// 将窗口设为前景窗口
		setForeground.Call(hwnd)
	}
}
