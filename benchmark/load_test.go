// Package benchmark 负载测试和性能基准
package benchmark

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// 负载测试：模拟真实使用场景

// TestLoadTest_LightLoad 轻负载测试（10个容器）
func TestLoadTest_LightLoad(t *testing.T) {
	runLoadTest(t, "轻负载(10容器)", 10, 5*time.Second, 100*time.Millisecond)
}

// TestLoadTest_MediumLoad 中等负载测试（50个容器）
func TestLoadTest_MediumLoad(t *testing.T) {
	runLoadTest(t, "中等负载(50容器)", 50, 5*time.Second, 100*time.Millisecond)
}

// TestLoadTest_HeavyLoad 重负载测试（100个容器）
func TestLoadTest_HeavyLoad(t *testing.T) {
	runLoadTest(t, "重负载(100容器)", 100, 5*time.Second, 100*time.Millisecond)
}

// TestLoadTest_ExtremeLoad 极端负载测试（200个容器）
func TestLoadTest_ExtremeLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过极端负载测试")
	}
	runLoadTest(t, "极端负载(200容器)", 200, 5*time.Second, 100*time.Millisecond)
}

// runLoadTest 运行负载测试
func runLoadTest(t *testing.T, name string, containerCount int, duration time.Duration, refreshInterval time.Duration) {
	client := NewMockDockerClient(containerCount, 10*time.Millisecond)

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	startTime := time.Now()
	iterations := 0
	var totalStatsTime time.Duration
	var totalListTime time.Duration

	// 模拟刷新循环
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			goto Done
		default:
			// 模拟容器列表刷新
			listStart := time.Now()
			containers, err := client.ListContainers(true)
			listElapsed := time.Since(listStart)
			totalListTime += listElapsed

			if err != nil {
				t.Errorf("列出容器失败: %v", err)
				continue
			}

			// 收集运行中的容器ID
			ids := make([]string, 0, len(containers))
			for _, c := range containers {
				ids = append(ids, c.ID)
			}

			// 模拟批量获取stats
			statsStart := time.Now()
			result := client.BatchGetStats(ids)
			statsElapsed := time.Since(statsStart)
			totalStatsTime += statsElapsed

			if result.Success != len(ids) {
				t.Logf("警告: 预期成功 %d, 实际成功 %d", len(ids), result.Success)
			}

			iterations++
			time.Sleep(refreshInterval)
		}
	}

Done:
	runtime.GC()
	runtime.ReadMemStats(&m2)

	elapsed := time.Since(startTime)
	opsPerSec := float64(iterations) / elapsed.Seconds()

	t.Logf("\n========== %s 测试结果 ==========", name)
	t.Logf("测试时长: %v", elapsed)
	t.Logf("迭代次数: %d", iterations)
	t.Logf("操作频率: %.2f ops/sec", opsPerSec)
	t.Logf("平均列表耗时: %v", totalListTime/time.Duration(iterations))
	t.Logf("平均Stats耗时: %v", totalStatsTime/time.Duration(iterations))
	t.Logf("内存增长: %d MB", (m2.Alloc-m1.Alloc)/1024/1024)
	t.Logf("最终内存: %d MB", m2.Alloc/1024/1024)
	t.Logf("Goroutine数量: %d", runtime.NumGoroutine())
	t.Logf("=========================================")
}

// 并发测试：测试并发安全性

// TestConcurrentAccess_10Workers 10个并发worker测试
func TestConcurrentAccess_10Workers(t *testing.T) {
	runConcurrentTest(t, 10, 100)
}

// TestConcurrentAccess_50Workers 50个并发worker测试
func TestConcurrentAccess_50Workers(t *testing.T) {
	runConcurrentTest(t, 50, 100)
}

// TestConcurrentAccess_100Workers 100个并发worker测试
func TestConcurrentAccess_100Workers(t *testing.T) {
	runConcurrentTest(t, 100, 100)
}

// runConcurrentTest 运行并发测试
func runConcurrentTest(t *testing.T, workerCount int, operationsPerWorker int) {
	client := NewMockDockerClient(100, 5*time.Millisecond)

	var wg sync.WaitGroup
	errors := make(chan error, workerCount*operationsPerWorker)
	startTime := time.Now()

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				containers, err := client.ListContainers(true)
				if err != nil {
					errors <- fmt.Errorf("list containers failed: %w", err)
					continue
				}

				ids := make([]string, len(containers))
				for i, c := range containers {
					ids[i] = c.ID
				}

				result := client.BatchGetStats(ids[:10])
				if result.Success < 10 {
					errors <- fmt.Errorf("batch stats failed: expected 10, got %d", result.Success)
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(startTime)
	totalOps := workerCount * operationsPerWorker
	opsPerSec := float64(totalOps) / elapsed.Seconds()

	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Logf("错误: %v", err)
		}
	}

	t.Logf("\n========== 并发测试结果 ==========")
	t.Logf("Worker数量: %d", workerCount)
	t.Logf("每个Worker操作数: %d", operationsPerWorker)
	t.Logf("总操作数: %d", totalOps)
	t.Logf("测试时长: %v", elapsed)
	t.Logf("操作频率: %.2f ops/sec", opsPerSec)
	t.Logf("错误数量: %d", errorCount)
	t.Logf("Goroutine数量: %d", runtime.NumGoroutine())
	t.Logf("====================================")

	if errorCount > 0 {
		t.Errorf("并发测试发现 %d 个错误", errorCount)
	}
}

