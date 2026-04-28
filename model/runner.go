package model

import "os"

// RunnerConfig menyimpan konfigurasi untuk satu GitHub Actions runner
type RunnerConfig struct {
	ID      string   `json:"id"`       // UUID unik per runner
	Name    string   `json:"name"`     // Nama tampilan runner
	RepoURL string   `json:"repo_url"` // https://github.com/owner/repo
	Token   string   `json:"token"`    // Registration token
	Labels  []string `json:"labels"`   // Optional custom labels
	WorkDir string   `json:"work_dir"` // Path folder instalasi runner
}

// RunnerStatus merepresentasikan status runner saat ini
type RunnerStatus string

const (
	StatusIdle       RunnerStatus = "Idle"
	StatusInstalling RunnerStatus = "Installing"
	StatusRunning    RunnerStatus = "Running"
	StatusStopped    RunnerStatus = "Stopped"
	StatusError      RunnerStatus = "Error"
)

// RunnerState menyimpan state runtime untuk satu runner
type RunnerState struct {
	Config   RunnerConfig
	Status   RunnerStatus
	LogChan  chan string // channel untuk streaming log ke UI
	Process  *os.Process // proses run.sh yang sedang berjalan
	ExitChan chan error  // channel untuk menerima exit status
	Stopping bool        // flag: runner sedang dihentikan secara sengaja
}

// ConfigFile adalah struktur file konfigurasi JSON
type ConfigFile struct {
	Runners []RunnerConfig `json:"runners"`
}
