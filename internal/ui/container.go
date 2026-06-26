// Package ui UI组件包
package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/wsl12105/docker-manager/internal/config"
	"github.com/wsl12105/docker-manager/internal/docker"
	"github.com/wsl12105/docker-manager/internal/version"
)

// ContainerUI
type ContainerUI struct {
	common       *Common
	docker       *docker.Client

	// Stats缓存（性能优化：使用LRU缓存策略）
	statsCache *LRUCache // 存储 *StatsCacheEntry，容量100

	// 刷新版本控制（修复竞态条件问题）
	refreshVersion  atomic.Int64 // 当前刷新版本号
	stopChan        chan struct{} // 停止通道

	// 增量更新缓存（性能优化：避免全量刷新）
	lastContainerList   []types.Container // 上次容器列表
	lastContainerMap    map[string]int    // 容器ID到行索引的映射
	containerListMutex  sync.RWMutex      // 保护容器列表缓存

	// TableCell对象池（性能优化：减少内存分配）
	cellPool *TableCellPool
}

// StatsCacheEntry Stats缓存条目
type StatsCacheEntry struct {
	CPU       string
	Memory    string
	Timestamp time.Time
	ExpiresAt time.Time
}

// LRUCacheEntry LRU缓存条目（双向链表节点）
type LRUCacheEntry struct {
	key       string
	value     *StatsCacheEntry
	prev      *LRUCacheEntry
	next      *LRUCacheEntry
}

// LRUCache LRU缓存结构
type LRUCache struct {
	capacity int                // 缓存容量
	size     int                // 当前大小
	cache    map[string]*LRUCacheEntry // 哈希表
	head     *LRUCacheEntry     // 双向链表头节点（最近使用）
	tail     *LRUCacheEntry     // 双向链表尾节点（最久未使用）
	mutex    sync.RWMutex       // 读写锁，保证并发安全
}

// NewLRUCache 创建LRU缓存
func NewLRUCache(capacity int) *LRUCache {
	lru := &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*LRUCacheEntry),
	}
	// 初始化哨兵节点（简化边界处理）
	lru.head = &LRUCacheEntry{}
	lru.tail = &LRUCacheEntry{}
	lru.head.next = lru.tail
	lru.tail.prev = lru.head
	return lru
}

// Get 获取缓存（如果存在则移动到头部）
func (lru *LRUCache) Get(key string) (*StatsCacheEntry, bool) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if entry, exists := lru.cache[key]; exists {
		// 缓存命中，移动到头部
		lru.moveToHead(entry)
		return entry.value, true
	}
	return nil, false
}

// Put 添加缓存（添加到头部，如果超出容量则删除尾部）
func (lru *LRUCache) Put(key string, value *StatsCacheEntry) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if entry, exists := lru.cache[key]; exists {
		// 已存在，更新值并移动到头部
		entry.value = value
		lru.moveToHead(entry)
	} else {
		// 不存在，创建新条目
		entry := &LRUCacheEntry{
			key:   key,
			value: value,
		}
		lru.cache[key] = entry
		lru.addToHead(entry)
		lru.size++

		// 如果超出容量，删除尾部（最久未使用）
		if lru.size > lru.capacity {
			tail := lru.removeTail()
			if tail != nil {
				delete(lru.cache, tail.key)
				lru.size--
			}
		}
	}
}

// moveToHead 移动节点到头部
func (lru *LRUCache) moveToHead(entry *LRUCacheEntry) {
	// 先从链表中删除
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
	// 再添加到头部
	lru.addToHead(entry)
}

// addToHead 添加节点到头部
func (lru *LRUCache) addToHead(entry *LRUCacheEntry) {
	entry.prev = lru.head
	entry.next = lru.head.next
	lru.head.next.prev = entry
	lru.head.next = entry
}

// removeTail 删除尾部节点（最久未使用）
func (lru *LRUCache) removeTail() *LRUCacheEntry {
	// 检查链表是否为空（只有哨兵节点）
	if lru.tail.prev == lru.head {
		return nil
	}
	entry := lru.tail.prev
	entry.prev.next = lru.tail
	lru.tail.prev = entry.prev
	return entry
}

// Size 获取缓存大小
func (lru *LRUCache) Size() int {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()
	return lru.size
}

