// Package ui UI组件包
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/wsl12105/docker-manager/internal/version"
)

// Common 公共UI组件
type Common struct {
	App         *tview.Application
	Pages       *tview.Pages
	Header      *tview.TextView
	InfoView    *tview.TextView
	Table       *tview.Table
	
	// 使用原子操作的字段（性能优化）
	selectedIDAtomic atomic.Value // 存储 string
	isOperatingAtomic atomic.Bool  // 存储 bool
	
	// 防抖相关字段
	debounceTimer    *time.Timer     // 防抖定时器
	debounceDelay    time.Duration   // 防抖延迟时间
	pendingSelection int             // 待处理的选中行
	debounceMutex    sync.Mutex      // 保护防抖定时器的锁
}

// NewCommon 创建公共UI组件
func NewCommon() *Common {
	c := &Common{
		App:            tview.NewApplication(),
		Pages:          tview.NewPages(),
		debounceDelay:  100 * time.Millisecond, // 防抖延迟100ms
	}

	c.Header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	c.Header.SetTextColor(tcell.ColorBlack).
		SetBackgroundColor(tcell.ColorGreen)

	c.InfoView = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	c.InfoView.SetBorder(true).SetTitle(" Command Help ")

	c.Table = tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 1)
	c.Table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorBlue).
		Foreground(tcell.ColorWhite))
	c.Table.SetBorder(true)

	c.resetHeader()

	c.Table.SetSelectionChangedFunc(func(row, col int) {
		if row > 0 {
			// 使用防抖机制，避免快速切换时的性能问题
			c.debounceMutex.Lock()
			
			// 取消之前的定时器
			if c.debounceTimer != nil {
				c.debounceTimer.Stop()
			}
			
			// 设置新的定时器（延迟处理）
			c.pendingSelection = row
			c.debounceTimer = time.AfterFunc(c.debounceDelay, func() {
				c.App.QueueUpdateDraw(func() {
					if cell := c.Table.GetCell(c.pendingSelection, 0); cell != nil {
						if ref := cell.GetReference(); ref != nil {
							// 安全的类型断言，避免panic
							if id, ok := ref.(string); ok {
								c.SetSelectedID(id)
							} else {
								c.SetSelectedID(cell.Text)
							}
						} else {
							c.SetSelectedID(cell.Text)
						}
					}
				})
			})
			
			c.debounceMutex.Unlock()
		}
	})

	return c
}

// SetSelectedID 设置选中的ID（使用原子操作，性能优化）
func (c *Common) SetSelectedID(id string) {
	c.selectedIDAtomic.Store(id)
}

// GetSelectedID 获取选中的ID（使用原子操作，性能优化，修复类型断言风险）
func (c *Common) GetSelectedID() string {
	if val := c.selectedIDAtomic.Load(); val != nil {
		// 使用安全的类型断言，避免panic（修复类型断言风险）
		if id, ok := val.(string); ok {
			return id
		}
		// 类型不匹配，返回空字符串（安全处理）
		return ""
	}
	return ""
}

// SetOperating 设置操作状态（使用原子操作，性能优化）
func (c *Common) SetOperating(operating bool) {
	c.isOperatingAtomic.Store(operating)
}

// GetOperating 获取操作状态（使用原子操作，性能优化）
func (c *Common) GetOperating() bool {
	return c.isOperatingAtomic.Load()
}

// GetVersionString 
func (c *Common) GetVersionString() string {
	return version.GetVersionString()
}

// resetHeader 
func (c *Common) resetHeader() {
	c.Header.SetText(fmt.Sprintf("\n[white::b]%s[-:-:-]", c.GetVersionString()))
}

// RunAsyncAction 执行异步操作
func (c *Common) RunAsyncAction(msg string, action func(), onComplete func()) {
	c.SetOperating(true)
	row, _ := c.Table.GetSelection()
	if row > 0 {
		c.Table.SetCell(row, 2, tview.NewTableCell("[yellow]"+msg))
	}

	go func() {
		action()
		c.App.QueueUpdateDraw(func() {
			c.SetOperating(false)
			if onComplete != nil {
				onComplete()
			}
		})
	}()
}

