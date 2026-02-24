// Package ui 
package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/wsl12105/docker-manager/internal/docker"
)

// ImageUI 
type ImageUI struct {
	common *Common
	docker *docker.Client
}

// NewImageUI 
func NewImageUI(common *Common, docker *docker.Client) *ImageUI {
	return &ImageUI{
		common: common,
		docker: docker,
	}
}

// RefreshList 
func (ui *ImageUI) RefreshList() {
	selRow, _ := ui.common.Table.GetSelection()
	ui.common.Table.Clear()

	
	headers := []string{"IMAGE ID", "REPOSITORY", "TAG", "SIZE"}
	for i, h := range headers {
		ui.common.Table.SetCell(0, i,
			tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetExpansion(1))
	}

	
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

	
	ui.restoreSelection(selRow)
}

// addRow
func (ui *ImageUI) addRow(row int, idShort, repo, tag, size, reference string) {
	ui.common.Table.SetCell(row, 0, tview.NewTableCell(idShort).SetReference(reference))
	ui.common.Table.SetCell(row, 1, tview.NewTableCell(repo))
	ui.common.Table.SetCell(row, 2, tview.NewTableCell(tag))
	ui.common.Table.SetCell(row, 3, tview.NewTableCell(size))
}

// restoreSelection 
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

// Tag 
func (ui *ImageUI) Tag(newTag string) {
	if newTag != "" && ui.common.SelectedID != "" {
		_ = ui.docker.TagImage(ui.common.SelectedID, newTag)
		ui.RefreshList()
	}
}

// ShowTagInput 
func (ui *ImageUI) ShowTagInput() {
	if ui.common.SelectedID != "" {
		ui.common.ShowInput("New Tag (repo:tag):", ui.Tag)
	}
}

// Delete 
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