// Clear 清空缓存
func (lru *LRUCache) Clear() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()
	lru.cache = make(map[string]*LRUCacheEntry)
	lru.head.next = lru.tail
	lru.tail.prev = lru.head
	lru.size = 0
}

// TableCellPool TableCell对象池（性能优化：减少内存分配）
type TableCellPool struct {
	pool sync.Pool
}

// NewTableCellPool 创建TableCell对象池
func NewTableCellPool() *TableCellPool {
	return &TableCellPool{
		pool: sync.Pool{
			New: func() interface{} {
				return tview.NewTableCell("")
			},
		},
	}
}

// Get 从池中获取TableCell（重置状态）
func (p *TableCellPool) Get() *tview.TableCell {
	cell := p.pool.Get().(*tview.TableCell)
	// 重置单元格状态，避免数据残留
	cell.SetText("")
	cell.SetTextColor(tcell.ColorWhite)
	cell.SetBackgroundColor(tcell.ColorDefault)
	cell.SetReference(nil)
	cell.SetSelectable(true)
	cell.SetExpansion(0)
	return cell
}

// Put 将TableCell放回池中
func (p *TableCellPool) Put(cell *tview.TableCell) {
	if cell == nil {
		return
	}
	p.pool.Put(cell)
}

// NewContainerUI
func NewContainerUI(common *Common, docker *docker.Client) *ContainerUI {
	ui := &ContainerUI{
		common:     common,
		docker:     docker,
		statsCache: NewLRUCache(100), // 初始化LRU缓存，容量100
		stopChan:   make(chan struct{}), // 初始化停止通道
		cellPool:   NewTableCellPool(), // 初始化TableCell对象池
	}

	return ui
}

// RefreshList 刷新容器列表（性能优化：使用增量更新）
func (ui *ContainerUI) RefreshList() {
	ui.RefreshListIncremental()
}

// updateStatsAsync 异步获取stats并更新UI（性能优化，修复竞态条件）
func (ui *ContainerUI) updateStatsAsync(list []types.Container, version int64) {
	// 收集运行容器ID
	runningIDs := []string{}
	runningIndices := make(map[string]int) // 记录容器ID对应的行索引
	
	for i, cont := range list {
		if strings.HasPrefix(cont.Status, "Up") {
			runningIDs = append(runningIDs, cont.ID)
			runningIndices[cont.ID] = i + 1 // 行索引（从1开始，第0行是表头）
		}
	}
	
	if len(runningIDs) == 0 {
		return // 无运行容器，无需获取stats
	}
	
	// 使用带缓存的批量获取
	statsMap := ui.batchGetStatsWithCache(runningIDs)
	
	// 更新UI（在主线程中执行）
	ui.common.App.QueueUpdateDraw(func() {
		// 检查版本号是否匹配（修复竞态条件问题）
		if ui.refreshVersion.Load() != version {
			return // 版本不匹配，放弃更新，避免数据错乱
		}
		
		for id, stats := range statsMap {
			if rowIndex, exists := runningIndices[id]; exists {
				ui.common.Table.SetCell(rowIndex, 3, tview.NewTableCell(stats.CPU).SetTextColor(tcell.ColorGreen))
				ui.common.Table.SetCell(rowIndex, 4, tview.NewTableCell(stats.Memory).SetTextColor(tcell.ColorGreen))
			}
		}
	})
}

// batchGetStatsWithCache 批量获取stats（性能优化：使用LRU缓存）
func (ui *ContainerUI) batchGetStatsWithCache(containerIDs []string) map[string]docker.ContainerStats {
	results := make(map[string]docker.ContainerStats)

	if len(containerIDs) == 0 {
		return results
	}

	// 检查缓存，过滤出需要重新获取的容器ID
	needRefreshIDs := []string{}
	now := time.Now()

	for _, id := range containerIDs {
		// 使用LRU缓存的Get方法（自动移动到头部）
		if entry, ok := ui.statsCache.Get(id); ok {
			if now.Before(entry.ExpiresAt) {
				// 缓存有效，直接使用
				results[id] = docker.ContainerStats{
					CPU:    entry.CPU,
					Memory: entry.Memory,
				}
			} else {
				// 缓存过期，需要重新获取
				needRefreshIDs = append(needRefreshIDs, id)
			}
		} else {
			// 无缓存，需要获取
			needRefreshIDs = append(needRefreshIDs, id)
		}
	}

	// 批量获取需要刷新的容器stats
	if len(needRefreshIDs) > 0 {
		batchResult := ui.docker.BatchGetStats(needRefreshIDs)

		// 更新缓存和结果（使用LRU缓存的Put方法）
		for id, stats := range batchResult.Stats {
			results[id] = stats
			ui.statsCache.Put(id, &StatsCacheEntry{
				CPU:       stats.CPU,
				Memory:    stats.Memory,
				Timestamp: now,
				ExpiresAt: now.Add(config.StatsCacheExpiry), // 使用配置常量
			})
		}

		// 记录错误（可选：可以添加错误日志）
		// for id, err := range batchResult.Errors {
		//     log.Printf("Failed to get stats for container %s: %v", id, err)
		// }
	}

	return results
}

