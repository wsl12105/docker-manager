// Package benchmark 性能测试和基准测试
package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/wsl12105/docker-manager/internal/docker"
)

// MockDockerClient 模拟Docker客户端用于性能测试
type MockDockerClient struct {
	containerCount int
	statsDelay     time.Duration // 模拟stats获取延迟
}

// NewMockDockerClient 创建模拟客户端
func NewMockDockerClient(containerCount int, statsDelay time.Duration) *MockDockerClient {
	return &MockDockerClient{
		containerCount: containerCount,
		statsDelay:     statsDelay,
	}
}

// ListContainers 模拟列出容器
func (m *MockDockerClient) ListContainers(all bool) ([]types.Container, error) {
	containers := make([]types.Container, m.containerCount)
	for i := 0; i < m.containerCount; i++ {
		status := "Up 2 hours"
		if i%3 == 0 {
			status = "Exited (0) 1 hour ago"
		}
		containers[i] = types.Container{
			ID:     fmt.Sprintf("container%d", i),
			Names:  []string{fmt.Sprintf("/container%d", i)},
			Image:  fmt.Sprintf("image%d:latest", i%10),
			Status: status,
			Ports:  []types.Port{},
		}
	}
	return containers, nil
}

// GetContainerStats 模拟获取容器统计信息
func (m *MockDockerClient) GetContainerStats(containerID string) (map[string]interface{}, error) {
	// 模拟网络延迟
	time.Sleep(m.statsDelay)

	return map[string]interface{}{
		"cpu_stats": map[string]interface{}{
			"cpu_usage": map[string]interface{}{
				"total_usage": rand.Float64() * 1000000000,
			},
			"system_cpu_usage": rand.Float64() * 10000000000,
		},
		"precpu_stats": map[string]interface{}{
			"cpu_usage": map[string]interface{}{
				"total_usage": rand.Float64() * 1000000000,
			},
			"system_cpu_usage": rand.Float64() * 10000000000,
		},
		"memory_stats": map[string]interface{}{
			"usage": rand.Float64() * 1024 * 1024 * 100,
		},
	}, nil
}

// BatchGetStats 模拟批量获取统计信息
func (m *MockDockerClient) BatchGetStats(containerIDs []string) docker.BatchStatsResult {
	result := docker.BatchStatsResult{
		Stats:  make(map[string]docker.ContainerStats),
		Errors: make(map[string]error),
	}

	// 使用并发池，限制并发数量（最多5个）
	pool := make(chan struct{}, 5)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, id := range containerIDs {
		wg.Add(1)
		pool <- struct{}{} // 获取并发槽

		go func(containerID string) {
			defer wg.Done()
			defer func() { <-pool }() // 释放并发槽

			data, err := m.GetContainerStats(containerID)
			if err != nil {
				mu.Lock()
				result.Stats[containerID] = docker.ContainerStats{CPU: "0.00%", Memory: "0MB"}
				result.Errors[containerID] = err
				result.Failed++
				mu.Unlock()
				return
			}

			// 解析统计数据
			cpu, mem := parseStats(data)

			mu.Lock()
			result.Stats[containerID] = docker.ContainerStats{CPU: cpu, Memory: mem}
			result.Success++
			mu.Unlock()
		}(id)
	}

	wg.Wait()
	return result
}

// parseStats 解析统计数据
func parseStats(data map[string]interface{}) (string, string) {
	memVal := 0.0
	if ms, ok := data["memory_stats"].(map[string]interface{}); ok {
		if u, ok := ms["usage"].(float64); ok {
			memVal = u / 1024 / 1024
		}
	}

	cpuP := 0.0
	cs, okCS := data["cpu_stats"].(map[string]interface{})
	ps, okPS := data["precpu_stats"].(map[string]interface{})
	if okCS && okPS {
		var curU, preU, curS, preS float64
		if u, ok := cs["cpu_usage"].(map[string]interface{}); ok {
			if v, ok := u["total_usage"].(float64); ok {
				curU = v
			}
		}
		if u, ok := ps["cpu_usage"].(map[string]interface{}); ok {
			if v, ok := u["total_usage"].(float64); ok {
				preU = v
			}
		}
		if v, ok := cs["system_cpu_usage"].(float64); ok {
			curS = v
		}
		if v, ok := ps["system_cpu_usage"].(float64); ok {
			preS = v
		}
		deltaC := curU - preU
		deltaS := curS - preS
		if deltaS > 0 && deltaC > 0 {
			cpuP = (deltaC / deltaS) * 100.0
		}
	}

	return fmt.Sprintf("%.2f%%", cpuP), fmt.Sprintf("%.1fMB", memVal)
}

