package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"

	"github-runner-manager/model"
)

// RunnerForm adalah dialog/form untuk menambah runner baru
type RunnerForm struct {
	window     fyne.Window
	onSubmit   func(config model.RunnerConfig)
	onCancel   func()
	formDialog dialog.Dialog
}

// NewRunnerForm membuat instance RunnerForm baru
func NewRunnerForm(window fyne.Window, onSubmit func(config model.RunnerConfig), onCancel func()) *RunnerForm {
	return &RunnerForm{
		window:   window,
		onSubmit: onSubmit,
		onCancel: onCancel,
	}
}

// Show menampilkan form untuk menambah runner
func (rf *RunnerForm) Show() {
	// Field inputs
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("my-runner")

	repoEntry := widget.NewEntry()
	repoEntry.SetPlaceHolder("https://github.com/owner/repo")

	tokenEntry := widget.NewPasswordEntry()
	tokenEntry.SetPlaceHolder("ghp_xxxxxxxxxxxx")

	labelsEntry := widget.NewEntry()
	labelsEntry.SetPlaceHolder("self-hosted, linux, x64 (optional)")

	// Get default work directory
	homeDir, _ := os.UserHomeDir()
	defaultWorkDir := filepath.Join(homeDir, "runners", "runner-"+uuid.New().String()[:8])

	workDirEntry := widget.NewEntry()
	workDirEntry.SetText(defaultWorkDir)

	// Directory picker button
	dirButton := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, rf.window)
				return
			}
			if uri != nil {
				workDirEntry.SetText(uri.Path())
			}
		}, rf.window)
	})

	workDirContainer := container.NewBorder(nil, nil, nil, dirButton, workDirEntry)

	// Form items
	formItems := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Repository URL", repoEntry),
		widget.NewFormItem("Token", tokenEntry),
		widget.NewFormItem("Labels", labelsEntry),
		widget.NewFormItem("Work Directory", workDirContainer),
	}

	// Create form dialog
	form := widget.NewForm(formItems...)
	form.SubmitText = "Add Runner"
	form.OnSubmit = func() {
		// Validasi input
		if nameEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("name is required"), rf.window)
			return
		}
		if repoEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("repository URL is required"), rf.window)
			return
		}
		if tokenEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("token is required"), rf.window)
			return
		}

		// Parse labels
		var labels []string
		if labelsEntry.Text != "" {
			labels = strings.Split(labelsEntry.Text, ",")
			for i := range labels {
				labels[i] = strings.TrimSpace(labels[i])
			}
		}

		// Resolve work directory
		workDir := workDirEntry.Text
		if strings.HasPrefix(workDir, "~/") {
			home, _ := os.UserHomeDir()
			workDir = filepath.Join(home, workDir[2:])
		}

		// Buat config
		config := model.RunnerConfig{
			ID:      uuid.New().String(),
			Name:    nameEntry.Text,
			RepoURL: repoEntry.Text,
			Token:   tokenEntry.Text,
			Labels:  labels,
			WorkDir: workDir,
		}

		rf.formDialog.Hide()
		if rf.onSubmit != nil {
			rf.onSubmit(config)
		}
	}

	form.OnCancel = func() {
		rf.formDialog.Hide()
		if rf.onCancel != nil {
			rf.onCancel()
		}
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("Add New GitHub Actions Runner", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		form,
	)

	rf.formDialog = dialog.NewCustom("Add Runner", "Cancel", content, rf.window)
	rf.formDialog.Resize(fyne.NewSize(600, 500))
	rf.formDialog.Show()
}

// Hide menyembunyikan form
func (rf *RunnerForm) Hide() {
	if rf.formDialog != nil {
		rf.formDialog.Hide()
	}
}