// Cleanup 停止ContainerUI，清理资源
func (ui *ContainerUI) Cleanup() {
	// LRU缓存无需手动清理，会自动管理内存
	// 但如果需要，可以清空缓存
	if ui.statsCache != nil {
		ui.statsCache.Clear()
	}
}

// getStats 获取单个容器stats（性能优化：使用结构体）
func (ui *ContainerUI) getStats(containerID string) (string, string) {
	data, err := ui.docker.GetContainerStats(containerID)
	if err != nil {
		return "0.00%", "0MB"
	}

	// 使用优化后的ParseStats函数
	return ui.docker.ParseStats(data)
}

// restoreSelection
func (ui *ContainerUI) restoreSelection(selRow int) {
	if selRow >= ui.common.Table.GetRowCount() {
		selRow = ui.common.Table.GetRowCount() - 1
	}
	if selRow > 0 {
		ui.common.Table.Select(selRow, 0)
	} else if ui.common.Table.GetRowCount() > 1 {
		ui.common.Table.Select(1, 0)
	}
}

// buildContainerMap 构建容器ID到行索引的映射
func (ui *ContainerUI) buildContainerMap(list []types.Container) map[string]int {
	containerMap := make(map[string]int)
	for i, cont := range list {
		containerMap[cont.ID] = i + 1 // 行索引从1开始（第0行是表头）
	}
	return containerMap
}

// containerChanges 容器变化结果
type containerChanges struct {
	added   []types.Container // 新增的容器
	removed []string          // 删除的容器ID
	updated []types.Container // 更新的容器
}

// compareContainerChanges 对比容器变化
func (ui *ContainerUI) compareContainerChanges(newList []types.Container) *containerChanges {
	ui.containerListMutex.RLock()
	defer ui.containerListMutex.RUnlock()

	changes := &containerChanges{
		added:   []types.Container{},
		removed: []string{},
		updated: []types.Container{},
	}

	// 构建新容器ID集合
	newIDSet := make(map[string]bool)
	for _, cont := range newList {
		newIDSet[cont.ID] = true
	}

	// 检查删除和更新的容器
	if ui.lastContainerMap != nil {
		for id := range ui.lastContainerMap {
			if !newIDSet[id] {
				changes.removed = append(changes.removed, id)
			}
		}
	}

	// 构建旧容器ID到容器的映射
	oldContainerMap := make(map[string]types.Container)
	for _, cont := range ui.lastContainerList {
		oldContainerMap[cont.ID] = cont
	}

	// 检查新增和更新的容器
	for _, newCont := range newList {
		oldCont, exists := oldContainerMap[newCont.ID]
		if !exists {
			// 新增容器
			changes.added = append(changes.added, newCont)
		} else {
			// 检查是否需要更新（状态、镜像、端口等变化）
			if ui.containerChanged(oldCont, newCont) {
				changes.updated = append(changes.updated, newCont)
			}
		}
	}

	return changes
}

// containerChanged 检查容器是否发生变化
func (ui *ContainerUI) containerChanged(old, new types.Container) bool {
	// 检查状态变化
	if old.Status != new.Status {
		return true
	}

	// 检查镜像变化
	if old.Image != new.Image {
		return true
	}

	// 检查名称变化
	if len(old.Names) != len(new.Names) {
		return true
	}
	if len(old.Names) > 0 && len(new.Names) > 0 {
		if old.Names[0] != new.Names[0] {
			return true
		}
	}

	// 检查端口变化
	if len(old.Ports) != len(new.Ports) {
		return true
	}

	return false
}

