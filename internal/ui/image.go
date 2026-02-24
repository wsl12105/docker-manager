// Package ui 用户界面组件
package ui

import (
	"fmt"
	"strings"

	// 这个包用于类型定义，虽然代码中没有直接使用，但编译时需要
	_ "github.com/docker/docker/api/types/image"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/wsl12105/docker-manager/internal/docker"
)

// ImageUI 镜像UI管理器
type ImageUI struct {
	common *Common
	docker *docker.Client
}

// NewImageUI 创建镜像UI管理器
func NewImageUI(common *Common, docker *docker.Client) *ImageUI {
	return &ImageUI{
		common: common,
		docker: docker,
	}
}

// RefreshList 刷新镜像列表
func (ui *ImageUI) RefreshList() {
	selRow, _ := ui.common.Table.GetSelection()
	ui.common.Table.Clear()

	// 设置表头
	headers := []string{"IMAGE ID", "REPOSITORY", "TAG", "SIZE"}
	for i, h := range headers {
		ui.common.Table.SetCell(0, i,
			tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetExpansion(1))
	}

	// 获取镜像列表
	list, err := ui.docker.ListImages()
	if err != nil {
		return
	}

	rowIdx := 1
	for _, img := range list {
		idShort := img.ID[7:19]
		sizeStr := fmt.Sprintf("%.2fMB", float64(img.Size)/1024/1024)

		tags := img.RepoTags
		if len(tags) == 0 {
			ui.addRow(rowIdx, idShort, "<none>", "<none>", sizeStr, img.ID)
			rowIdx++
		} else {
			for _, fullTag := range tags {
				repo := "<none>"
				tag := "<none>"
				parts := strings.Split(fullTag, ":")
				if len(parts) >= 2 {
					repo = strings.Join(parts[:len(parts)-1], ":")
					tag = parts[len(parts)-1]
				}
				ui.addRow(rowIdx, idShort, repo, tag, sizeStr, fullTag)
				rowIdx++
			}
		}
	}

	// 恢复选择
	ui.restoreSelection(selRow)
}

// addRow 添加表格行
func (ui *ImageUI) addRow(row int, idShort, repo, tag, size, reference string) {
	ui.common.Table.SetCell(row, 0, tview.NewTableCell(idShort).SetReference(reference))
	ui.common.Table.SetCell(row, 1, tview.NewTableCell(repo))
	ui.common.Table.SetCell(row, 2, tview.NewTableCell(tag))
	ui.common.Table.SetCell(row, 3, tview.NewTableCell(size))
}

// restoreSelection 恢复选择
func (ui *ImageUI) restoreSelection(selRow int) {
	if selRow >= ui.common.Table.GetRowCount() {
		selRow = ui.common.Table.GetRowCount() - 1
	}
	if selRow > 0 {
		ui.common.Table.Select(selRow, 0)
	} else if ui.common.Table.GetRowCount() > 1 {
		ui.common.Table.Select(1, 0)
	}
}

// Tag 标记镜像
func (ui *ImageUI) Tag(newTag string) {
	if newTag != "" && ui.common.SelectedID != "" {
		_ = ui.docker.TagImage(ui.common.SelectedID, newTag)
		ui.RefreshList()
	}
}

// ShowTagInput 显示标签输入对话框
func (ui *ImageUI) ShowTagInput() {
	if ui.common.SelectedID != "" {
		ui.common.ShowInput("New Tag (repo:tag):", ui.Tag)
	}
}

// Delete 删除镜像
func (ui *ImageUI) Delete() {
	if ui.common.SelectedID == "" {
		return
	}
	ui.common.ShowConfirm("Delete image "+ui.common.SelectedID+"?",
		func() {
			ui.common.RunAsyncAction("Deleting...",
				func() { _, _ = ui.docker.RemoveImage(ui.common.SelectedID, false) },
				ui.RefreshList)
		}, nil)
}
