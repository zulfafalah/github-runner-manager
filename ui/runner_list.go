package ui

import (
	"fmt"
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github-runner-manager/model"
)

// RunnerListItem merepresentasikan item di daftar runner
type RunnerListItem struct {
	ID       string
	Name     string
	Status   model.RunnerStatus
	OnSelect func(id string)
}

// RunnerList adalah widget daftar runner di sidebar
type RunnerList struct {
	container  *fyne.Container
	items      map[string]*fyne.Container
	onSelect   func(id string)
	selectedID string
	scroll     *container.Scroll
	listBox    *fyne.Container
}

// hexToColor mengkonversi hex color string ke color.Color
func hexToColor(hex string) color.Color {
	// Remove # if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	
	if len(hex) != 6 {
		return color.Gray{128}
	}
	
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
}

// statusColors adalah mapping status ke warna
var statusColors = map[model.RunnerStatus]string{
	model.StatusIdle:       "#9E9E9E", // Gray
	model.StatusInstalling: "#FFC107", // Amber
	model.StatusRunning:    "#4CAF50", // Green
	model.StatusStopped:    "#F44336", // Red
	model.StatusError:      "#FF5722", // Deep Orange
}

// statusIcons adalah mapping status ke ikon
var statusIcons = map[model.RunnerStatus]fyne.Resource{
	model.StatusIdle:       theme.RadioButtonIcon(),
	model.StatusInstalling: theme.ViewRefreshIcon(),
	model.StatusRunning:    theme.ConfirmIcon(),
	model.StatusStopped:    theme.CancelIcon(),
	model.StatusError:      theme.WarningIcon(),
}

// NewRunnerList membuat instance RunnerList baru
func NewRunnerList(onSelect func(id string)) *RunnerList {
	rl := &RunnerList{
		items:    make(map[string]*fyne.Container),
		onSelect: onSelect,
	}

	// Header
	header := widget.NewLabelWithStyle("GitHub Runners", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Container untuk list
	rl.listBox = container.NewVBox()
	rl.scroll = container.NewScroll(rl.listBox)

	rl.container = container.NewBorder(header, nil, nil, nil, rl.scroll)
	rl.container.Resize(fyne.NewSize(250, 400))

	return rl
}

// AddRunner menambahkan runner ke daftar
func (rl *RunnerList) AddRunner(state *model.RunnerState) {
	// Jika sudah ada, update saja
	if _, exists := rl.items[state.Config.ID]; exists {
		rl.UpdateRunner(state)
		return
	}
	
	item := rl.createListItem(state)
	rl.items[state.Config.ID] = item
	rl.refreshList()
}

// RemoveRunner menghapus runner dari daftar
func (rl *RunnerList) RemoveRunner(id string) {
	delete(rl.items, id)
	if rl.selectedID == id {
		rl.selectedID = ""
	}
	rl.refreshList()
}

// UpdateRunner memperbarui tampilan runner
func (rl *RunnerList) UpdateRunner(state *model.RunnerState) {
	// Rebuild item untuk update sederhana
	if _, exists := rl.items[state.Config.ID]; exists {
		newItem := rl.createListItem(state)
		rl.items[state.Config.ID] = newItem
		rl.refreshList()
	}
}

// createListItem membuat item daftar untuk satu runner
func (rl *RunnerList) createListItem(state *model.RunnerState) *fyne.Container {
	id := state.Config.ID

	// Status indicator (circle)
	colorStr := statusColors[state.Status]
	statusCircle := canvas.NewCircle(hexToColor(colorStr))
	statusCircle.Resize(fyne.NewSize(12, 12))
	
	// Label
	label := widget.NewLabel(fmt.Sprintf("%s\n%s", state.Config.Name, state.Status))

	// Button
	btn := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		rl.SelectRunner(id)
	})
	btn.Importance = widget.LowImportance

	// Layout untuk circle
	circleContainer := container.NewWithoutLayout(statusCircle)
	circleContainer.Resize(fyne.NewSize(20, 20))

	// Main layout
	content := container.NewBorder(nil, nil, circleContainer, btn, label)
	
	// Make clickable
	tappable := newTappableContainer(content, func() {
		rl.SelectRunner(id)
	})

	return container.NewBorder(nil, widget.NewSeparator(), nil, nil, tappable)
}

// SelectRunner menangani pemilihan runner (publik)
func (rl *RunnerList) SelectRunner(id string) {
	rl.selectedID = id
	if rl.onSelect != nil {
		rl.onSelect(id)
	}
	rl.refreshList()
}

// refreshList memperbarui tampilan daftar
func (rl *RunnerList) refreshList() {
	rl.listBox.Objects = nil
	
	for _, item := range rl.items {
		rl.listBox.Add(item)
	}

	if len(rl.items) == 0 {
		emptyLabel := widget.NewLabel("No runners added")
		emptyLabel.Alignment = fyne.TextAlignCenter
		rl.listBox.Add(container.NewCenter(emptyLabel))
	}

	rl.listBox.Refresh()
}

// GetContainer mengembalikan container untuk ditambahkan ke layout
func (rl *RunnerList) GetContainer() fyne.CanvasObject {
	return rl.container
}

// tappableContainer adalah container yang bisa diklik
type tappableContainer struct {
	fyne.CanvasObject
	onTapped func()
}

func newTappableContainer(content fyne.CanvasObject, onTapped func()) *tappableContainer {
	return &tappableContainer{
		CanvasObject: content,
		onTapped:     onTapped,
	}
}

func (tc *tappableContainer) Tapped(*fyne.PointEvent) {
	if tc.onTapped != nil {
		tc.onTapped()
	}
}

func (tc *tappableContainer) TappedSecondary(*fyne.PointEvent) {}

var _ fyne.Tappable = (*tappableContainer)(nil)