// 基准测试：测试不同容器数量下的性能表现

// BenchmarkListContainers_10Containers 测试10个容器的列表性能
func BenchmarkListContainers_10Containers(b *testing.B) {
	client := NewMockDockerClient(10, 10*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ListContainers(true)
	}
}

// BenchmarkListContainers_50Containers 测试50个容器的列表性能
func BenchmarkListContainers_50Containers(b *testing.B) {
	client := NewMockDockerClient(50, 10*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ListContainers(true)
	}
}

// BenchmarkListContainers_100Containers 测试100个容器的列表性能
func BenchmarkListContainers_100Containers(b *testing.B) {
	client := NewMockDockerClient(100, 10*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ListContainers(true)
	}
}

// BenchmarkListContainers_200Containers 测试200个容器的列表性能
func BenchmarkListContainers_200Containers(b *testing.B) {
	client := NewMockDockerClient(200, 10*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ListContainers(true)
	}
}

// BenchmarkBatchGetStats_10Containers 测试10个容器的批量stats获取性能
func BenchmarkBatchGetStats_10Containers(b *testing.B) {
	client := NewMockDockerClient(10, 10*time.Millisecond)
	containers, _ := client.ListContainers(true)
	ids := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.BatchGetStats(ids)
	}
}

// BenchmarkBatchGetStats_50Containers 测试50个容器的批量stats获取性能
func BenchmarkBatchGetStats_50Containers(b *testing.B) {
	client := NewMockDockerClient(50, 10*time.Millisecond)
	containers, _ := client.ListContainers(true)
	ids := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.BatchGetStats(ids)
	}
}

// BenchmarkBatchGetStats_100Containers 测试100个容器的批量stats获取性能
func BenchmarkBatchGetStats_100Containers(b *testing.B) {
	client := NewMockDockerClient(100, 10*time.Millisecond)
	containers, _ := client.ListContainers(true)
	ids := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.BatchGetStats(ids)
	}
}

// BenchmarkBatchGetStats_200Containers 测试200个容器的批量stats获取性能
func BenchmarkBatchGetStats_200Containers(b *testing.B) {
	client := NewMockDockerClient(200, 10*time.Millisecond)
	containers, _ := client.ListContainers(true)
	ids := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.BatchGetStats(ids)
	}
}

