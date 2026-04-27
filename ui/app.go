package ui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"

	"github-runner-manager/model"
	"github-runner-manager/runner"
)

// App adalah root aplikasi Fyne
type App struct {
	myApp        fyne.App
	window       fyne.Window
	manager      *runner.RunnerManager
	runnerForm   *RunnerForm
	runnerList   *RunnerList
	runnerDetail *RunnerDetail
	toolbar      *AppToolbar

	logListeners  map[string]chan struct{} // done channel per listener
	stopListeners chan struct{}
}

// NewApp membuat instance aplikasi baru
func NewApp() *App {
	myApp := app.New()
	window := myApp.NewWindow("GitHub Runner Manager")
	window.Resize(fyne.NewSize(1200, 800))
	window.CenterOnScreen()

	return &App{
		myApp:         myApp,
		window:        window,
		manager:       runner.NewRunnerManager(),
		logListeners:  make(map[string]chan struct{}),
		stopListeners: make(chan struct{}),
	}
}

// Initialize mengatur UI dan komponen
func (a *App) Initialize() {
	// Buat komponen UI
	a.runnerList = NewRunnerList(a.onRunnerSelect)
	a.runnerDetail = NewRunnerDetail(
		a.onStartRunner,
		a.onStopRunner,
		a.onRemoveRunner,
		a.onClearLog,
		a.onSaveLog,
	)
	a.toolbar = NewAppToolbar(
		a.window,
		a.onAddRunner,
		a.onSaveConfig,
		a.onLoadConfig,
	)

	// Runner form (lazy init)
	a.runnerForm = NewRunnerForm(a.window, a.onRunnerFormSubmit, nil)

	// Layout utama
	mainContent := container.NewBorder(
		a.toolbar.GetContainer(),
		nil,
		a.runnerList.GetContainer(),
		nil,
		a.runnerDetail.GetContainer(),
	)

	// Set padding dan border
	mainContent = container.NewPadded(mainContent)

	a.window.SetContent(mainContent)

	// Setup window close handler
	a.window.SetCloseIntercept(func() {
		a.cleanup()
		a.myApp.Quit()
	})

	// Mulai goroutine untuk listening log
	go a.logListener()

	// Load config default jika ada
	a.loadDefaultConfig()
}

// Run menjalankan aplikasi
func (a *App) Run() {
	a.window.ShowAndRun()
}

// cleanup membersihkan resources sebelum keluar
func (a *App) cleanup() {
	// Hentikan semua runner
	a.manager.StopAll()

	// Tutup semua listener
	select {
	case <-a.stopListeners:
	default:
		close(a.stopListeners)
	}
}

// Event handlers

func (a *App) onAddRunner() {
	a.runnerForm.Show()
}

func (a *App) onRunnerFormSubmit(config model.RunnerConfig) {
	// Tambahkan runner ke manager
	state := a.manager.Add(config)

	// Tambahkan ke UI list
	a.runnerList.AddRunner(state)

	// Pilih runner baru
	a.runnerList.SelectRunner(config.ID)

	// Auto-start listener untuk log
	a.startLogListener(config.ID, state.LogChan)

	// Simpan config
	a.saveDefaultConfig()
}

func (a *App) onRunnerSelect(id string) {
	state, exists := a.manager.Get(id)
	if !exists {
		return
	}

	a.runnerDetail.SetRunner(state)
	// Log sudah di-forward oleh startLogListener — tidak perlu goroutine tambahan
}

func (a *App) onStartRunner(id string) {
	err := a.manager.Start(id)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	state, _ := a.manager.Get(id)
	a.runnerList.UpdateRunner(state)
	a.runnerDetail.Refresh()

	// Start listening log
	a.startLogListener(id, state.LogChan)
}

func (a *App) onStopRunner(id string) {
	err := a.manager.Stop(id)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	state, _ := a.manager.Get(id)
	a.runnerList.UpdateRunner(state)
	a.runnerDetail.Refresh()
}

