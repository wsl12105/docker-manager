// Package ui 用户界面主包
package ui

import (
	"time"

	"github.com/rivo/tview"
	"github.com/wsl12105/docker-manager/internal/docker"
)

// App 应用程序主结构
type App struct {
	common      *Common
	docker      *docker.Client
	containerUI *ContainerUI
	imageUI     *ImageUI
	currentMode string
}

// NewApp 创建新应用
func NewApp(dockerClient *docker.Client) *App {
	common := NewCommon("DM (Docker Manager)", "v1.4.9")
	app := &App{
		common:      common,
		docker:      dockerClient,
		containerUI: NewContainerUI(common, dockerClient),
		imageUI:     NewImageUI(common, dockerClient),
	}

	app.setupUI()
	app.setupHandlers()
	app.startRefreshLoop()

	return app
}

// setupUI 设置UI布局
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

// setupHandlers 设置事件处理器
func (a *App) setupHandlers() {
	handlers := map[rune]func(){
		'c': a.switchToContainerMode,
		'C': a.switchToContainerMode,
		'm': a.switchToImageMode,
		'M': a.switchToImageMode,
		'i': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.containerUI.ShowInspect(a.common.SelectedID)
			}
		},
		'I': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.containerUI.ShowInspect(a.common.SelectedID)
			}
		},
		'l': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.containerUI.ShowLogs(a.common.SelectedID)
			}
		},
		'L': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.containerUI.ShowLogs(a.common.SelectedID)
			}
		},
		'e': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.common.RunExec(a.common.SelectedID)
			}
		},
		'E': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.common.RunExec(a.common.SelectedID)
			}
		},
		'r': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.containerUI.Start()
			}
		},
		'R': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.containerUI.Start()
			}
		},
		's': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.containerUI.Stop()
			}
		},
		'S': func() {
			if a.currentMode == "container" && a.common.SelectedID != "" {
				a.containerUI.Stop()
			}
		},
		'd': func() {
			if a.common.SelectedID != "" {
				if a.currentMode == "container" {
					a.containerUI.Delete()
				} else if a.currentMode == "image" {
					a.imageUI.Delete()
				}
			}
		},
		'D': func() {
			if a.common.SelectedID != "" {
				if a.currentMode == "container" {
					a.containerUI.Delete()
				} else if a.currentMode == "image" {
					a.imageUI.Delete()
				}
			}
		},
		't': func() {
			if a.currentMode == "image" && a.common.SelectedID != "" {
				a.imageUI.ShowTagInput()
			}
		},
		'T': func() {
			if a.currentMode == "image" && a.common.SelectedID != "" {
				a.imageUI.ShowTagInput()
			}
		},
	}

	a.common.SetupInputCapture(handlers)
}

// startRefreshLoop 启动刷新循环
func (a *App) startRefreshLoop() {
	go func() {
		for {
			time.Sleep(2 * time.Second)
			a.common.App.QueueUpdateDraw(func() {
				front, _ := a.common.Pages.GetFrontPage()
				if front == "main" && !a.common.IsOperating {
					if a.currentMode == "container" {
						a.containerUI.RefreshList()
					} else {
						a.imageUI.RefreshList()
					}
				}
			})
		}
	}()
}

// switchToContainerMode 切换到容器模式
func (a *App) switchToContainerMode() {
	a.currentMode = "container"
	a.common.Table.SetTitle(" Containers List ")
	a.common.InfoView.SetText(" [yellow]Manage Container:[white]\n [blue::b]<i>[-:-:-] Inspect [blue::b]<l>[-:-:-] Logs [blue::b]<s>[-:-:-] Stop [blue::b]<r>[-:-:-] Restart [blue::b]<e>[-:-:-] Exec [blue::b]<d>[-:-:-] Delete [blue::b]<m>[-:-:-] Manage Image [red::b]Ctrl+C EXIT")
	a.containerUI.RefreshList()
}

// switchToImageMode 切换到镜像模式
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