// addRow 添加新行
func (ui *ContainerUI) addRow(cont types.Container, rowIndex int) {
	// 安全获取容器ID前12位，避免越界
	id := cont.ID
	if len(id) > config.ContainerIDLength {
		id = id[:config.ContainerIDLength]
	}

	color := tcell.ColorWhite
	cpu, mem := "-", "-"

	if strings.HasPrefix(cont.Status, "Up") {
		color = tcell.ColorGreen
		cpu, mem = "...", "..." // 占位符，表示正在获取
	}

	var portStrs []string
	for _, p := range cont.Ports {
		if p.PublicPort != 0 {
			portStrs = append(portStrs, fmt.Sprintf("%d->%d", p.PublicPort, p.PrivatePort))
		} else {
			portStrs = append(portStrs, fmt.Sprintf("%d", p.PrivatePort))
		}
	}

	name := ""
	if len(cont.Names) > 0 {
		name = strings.TrimPrefix(cont.Names[0], "/")
	}

	// 使用对象池获取TableCell（性能优化：减少内存分配）
	ui.common.Table.SetCell(rowIndex, 0, ui.cellPool.Get().SetText(id).SetTextColor(color).SetReference(cont.ID))
	ui.common.Table.SetCell(rowIndex, 1, ui.cellPool.Get().SetText(cont.Image).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 2, ui.cellPool.Get().SetText(cont.Status).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 3, ui.cellPool.Get().SetText(cpu).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 4, ui.cellPool.Get().SetText(mem).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 5, ui.cellPool.Get().SetText(name).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 6, ui.cellPool.Get().SetText(strings.Join(portStrs, ",")).SetTextColor(color))
}

// removeRow 删除行
func (ui *ContainerUI) removeRow(rowIndex int) {
	ui.common.Table.RemoveRow(rowIndex)
}

// recycleCells 回收表格中的所有单元格到对象池（性能优化：减少内存分配）
func (ui *ContainerUI) recycleCells() {
	rowCount := ui.common.Table.GetRowCount()
	colCount := ui.common.Table.GetColumnCount()

	// 回收所有单元格（跳过表头行）
	for row := 1; row < rowCount; row++ {
		for col := 0; col < colCount; col++ {
			if cell := ui.common.Table.GetCell(row, col); cell != nil {
				ui.cellPool.Put(cell)
			}
		}
	}
}

// updateRow 更新现有行
func (ui *ContainerUI) updateRow(cont types.Container, rowIndex int) {
	// 安全获取容器ID前12位，避免越界
	id := cont.ID
	if len(id) > config.ContainerIDLength {
		id = id[:config.ContainerIDLength]
	}

	color := tcell.ColorWhite
	cpu, mem := "-", "-"

	if strings.HasPrefix(cont.Status, "Up") {
		color = tcell.ColorGreen
		cpu, mem = "...", "..." // 占位符，表示正在获取
	}

	var portStrs []string
	for _, p := range cont.Ports {
		if p.PublicPort != 0 {
			portStrs = append(portStrs, fmt.Sprintf("%d->%d", p.PublicPort, p.PrivatePort))
		} else {
			portStrs = append(portStrs, fmt.Sprintf("%d", p.PrivatePort))
		}
	}

	name := ""
	if len(cont.Names) > 0 {
		name = strings.TrimPrefix(cont.Names[0], "/")
	}

	// 使用对象池获取TableCell（性能优化：减少内存分配）
	ui.common.Table.SetCell(rowIndex, 0, ui.cellPool.Get().SetText(id).SetTextColor(color).SetReference(cont.ID))
	ui.common.Table.SetCell(rowIndex, 1, ui.cellPool.Get().SetText(cont.Image).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 2, ui.cellPool.Get().SetText(cont.Status).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 3, ui.cellPool.Get().SetText(cpu).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 4, ui.cellPool.Get().SetText(mem).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 5, ui.cellPool.Get().SetText(name).SetTextColor(color))
	ui.common.Table.SetCell(rowIndex, 6, ui.cellPool.Get().SetText(strings.Join(portStrs, ",")).SetTextColor(color))
}

