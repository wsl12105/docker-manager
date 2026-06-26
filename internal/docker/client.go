package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/wsl12105/docker-manager/internal/config"
)

// ContainerStats 容器统计信息结构体
type ContainerStats struct {
	CPU    string
	Memory string
}

// StatsResponse Docker stats API响应结构体（性能优化：使用结构体代替map[string]interface{}）
type StatsResponse struct {
	CPUStats    CPUStats    `json:"cpu_stats"`
	PreCPUStats CPUStats    `json:"precpu_stats"`
	MemoryStats MemoryStats `json:"memory_stats"`
}

// CPUStats CPU统计信息结构体
type CPUStats struct {
	CPUUsage       CPUUsage `json:"cpu_usage"`
	SystemCPUUsage float64  `json:"system_cpu_usage"`
}

// CPUUsage CPU使用信息结构体
type CPUUsage struct {
	TotalUsage float64 `json:"total_usage"`
}

// MemoryStats 内存统计信息结构体
type MemoryStats struct {
	Usage float64 `json:"usage"`
}

// BatchStatsResult 批量Stats获取结果（修复错误处理问题）
type BatchStatsResult struct {
	Stats   map[string]ContainerStats // 成功获取的stats
	Errors  map[string]error          // 记录每个容器的错误
	Success int                       // 成功数量
	Failed  int                       // 失败数量
}

// Client Docker客户端封装
type Client struct {
	cli *client.Client
	ctx context.Context
}

// NewClient 创建Docker客户端实例
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

// Close 关闭Docker客户端连接
func (c *Client) Close() error {
	return c.cli.Close()
}

// CheckDockerRunning 检查Docker服务是否运行
func (c *Client) CheckDockerRunning() error {
	ctx, cancel := context.WithTimeout(c.ctx, config.PingTimeout)
	defer cancel()
	
	_, err := c.cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Docker service is not running: %w", err)
	}
	return nil
}

// ListContainers 列出所有容器
func (c *Client) ListContainers(all bool) ([]types.Container, error) {
	ctx, cancel := context.WithTimeout(c.ctx, config.ListTimeout)
	defer cancel()
	return c.cli.ContainerList(ctx, container.ListOptions{All: all})
}

// ListImages 列出所有镜像
func (c *Client) ListImages() ([]image.Summary, error) {
	ctx, cancel := context.WithTimeout(c.ctx, config.ListTimeout)
	defer cancel()
	return c.cli.ImageList(ctx, image.ListOptions{})
}

