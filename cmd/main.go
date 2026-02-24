package main

import (
	"log"

	"github.com/wsl12105/docker-manager/internal/docker"
	"github.com/wsl12105/docker-manager/internal/ui"
)

func main() {
	// 创建Docker客户端
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("无法创建Docker客户端: %v", err)
	}
	defer dockerClient.Close()

	// 创建并运行应用
	app := ui.NewApp(dockerClient)
	if err := app.Run(); err != nil {
		log.Fatalf("应用运行失败: %v", err)
	}
}
