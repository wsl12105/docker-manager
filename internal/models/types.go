// Package models 
package models

// AppConfig 
type AppConfig struct {
	Name    string
	Version string
}

// ContainerStats 
type ContainerStats struct {
	CPU string
	MEM string
}

// TableRow 
type TableRow struct {
	ID      string
	Columns []string
	Color   string
	Ref     interface{}
}

// ModalAction 
type ModalAction struct {
	Title   string
	Message string
	Confirm func()
	Cancel  func()
}