// 内存泄漏测试

// TestMemoryLeak 内存泄漏测试
func TestMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过内存泄漏测试")
	}

	client := NewMockDockerClient(100, 5*time.Millisecond)

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 执行大量操作
	for i := 0; i < 1000; i++ {
		containers, _ := client.ListContainers(true)
		ids := make([]string, len(containers))
		for i, c := range containers {
			ids[i] = c.ID
		}
		_ = client.BatchGetStats(ids)

		// 每100次强制GC
		if i%100 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	memGrowth := int64(m2.Alloc) - int64(m1.Alloc)
	t.Logf("\n========== 内存泄漏测试结果 ==========")
	t.Logf("初始内存: %d MB", m1.Alloc/1024/1024)
	t.Logf("最终内存: %d MB", m2.Alloc/1024/1024)
	t.Logf("内存增长: %d MB", memGrowth/1024/1024)
	t.Logf("GC次数: %d", m2.NumGC-m1.NumGC)
	t.Logf("========================================")

	// 如果内存增长超过50MB，可能存在内存泄漏
	if memGrowth > 50*1024*1024 {
		t.Errorf("可能存在内存泄漏: 内存增长 %d MB", memGrowth/1024/1024)
	}
}

// 性能回归测试

// BenchmarkPerformanceRegression 性能回归基准测试
func BenchmarkPerformanceRegression(b *testing.B) {
	client := NewMockDockerClient(100, 10*time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containers, _ := client.ListContainers(true)
		ids := make([]string, len(containers))
		for i, c := range containers {
			ids[i] = c.ID
		}
		_ = client.BatchGetStats(ids)
	}
}

// 吞吐量测试

// TestThroughput 测试系统吞吐量
func TestThroughput(t *testing.T) {
	client := NewMockDockerClient(100, 5*time.Millisecond)

	duration := 5 * time.Second
	var wg sync.WaitGroup
	operations := make(chan int, 1000)
	stopChan := make(chan struct{})

	// 启动多个worker
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					containers, _ := client.ListContainers(true)
					ids := make([]string, len(containers))
					for i, c := range containers {
						ids[i] = c.ID
					}
					_ = client.BatchGetStats(ids[:20])
					operations <- 1
				}
			}
		}()
	}

	// 运行指定时长
	time.Sleep(duration)
	close(stopChan)
	wg.Wait()
	close(operations)

	// 统计操作数
	totalOps := 0
	for range operations {
		totalOps++
	}

	throughput := float64(totalOps) / duration.Seconds()
	t.Logf("\n========== 吞吐量测试结果 ==========")
	t.Logf("测试时长: %v", duration)
	t.Logf("总操作数: %d", totalOps)
	t.Logf("吞吐量: %.2f ops/sec", throughput)
	t.Logf("========================================")
}

// 延迟测试

// TestLatency 测试操作延迟
func TestLatency(t *testing.T) {
	client := NewMockDockerClient(100, 10*time.Millisecond)

	samples := 100
	latencies := make([]time.Duration, samples)

	for i := 0; i < samples; i++ {
		start := time.Now()
		containers, _ := client.ListContainers(true)
		ids := make([]string, len(containers))
		for i, c := range containers {
			ids[i] = c.ID
		}
		_ = client.BatchGetStats(ids)
		latencies[i] = time.Since(start)
	}

	// 计算统计数据
	var total time.Duration
	minLatency := latencies[0]
	maxLatency := latencies[0]

	for _, lat := range latencies {
		total += lat
		if lat < minLatency {
			minLatency = lat
		}
		if lat > maxLatency {
			maxLatency = lat
		}
	}

	avgLatency := total / time.Duration(samples)

	t.Logf("\n========== 延迟测试结果 ==========")
	t.Logf("样本数量: %d", samples)
	t.Logf("平均延迟: %v", avgLatency)
	t.Logf("最小延迟: %v", minLatency)
	t.Logf("最大延迟: %v", maxLatency)
	t.Logf("====================================")
}