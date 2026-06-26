// Package ui UI组件单元测试
package ui

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ==================== LRUCache 单元测试 ====================

// TestLRUCache_New 测试创建LRU缓存
func TestLRUCache_New(t *testing.T) {
	cache := NewLRUCache(10)
	if cache == nil {
		t.Fatal("创建LRU缓存失败")
	}
	if cache.capacity != 10 {
		t.Errorf("缓存容量错误: 期望10, 实际%d", cache.capacity)
	}
	if cache.Size() != 0 {
		t.Errorf("初始缓存大小错误: 期望0, 实际%d", cache.Size())
	}
}

// TestLRUCache_PutAndGet 测试Put和Get操作
func TestLRUCache_PutAndGet(t *testing.T) {
	cache := NewLRUCache(3)

	// 添加缓存条目
	entry1 := &StatsCacheEntry{CPU: "10.5%", Memory: "50MB", Timestamp: time.Now()}
	cache.Put("key1", entry1)

	entry2 := &StatsCacheEntry{CPU: "20.5%", Memory: "100MB", Timestamp: time.Now()}
	cache.Put("key2", entry2)

	// 测试Get操作
	if val, ok := cache.Get("key1"); !ok {
		t.Error("获取key1失败")
	} else if val.CPU != "10.5%" {
		t.Errorf("key1的CPU值错误: 期望10.5%, 实际%s", val.CPU)
	}

	if val, ok := cache.Get("key2"); !ok {
		t.Error("获取key2失败")
	} else if val.Memory != "100MB" {
		t.Errorf("key2的Memory值错误: 期望100MB, 实际%s", val.Memory)
	}

	// 测试不存在的key
	if _, ok := cache.Get("key3"); ok {
		t.Error("获取不存在的key3应该失败")
	}
}

// TestLRUCache_Capacity 测试缓存容量限制
func TestLRUCache_Capacity(t *testing.T) {
	cache := NewLRUCache(3)

	// 添加4个条目（超过容量）
	for i := 1; i <= 4; i++ {
		entry := &StatsCacheEntry{
			CPU:       "10%",
			Memory:    "50MB",
			Timestamp: time.Now(),
		}
		cache.Put(fmt.Sprintf("key%d", i), entry)
	}

	// 缓存大小应该是3（容量限制）
	if cache.Size() != 3 {
		t.Errorf("缓存大小错误: 期望3, 实际%d", cache.Size())
	}

	// key1应该被淘汰（最久未使用）
	if _, ok := cache.Get("key1"); ok {
		t.Error("key1应该被淘汰")
	}

	// key2, key3, key4应该存在
	for i := 2; i <= 4; i++ {
		if _, ok := cache.Get(fmt.Sprintf("key%d", i)); !ok {
			t.Errorf("key%d应该存在", i)
		}
	}
}

// TestLRUCache_Update 测试更新缓存条目
func TestLRUCache_Update(t *testing.T) {
	cache := NewLRUCache(3)

	// 添加初始条目
	entry1 := &StatsCacheEntry{CPU: "10%", Memory: "50MB", Timestamp: time.Now()}
	cache.Put("key1", entry1)

	// 更新条目
	entry1Updated := &StatsCacheEntry{CPU: "20%", Memory: "100MB", Timestamp: time.Now()}
	cache.Put("key1", entry1Updated)

	// 验证更新
	if val, ok := cache.Get("key1"); !ok {
		t.Error("获取key1失败")
	} else if val.CPU != "20%" {
		t.Errorf("key1的CPU值错误: 期望20%, 实际%s", val.CPU)
	}

	// 缓存大小应该是1（更新不增加大小）
	if cache.Size() != 1 {
		t.Errorf("缓存大小错误: 期望1, 实际%d", cache.Size())
	}
}

