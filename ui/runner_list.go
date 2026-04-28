package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

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
	order      []string // menyimpan urutan insert agar tampilan tidak acak
	onSelect   func(id string)
	selectedID string
	scroll     *container.Scroll
	listBox    *fyne.Container
}

// hexToColor mengkonversi hex color string ke color.Color
func hexToColor(hex string) color.Color {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return color.Gray{Y: 128}
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

// extractDisplayName mengekstrak nama tampilan dari nama runner.
// Jika nama berupa URL GitHub (mis. https://github.com/owner/repo),
// hanya bagian repo-nya yang dikembalikan. Selain itu dikembalikan apa adanya.
func extractDisplayName(name string) string {
	// Trim whitespace dan trailing slash
	name = strings.TrimSpace(strings.TrimRight(name, "/"))
	if name == "" {
		return name
	}
	// Jika mengandung '/', ambil segment terakhir sebagai nama
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		// Cari segment terakhir yang tidak kosong
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" {
				return parts[i]
			}
		}
	}
	return name
}

// statusColors adalah mapping status ke warna dot
var statusColors = map[model.RunnerStatus]string{
	model.StatusIdle:       "#9E9E9E",
	model.StatusInstalling: "#FFC107",
	model.StatusRunning:    "#4CAF50",
	model.StatusStopped:    "#F44336",
	model.StatusError:      "#FF5722",
}

// statusIcons (dipakai untuk referensi, tidak dipakai langsung di list item)
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
		order:    []string{},
		onSelect: onSelect,
	}

	header := widget.NewLabelWithStyle("GitHub Runners", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	rl.listBox = container.NewVBox()
	rl.scroll = container.NewScroll(rl.listBox)
	rl.container = container.NewBorder(header, nil, nil, nil, rl.scroll)

	return rl
}

// AddRunner menambahkan runner ke daftar
func (rl *RunnerList) AddRunner(state *model.RunnerState) {
	if _, exists := rl.items[state.Config.ID]; exists {
		rl.UpdateRunner(state)
		return
	}
	item := rl.createListItem(state)
	rl.items[state.Config.ID] = item
	rl.order = append(rl.order, state.Config.ID) // simpan urutan insert
	rl.refreshList()
}

// RemoveRunner menghapus runner dari daftar
func (rl *RunnerList) RemoveRunner(id string) {
	delete(rl.items, id)
	// Hapus dari order slice
	for i, oid := range rl.order {
		if oid == id {
			rl.order = append(rl.order[:i], rl.order[i+1:]...)
			break
		}
	}
	if rl.selectedID == id {
		rl.selectedID = ""
	}
	rl.refreshList()
}

// UpdateRunner memperbarui tampilan runner
func (rl *RunnerList) UpdateRunner(state *model.RunnerState) {
	if _, exists := rl.items[state.Config.ID]; exists {
		rl.items[state.Config.ID] = rl.createListItem(state)
		rl.refreshList()
	}
}

// createListItem membuat item daftar untuk satu runner.
//
// Penyebab teks tidak terlihat sebelumnya:
//  1. container.NewWithoutLayout → MinSize = 0, menyebabkan label tertimpa
//  2. tappableContainer bukan widget proper → Fyne tidak me-render children-nya
//
// Solusi: canvas.Rectangle dengan SetMinSize sebagai dot, lalu
// container.NewStack(invisibleBtn, row) agar baris bisa diklik DAN label terlihat.
func (rl *RunnerList) createListItem(state *model.RunnerState) *fyne.Container {
	id := state.Config.ID

	// Status dot — canvas.Rectangle punya MinSize yang dipatuhi layout
	colorStr, ok := statusColors[state.Status]
	if !ok {
		colorStr = "#9E9E9E"
	}
	dot := canvas.NewRectangle(hexToColor(colorStr))
	dot.SetMinSize(fyne.NewSize(10, 10))
	dot.CornerRadius = 5

	// Label nama dan status — tampilkan nama repo jika nama berupa URL
	nameLabel := widget.NewLabelWithStyle(extractDisplayName(state.Config.Name), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	statusLabel := widget.NewLabel(fmt.Sprintf("● %s", state.Status))

	// Susun dot di kiri, info di tengah
	info := container.NewVBox(nameLabel, statusLabel)
	row := container.NewBorder(nil, nil, container.NewCenter(dot), nil, info)

	// Background highlight untuk item yang sedang dipilih
	bg := canvas.NewRectangle(color.Transparent)
	if id == rl.selectedID {
		bg.FillColor = color.RGBA{R: 127, G: 168, B: 246, A: 180} // #7fa8f6 lebih muda
		bg.CornerRadius = 6
	}

	// Button transparan untuk clickable — ditumpuk di bawah row via Stack
	// (Stack menggambar dari indeks 0 ke atas; widget di indeks tinggi menimpa yang di bawah)
	invisibleBtn := widget.NewButton("", func() {
		rl.SelectRunner(id)
	})
	invisibleBtn.Importance = widget.LowImportance

	// bg → btn → row: background di paling bawah, klik tetap terdaftar, label terlihat
	card := container.NewStack(bg, invisibleBtn, row)

	return container.NewBorder(nil, widget.NewSeparator(), nil, nil, card)
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

	// Iterasi berdasarkan order slice, bukan map, agar urutan selalu konsisten
	for _, id := range rl.order {
		if item, ok := rl.items[id]; ok {
			rl.listBox.Add(item)
		}
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