// BenchmarkStatsParsing 测试stats解析性能
func BenchmarkStatsParsing(b *testing.B) {
	data := map[string]interface{}{
		"cpu_stats": map[string]interface{}{
			"cpu_usage": map[string]interface{}{
				"total_usage": 1234567890.0,
			},
			"system_cpu_usage": 12345678900.0,
		},
		"precpu_stats": map[string]interface{}{
			"cpu_usage": map[string]interface{}{
				"total_usage": 1234567880.0,
			},
			"system_cpu_usage": 12345678800.0,
		},
		"memory_stats": map[string]interface{}{
			"usage": 104857600.0, // 100MB
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseStats(data)
	}
}

// BenchmarkJSONDecoding 测试JSON解码性能
func BenchmarkJSONDecoding(b *testing.B) {
	jsonData := `{
		"cpu_stats": {
			"cpu_usage": {
				"total_usage": 1234567890
			},
			"system_cpu_usage": 12345678900
		},
		"precpu_stats": {
			"cpu_usage": {
				"total_usage": 1234567880
			},
			"system_cpu_usage": 12345678800
		},
		"memory_stats": {
			"usage": 104857600
		}
	}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var data map[string]interface{}
		_ = json.Unmarshal([]byte(jsonData), &data)
	}
}

// BenchmarkConcurrentStatsFetch 测试并发stats获取性能
func BenchmarkConcurrentStatsFetch(b *testing.B) {
	client := NewMockDockerClient(50, 10*time.Millisecond)
	containers, _ := client.ListContainers(true)
	ids := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, id := range ids {
			wg.Add(1)
			go func(containerID string) {
				defer wg.Done()
				_, _ = client.GetContainerStats(containerID)
			}(id)
		}
		wg.Wait()
	}
}

// BenchmarkMemoryAllocation 测试内存分配性能
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = make([]types.Container, 100)
		_ = make(map[string]docker.ContainerStats, 100)
		_ = make(map[string]interface{}, 10)
	}
}

// 性能测试：测试内存使用情况
func TestMemoryUsage(t *testing.T) {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)

	t.Logf("内存使用情况:")
	t.Logf("  Alloc = %v MB", m.Alloc/1024/1024)
	t.Logf("  TotalAlloc = %v MB", m.TotalAlloc/1024/1024)
	t.Logf("  Sys = %v MB", m.Sys/1024/1024)
	t.Logf("  NumGC = %v", m.NumGC)
}

// 性能测试：测试并发安全性
func TestConcurrentSafety(t *testing.T) {
	client := NewMockDockerClient(100, 5*time.Millisecond)
	containers, _ := client.ListContainers(true)
	ids := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
	}

	// 并发测试
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := client.BatchGetStats(ids[:10]) // 每次获取10个
			if result.Success < 10 {
				errors <- fmt.Errorf("expected 10 successes, got %d", result.Success)
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("并发安全测试失败: %v", err)
		}
	}
}

// 性能测试：测试不同延迟下的性能
func TestStatsFetchWithDifferentLatencies(t *testing.T) {
	latencies := []time.Duration{
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		20 * time.Millisecond,
		50 * time.Millisecond,
	}

	for _, latency := range latencies {
		t.Run(fmt.Sprintf("Latency_%dms", latency.Milliseconds()), func(t *testing.T) {
			client := NewMockDockerClient(50, latency)
			containers, _ := client.ListContainers(true)
			ids := make([]string, len(containers))
			for i, c := range containers {
				ids[i] = c.ID
			}

			start := time.Now()
			result := client.BatchGetStats(ids)
			elapsed := time.Since(start)

			t.Logf("延迟 %dms: 获取50个容器stats耗时 %v, 成功 %d, 失败 %d",
				latency.Milliseconds(), elapsed, result.Success, result.Failed)

			if result.Success != 50 {
				t.Errorf("预期成功50个，实际成功%d个", result.Success)
			}
		})
	}
}

// 性能测试：测试缓存效果
func TestCacheEffectiveness(t *testing.T) {
	client := NewMockDockerClient(100, 10*time.Millisecond)
	containers, _ := client.ListContainers(true)
	ids := make([]string, len(containers))
	for i, c := range containers {
		ids[i] = c.ID
	}

	// 第一次获取（无缓存）
	start1 := time.Now()
	result1 := client.BatchGetStats(ids)
	elapsed1 := time.Since(start1)

	// 第二次获取（模拟缓存）
	start2 := time.Now()
	result2 := client.BatchGetStats(ids)
	elapsed2 := time.Since(start2)

	t.Logf("第一次获取: %v (成功 %d)", elapsed1, result1.Success)
	t.Logf("第二次获取: %v (成功 %d)", elapsed2, result2.Success)

	// 第二次应该更快（因为并发池限制，实际可能差不多）
	if elapsed2 > elapsed1*2 {
		t.Logf("警告: 第二次获取比第一次慢很多")
	}
}

// 压力测试：测试极端情况下的性能
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	client := NewMockDockerClient(200, 5*time.Millisecond)

	// 持续运行10秒
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	operations := 0
	mu := sync.Mutex{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					containers, _ := client.ListContainers(true)
					ids := make([]string, len(containers))
					for i, c := range containers {
						ids[i] = c.ID
					}
					_ = client.BatchGetStats(ids[:20])

					mu.Lock()
					operations++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	opsPerSec := float64(operations) / 10.0
	t.Logf("压力测试: 10秒内完成 %d 次操作 (%.2f ops/sec)", operations, opsPerSec)
}