// Package docker Docker客户端封装
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Client Docker客户端包装器
type Client struct {
	cli *client.Client
	ctx context.Context
}

// NewClient 创建新的Docker客户端
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("创建Docker客户端失败: %w", err)
	}
	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	return c.cli.Close()
}

// ListContainers 列出所有容器
func (c *Client) ListContainers(all bool) ([]types.Container, error) {
	return c.cli.ContainerList(c.ctx, container.ListOptions{All: all})
}

// ListImages 列出所有镜像
func (c *Client) ListImages() ([]image.Summary, error) {
	return c.cli.ImageList(c.ctx, image.ListOptions{})
}

// GetContainerStats 获取容器统计信息
func (c *Client) GetContainerStats(containerID string) (map[string]interface{}, error) {
	stats, err := c.cli.ContainerStatsOneShot(c.ctx, containerID)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(stats.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// StartContainer 启动容器
func (c *Client) StartContainer(containerID string) error {
	return c.cli.ContainerStart(c.ctx, containerID, container.StartOptions{})
}

// StopContainer 停止容器
func (c *Client) StopContainer(containerID string) error {
	return c.cli.ContainerStop(c.ctx, containerID, container.StopOptions{})
}

// RemoveContainer 删除容器
func (c *Client) RemoveContainer(containerID string, force bool) error {
	return c.cli.ContainerRemove(c.ctx, containerID, container.RemoveOptions{Force: force})
}

// GetContainerLogs 获取容器日志
func (c *Client) GetContainerLogs(containerID string, tail string) (io.ReadCloser, error) {
	return c.cli.ContainerLogs(c.ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	})
}

// InspectContainer 查看容器详情
func (c *Client) InspectContainer(containerID string) (types.ContainerJSON, error) {
	resp, _, err := c.cli.ContainerInspectWithRaw(c.ctx, containerID, false)
	return resp, err
}

// TagImage 标记镜像
func (c *Client) TagImage(imageID, tag string) error {
	return c.cli.ImageTag(c.ctx, imageID, tag)
}

// RemoveImage 删除镜像
func (c *Client) RemoveImage(imageID string, force bool) ([]image.DeleteResponse, error) {
	return c.cli.ImageRemove(c.ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
}
