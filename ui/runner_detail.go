package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github-runner-manager/model"
)

// RunnerDetail menampilkan detail dan kontrol untuk satu runner
type RunnerDetail struct {
	container    *fyne.Container
	nameLabel    *widget.Label
	repoLabel    *widget.Label
	statusLabel  *widget.Label
	statusBadge  *canvas.Circle
	startBtn     *widget.Button
	stopBtn      *widget.Button
	removeBtn    *widget.Button
	logPanel     *LogPanel

	currentState *model.RunnerState
	onStart      func(id string)
	onStop       func(id string)
	onRemove     func(id string)
	onClearLog   func()
	onSaveLog    func()
}

// statusColors adalah mapping status ke warna
var detailStatusColors = map[model.RunnerStatus]color.Color{
	model.StatusIdle:       color.RGBA{158, 158, 158, 255},   // Gray
	model.StatusInstalling: color.RGBA{255, 193, 7, 255},     // Amber
	model.StatusRunning:    color.RGBA{76, 175, 80, 255},     // Green
	model.StatusStopped:    color.RGBA{244, 67, 54, 255},     // Red
	model.StatusError:      color.RGBA{255, 87, 34, 255},     // Deep Orange
}

// NewRunnerDetail membuat instance RunnerDetail baru
func NewRunnerDetail(
	onStart func(id string),
	onStop func(id string),
	onRemove func(id string),
	onClearLog func(),
	onSaveLog func(),
) *RunnerDetail {
	rd := &RunnerDetail{
		onStart:    onStart,
		onStop:     onStop,
		onRemove:   onRemove,
		onClearLog: onClearLog,
		onSaveLog:  onSaveLog,
	}

	// Title
	title := widget.NewLabelWithStyle("Runner Details", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Info section
	rd.nameLabel = widget.NewLabel("Select a runner")
	rd.repoLabel = widget.NewLabel("")
	rd.statusLabel = widget.NewLabelWithStyle("Status: -", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Status badge
	rd.statusBadge = canvas.NewCircle(detailStatusColors[model.StatusIdle])
	rd.statusBadge.Resize(fyne.NewSize(12, 12))
	statusContainer := container.NewHBox(rd.statusBadge, rd.statusLabel)

	infoCard := widget.NewCard("", "", container.NewVBox(
		container.NewHBox(widget.NewLabel("Name:"), rd.nameLabel),
		container.NewHBox(widget.NewLabel("Repository:"), rd.repoLabel),
		statusContainer,
	))

	// Action buttons
	rd.startBtn = widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), func() {
		if rd.currentState != nil && rd.onStart != nil {
			rd.onStart(rd.currentState.Config.ID)
		}
	})

	rd.stopBtn = widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		if rd.currentState != nil && rd.onStop != nil {
			rd.onStop(rd.currentState.Config.ID)
		}
	})

	rd.removeBtn = widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {
		if rd.currentState != nil && rd.onRemove != nil {
			rd.onRemove(rd.currentState.Config.ID)
		}
	})
	rd.removeBtn.Importance = widget.DangerImportance

	buttonContainer := container.NewHBox(rd.startBtn, rd.stopBtn, rd.removeBtn)

	// Log panel
	rd.logPanel = NewLogPanel()

	// Log toolbar
	clearBtn := widget.NewButtonWithIcon("Clear Log", theme.ContentClearIcon(), func() {
		rd.logPanel.Clear()
		if rd.onClearLog != nil {
			rd.onClearLog()
		}
	})

	saveBtn := widget.NewButtonWithIcon("Save Log", theme.DocumentSaveIcon(), func() {
		if rd.onSaveLog != nil {
			rd.onSaveLog()
		}
	})

	logToolbar := container.NewHBox(
		widget.NewLabelWithStyle("Live Log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		clearBtn,
		saveBtn,
	)

	logSection := container.NewBorder(logToolbar, nil, nil, nil, rd.logPanel.GetContainer())

	// Main layout
	rd.container = container.NewBorder(
		container.NewVBox(title, widget.NewSeparator(), infoCard, buttonContainer, widget.NewSeparator()),
		nil, nil, nil,
		logSection,
	)

	rd.updateButtonStates()

	return rd
}

// SetRunner menampilkan detail untuk runner tertentu
func (rd *RunnerDetail) SetRunner(state *model.RunnerState) {
	rd.currentState = state

	if state == nil {
		rd.nameLabel.SetText("Select a runner")
		rd.repoLabel.SetText("")
		rd.statusLabel.SetText("Status: -")
		rd.logPanel.Clear()
		rd.updateStatus(model.StatusIdle)
	} else {
		rd.nameLabel.SetText(state.Config.Name)
		rd.repoLabel.SetText(state.Config.RepoURL)
		rd.updateStatus(state.Status)
	}

	rd.updateButtonStates()
}

// updateStatus memperbarui tampilan status
func (rd *RunnerDetail) updateStatus(status model.RunnerStatus) {
	rd.statusLabel.SetText(fmt.Sprintf("Status: %s", status))

	// Update badge color
	if color, ok := detailStatusColors[status]; ok {
		rd.statusBadge.FillColor = color
		rd.statusBadge.Refresh()
	}
}

// updateButtonStates memperbarui state tombol berdasarkan status
func (rd *RunnerDetail) updateButtonStates() {
	if rd.currentState == nil {
		rd.startBtn.Disable()
		rd.stopBtn.Disable()
		rd.removeBtn.Disable()
		return
	}

	switch rd.currentState.Status {
	case model.StatusRunning:
		rd.startBtn.Disable()
		rd.stopBtn.Enable()
		rd.removeBtn.Disable()
	case model.StatusStopped, model.StatusIdle, model.StatusError:
		rd.startBtn.Enable()
		rd.stopBtn.Disable()
		rd.removeBtn.Enable()
	case model.StatusInstalling:
		rd.startBtn.Disable()
		rd.stopBtn.Disable()
		rd.removeBtn.Disable()
	}
}

// AppendLog menambahkan log ke panel
func (rd *RunnerDetail) AppendLog(line string) {
	rd.logPanel.Append(line)
}

// GetLogText mengembalikan semua teks log
func (rd *RunnerDetail) GetLogText() string {
	return rd.logPanel.GetText()
}

// ClearLog menghapus semua log
func (rd *RunnerDetail) ClearLog() {
	rd.logPanel.Clear()
}

// Refresh memperbarui tampilan detail
func (rd *RunnerDetail) Refresh() {
	if rd.currentState != nil {
		rd.updateStatus(rd.currentState.Status)
	}
	rd.updateButtonStates()
}

// GetContainer mengembalikan container untuk ditambahkan ke layout
func (rd *RunnerDetail) GetContainer() fyne.CanvasObject {
	return rd.container
}
