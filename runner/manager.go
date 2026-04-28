package runner

import (
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github-runner-manager/model"
)

// RunnerManager mengelola koleksi runner dan operasinya
type RunnerManager struct {
	mu      sync.RWMutex
	runners map[string]*model.RunnerState
}

// NewRunnerManager membuat instance RunnerManager baru
func NewRunnerManager() *RunnerManager {
	return &RunnerManager{
		runners: make(map[string]*model.RunnerState),
	}
}

// Add menambahkan runner baru ke manager
func (rm *RunnerManager) Add(config model.RunnerConfig) *model.RunnerState {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Generate ID jika belum ada
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// Buat state baru
	state := &model.RunnerState{
		Config:  config,
		Status:  model.StatusIdle,
		LogChan: make(chan string, 1000),
	}

	rm.runners[config.ID] = state
	return state
}

// Remove menghapus runner dari manager
func (rm *RunnerManager) Remove(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	state, exists := rm.runners[id]
	if !exists {
		return nil
	}

	// Hentikan runner jika sedang berjalan
	if state.Status == model.StatusRunning && state.Process != nil {
		StopRunner(state)
	}

	// Tutup log channel
	if state.LogChan != nil {
		close(state.LogChan)
	}

	delete(rm.runners, id)
	return nil
}

// Get mengembalikan state runner berdasarkan ID
func (rm *RunnerManager) Get(id string) (*model.RunnerState, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	state, exists := rm.runners[id]
	return state, exists
}

// GetAll mengembalikan semua runner
func (rm *RunnerManager) GetAll() []*model.RunnerState {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	states := make([]*model.RunnerState, 0, len(rm.runners))
	for _, state := range rm.runners {
		states = append(states, state)
	}
	return states
}

// GetConfigs mengembalikan konfigurasi semua runner
func (rm *RunnerManager) GetConfigs() []model.RunnerConfig {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	configs := make([]model.RunnerConfig, 0, len(rm.runners))
	for _, state := range rm.runners {
		configs = append(configs, state.Config)
	}
	return configs
}

// Start memulai runner dengan ID tertentu
func (rm *RunnerManager) Start(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	state, exists := rm.runners[id]
	if !exists {
		return nil
	}

	// Cek apakah runner sudah berjalan
	if state.Status == model.StatusRunning {
		return nil
	}

	// Cek apakah runner sudah terinstal
	if !CheckRunnerInstalled(state.Config.WorkDir) {
		state.Status = model.StatusInstalling

		// Jalankan instalasi di goroutine terpisah
		go func() {
			err := InstallRunner(state.Config, state.LogChan)
			if err != nil {
				state.Status = model.StatusError
				state.LogChan <- "Error: " + err.Error()
				return
			}

			// Mulai runner setelah instalasi
			err = StartRunner(state)
			if err != nil {
				state.Status = model.StatusError
				state.LogChan <- "Error: " + err.Error()
			}
		}()

		return nil
	}

	// Runner sudah terinstal, cek apakah sudah terkonfigurasi
	if CheckRunnerConfigured(state.Config.WorkDir) {
		// Sudah terkonfigurasi, langsung jalankan tanpa reconfigure
		go func() {
			if err := StartRunner(state); err != nil {
				state.Status = model.StatusError
				state.LogChan <- "Error: " + err.Error()
			}
		}()
		return nil
	}

	// Terinstal tapi belum terkonfigurasi, jalankan config dulu
	go func() {
		state.LogChan <- "Configuring runner..."
		err := runConfigScript(state.Config.WorkDir, state.Config.RepoURL, state.Config.Token, state.Config.Name, state.Config.Labels, state.LogChan)
		if err != nil {
			state.Status = model.StatusError
			state.LogChan <- "Error configuring: " + err.Error()
			return
		}

		// Mulai runner setelah configure
		if err := StartRunner(state); err != nil {
			state.Status = model.StatusError
			state.LogChan <- "Error: " + err.Error()
		}
	}()

	return nil
}

