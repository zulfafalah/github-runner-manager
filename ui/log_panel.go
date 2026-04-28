package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LogPanel adalah widget untuk menampilkan log secara real-time
type LogPanel struct {
	container *fyne.Container
	textArea  *widget.Entry
	lines     []string
	maxLines  int
}

// NewLogPanel membuat instance LogPanel baru
func NewLogPanel() *LogPanel {
	textArea := widget.NewMultiLineEntry()
	textArea.Wrapping = fyne.TextWrapOff
	textArea.SetPlaceHolder("")

	lp := &LogPanel{
		textArea: textArea,
		lines:    make([]string, 0),
		maxLines: 5000, // Batasi jumlah baris untuk performa
	}

	scroll := container.NewScroll(textArea)
	scroll.SetMinSize(fyne.NewSize(400, 300))

	lp.container = container.NewBorder(nil, nil, nil, nil, scroll)
	return lp
}

// Append menambahkan baris log baru
func (lp *LogPanel) Append(text string) {
	lp.lines = append(lp.lines, text)

	// Batasi jumlah baris
	if len(lp.lines) > lp.maxLines {
		lp.lines = lp.lines[len(lp.lines)-lp.maxLines:]
	}

	// Update text area
	content := strings.Join(lp.lines, "\n")
	lp.textArea.SetText(content)
}

// Clear menghapus semua log
func (lp *LogPanel) Clear() {
	lp.lines = lp.lines[:0]
	lp.textArea.SetText("")
}

// GetContainer mengembalikan container untuk ditambahkan ke layout
func (lp *LogPanel) GetContainer() fyne.CanvasObject {
	return lp.container
}

// GetText mengembalikan semua teks log
func (lp *LogPanel) GetText() string {
	return strings.Join(lp.lines, "\n")
}
