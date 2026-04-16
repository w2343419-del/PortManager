package main

import (
	"log"
	"portmanager/internal/ui"
	"portmanager/internal/util"
)

func main() {
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