// RefreshListIncremental 增量更新表格（性能优化）
func (ui *ContainerUI) RefreshListIncremental() {
	// 增加刷新版本号（修复竞态条件问题）
	currentVersion := ui.refreshVersion.Add(1)

	// 获取容器列表（快速）
	list, err := ui.docker.ListContainers(true)
	if err != nil {
		ui.common.ShowError(fmt.Sprintf("Failed to list containers: %v", err), nil)
		return
	}

	// 保存选中状态
	selRow, _ := ui.common.Table.GetSelection()
	var selectedID string
	if selRow > 0 && selRow < ui.common.Table.GetRowCount() {
		cell := ui.common.Table.GetCell(selRow, 0)
		if cell != nil {
			selectedID = fmt.Sprintf("%v", cell.GetReference())
		}
	}

	// 判断是否需要全量刷新
	needFullRefresh := ui.needFullRefresh(list)

	if needFullRefresh {
		// 执行全量刷新
		ui.fullRefresh(list, currentVersion)
	} else {
		// 执行增量更新
		ui.incrementalRefresh(list, currentVersion)
	}

	// 更新缓存
	ui.updateCache(list)

	// 恢复选中状态
	ui.restoreSelectionByID(selectedID, list)
}

// needFullRefresh 判断是否需要全量刷新（优化阈值）
func (ui *ContainerUI) needFullRefresh(newList []types.Container) bool {
	ui.containerListMutex.RLock()
	defer ui.containerListMutex.RUnlock()

	// 首次刷新，需要全量刷新
	if ui.lastContainerList == nil || ui.lastContainerMap == nil {
		return true
	}

	// 容器数量变化超过50%，执行全量刷新（优化阈值，从20%提高到50%）
	oldCount := len(ui.lastContainerList)
	newCount := len(newList)
	if oldCount == 0 {
		return true
	}

	changeRatio := float64(abs(newCount-oldCount)) / float64(oldCount)
	// 优化：阈值从20%提高到50%，减少频繁全量刷新
	if changeRatio > 0.5 {
		return true
	}

	// 检查是否有删除或添加操作（如果有，执行全量刷新以保持顺序）
	newIDSet := make(map[string]bool)
	for _, cont := range newList {
		newIDSet[cont.ID] = true
	}

	for _, cont := range ui.lastContainerList {
		if !newIDSet[cont.ID] {
			// 有删除操作，执行全量刷新
			return true
		}
	}

	oldIDSet := make(map[string]bool)
	for _, cont := range ui.lastContainerList {
		oldIDSet[cont.ID] = true
	}

	for _, cont := range newList {
		if !oldIDSet[cont.ID] {
			// 有添加操作，执行全量刷新
			return true
		}
	}

	return false
}

// abs 返回绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// fullRefresh 全量刷新表格
func (ui *ContainerUI) fullRefresh(list []types.Container, version int64) {
	// 回收旧的TableCell对象到对象池（性能优化：减少内存分配）
	ui.recycleCells()

	ui.common.Table.Clear()

	// 设置表头
	headers := []string{"ID", "IMAGE", "STATUS", "CPU", "MEM", "NAMES", "PORTS"}
	expansions := []int{1, 3, 2, 1, 1, 2, 3}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(expansions[i])
		ui.common.Table.SetCell(0, i, cell)
	}

	// 添加所有容器行
	for i, cont := range list {
		ui.addRow(cont, i+1)
	}

	// 异步获取stats并更新
	go ui.updateStatsAsync(list, version)
}

// incrementalRefresh 增量更新表格（仅处理状态变化）
func (ui *ContainerUI) incrementalRefresh(list []types.Container, version int64) {
	// 对比容器变化
	changes := ui.compareContainerChanges(list)

	// 如果没有变化，直接返回
	if len(changes.updated) == 0 {
		return
	}

	// 构建容器ID到行索引的映射
	containerMap := ui.buildContainerMap(list)

	// 处理更新的容器
	for _, cont := range changes.updated {
		if rowIndex, exists := containerMap[cont.ID]; exists {
			ui.updateRow(cont, rowIndex)
		}
	}

	// 异步获取stats并更新（只更新变化的容器）
	if len(changes.updated) > 0 {
		go ui.updateStatsAsync(changes.updated, version)
	}
}

// updateCache 更新容器列表缓存
func (ui *ContainerUI) updateCache(list []types.Container) {
	ui.containerListMutex.Lock()
	defer ui.containerListMutex.Unlock()

	ui.lastContainerList = list
	ui.lastContainerMap = ui.buildContainerMap(list)
}

