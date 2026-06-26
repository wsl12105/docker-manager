package monitor

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo    AlertLevel = "INFO"
	AlertLevelWarning AlertLevel = "WARNING"
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// Metric 性能指标
type Metric struct {
	Name      string    // 指标名称
	Value     float64   // 当前值
	Threshold float64   // 阈值
	Unit      string    // 单位
	Timestamp time.Time // 时间戳
	History   []float64 // 历史记录（最近100个数据点）
}

// Alert 告警
type Alert struct {
	Level     AlertLevel // 告警级别
	Metric    string     // 指标名称
	Value     float64    // 当前值
	Threshold float64    // 阈值
	Message   string     // 告警消息
	Timestamp time.Time  // 时间戳
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	metrics    map[string]*Metric // 性能指标映射
	alerts     []Alert            // 告警列表
	alertsLock sync.Mutex        // 告警列表锁
	stopChan   chan struct{}     // 停止信号通道
	running    bool               // 是否正在运行
	runningLock sync.Mutex       // 运行状态锁

	// API调用计数器
	apiCallCount int64       // API调用计数
	apiCallLock  sync.Mutex  // API调用锁
	lastAPICheck time.Time   // 上次API调用检查时间
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor() *PerformanceMonitor {
	pm := &PerformanceMonitor{
		metrics:      make(map[string]*Metric),
		alerts:       make([]Alert, 0),
		stopChan:     make(chan struct{}),
		lastAPICheck: time.Now(),
	}

	// 初始化性能指标
	pm.initMetrics()

	// 启动监控
	pm.startMonitoring()

	return pm
}

// initMetrics 初始化性能指标
func (pm *PerformanceMonitor) initMetrics() {
	// CPU使用率，阈值30%
	pm.metrics["cpu_usage"] = &Metric{
		Name:      "CPU Usage",
		Value:     0,
		Threshold: 30.0,
		Unit:      "%",
		Timestamp: time.Now(),
		History:   make([]float64, 0, 100),
	}

	// 内存使用量，阈值100MB
	pm.metrics["memory_usage"] = &Metric{
		Name:      "Memory Usage",
		Value:     0,
		Threshold: 100.0, // 100MB
		Unit:      "MB",
		Timestamp: time.Now(),
		History:   make([]float64, 0, 100),
	}

	// 响应时间，阈值200ms
	pm.metrics["response_time"] = &Metric{
		Name:      "Response Time",
		Value:     0,
		Threshold: 200.0,
		Unit:      "ms",
		Timestamp: time.Now(),
		History:   make([]float64, 0, 100),
	}

	// API调用次数，阈值100次/分钟
	pm.metrics["api_calls"] = &Metric{
		Name:      "API Calls/min",
		Value:     0,
		Threshold: 100.0,
		Unit:      "calls/min",
		Timestamp: time.Now(),
		History:   make([]float64, 0, 100),
	}

	// Goroutine数量，阈值50个
	pm.metrics["goroutine_count"] = &Metric{
		Name:      "Goroutine Count",
		Value:     0,
		Threshold: 50.0,
		Unit:      "count",
		Timestamp: time.Now(),
		History:   make([]float64, 0, 100),
	}
}

// startMonitoring 启动监控循环（5秒间隔）
func (pm *PerformanceMonitor) startMonitoring() {
	pm.runningLock.Lock()
	defer pm.runningLock.Unlock()

	if pm.running {
		return
	}

	pm.running = true

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pm.collectMetrics()
				pm.checkThresholds()
			case <-pm.stopChan:
				return
			}
		}
	}()
}

// collectMetrics 收集性能指标
func (pm *PerformanceMonitor) collectMetrics() {
	// 收集CPU使用率
	cpuUsage := pm.getCPUUsage()
	pm.updateMetric("cpu_usage", cpuUsage)

	// 收集内存使用量
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryMB := float64(memStats.Alloc) / 1024 / 1024 // 转换为MB
	pm.updateMetric("memory_usage", memoryMB)

	// 收集Goroutine数量
	goroutineCount := float64(runtime.NumGoroutine())
	pm.updateMetric("goroutine_count", goroutineCount)

	// 收集API调用次数（每分钟）
	pm.apiCallLock.Lock()
	now := time.Now()
	elapsed := now.Sub(pm.lastAPICheck).Minutes()
	if elapsed >= 1.0 {
		// 每分钟重置计数
		apiCallsPerMin := float64(pm.apiCallCount)
		pm.updateMetric("api_calls", apiCallsPerMin)
		pm.apiCallCount = 0
		pm.lastAPICheck = now
	}
	pm.apiCallLock.Unlock()
}

// updateMetric 更新指标值和历史记录
func (pm *PerformanceMonitor) updateMetric(metricName string, value float64) {
	metric, exists := pm.metrics[metricName]
	if !exists {
		return
	}

	metric.Value = value
	metric.Timestamp = time.Now()

	// 更新历史记录（保留最近100个数据点）
	metric.History = append(metric.History, value)
	if len(metric.History) > 100 {
		metric.History = metric.History[1:]
	}
}

