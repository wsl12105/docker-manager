// Package monitor 提供性能监控和告警系统
//
// 该包实现了应用性能监控器，用于实时监控应用性能指标，
// 并在超过阈值时生成告警，包括：
//
// - CPU使用率监控（阈值：30%）
// - 内存使用量监控（阈值：100MB）
// - 响应时间监控（阈值：200ms）
// - API调用频率监控（阈值：100次/分钟）
// - Goroutine数量监控（阈值：50个）
//
// 主要特性：
//   - 实时监控：每5秒收集一次性能指标
//   - 自动告警：超过阈值时自动生成告警
//   - 历史记录：保留最近100个数据点
//   - 告警分级：INFO、WARNING、CRITICAL三个级别
//   - 外部接口：提供RecordAPICall、RecordResponseTime等接口
//
// 使用示例：
//
//	monitor := monitor.NewPerformanceMonitor()
//	defer monitor.Stop()
//
//	// 记录API调用
//	monitor.RecordAPICall()
//
//	// 记录响应时间
//	start := time.Now()
//	// ... 执行操作
//	monitor.RecordResponseTime(time.Since(start))
//
//	// 获取性能指标
//	metrics := monitor.GetMetrics()
//	for name, metric := range metrics {
//	    fmt.Printf("%s: %.2f%s\n", name, metric.Value, metric.Unit)
//	}
//
// 注意事项：
//   - 需要在应用启动时创建监控器
//   - 需要在应用退出时停止监控器（调用Stop()）
//   - CPU使用率估算基于Goroutine、GC暂停时间和内存分配速率
package monitor