// restoreSelectionByID 通过容器ID恢复选中状态
func (ui *ContainerUI) restoreSelectionByID(selectedID string, list []types.Container) {
	if selectedID == "" {
		return
	}

	// 查找容器ID对应的行索引
	for i, cont := range list {
		if cont.ID == selectedID {
			ui.common.Table.Select(i+1, 0)
			return
		}
	}

	// 如果找不到，选择第一行
	if ui.common.Table.GetRowCount() > 1 {
		ui.common.Table.Select(1, 0)
	}
}

// ShowInspect 显示容器详细信息
func (ui *ContainerUI) ShowInspect(containerID string) {
	ui.common.Header.SetText(fmt.Sprintf("\n[white::b]%s[-:-:-] [yellow::] (Inspect: %s)[-:-:-]",
		version.GetVersionString(), containerID))

	data, err := ui.docker.InspectContainer(containerID)
	if err != nil {
		ui.common.ShowError(fmt.Sprintf("Failed to inspect container: %v", err), nil)
		return
	}

	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		ui.common.ShowError(fmt.Sprintf("Failed to format inspect data: %v", err), nil)
		return
	}
	
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	view.SetBorder(true).SetTitle(" Inspect Detail (ESC Return) ")
	view.SetText(string(pretty))

	ui.common.Pages.AddPage("inspect", view, true, true)
	ui.common.App.SetFocus(view)
}

// ShowLogs 显示容器日志
func (ui *ContainerUI) ShowLogs(containerID string) {
	ui.common.Header.SetText(fmt.Sprintf("\n[white::b]%s[-:-:-] [yellow::] (Logs: %s)[-:-:-]",
		version.GetVersionString(), containerID))

	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	view.SetBorder(true).SetTitle(" Container Logs (ESC Return) ")

	logs, err := ui.docker.GetContainerLogs(containerID, config.LogTailLines)
	if err != nil {
		ui.common.ShowError(fmt.Sprintf("Failed to get container logs: %v", err), nil)
		return
	}

	go func() {
		defer logs.Close()
		if _, err := io.Copy(tview.ANSIWriter(view), logs); err != nil {
			// 在主线程中显示错误提示（修复错误处理不完善问题）
			ui.common.App.QueueUpdateDraw(func() {
				// 检查日志视图是否仍然存在
				if front, _ := ui.common.Pages.GetFrontPage(); front == "log" {
					// 在日志视图中添加错误提示
					fmt.Fprintf(view, "\n[red::b]❌ Error reading logs: %v[-:-:-]\n", err)
				}
			})
		}
	}()

	ui.common.Pages.AddPage("log", view, true, true)
	ui.common.App.SetFocus(view)
}

// Start 启动容器
func (ui *ContainerUI) Start() {
	if ui.common.GetSelectedID() == "" {
		return
	}
	ui.common.RunAsyncAction("Starting...",
		func() {
			if err := ui.docker.StartContainer(ui.common.GetSelectedID()); err != nil {
				ui.common.App.QueueUpdateDraw(func() {
					ui.common.ShowError(fmt.Sprintf("Failed to start container: %v", err), nil)
				})
			}
		},
		ui.RefreshList)
}

// Stop 停止容器
func (ui *ContainerUI) Stop() {
	if ui.common.GetSelectedID() == "" {
		return
	}
	ui.common.ShowConfirm("Stop container "+ui.common.GetSelectedID()+"?",
		func() {
			ui.common.RunAsyncAction("Stopping...",
				func() {
					if err := ui.docker.StopContainer(ui.common.GetSelectedID()); err != nil {
						ui.common.App.QueueUpdateDraw(func() {
							ui.common.ShowError(fmt.Sprintf("Failed to stop container: %v", err), nil)
						})
					}
				},
				ui.RefreshList)
		}, nil)
}

// Delete 删除容器
func (ui *ContainerUI) Delete() {
	if ui.common.GetSelectedID() == "" {
		return
	}
	ui.common.ShowConfirm("Delete container "+ui.common.GetSelectedID()+"?",
		func() {
			ui.common.RunAsyncAction("Deleting...",
				func() {
					if err := ui.docker.RemoveContainer(ui.common.GetSelectedID(), true); err != nil {
						ui.common.App.QueueUpdateDraw(func() {
							ui.common.ShowError(fmt.Sprintf("Failed to delete container: %v", err), nil)
						})
					}
				},
				ui.RefreshList)
		}, nil)
}