// TestLRUCache_Clear 测试清空缓存
func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRUCache(3)

	// 添加条目
	for i := 1; i <= 3; i++ {
		entry := &StatsCacheEntry{CPU: "10%", Memory: "50MB", Timestamp: time.Now()}
		cache.Put(fmt.Sprintf("key%d", i), entry)
	}

	// 清空缓存
	cache.Clear()

	// 验证缓存已清空
	if cache.Size() != 0 {
		t.Errorf("清空后缓存大小错误: 期望0, 实际%d", cache.Size())
	}

	// 所有key应该不存在
	for i := 1; i <= 3; i++ {
		if _, ok := cache.Get(fmt.Sprintf("key%d", i)); ok {
			t.Errorf("key%d应该不存在", i)
		}
	}
}

// TestLRUCache_Concurrent 测试并发安全性
func TestLRUCache_Concurrent(t *testing.T) {
	cache := NewLRUCache(100)
	var wg sync.WaitGroup

	// 并发写入
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			entry := &StatsCacheEntry{
				CPU:       "10%",
				Memory:    "50MB",
				Timestamp: time.Now(),
			}
			cache.Put(fmt.Sprintf("key%d", id), entry)
		}(i)
	}

	// 并发读取
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cache.Get(fmt.Sprintf("key%d", id))
		}(i)
	}

	wg.Wait()

	// 验证缓存大小不超过容量
	if cache.Size() > 100 {
		t.Errorf("缓存大小超过容量: %d", cache.Size())
	}
}

// ==================== TableCellPool 单元测试 ====================

// TestTableCellPool_New 测试创建TableCell对象池
func TestTableCellPool_New(t *testing.T) {
	pool := NewTableCellPool()
	if pool == nil {
		t.Fatal("创建TableCell对象池失败")
	}
}

// TestTableCellPool_GetAndPut 测试Get和Put操作
func TestTableCellPool_GetAndPut(t *testing.T) {
	pool := NewTableCellPool()

	// 从池中获取TableCell
	cell1 := pool.Get()
	if cell1 == nil {
		t.Fatal("从池中获取TableCell失败")
	}

	// 设置单元格属性
	cell1.SetText("test").SetTextColor(tcell.ColorGreen)

	// 放回池中
	pool.Put(cell1)

	// 再次获取（应该重置状态）
	cell2 := pool.Get()
	if cell2 == nil {
		t.Fatal("再次从池中获取TableCell失败")
	}

	// 验证状态已重置
	if cell2.GetText() != "" {
		t.Errorf("单元格文本未重置: 期望空字符串, 实际%s", cell2.GetText())
	}
	if cell2.GetTextColor() != tcell.ColorWhite {
		t.Errorf("单元格颜色未重置: 期望白色, 实际%v", cell2.GetTextColor())
	}
}

// TestTableCellPool_MultipleGet 测试多次获取
func TestTableCellPool_MultipleGet(t *testing.T) {
	pool := NewTableCellPool()

	// 多次获取TableCell
	cells := make([]*tview.TableCell, 10)
	for i := 0; i < 10; i++ {
		cells[i] = pool.Get()
		if cells[i] == nil {
			t.Errorf("第%d次获取TableCell失败", i)
		}
	}

	// 所有单元格应该可用
	for i := 0; i < 10; i++ {
		if cells[i] == nil {
			t.Errorf("第%d个单元格不可用", i)
		}
	}

	// 放回池中
	for i := 0; i < 10; i++ {
		pool.Put(cells[i])
	}
}

// TestTableCellPool_Concurrent 测试并发安全性
func TestTableCellPool_Concurrent(t *testing.T) {
	pool := NewTableCellPool()
	var wg sync.WaitGroup

	// 并发获取和放回
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cell := pool.Get()
			if cell != nil {
				// 模拟使用
				cell.SetText("test")
				// 放回池中
				pool.Put(cell)
			}
		}()
	}

	wg.Wait()
}

// TestTableCellPool_PutNil 测试放回nil值
func TestTableCellPool_PutNil(t *testing.T) {
	pool := NewTableCellPool()

	// 放回nil值（应该被忽略）
	pool.Put(nil)

	// 验证池仍然可用
	cell := pool.Get()
	if cell == nil {
		t.Error("放回nil后池应该仍然可用")
	}
}

