// Package ui 
package ui

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/wsl12105/docker-manager/internal/version"
)

// Common 
type Common struct {
	App         *tview.Application
	Pages       *tview.Pages
	Header      *tview.TextView
	InfoView    *tview.TextView
	Table       *tview.Table
	IsOperating bool
	SelectedID  string
}

// NewCommon 
func NewCommon() *Common {
	c := &Common{
		App:   tview.NewApplication(),
		Pages: tview.NewPages(),
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
			if cell := c.Table.GetCell(row, 0); cell != nil {
				if ref := cell.GetReference(); ref != nil {
					c.SelectedID = ref.(string)
				} else {
					c.SelectedID = cell.Text
				}
			}
		}
	})

	return c
}

// GetVersionString 
func (c *Common) GetVersionString() string {
	return version.GetVersionString()
}

// resetHeader 
func (c *Common) resetHeader() {
	c.Header.SetText(fmt.Sprintf("\n[white::b]%s[-:-:-]", c.GetVersionString()))
}

// RunAsyncAction 
func (c *Common) RunAsyncAction(msg string, action func(), onComplete func()) {
	c.IsOperating = true
	row, _ := c.Table.GetSelection()
	if row > 0 {
		c.Table.SetCell(row, 2, tview.NewTableCell("[yellow]"+msg))
	}

	go func() {
		action()
		c.App.QueueUpdateDraw(func() {
			c.IsOperating = false
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

// RunExec 
func (c *Common) RunExec(containerID string) {
	c.App.Suspend(func() {
		cmd := exec.Command("docker", "exec", "-it", containerID, "/bin/sh")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
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

// SetupInputCapture 
func (c *Common) SetupInputCapture(handlers map[rune]func()) {
	c.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Ctrl+C 
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
				return nil
			}
			return event
		}

		if c.IsOperating {
			return nil
		}

		if handler, exists := handlers[event.Rune()]; exists {
			handler()
			return nil
		}

		return event
	})
}
