// Package ui 用户界面组件
package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	// 这个包用于类型定义，虽然代码中没有直接使用，但编译时需要
	_ "github.com/docker/docker/api/types"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/wsl12105/docker-manager/internal/docker"
)

// ContainerUI 容器UI管理器
type ContainerUI struct {
	common *Common
	docker *docker.Client
}

// NewContainerUI 创建容器UI管理器
func NewContainerUI(common *Common, docker *docker.Client) *ContainerUI {
	return &ContainerUI{
		common: common,
		docker: docker,
	}
}

// RefreshList 刷新容器列表
func (ui *ContainerUI) RefreshList() {
	selRow, _ := ui.common.Table.GetSelection()
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

	// 获取容器列表
	list, err := ui.docker.ListContainers(true)
	if err != nil {
		return
	}

	for i, cont := range list {
		id := cont.ID[:12]
		color := tcell.ColorWhite
		cpu, mem := "-", "-"

		if strings.HasPrefix(cont.Status, "Up") {
			color = tcell.ColorGreen
			cpu, mem = ui.getStats(cont.ID)
		}

		// 处理端口信息
		var portStrs []string
		for _, p := range cont.Ports {
			if p.PublicPort != 0 {
				portStrs = append(portStrs, fmt.Sprintf("%d->%d", p.PublicPort, p.PrivatePort))
			} else {
				portStrs = append(portStrs, fmt.Sprintf("%d", p.PrivatePort))
			}
		}

		// 处理容器名称
		name := ""
		if len(cont.Names) > 0 {
			name = strings.TrimPrefix(cont.Names[0], "/")
		}

		// 设置单元格
		ui.common.Table.SetCell(i+1, 0, tview.NewTableCell(id).SetTextColor(color).SetReference(cont.ID))
		ui.common.Table.SetCell(i+1, 1, tview.NewTableCell(cont.Image).SetTextColor(color))
		ui.common.Table.SetCell(i+1, 2, tview.NewTableCell(cont.Status).SetTextColor(color))
		ui.common.Table.SetCell(i+1, 3, tview.NewTableCell(cpu).SetTextColor(color))
		ui.common.Table.SetCell(i+1, 4, tview.NewTableCell(mem).SetTextColor(color))
		ui.common.Table.SetCell(i+1, 5, tview.NewTableCell(name).SetTextColor(color))
		ui.common.Table.SetCell(i+1, 6, tview.NewTableCell(strings.Join(portStrs, ",")).SetTextColor(color))
	}

	// 恢复选择
	ui.restoreSelection(selRow)
}

// getStats 获取容器统计信息
func (ui *ContainerUI) getStats(containerID string) (string, string) {
	data, err := ui.docker.GetContainerStats(containerID)
	if err != nil {
		return "0.00%", "0MB"
	}

	memVal := 0.0
	if ms, ok := data["memory_stats"].(map[string]interface{}); ok {
		if u, ok := ms["usage"].(float64); ok {
			memVal = u / 1024 / 1024
		}
	}

	cpuP := 0.0
	cs, okCS := data["cpu_stats"].(map[string]interface{})
	ps, okPS := data["precpu_stats"].(map[string]interface{})
	if okCS && okPS {
		var curU, preU, curS, preS float64
		if u, ok := cs["cpu_usage"].(map[string]interface{}); ok {
			if v, ok := u["total_usage"].(float64); ok {
				curU = v
			}
		}
		if u, ok := ps["cpu_usage"].(map[string]interface{}); ok {
			if v, ok := u["total_usage"].(float64); ok {
				preU = v
			}
		}
		if v, ok := cs["system_cpu_usage"].(float64); ok {
			curS = v
		}
		if v, ok := ps["system_cpu_usage"].(float64); ok {
			preS = v
		}
		deltaC := curU - preU
		deltaS := curS - preS
		if deltaS > 0 && deltaC > 0 {
			cpuP = (deltaC / deltaS) * 100.0
		}
	}

	return fmt.Sprintf("%.2f%%", cpuP), fmt.Sprintf("%.1fMB", memVal)
}

// restoreSelection 恢复选择
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

// ShowInspect 显示容器详情
func (ui *ContainerUI) ShowInspect(containerID string) {
	ui.common.Header.SetText(fmt.Sprintf("\n[white::b]%s[-:-:-] [yellow::] (Inspect: %s)[-:-:-]",
		ui.common.AppName, containerID))

	data, err := ui.docker.InspectContainer(containerID)
	if err != nil {
		return
	}

	pretty, _ := json.MarshalIndent(data, "", "  ")
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
		ui.common.AppName, containerID))

	view := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	view.SetBorder(true).SetTitle(" Container Logs (ESC Return) ")

	logs, err := ui.docker.GetContainerLogs(containerID, "200")
	if err == nil {
		go func() {
			defer logs.Close()
			_, _ = io.Copy(tview.ANSIWriter(view), logs)
		}()
	}

	ui.common.Pages.AddPage("log", view, true, true)
	ui.common.App.SetFocus(view)
}

// Start 启动容器
func (ui *ContainerUI) Start() {
	if ui.common.SelectedID == "" {
		return
	}
	ui.common.RunAsyncAction("Starting...",
		func() { _ = ui.docker.StartContainer(ui.common.SelectedID) },
		ui.RefreshList)
}

// Stop 停止容器
func (ui *ContainerUI) Stop() {
	if ui.common.SelectedID == "" {
		return
	}
	ui.common.ShowConfirm("Stop container "+ui.common.SelectedID+"?",
		func() {
			ui.common.RunAsyncAction("Stopping...",
				func() { _ = ui.docker.StopContainer(ui.common.SelectedID) },
				ui.RefreshList)
		}, nil)
}

// Delete 删除容器
func (ui *ContainerUI) Delete() {
	if ui.common.SelectedID == "" {
		return
	}
	ui.common.ShowConfirm("Delete container "+ui.common.SelectedID+"?",
		func() {
			ui.common.RunAsyncAction("Deleting...",
				func() { _ = ui.docker.RemoveContainer(ui.common.SelectedID, true) },
				ui.RefreshList)
		}, nil)
}