// ==================== 边界场景测试 ====================

// TestLRUCache_EmptyCapacity 测试容量为0的缓存
func TestLRUCache_EmptyCapacity(t *testing.T) {
	cache := NewLRUCache(0)

	// 添加条目（应该被立即淘汰）
	entry := &StatsCacheEntry{CPU: "10%", Memory: "50MB", Timestamp: time.Now()}
	cache.Put("key1", entry)

	// 缓存大小应该是0
	if cache.Size() != 0 {
		t.Errorf("容量为0的缓存大小错误: 期望0, 实际%d", cache.Size())
	}

	// key1应该不存在
	if _, ok := cache.Get("key1"); ok {
		t.Error("容量为0的缓存不应该存储任何条目")
	}
}

// TestLRUCache_SingleCapacity 测试容量为1的缓存
func TestLRUCache_SingleCapacity(t *testing.T) {
	cache := NewLRUCache(1)

	// 添加第一个条目
	entry1 := &StatsCacheEntry{CPU: "10%", Memory: "50MB", Timestamp: time.Now()}
	cache.Put("key1", entry1)

	// 添加第二个条目（应该淘汰第一个）
	entry2 := &StatsCacheEntry{CPU: "20%", Memory: "100MB", Timestamp: time.Now()}
	cache.Put("key2", entry2)

	// 缓存大小应该是1
	if cache.Size() != 1 {
		t.Errorf("容量为1的缓存大小错误: 期望1, 实际%d", cache.Size())
	}

	// key1应该被淘汰
	if _, ok := cache.Get("key1"); ok {
		t.Error("key1应该被淘汰")
	}

	// key2应该存在
	if _, ok := cache.Get("key2"); !ok {
		t.Error("key2应该存在")
	}
}

// TestLRUCache_LargeCapacity 测试大容量缓存
func TestLRUCache_LargeCapacity(t *testing.T) {
	cache := NewLRUCache(1000)

	// 添加1000个条目
	for i := 0; i < 1000; i++ {
		entry := &StatsCacheEntry{
			CPU:       "10%",
			Memory:    "50MB",
			Timestamp: time.Now(),
		}
		cache.Put(fmt.Sprintf("key%d", i), entry)
	}

	// 缓存大小应该是1000
	if cache.Size() != 1000 {
		t.Errorf("大容量缓存大小错误: 期望1000, 实际%d", cache.Size())
	}
}

// ==================== 性能基准测试 ====================

// BenchmarkLRUCache_Put 测试Put操作性能
func BenchmarkLRUCache_Put(b *testing.B) {
	cache := NewLRUCache(100)
	entry := &StatsCacheEntry{CPU: "10%", Memory: "50MB", Timestamp: time.Now()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%100)
		cache.Put(key, entry)
	}
}

// BenchmarkLRUCache_Get 测试Get操作性能
func BenchmarkLRUCache_Get(b *testing.B) {
	cache := NewLRUCache(100)

	// 预填充缓存
	for i := 0; i < 100; i++ {
		entry := &StatsCacheEntry{CPU: "10%", Memory: "50MB", Timestamp: time.Now()}
		key := fmt.Sprintf("key%d", i)
		cache.Put(key, entry)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%100)
		cache.Get(key)
	}
}

// BenchmarkTableCellPool_Get 测试Get操作性能
func BenchmarkTableCellPool_Get(b *testing.B) {
	pool := NewTableCellPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cell := pool.Get()
		pool.Put(cell)
	}
}

// BenchmarkLRUCache_Concurrent 测试并发性能
func BenchmarkLRUCache_Concurrent(b *testing.B) {
	cache := NewLRUCache(1000)
	entry := &StatsCacheEntry{CPU: "10%", Memory: "50MB", Timestamp: time.Now()}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%100)
			if i%2 == 0 {
				cache.Put(key, entry)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}