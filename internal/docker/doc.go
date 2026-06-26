// Package docker 提供Docker API客户端封装
//
// 该包封装了Docker官方SDK，提供简化的API接口，
// 用于管理容器和镜像，包括：
//
// - 容器管理：列出、启动、停止、删除、查看日志、执行命令、查看详情
// - 镜像管理：列出、删除、添加标签
// - 性能监控：获取容器实时Stats（CPU、内存）
//
// 主要特性：
//   - Context超时控制：所有API调用都有超时限制
//   - 批量Stats获取：并发获取多个容器的Stats数据
//   - 动态并发池：根据容器数量动态调整并发池大小
//   - 结构体解码：使用结构体代替map解码JSON，提高性能
//   - 错误处理：记录批量操作中的错误，不中断流程
//
// 使用示例：
//
//	client, err := docker.NewClient()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	containers, err := client.ListContainers(true)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	stats := client.BatchGetStats(containerIDs)
//	for id, stat := range stats.Stats {
//	    fmt.Printf("%s: CPU=%s, Memory=%s\n", id, stat.CPU, stat.Memory)
//	}
//
// 注意事项：
//   - 需要确保Docker服务正在运行
//   - 批量操作可能失败，需要检查Errors字段
//   - Stats数据有缓存有效期（5秒）
package docker