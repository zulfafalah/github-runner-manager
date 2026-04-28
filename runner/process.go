package runner

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github-runner-manager/model"
)

// StartRunner memulai runner process dan mengalirkan log ke channel
func StartRunner(state *model.RunnerState) error {
	workDir := state.Config.WorkDir

	// Tentukan script yang akan dijalankan
	var scriptName string
	if runtime.GOOS == "windows" {
		scriptName = "run.cmd"
	} else {
		scriptName = "run.sh"
	}

	scriptPath := filepath.Join(workDir, scriptName)

	// Verifikasi script ada
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("runner script not found at %s", scriptPath)
	}

	// Buat command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", scriptPath)
	} else {
		// Pastikan script executable
		os.Chmod(scriptPath, 0755)
		cmd = exec.Command(scriptPath)
	}

	// Set working directory
	cmd.Dir = workDir

	// Set environment variables jika diperlukan
	cmd.Env = os.Environ()

	// Gunakan process group agar sinyal diteruskan ke semua child processes
	// (termasuk runner binary yang dijalankan oleh run.sh)
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	// Setup pipes untuk stdout dan stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Jalankan process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start runner: %w", err)
	}

	// Simpan process reference
	state.Process = cmd.Process
	state.Status = model.StatusRunning
	state.ExitChan = make(chan error, 1)

	// Goroutine untuk membaca stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case state.LogChan <- fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), line):
			default:
				// Channel penuh, skip log
			}
		}
	}()

	// Goroutine untuk membaca stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case state.LogChan <- fmt.Sprintf("[%s] [ERR] %s", time.Now().Format("15:04:05"), line):
			default:
				// Channel penuh, skip log
			}
		}
	}()

	// Goroutine untuk menunggu process selesai
	go func() {
		err := cmd.Wait()
		state.ExitChan <- err
		close(state.ExitChan)

		if state.Stopping {
			// Dihentikan secara sengaja — bukan error
			state.Status = model.StatusStopped
			state.Stopping = false
			select {
			case state.LogChan <- fmt.Sprintf("[%s] Runner stopped", time.Now().Format("15:04:05")):
			default:
			}
		} else if err != nil {
			state.Status = model.StatusError
			select {
			case state.LogChan <- fmt.Sprintf("[%s] Runner exited with error: %v", time.Now().Format("15:04:05"), err):
			default:
			}
		} else {
			state.Status = model.StatusStopped
			select {
			case state.LogChan <- fmt.Sprintf("[%s] Runner stopped", time.Now().Format("15:04:05")):
			default:
			}
		}

		// Reset process reference
		state.Process = nil
	}()

	return nil
}

// StopRunner menghentikan runner process
func StopRunner(state *model.RunnerState) error {
	if state.Process == nil {
		return fmt.Errorf("runner is not running")
	}

	// Tandai bahwa ini penghentian yang disengaja
	state.Stopping = true

	var err error
	if runtime.GOOS == "windows" {
		err = state.Process.Kill()
	} else {
		// Kirim SIGTERM ke seluruh process group (negatif PID).
		// Ini memastikan sinyal sampai ke runner binary (child dari run.sh),
		// sehingga runner sempat unregister session dari GitHub.
		pgid := state.Process.Pid
		err = syscall.Kill(-pgid, syscall.SIGTERM)
		if err != nil {
			// Fallback: kirim langsung ke process
			err = state.Process.Signal(syscall.SIGTERM)
			if err != nil {
				err = state.Process.Kill()
			}
		}
	}

	if err != nil {
		state.Stopping = false
		return fmt.Errorf("failed to stop runner: %w", err)
	}

	// Tunggu process benar-benar selesai (dengan timeout)
	// Status akan di-set oleh goroutine Wait setelah proses selesai
	if state.ExitChan != nil {
		select {
		case <-state.ExitChan:
			// Process selesai dengan bersih
		case <-time.After(15 * time.Second):
			// Timeout, force kill seluruh process group
			if state.Process != nil {
				syscall.Kill(-state.Process.Pid, syscall.SIGKILL)
				state.Process.Kill()
			}
			// Tunggu lagi sebentar setelah force kill
			select {
			case <-state.ExitChan:
			case <-time.After(3 * time.Second):
			}
		}
	}

	return nil
}

// IsRunning memeriksa apakah runner sedang berjalan
func IsRunning(state *model.RunnerState) bool {
	if state.Process == nil {
		return false
	}

	// Cek apakah process masih ada
	err := state.Process.Signal(os.Signal(nil))
	return err == nil
}

// GetRunnerLogFile mengembalikan path ke file log runner (jika ada)
func GetRunnerLogFile(workDir string) string {
	// GitHub Actions runner menyimpan log di _diag folder
	diagDir := filepath.Join(workDir, "_diag")
	return diagDir
}