// Reconfigure menjalankan ulang config.sh --replace untuk mendapatkan sesi baru dari GitHub.
// Berguna saat muncul error "A session for this runner already exists".
func (rm *RunnerManager) Reconfigure(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	state, exists := rm.runners[id]
	if !exists {
		return fmt.Errorf("runner not found")
	}

	if state.Status == model.StatusRunning {
		return fmt.Errorf("stop the runner before reconfiguring")
	}

	if !CheckRunnerInstalled(state.Config.WorkDir) {
		return fmt.Errorf("runner is not installed yet")
	}

	state.Status = model.StatusInstalling
	go func() {
		state.LogChan <- "Re-configuring runner (replacing existing session)..."

		// Hapus konfigurasi lama dulu agar config.sh tidak menolak
		if err := removeRunnerConfig(state.Config.WorkDir, state.LogChan); err != nil {
			state.Status = model.StatusError
			state.LogChan <- "Error removing old config: " + err.Error()
			return
		}

		err := runConfigScript(
			state.Config.WorkDir,
			state.Config.RepoURL,
			state.Config.Token,
			state.Config.Name,
			state.Config.Labels,
			state.LogChan,
		)
		if err != nil {
			state.Status = model.StatusError
			state.LogChan <- "Error reconfiguring: " + err.Error()
			return
		}
		state.Status = model.StatusIdle
		state.LogChan <- "Runner reconfigured successfully. You can now start it."
	}()
	return nil
}

// IsWorkDirInUse memeriksa apakah workDir sudah digunakan oleh runner lain
func (rm *RunnerManager) IsWorkDirInUse(workDir string, excludeID string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	for _, state := range rm.runners {
		if state.Config.ID != excludeID && state.Config.WorkDir == workDir {
			return true
		}
	}
	return false
}

// Stop menghentikan runner dengan ID tertentu
func (rm *RunnerManager) Stop(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	state, exists := rm.runners[id]
	if !exists {
		return nil
	}

	if state.Status != model.StatusRunning {
		return nil
	}

	return StopRunner(state)
}

// StopAll menghentikan semua runner
func (rm *RunnerManager) StopAll() {
	// Ambil snapshot runners tanpa menahan lock saat blocking
	rm.mu.RLock()
	states := make([]*model.RunnerState, 0, len(rm.runners))
	for _, state := range rm.runners {
		if state.Status == model.StatusRunning && state.Process != nil {
			states = append(states, state)
		}
	}
	rm.mu.RUnlock()

	for _, state := range states {
		StopRunner(state)
	}
}

// LoadFromConfigs memuat runner dari daftar konfigurasi
func (rm *RunnerManager) LoadFromConfigs(configs []model.RunnerConfig) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Hentikan dan hapus runner yang ada
	for _, state := range rm.runners {
		if state.Status == model.StatusRunning && state.Process != nil {
			StopRunner(state)
		}
		if state.LogChan != nil {
			close(state.LogChan)
		}
	}

	// Reset map
	rm.runners = make(map[string]*model.RunnerState)

	// Tambahkan runner dari konfigurasi
	for _, config := range configs {
		state := &model.RunnerState{
			Config:  config,
			Status:  model.StatusIdle,
			LogChan: make(chan string, 1000),
		}
		rm.runners[config.ID] = state
	}
}

// SaveToFile menyimpan semua konfigurasi runner ke file
func (rm *RunnerManager) SaveToFile(path string) error {
	configs := rm.GetConfigs()
	return SaveConfig(configs, path)
}

// LoadFromFile memuat konfigurasi runner dari file
func (rm *RunnerManager) LoadFromFile(path string) error {
	configs, err := LoadConfig(path)
	if err != nil {
		return err
	}
	rm.LoadFromConfigs(configs)
	return nil
}
