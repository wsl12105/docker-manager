// Package models 定义数据模型
package models

// AppConfig 应用配置
type AppConfig struct {
	Name    string
	Version string
}

// ContainerStats 容器统计信息
type ContainerStats struct {
	CPU string
	MEM string
}

// TableRow 表格行数据
type TableRow struct {
	ID      string
	Columns []string
	Color   string
	Ref     interface{}
}

// ModalAction 模态框操作
type ModalAction struct {
	Title   string
	Message string
	Confirm func()
	Cancel  func()
}
