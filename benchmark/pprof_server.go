// Package benchmark 提供pprof性能分析支持
package benchmark

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" // 导入pprof，自动注册性能分析端点
	"runtime"
	"time"
)

// StartPprofServer 启动pprof性能分析服务器
// 访问 http://localhost:6060/debug/pprof/ 查看性能数据
func StartPprofServer(port int) error {
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("pprof服务器启动在 http://localhost%s/debug/pprof/\n", addr)
	fmt.Println("可用的性能分析端点:")
	fmt.Println("  - /debug/pprof/           性能分析首页")
	fmt.Println("  - /debug/pprof/heap       堆内存分析")
	fmt.Println("  - /debug/pprof/goroutine  Goroutine分析")
	fmt.Println("  - /debug/pprof/cpu        CPU分析")
	fmt.Println("  - /debug/pprof/block      阻塞分析")
	fmt.Println("  - /debug/pprof/mutex      互斥锁分析")
	fmt.Println("\n使用go tool pprof命令行工具:")
	fmt.Printf("  go tool pprof http://localhost%s/debug/pprof/profile?seconds=30\n", addr)
	fmt.Printf("  go tool pprof http://localhost%s/debug/pprof/heap\n", addr)
	fmt.Printf("  go tool pprof http://localhost%s/debug/pprof/goroutine\n", addr)

	return http.ListenAndServe(addr, nil)
}

// PerformanceMetrics 性能指标结构
type PerformanceMetrics struct {
	Timestamp          time.Time
	CPUUsage          float64
	MemoryAllocMB     uint64
	MemoryTotalAllocMB uint64
	MemorySysMB       uint64
	NumGoroutine      int
	NumGC             uint32
	GCPauseTotalMs    uint64
}

// GetPerformanceMetrics 获取当前性能指标
func GetPerformanceMetrics() PerformanceMetrics {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)

	return PerformanceMetrics{
		Timestamp:          time.Now(),
		MemoryAllocMB:      m.Alloc / 1024 / 1024,
		MemoryTotalAllocMB: m.TotalAlloc / 1024 / 1024,
		MemorySysMB:        m.Sys / 1024 / 1024,
		NumGoroutine:       runtime.NumGoroutine(),
		NumGC:              m.NumGC,
		GCPauseTotalMs:     m.PauseTotalNs / 1000000,
	}
}

// PrintPerformanceMetrics 打印性能指标
func PrintPerformanceMetrics() {
	metrics := GetPerformanceMetrics()
	fmt.Println("\n========== 性能指标 ==========")
	fmt.Printf("时间: %v\n", metrics.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("内存分配: %d MB\n", metrics.MemoryAllocMB)
	fmt.Printf("总内存分配: %d MB\n", metrics.MemoryTotalAllocMB)
	fmt.Printf("系统内存: %d MB\n", metrics.MemorySysMB)
	fmt.Printf("Goroutine数量: %d\n", metrics.NumGoroutine)
	fmt.Printf("GC次数: %d\n", metrics.NumGC)
	fmt.Printf("GC暂停总时间: %d ms\n", metrics.GCPauseTotalMs)
	fmt.Println("================================")
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	samples    []PerformanceMetrics
	maxSamples int
	stopChan   chan struct{}
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(maxSamples int) *PerformanceMonitor {
	return &PerformanceMonitor{
		samples:    make([]PerformanceMetrics, 0, maxSamples),
		maxSamples: maxSamples,
		stopChan:   make(chan struct{}),
	}
}

// Start 开始监控
func (pm *PerformanceMonitor) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := GetPerformanceMetrics()
			pm.samples = append(pm.samples, metrics)

			// 保持样本数量不超过最大值
			if len(pm.samples) > pm.maxSamples {
				pm.samples = pm.samples[1:]
			}
		case <-pm.stopChan:
			return
		}
	}
}

// Stop 停止监控
func (pm *PerformanceMonitor) Stop() {
	close(pm.stopChan)
}

// GetSamples 获取样本数据
func (pm *PerformanceMonitor) GetSamples() []PerformanceMetrics {
	return pm.samples
}

// GetAverageMetrics 获取平均性能指标
func (pm *PerformanceMonitor) GetAverageMetrics() PerformanceMetrics {
	if len(pm.samples) == 0 {
		return PerformanceMetrics{}
	}

	var totalAlloc, totalSys, totalGoroutine uint64
	var totalGC uint32
	var totalGCPause uint64

	for _, sample := range pm.samples {
		totalAlloc += sample.MemoryAllocMB
		totalSys += sample.MemorySysMB
		totalGoroutine += uint64(sample.NumGoroutine)
		totalGC += sample.NumGC
		totalGCPause += sample.GCPauseTotalMs
	}

	count := uint64(len(pm.samples))
	return PerformanceMetrics{
		Timestamp:          time.Now(),
		MemoryAllocMB:      totalAlloc / count,
		MemorySysMB:        totalSys / count,
		NumGoroutine:       int(totalGoroutine / count),
		NumGC:             totalGC / uint32(count),
		GCPauseTotalMs:    totalGCPause / count,
	}
}

// AnalyzePerformance 分析性能数据
func AnalyzePerformance(samples []PerformanceMetrics) {
	if len(samples) == 0 {
		fmt.Println("无性能数据")
		return
	}

	fmt.Println("\n========== 性能分析报告 ==========")
	fmt.Printf("样本数量: %d\n", len(samples))
	fmt.Printf("时间范围: %v 到 %v\n",
		samples[0].Timestamp.Format("15:04:05"),
		samples[len(samples)-1].Timestamp.Format("15:04:05"))

	// 内存分析
	minMem, maxMem := samples[0].MemoryAllocMB, samples[0].MemoryAllocMB
	for _, s := range samples {
		if s.MemoryAllocMB < minMem {
			minMem = s.MemoryAllocMB
		}
		if s.MemoryAllocMB > maxMem {
			maxMem = s.MemoryAllocMB
		}
	}
	fmt.Printf("内存使用范围: %d MB - %d MB\n", minMem, maxMem)

	// Goroutine分析
	minGoroutine, maxGoroutine := samples[0].NumGoroutine, samples[0].NumGoroutine
	for _, s := range samples {
		if s.NumGoroutine < minGoroutine {
			minGoroutine = s.NumGoroutine
		}
		if s.NumGoroutine > maxGoroutine {
			maxGoroutine = s.NumGoroutine
		}
	}
	fmt.Printf("Goroutine数量范围: %d - %d\n", minGoroutine, maxGoroutine)

	// GC分析
	if len(samples) >= 2 {
		gcIncrease := samples[len(samples)-1].NumGC - samples[0].NumGC
		fmt.Printf("GC次数增加: %d\n", gcIncrease)
	}

	fmt.Println("====================================")
}