// GetContainerStats 获取容器实时统计信息（性能优化：使用结构体解码）
func (c *Client) GetContainerStats(containerID string) (*StatsResponse, error) {
	ctx, cancel := context.WithTimeout(c.ctx, config.StatsTimeout)
	defer cancel()

	stats, err := c.cli.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var data StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

// BatchGetStats 批量获取容器统计信息（性能优化：动态并发池）
func (c *Client) BatchGetStats(containerIDs []string) BatchStatsResult {
	result := BatchStatsResult{
		Stats:  make(map[string]ContainerStats),
		Errors: make(map[string]error),
	}

	if len(containerIDs) == 0 {
		return result
	}

	// 动态调整并发池大小（性能优化）
	poolSize := c.calculatePoolSize(len(containerIDs))
	pool := make(chan struct{}, poolSize)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, id := range containerIDs {
		wg.Add(1)
		pool <- struct{}{} // 获取并发槽

		go func(containerID string) {
			defer wg.Done()
			defer func() { <-pool }() // 释放并发槽

			data, err := c.GetContainerStats(containerID)
			if err != nil {
				mu.Lock()
				result.Stats[containerID] = ContainerStats{CPU: "0.00%", Memory: "0MB"}
				result.Errors[containerID] = err // 记录错误（修复错误处理问题）
				result.Failed++
				mu.Unlock()
				return
			}

			// 解析统计数据（使用优化后的ParseStats）
			cpu, mem := c.ParseStats(data)

			mu.Lock()
			result.Stats[containerID] = ContainerStats{CPU: cpu, Memory: mem}
			result.Success++
			mu.Unlock()
		}(id)
	}

	wg.Wait()
	return result
}

// calculatePoolSize 计算并发池大小（性能优化：动态调整）
func (c *Client) calculatePoolSize(containerCount int) int {
	// 根据容器数量动态调整并发池大小
	// 小容器数(≤20): 3个并发
	// 中等容器数(20-50): 5个并发
	// 大容器数(50-100): 8个并发
	// 极大容器数(>100): 10个并发
	switch {
	case containerCount <= 20:
		return 3
	case containerCount <= 50:
		return 5
	case containerCount <= 100:
		return 8
	default:
		return 10
	}
}

// ParseStats 解析统计数据（性能优化：使用结构体和strconv）
func (c *Client) ParseStats(data *StatsResponse) (string, string) {
	// 内存计算（使用结构体直接访问，无需类型断言）
	memVal := data.MemoryStats.Usage / 1024 / 1024

	// CPU计算（使用结构体直接访问，无需类型断言）
	cpuP := 0.0
	deltaC := data.CPUStats.CPUUsage.TotalUsage - data.PreCPUStats.CPUUsage.TotalUsage
	deltaS := data.CPUStats.SystemCPUUsage - data.PreCPUStats.SystemCPUUsage
	if deltaS > 0 && deltaC > 0 {
		cpuP = (deltaC / deltaS) * 100.0
	}

	// 使用strconv代替fmt.Sprintf（性能优化）
	cpuStr := formatPercent(cpuP)
	memStr := formatMemory(memVal)

	return cpuStr, memStr
}

// formatPercent 格式化百分比（性能优化：使用strconv）
func formatPercent(value float64) string {
	// 使用strconv.FormatFloat代替fmt.Sprintf
	// 格式化为两位小数百分比，如 "12.34%"
	str := strconv.FormatFloat(value, 'f', 2, 64)
	return str + "%"
}

// formatMemory 格式化内存（性能优化：使用strconv）
func formatMemory(valueMB float64) string {
	// 格式化为一位小数MB，如 "123.4MB"
	str := strconv.FormatFloat(valueMB, 'f', 1, 64)
	return str + "MB"
}

// StartContainer 启动容器
func (c *Client) StartContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(c.ctx, config.OperationTimeout)
	defer cancel()
	return c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

// StopContainer 停止容器
func (c *Client) StopContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(c.ctx, config.OperationTimeout)
	defer cancel()
	return c.cli.ContainerStop(ctx, containerID, container.StopOptions{})
}

// RemoveContainer 删除容器
func (c *Client) RemoveContainer(containerID string, force bool) error {
	ctx, cancel := context.WithTimeout(c.ctx, config.OperationTimeout)
	defer cancel()
	return c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: force})
}

// GetContainerLogs 获取容器日志
func (c *Client) GetContainerLogs(containerID string, tail string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(c.ctx, config.DefaultTimeout)
	defer cancel()
	return c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	})
}

// InspectContainer 查看容器详细信息
func (c *Client) InspectContainer(containerID string) (types.ContainerJSON, error) {
	ctx, cancel := context.WithTimeout(c.ctx, config.InspectTimeout)
	defer cancel()
	resp, _, err := c.cli.ContainerInspectWithRaw(ctx, containerID, false)
	return resp, err
}

// TagImage 为镜像添加标签
func (c *Client) TagImage(imageID, tag string) error {
	ctx, cancel := context.WithTimeout(c.ctx, config.OperationTimeout)
	defer cancel()
	return c.cli.ImageTag(ctx, imageID, tag)
}

// RemoveImage 删除镜像
func (c *Client) RemoveImage(imageID string, force bool) ([]image.DeleteResponse, error) {
	ctx, cancel := context.WithTimeout(c.ctx, config.OperationTimeout)
	defer cancel()
	return c.cli.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
}
