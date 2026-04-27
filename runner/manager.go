package runner

import (
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

	// Mulai runner
	err := StartRunner(state)
	if err != nil {
		state.Status = model.StatusError
		return err
	}

	return nil
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
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, state := range rm.runners {
		if state.Status == model.StatusRunning && state.Process != nil {
			StopRunner(state)
		}
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
