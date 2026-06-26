// Package config 提供应用配置集中管理
//
// 该包集中管理应用的配置常量，包括：
//
// - 应用名称和版本信息
// - 刷新间隔配置
// - 超时配置（默认超时、列表超时、操作超时等）
// - 缓存配置（Stats缓存有效期）
// - Shell路径配置（用于容器内执行命令）
//
// 主要配置项：
//   - RefreshInterval: 刷新间隔（2秒）
//   - DefaultTimeout: 默认超时（30秒）
//   - ListTimeout: 列表操作超时（10秒）
//   - OperationTimeout: 容器操作超时（60秒）
//   - StatsTimeout: Stats获取超时（5秒）
//   - StatsCacheExpiry: Stats缓存有效期（5秒）
//
// 使用示例：
//
//	// 使用超时配置
//	ctx, cancel := context.WithTimeout(context.Background(), config.ListTimeout)
//	defer cancel()
//	containers, err := client.ContainerList(ctx, options)
//
//	// 使用缓存配置
//	cacheEntry := &StatsCacheEntry{
//	    ExpiresAt: time.Now().Add(config.StatsCacheExpiry),
//	}
//
// 注意事项：
//   - 所有超时配置都是time.Duration类型
//   - Shell路径按优先级排序，第一个可用的将被使用
//   - 配置常量不应在运行时修改
package config