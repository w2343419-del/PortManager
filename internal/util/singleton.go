package util

import (
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	createMutexW     = kernel32.NewProc("CreateMutexW")
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

	ret, _, callErr := createMutexW.Call(
		0, // lpMutexAttributes = NULL
		0, // bInitialOwner = FALSE
		uintptr(unsafe.Pointer(mutexNameUTF16)),
	)

	if ret == 0 {
		// 访问被拒绝通常意味着同名互斥体已存在（不同权限级别下常见），
		// 这里按“已有实例”处理，避免继续启动第二个实例。
		if errno, ok := callErr.(syscall.Errno); ok && errno == syscall.ERROR_ACCESS_DENIED {
			if windowClass != "" {
				activateExistingWindow(windowClass)
			}
			return true
		}
		return true
	}

	singleInstanceMutex = ret

	// 通过CreateMutexW的返回错误判断是否已存在，避免GetLastError不稳定问题。
	if errno, ok := callErr.(syscall.Errno); ok && errno == syscall.ERROR_ALREADY_EXISTS {
		// 互斥体已存在，说明有实例在运行
		// 尝试激活现有窗口
		if windowClass != "" {
			activateExistingWindow(windowClass)
		}

		// 关闭互斥体句柄
		closeSemaphore.Call(ret)
		singleInstanceMutex = 0
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
