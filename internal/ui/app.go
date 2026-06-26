package ui

import (
	"strings"
	"sync/atomic"
	"time"

	"github.com/rivo/tview"
	"github.com/wsl12105/docker-manager/internal/docker"
	"github.com/wsl12105/docker-manager/internal/monitor"
)

// App 应用主结构
type App struct {
	common      *Common
	docker      *docker.Client
	containerUI *ContainerUI
	imageUI     *ImageUI
	currentMode string
	stopChan    chan struct{} // 用于停止刷新循环的信号通道

	// 性能监控器
	performanceMonitor *monitor.PerformanceMonitor

	// 智能刷新相关字段
	currentRefreshInterval time.Duration // 当前刷新间隔
	minRefreshInterval     time.Duration // 最小刷新间隔
	maxRefreshInterval     time.Duration // 最大刷新间隔
	baseRefreshInterval    time.Duration // 基础刷新间隔

	// 用户操作状态跟踪（智能刷新优化）
	userActive      atomic.Bool  // 用户是否活跃（正在操作）
	lastUserAction  atomic.Int64 // 上次用户操作时间戳
	viewingDetails  atomic.Bool  // 是否正在查看详情（日志、inspect等）
}

// NewApp 创建应用实例
func NewApp(dockerClient *docker.Client) *App {
	common := NewCommon()
	app := &App{
		common:                common,
		docker:                dockerClient,
		containerUI:           NewContainerUI(common, dockerClient),
		imageUI:               NewImageUI(common, dockerClient),
		stopChan:              make(chan struct{}), // 初始化停止通道
		performanceMonitor:    monitor.NewPerformanceMonitor(), // 初始化性能监控器

		// 初始化智能刷新参数
		minRefreshInterval:     1 * time.Second,   // 最小刷新间隔（大量运行容器）
		maxRefreshInterval:     10 * time.Second,  // 最大刷新间隔（无运行容器）
		baseRefreshInterval:    2 * time.Second,   // 基础刷新间隔（正常情况）
		currentRefreshInterval: 2 * time.Second,   // 初始刷新间隔
	}

	app.setupUI()
	app.setupHandlers()
	app.startSmartRefreshLoop()

	return app
}

// setupUI 
func (a *App) setupUI() {
	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.common.InfoView, 6, 0, false).
		AddItem(a.common.Table, 0, 1, true)

	a.common.Pages.AddPage("main", mainFlex, true, true)

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.common.Header, 3, 0, false).
		AddItem(a.common.Pages, 0, 1, true)

	a.common.App.SetRoot(layout, true)
}

// setupHandlers 设置键盘事件处理器
func (a *App) setupHandlers() {
	// 使用辅助函数避免代码重复，并设置用户活跃状态
	handlerWrapper := func(fn func()) func() {
		return func() {
			// 设置用户活跃状态
			a.SetUserActive(true)
			// 执行实际的处理函数
			fn()
		}
	}

	// 定义所有按键处理器（大小写统一处理）
	handlers := map[rune]func(){
		'c': handlerWrapper(a.switchToContainerMode),
		'C': handlerWrapper(a.switchToContainerMode),
		'm': handlerWrapper(a.switchToImageMode),
		'M': handlerWrapper(a.switchToImageMode),
		'i': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				// 设置查看详情状态
				a.SetViewingDetails(true)
				a.containerUI.ShowInspect(a.common.GetSelectedID())
			}
		}),
		'I': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				// 设置查看详情状态
				a.SetViewingDetails(true)
				a.containerUI.ShowInspect(a.common.GetSelectedID())
			}
		}),
		'l': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				// 设置查看详情状态
				a.SetViewingDetails(true)
				a.containerUI.ShowLogs(a.common.GetSelectedID())
			}
		}),
		'L': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				// 设置查看详情状态
				a.SetViewingDetails(true)
				a.containerUI.ShowLogs(a.common.GetSelectedID())
			}
		}),
		'e': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				a.common.RunExec(a.common.GetSelectedID())
			}
		}),
		'E': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				a.common.RunExec(a.common.GetSelectedID())
			}
		}),
		'r': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				a.containerUI.Start()
			}
		}),
		'R': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				a.containerUI.Start()
			}
		}),
		's': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				a.containerUI.Stop()
			}
		}),
		'S': handlerWrapper(func() {
			if a.currentMode == "container" && a.common.GetSelectedID() != "" {
				a.containerUI.Stop()
			}
		}),
		'd': handlerWrapper(func() {
			if a.common.GetSelectedID() != "" {
				if a.currentMode == "container" {
					a.containerUI.Delete()
				} else if a.currentMode == "image" {
					a.imageUI.Delete()
				}
			}
		}),
		'D': handlerWrapper(func() {
			if a.common.GetSelectedID() != "" {
				if a.currentMode == "container" {
					a.containerUI.Delete()
				} else if a.currentMode == "image" {
					a.imageUI.Delete()
				}
			}
		}),
		't': handlerWrapper(func() {
			if a.currentMode == "image" && a.common.GetSelectedID() != "" {
				a.imageUI.ShowTagInput()
			}
		}),
		'T': handlerWrapper(func() {
			if a.currentMode == "image" && a.common.GetSelectedID() != "" {
				a.imageUI.ShowTagInput()
			}
		}),
	}

	// 定义返回回调函数（重置查看详情状态）
	onReturn := func() {
		a.SetViewingDetails(false)
	}

	a.common.SetupInputCapture(handlers, onReturn)
}