// checkThresholds 检查阈值并生成告警
func (pm *PerformanceMonitor) checkThresholds() {
	for _, metric := range pm.metrics {
		if metric.Value > metric.Threshold {
			// 超过阈值，生成告警
			alertLevel := pm.getAlertLevel(metric.Value, metric.Threshold)
			alert := Alert{
				Level:     alertLevel,
				Metric:    metric.Name,
				Value:     metric.Value,
				Threshold: metric.Threshold,
				Message:   fmt.Sprintf("%s: %.2f%s (threshold: %.2f%s)", metric.Name, metric.Value, metric.Unit, metric.Threshold, metric.Unit),
				Timestamp: time.Now(),
			}

			pm.alertsLock.Lock()
			pm.alerts = append(pm.alerts, alert)
			// 只保留最近100条告警
			if len(pm.alerts) > 100 {
				pm.alerts = pm.alerts[1:]
			}
			pm.alertsLock.Unlock()

			// 输出告警日志
			fmt.Printf("[%s] %s\n", alertLevel, alert.Message)
		}
	}
}

// getAlertLevel 根据超过程度确定告警级别
func (pm *PerformanceMonitor) getAlertLevel(value, threshold float64) AlertLevel {
	overPercent := (value - threshold) / threshold * 100

	if overPercent >= 100 {
		return AlertLevelCritical
	} else if overPercent >= 50 {
		return AlertLevelWarning
	} else {
		return AlertLevelInfo
	}
}

// getCPUUsage 获取应用负载指标（改进的估算方法）
func (pm *PerformanceMonitor) getCPUUsage() float64 {
	// 使用更准确的应用负载估算方法，结合多个指标：
	// 1. Goroutine数量（调度负载）
	// 2. GC暂停时间（CPU消耗）
	// 3. 内存分配速率（处理负载）

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 1. Goroutine调度负载（权重：40%）
	goroutineCount := runtime.NumGoroutine()
	// 假设正常情况下20个Goroutine为基准，超过则增加负载
	goroutineLoad := float64(goroutineCount) / 20.0 * 40.0
	if goroutineLoad > 40 {
		goroutineLoad = 40
	}

	// 2. GC暂停时间负载（权重：30%）
	// 计算最近GC暂停时间的平均值（转换为百分比）
	// 修复：防止除零错误（NumGC可能为0）
	gcLoad := 0.0
	if memStats.NumGC > 0 {
		gcPauseAvg := float64(memStats.PauseTotalNs) / float64(memStats.NumGC) / 1e6 // 转换为毫秒
		// 假设1ms GC暂停为基准，超过则增加负载
		gcLoad = gcPauseAvg / 1.0 * 30.0
		if gcLoad > 30 {
			gcLoad = 30
		}
	}

	// 3. 内存分配速率负载（权重：30%）
	// 计算内存分配量（MB）
	// 修复：使用Alloc（当前分配量）而不是TotalAlloc（累计分配量）
	allocMB := float64(memStats.Alloc) / 1024 / 1024 // 转换为MB
	// 假设50MB分配为基准，超过则增加负载
	allocLoad := allocMB / 50.0 * 30.0
	if allocLoad > 30 {
		allocLoad = 30
	}

	// 综合负载指标（0-100）
	totalLoad := goroutineLoad + gcLoad + allocLoad

	// 限制最大值为100%
	if totalLoad > 100 {
		totalLoad = 100
	}

	return totalLoad
}

// RecordAPICall 记录API调用（供外部调用）
func (pm *PerformanceMonitor) RecordAPICall() {
	pm.apiCallLock.Lock()
	defer pm.apiCallLock.Unlock()
	pm.apiCallCount++
}

// RecordResponseTime 记录响应时间（供外部调用）
func (pm *PerformanceMonitor) RecordResponseTime(duration time.Duration) {
	responseTimeMS := float64(duration.Milliseconds())
	pm.updateMetric("response_time", responseTimeMS)
}

// GetMetrics 获取当前性能指标（供外部查询）
func (pm *PerformanceMonitor) GetMetrics() map[string]*Metric {
	return pm.metrics
}

// GetAlerts 获取告警列表（供外部查询）
func (pm *PerformanceMonitor) GetAlerts() []Alert {
	pm.alertsLock.Lock()
	defer pm.alertsLock.Unlock()

	// 返回告警的副本
	alerts := make([]Alert, len(pm.alerts))
	copy(alerts, pm.alerts)
	return alerts
}

// Stop 停止监控
func (pm *PerformanceMonitor) Stop() {
	pm.runningLock.Lock()
	defer pm.runningLock.Unlock()

	if !pm.running {
		return
	}

	pm.running = false
	close(pm.stopChan)
}