// Package config 应用配置包
package config

import "time"

// 应用配置常量
const (
	// 应用名称
	AppName = "DM (Docker Manager)"
	
	// 刷新间隔
	RefreshInterval = 2 * time.Second
	
	// 日志显示行数
	LogTailLines = "200"
	
	// 容器ID显示长度
	ContainerIDLength = 12
	
	// 镜像ID前缀长度
	ImageIDPrefixLength = 7
	
	// 镜像ID显示长度
	ImageIDDisplayLength = 19
)

// Shell路径配置（按优先级排序）
var ShellPaths = []string{
	"/bin/bash",
	"/bin/sh",
	"/bin/ash",
	"/usr/bin/bash",
	"/usr/bin/sh",
}

// 超时配置
const (
	DefaultTimeout     = 30 * time.Second
	ListTimeout        = 10 * time.Second
	OperationTimeout   = 60 * time.Second
	StatsTimeout       = 5 * time.Second
	InspectTimeout     = 10 * time.Second
	PingTimeout        = 2 * time.Second
)

// 缓存配置（修复硬编码问题）
const (
	StatsCacheExpiry = 5 * time.Second // Stats缓存有效期
)