// Package ui 提供Docker Manager的终端用户界面组件
//
// 该包实现了基于tview库的TUI（Terminal User Interface）应用，
// 提供容器和镜像的管理界面，包括：
//
// - 容器列表显示和管理（启动、停止、删除、查看日志、执行命令等）
// - 镜像列表显示和管理（删除、添加标签等）
// - 实时性能监控（CPU、内存使用率）
// - 智能刷新策略（根据用户操作动态调整刷新频率）
//
// 主要组件：
//   - App: 应用主结构，协调各个UI组件
//   - ContainerUI: 容器管理界面
//   - ImageUI: 镜像管理界面
//   - Common: 公共UI组件（表格、页面、对话框等）
//
// 性能优化特性：
//   - 增量更新：只更新变化的容器，避免全量刷新
//   - LRU缓存：缓存容器Stats数据，减少API调用
//   - 对象池：重用TableCell对象，减少内存分配
//   - 防抖机制：避免快速切换时的性能问题
//   - 智能刷新：根据用户状态动态调整刷新频率
//
// 使用示例：
//
//	dockerClient, err := docker.NewClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	app := ui.NewApp(dockerClient)
//	if err := app.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// 注意事项：
//   - 需要在主线程中运行UI应用
//   - 需要正确清理资源（调用app.Stop()）
//   - 并发操作需要使用原子操作或锁保护
package ui