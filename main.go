package main

import (
	"log"
	"portmanager/internal/ui"
	"portmanager/internal/util"
)

func main() {
	// 检查单实例
	// 如果已有实例在运行，激活现有窗口然后退出
	if util.CheckSingleInstance("Global\\PortManager_SingleInstance_Mutex", "WALK_MAINWINDOW_CLASS_NAME") {
		return
	}
	defer util.ReleaseSingleInstance()

	// 初始化配置
	if err := util.Init(); err != nil {
		log.Fatalf("配置初始化失败: %v\n", err)
	}

	// 创建并运行应用
	app := ui.NewApp()
	if err := app.Run(); err != nil {
		log.Fatalf("应用启动失败: %v\n", err)
	}
}
