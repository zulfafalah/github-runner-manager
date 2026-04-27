package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// AppToolbar adalah toolbar untuk aksi utama aplikasi
type AppToolbar struct {
	container    *fyne.Container
	window       fyne.Window
	onAddRunner  func()
	onSaveConfig func(path string)
	onLoadConfig func(path string)
}

// NewAppToolbar membuat instance AppToolbar baru
func NewAppToolbar(
	window fyne.Window,
	onAddRunner func(),
	onSaveConfig func(path string),
	onLoadConfig func(path string),
) *AppToolbar {
	tb := &AppToolbar{
		window:       window,
		onAddRunner:  onAddRunner,
		onSaveConfig: onSaveConfig,
		onLoadConfig: onLoadConfig,
	}

	// Add Runner button
	addBtn := widget.NewButtonWithIcon("Add Runner", theme.ContentAddIcon(), func() {
		if tb.onAddRunner != nil {
			tb.onAddRunner()
		}
	})
	addBtn.Importance = widget.HighImportance

	// Save Config button
	saveBtn := widget.NewButtonWithIcon("Save Config", theme.DocumentSaveIcon(), func() {
		tb.showSaveDialog()
	})

	// Load Config button
	loadBtn := widget.NewButtonWithIcon("Load Config", theme.FolderOpenIcon(), func() {
		tb.showLoadDialog()
	})

	// Help/About button (optional)
	aboutBtn := widget.NewButtonWithIcon("About", theme.InfoIcon(), func() {
		tb.showAboutDialog()
	})

	// Layout
	tb.container = container.NewHBox(
		addBtn,
		widget.NewSeparator(),
		saveBtn,
		loadBtn,
		layout.NewSpacer(),
		aboutBtn,
	)

	return tb
}

// showSaveDialog menampilkan dialog untuk menyimpan konfigurasi
func (tb *AppToolbar) showSaveDialog() {
	dlg := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, tb.window)
			return
		}
		if writer == nil {
			return // User cancelled
		}
		defer writer.Close()

		if tb.onSaveConfig != nil {
			tb.onSaveConfig(writer.URI().Path())
		}
	}, tb.window)

	// Set default filter untuk JSON files
	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	dlg.SetFileName("runners.json")
	dlg.Show()
}

// showLoadDialog menampilkan dialog untuk memuat konfigurasi
func (tb *AppToolbar) showLoadDialog() {
	dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, tb.window)
			return
		}
		if reader == nil {
			return // User cancelled
		}
		defer reader.Close()

		if tb.onLoadConfig != nil {
			tb.onLoadConfig(reader.URI().Path())
		}
	}, tb.window)

	// Set filter untuk JSON files
	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	dlg.Show()
}

// showAboutDialog menampilkan dialog tentang aplikasi
func (tb *AppToolbar) showAboutDialog() {
	content := container.NewVBox(
		widget.NewLabelWithStyle("GitHub Runner Manager", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Version 1.0.0"),
		widget.NewSeparator(),
		widget.NewLabel("A desktop application for managing multiple"),
		widget.NewLabel("GitHub Actions self-hosted runners."),
		widget.NewLabel(""),
		widget.NewLabel("Built with Go and Fyne.io"),
	)

	dialog.ShowCustom("About", "Close", content, tb.window)
}

// GetContainer mengembalikan container untuk ditambahkan ke layout
func (tb *AppToolbar) GetContainer() fyne.CanvasObject {
	return tb.container
}
