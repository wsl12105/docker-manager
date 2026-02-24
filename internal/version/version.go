// Package version 
package version

import (
	"fmt"
)

const (
	// AppName 
	AppName = "DM (Docker Manager)"
	
	// Version 
	Version = "v1.5.0"
	
	// BuildDate 
	BuildDate = "unknown"
	
	// GitCommit Git）
	GitCommit = "unknown"
)

// GetVersionString 
func GetVersionString() string {
	return fmt.Sprintf("%s %s", AppName, Version)
}

// GetFullVersionInfo 
func GetFullVersionInfo() string {
	return fmt.Sprintf("%s %s (build: %s, commit: %s)", 
		AppName, Version, BuildDate, GitCommit)
}

// GetVersion 
func GetVersion() string {
	return Version
}

// GetAppName 
func GetAppName() string {
	return AppName
}