func (a *App) onRemoveRunner(id string) {
	// Konfirmasi
	dialog.ShowConfirm(
		"Remove Runner",
		"Are you sure you want to remove this runner?",
		func(confirmed bool) {
			if confirmed {
				// Hentikan listener goroutine via done channel
				if done, exists := a.logListeners[id]; exists {
					close(done)
					delete(a.logListeners, id)
				}

				// Hapus dari manager (juga menutup state.LogChan)
				err := a.manager.Remove(id)
				if err != nil {
					dialog.ShowError(err, a.window)
					return
				}

				// Update UI
				a.runnerList.RemoveRunner(id)

				// Clear detail jika yang dihapus sedang dipilih
				a.runnerDetail.SetRunner(nil)

				// Simpan config
				a.saveDefaultConfig()
			}
		},
		a.window,
	)
}

func (a *App) onClearLog() {
	// Log sudah di-clear di runnerDetail
}

func (a *App) onSaveLog() {
	text := a.runnerDetail.GetLogText()
	if text == "" {
		dialog.ShowInformation("Save Log", "Log is empty", a.window)
		return
	}

	dialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, a.window)
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()

		_, err = writer.Write([]byte(text))
		if err != nil {
			dialog.ShowError(err, a.window)
			return
		}

		dialog.ShowInformation("Save Log", "Log saved successfully", a.window)
	}, a.window)

	dialog.SetFilter(storage.NewExtensionFileFilter([]string{".log", ".txt"}))
	dialog.SetFileName("runner.log")
	dialog.Show()
}

func (a *App) onSaveConfig(path string) {
	err := a.manager.SaveToFile(path)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	dialog.ShowInformation("Save Config", "Configuration saved successfully", a.window)
}

func (a *App) onLoadConfig(path string) {
	err := a.manager.LoadFromFile(path)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	// Refresh UI
	a.refreshRunnerList()
	dialog.ShowInformation("Load Config", "Configuration loaded successfully", a.window)
}

// Helper methods

func (a *App) startLogListener(id string, logChan chan string) {
	// Hentikan listener lama jika ada
	if done, exists := a.logListeners[id]; exists {
		close(done)
		delete(a.logListeners, id)
	}

	// done channel sebagai sinyal stop untuk goroutine ini
	done := make(chan struct{})
	a.logListeners[id] = done

	// Goroutine untuk membaca log dan update UI
	go func() {
		for {
			select {
			case <-a.stopListeners:
				return
			case <-done:
				return
			case line, ok := <-logChan:
				if !ok {
					// logChan ditutup (runner dihapus), hentikan goroutine
					return
				}
				// Update UI di main thread
				fyne.Do(func() {
					a.runnerDetail.AppendLog(line)
				})
			}
		}
	}()
}

func (a *App) logListener() {
	for {
		select {
		case <-a.stopListeners:
			return
		case <-time.After(500 * time.Millisecond):
			// Check untuk update status
			for _, state := range a.manager.GetAll() {
				fyne.Do(func() {
					a.runnerList.UpdateRunner(state)
					a.runnerDetail.Refresh()
				})
			}
		}
	}
}

func (a *App) refreshRunnerList() {
	// Clear existing - rebuild dari manager
	for _, state := range a.manager.GetAll() {
		a.runnerList.AddRunner(state)
	}
}

func (a *App) saveDefaultConfig() {
	// Simpan ke default location (silently)
	_ = a.manager.SaveToFile("")
}

func (a *App) loadDefaultConfig() {
	configs, err := runner.LoadRunnersFromDefault()
	if err != nil {
		return
	}

	for _, config := range configs {
		state := a.manager.Add(config)
		a.runnerList.AddRunner(state)
		a.startLogListener(config.ID, state.LogChan)
	}
}

// ShowError menampilkan error dialog
func (a *App) ShowError(err error) {
	dialog.ShowError(err, a.window)
}

// ShowInfo menampilkan info dialog
func (a *App) ShowInfo(title, message string) {
	dialog.ShowInformation(title, message, a.window)
}

// Helper functions

// resolveHomeDir mengubah ~ atau $HOME ke home directory aktual
func resolveHomeDir(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// parseInt helper
func parseInt(s string) int64 {
	i, _ := strconv.ParseInt(s, 16, 64)
	return i
}