// startSmartRefreshLoop 启动智能自动刷新循环（性能优化，修复调整时机问题）
func (a *App) startSmartRefreshLoop() {
	go func() {
		for {
			select {
			case <-time.After(a.currentRefreshInterval):
				// 在后台线程调整刷新频率（修复调整时机问题）
				a.adjustRefreshInterval()
				
				a.common.App.QueueUpdateDraw(func() {
					front, _ := a.common.Pages.GetFrontPage()
					if front == "main" && !a.common.GetOperating() {
						// 执行刷新
						if a.currentMode == "container" {
							a.containerUI.RefreshList()
						} else {
							a.imageUI.RefreshList()
						}
					}
				})
			case <-a.stopChan:
				// 收到停止信号，退出循环
				return
			}
		}
	}()
}

// adjustRefreshInterval 根据容器状态智能调整刷新间隔（性能优化：避免重复调用）
func (a *App) adjustRefreshInterval() {
	// 只在容器模式下调整刷新频率
	if a.currentMode != "container" {
		a.currentRefreshInterval = a.baseRefreshInterval
		return
	}

	// 智能刷新策略：根据用户状态调整刷新频率
	// 优先级：查看详情 > 用户操作 > 容器状态

	// 1. 如果用户正在查看详情（日志、inspect等），几乎暂停刷新
	if a.viewingDetails.Load() {
		a.currentRefreshInterval = 100 * time.Second
		return
	}

	// 2. 如果用户正在操作，降低刷新频率
	if a.userActive.Load() {
		// 检查用户操作是否超过5秒，超过则认为用户不再活跃
		lastAction := a.lastUserAction.Load()
		if time.Now().Unix()-lastAction > 5 {
			a.userActive.Store(false)
		} else {
			// 用户正在操作，使用较慢的刷新频率
			a.currentRefreshInterval = 6 * time.Second
			return
		}
	}

	// 3. 根据容器状态动态调整刷新频率（原有逻辑）
	// 使用缓存的容器状态统计，避免重复调用ListContainers
	// 容器状态统计会在RefreshList时更新，这里直接使用
	runningCount, stoppedCount := a.getContainerStatsCached()

	// 根据容器状态动态调整刷新频率
	if runningCount > 10 {
		// 大量运行容器，快速刷新以保持实时性
		a.currentRefreshInterval = a.minRefreshInterval
	} else if runningCount == 0 {
		// 无运行容器，慢速刷新以节省资源
		a.currentRefreshInterval = a.maxRefreshInterval
	} else if stoppedCount > runningCount {
		// 多停止容器，中速刷新
		a.currentRefreshInterval = a.baseRefreshInterval * 2
	} else {
		// 正常情况，基础刷新
		a.currentRefreshInterval = a.baseRefreshInterval
	}
}

// getContainerStatsCached 获取缓存的容器状态统计（性能优化：避免重复调用）
func (a *App) getContainerStatsCached() (int, int) {
	// 从表格中统计容器状态，避免额外的API调用
	running := 0
	stopped := 0

	for i := 1; i < a.common.Table.GetRowCount(); i++ {
		cell := a.common.Table.GetCell(i, 2) // Status列
		if cell != nil {
			status := cell.Text
			if strings.HasPrefix(status, "Up") {
				running++
			} else {
				stopped++
			}
		}
	}

	return running, stopped
}

// getContainerStats 获取容器状态统计（保留用于初始化）
func (a *App) getContainerStats() (int, int) {
	list, err := a.docker.ListContainers(true)
	if err != nil {
		return 0, 0
	}

	running := 0
	stopped := 0

	for _, cont := range list {
		if strings.HasPrefix(cont.Status, "Up") {
			running++
		} else {
			stopped++
		}
	}

	return running, stopped
}

// switchToContainerMode 
func (a *App) switchToContainerMode() {
	a.currentMode = "container"
	a.common.Table.SetTitle(" Containers List ")
	a.common.InfoView.SetText(" [yellow]Manage Container:[white]\n [blue::b]<i>[-:-:-] Inspect [blue::b]<l>[-:-:-] Logs [blue::b]<s>[-:-:-] Stop [blue::b]<r>[-:-:-] Restart [blue::b]<e>[-:-:-] Exec [blue::b]<d>[-:-:-] Delete [blue::b]<m>[-:-:-] Manage Image [red::b]Ctrl+C EXIT")
	a.containerUI.RefreshList()
}

// switchToImageMode 
func (a *App) switchToImageMode() {
	a.currentMode = "image"
	a.common.Table.SetTitle(" Images List ")
	a.common.InfoView.SetText(" [yellow]Manage Image:[white]\n [blue::b]<d>[-:-:-] Delete [blue::b]<t>[-:-:-] Tag [blue::b]<c>[-:-:-] Container [red::b]Ctrl+C EXIT")
	a.imageUI.RefreshList()
}

// Run 运行应用
func (a *App) Run() error {
	a.switchToContainerMode()
	return a.common.App.Run()
}

// SetUserActive 设置用户活跃状态（智能刷新优化）
func (a *App) SetUserActive(active bool) {
	a.userActive.Store(active)
	if active {
		a.lastUserAction.Store(time.Now().Unix())
	}
}

// SetViewingDetails 设置查看详情状态（智能刷新优化）
func (a *App) SetViewingDetails(viewing bool) {
	a.viewingDetails.Store(viewing)
}

// Stop 停止应用，清理资源
func (a *App) Stop() {
	// 清理防抖定时器（修复定时器泄漏问题）
	a.common.Cleanup()

	// 清理ContainerUI的资源（修复goroutine泄漏问题）
	a.containerUI.Cleanup()

	// 停止性能监控器
	if a.performanceMonitor != nil {
		a.performanceMonitor.Stop()
	}

	// 发送停止信号，终止刷新循环
	close(a.stopChan)
}
