package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Client Docker
type Client struct {
	cli *client.Client
	ctx context.Context
}

// NewClient 
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

// Close 
func (c *Client) Close() error {
	return c.cli.Close()
}

// CheckDockerRunning 
func (c *Client) CheckDockerRunning() error {
	ctx, cancel := context.WithTimeout(c.ctx, 2*time.Second)
	defer cancel()
	
	_, err := c.cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("DockerNotRunning: %w", err)
	}
	return nil
}

// ListContainers 
func (c *Client) ListContainers(all bool) ([]types.Container, error) {
	return c.cli.ContainerList(c.ctx, container.ListOptions{All: all})
}

// ListImages 
func (c *Client) ListImages() ([]image.Summary, error) {
	return c.cli.ImageList(c.ctx, image.ListOptions{})
}

// GetContainerStats 
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

// StartContainer 
func (c *Client) StartContainer(containerID string) error {
	return c.cli.ContainerStart(c.ctx, containerID, container.StartOptions{})
}

// StopContainer 
func (c *Client) StopContainer(containerID string) error {
	return c.cli.ContainerStop(c.ctx, containerID, container.StopOptions{})
}

// RemoveContainer 
func (c *Client) RemoveContainer(containerID string, force bool) error {
	return c.cli.ContainerRemove(c.ctx, containerID, container.RemoveOptions{Force: force})
}

// GetContainerLogs 
func (c *Client) GetContainerLogs(containerID string, tail string) (io.ReadCloser, error) {
	return c.cli.ContainerLogs(c.ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	})
}

// InspectContainer 
func (c *Client) InspectContainer(containerID string) (types.ContainerJSON, error) {
	resp, _, err := c.cli.ContainerInspectWithRaw(c.ctx, containerID, false)
	return resp, err
}

// TagImage 
func (c *Client) TagImage(imageID, tag string) error {
	return c.cli.ImageTag(c.ctx, imageID, tag)
}

// RemoveImage 
func (c *Client) RemoveImage(imageID string, force bool) ([]image.DeleteResponse, error) {
	return c.cli.ImageRemove(c.ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
}