// ShowConfirm 
func (c *Common) ShowConfirm(message string, onConfirm func(), onCancel func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Cancel","OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "OK" && onConfirm != nil {
				onConfirm()
			} else if onCancel != nil {
				onCancel()
			}
			c.Pages.RemovePage("modal")
			c.App.SetFocus(c.Table)
		})
	c.Pages.AddPage("modal", modal, true, true)
	c.App.SetFocus(modal)
}

// ShowInput 
func (c *Common) ShowInput(label string, onSubmit func(string)) {
	form := tview.NewForm()
	input := tview.NewInputField().SetLabel(label).SetFieldWidth(30)

	form.AddFormItem(input).
		AddButton("OK", func() {
			onSubmit(input.GetText())
			c.Pages.RemovePage("input")
			c.App.SetFocus(c.Table)
		}).
		AddButton("Cancel", func() {
			c.Pages.RemovePage("input")
			c.App.SetFocus(c.Table)
		})

	form.SetBorder(true).SetTitle(" Input ")

	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 11, 1, true).
			AddItem(nil, 0, 1, false), 50, 1, true).
		AddItem(nil, 0, 1, false)

	c.Pages.AddPage("input", flex, true, true)
	c.App.SetFocus(form)
}

// RunExec 在容器中执行交互式shell
func (c *Common) RunExec(containerID string) {
	c.App.Suspend(func() {
		// 尝试多个常见的shell路径，提高兼容性
		shells := []string{"/bin/bash", "/bin/sh", "/bin/ash", "/usr/bin/bash", "/usr/bin/sh"}
		
		var cmd *exec.Cmd
		for _, shell := range shells {
			cmd = exec.Command("docker", "exec", "-it", containerID, shell)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			if err := cmd.Run(); err == nil {
				// 成功执行，退出
				return
			}
			// 如果shell不存在，尝试下一个
		}
		
		// 所有shell都失败，显示错误信息
		fmt.Printf("\n❌ Unable to execute shell in container %s\n", containerID)
		fmt.Println("Please check if the container has a valid shell installed")
		fmt.Println("Press Enter to continue...")
		os.Stdin.Read(make([]byte, 1))
	})
}

// ShowError 
func (c *Common) ShowError(message string, onClose func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			c.Pages.RemovePage("error")
			c.App.SetFocus(c.Table)
			if onClose != nil {
				onClose()
			}
		})
	c.Pages.AddPage("error", modal, true, true)
	c.App.SetFocus(modal)
}

// SetupInputCapture 设置输入捕获处理
func (c *Common) SetupInputCapture(handlers map[rune]func(), onReturn func()) {
	c.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Ctrl+C 退出
		if event.Key() == tcell.KeyCtrlC {
			c.App.Stop()
			return nil
		}

		front, _ := c.Pages.GetFrontPage()
		if front != "main" {
			if event.Key() == tcell.KeyEsc {
				c.Pages.RemovePage(front)
				c.resetHeader()
				c.App.SetFocus(c.Table)
				// 调用返回回调函数（重置查看详情状态）
				if onReturn != nil {
					onReturn()
				}
				return nil
			}
			return event
		}

		if c.GetOperating() {
			return nil
		}

		if handler, exists := handlers[event.Rune()]; exists {
			handler()
			return nil
		}

		return event
	})
}

// Cleanup 清理资源（修复定时器泄漏问题）
func (c *Common) Cleanup() {
	c.debounceMutex.Lock()
	if c.debounceTimer != nil {
		c.debounceTimer.Stop()
		c.debounceTimer = nil // 清理定时器引用，避免内存泄漏
	}
	c.debounceMutex.Unlock()
}
