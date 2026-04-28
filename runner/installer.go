package runner

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github-runner-manager/model"
)

const githubRunnerRepo = "actions/runner"

// getPlatformSuffix mengembalikan os dan arch suffix untuk download
func getPlatformSuffix() (string, string, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	var platformOS string
	var platformArch string

	switch osName {
	case "linux":
		platformOS = "linux"
		if arch == "amd64" {
			platformArch = "x64"
		} else if arch == "arm64" {
			platformArch = "arm64"
		} else {
			return "", "", fmt.Errorf("unsupported architecture: %s", arch)
		}
	case "darwin":
		platformOS = "osx"
		if arch == "amd64" {
			platformArch = "x64"
		} else if arch == "arm64" {
			platformArch = "arm64"
		} else {
			return "", "", fmt.Errorf("unsupported architecture: %s", arch)
		}
	case "windows":
		platformOS = "win"
		platformArch = "x64" // Windows runner hanya tersedia untuk x64
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", osName)
	}

	return platformOS, platformArch, nil
}

// getLatestRunnerVersion mengambil versi terbaru dari GitHub API
func getLatestRunnerVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRunnerRepo)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode release info: %w", err)
	}

	// Hapus prefix 'v' jika ada
	version := strings.TrimPrefix(release.TagName, "v")
	return version, nil
}

// getDownloadURL mengembalikan URL download untuk platform saat ini
func getDownloadURL(version string) (string, string, error) {
	platformOS, platformArch, err := getPlatformSuffix()
	if err != nil {
		return "", "", err
	}

	var filename string
	var ext string

	if runtime.GOOS == "windows" {
		filename = fmt.Sprintf("actions-runner-%s-%s-%s", platformOS, platformArch, version)
		ext = ".zip"
	} else {
		filename = fmt.Sprintf("actions-runner-%s-%s-%s", platformOS, platformArch, version)
		ext = ".tar.gz"
	}

	url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s%s",
		githubRunnerRepo, version, filename, ext)

	return url, ext, nil
}

// downloadFile mengunduh file dari URL ke path tujuan
func downloadFile(url, destPath string, statusChan chan<- string) error {
	statusChan <- fmt.Sprintf("Downloading from: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	statusChan <- fmt.Sprintf("Downloaded %d bytes", written)
	return nil
}

// extractTarGz mengekstrak file tar.gz ke direktori tujuan
func extractTarGz(srcPath, destDir string, statusChan chan<- string) error {
	statusChan <- "Extracting archive..."

	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		targetPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file: %w", err)
			}
			outFile.Close()
		}
	}

	return nil
}

// extractZip mengekstrak file zip ke direktori tujuan
func extractZip(srcPath, destDir string, statusChan chan<- string) error {
	statusChan <- "Extracting archive..."

	zipReader, err := zip.OpenReader(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		targetPath := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(targetPath, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		srcFile, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		_, err = io.Copy(outFile, srcFile)
		srcFile.Close()
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}

	return nil
}

// runConfigScript menjalankan config.sh untuk mendaftarkan runner ke GitHub
func runConfigScript(workDir, repoURL, token, name string, labels []string, statusChan chan<- string) error {
	statusChan <- "Configuring runner..."

	var configScript string
	if runtime.GOOS == "windows" {
		configScript = filepath.Join(workDir, "config.cmd")
	} else {
		configScript = filepath.Join(workDir, "config.sh")
	}

	// Cek apakah script config ada
	if _, err := os.Stat(configScript); os.IsNotExist(err) {
		return fmt.Errorf("config script not found at %s", configScript)
	}

	// Pastikan config.sh executable
	if runtime.GOOS != "windows" {
		if err := os.Chmod(configScript, 0755); err != nil {
			return fmt.Errorf("failed to chmod config script: %w", err)
		}
	}

	// Parse owner dan repo dari URL
	// Format: https://github.com/owner/repo
	parts := strings.Split(strings.TrimPrefix(repoURL, "https://github.com/"), "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid repo URL format")
	}
	owner, repo := parts[0], strings.TrimSuffix(parts[1], ".git")

	// Buat command dan arguments
	var cmd *exec.Cmd
	args := []string{
		"--url", fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		"--token", token,
		"--name", name, // nama unik agar beberapa runner bisa berjalan bersamaan
		"--unattended", // mode otomatis tanpa interaksi
		"--replace",    // ganti jika runner dengan nama sama sudah ada
	}

	if len(labels) > 0 {
		args = append(args, "--labels", strings.Join(labels, ","))
	}

	if runtime.GOOS == "windows" {
		allArgs := append([]string{"/c", configScript}, args...)
		cmd = exec.Command("cmd", allArgs...)
	} else {
		cmd = exec.Command(configScript, args...)
	}

	cmd.Dir = workDir
	cmd.Env = os.Environ()

	// Capture output dan stream ke log channel
	statusChan <- "Running config script..."

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		errorMsg := strings.TrimSpace(errBuf.String())
		if errorMsg == "" {
			errorMsg = err.Error()
		}
		return fmt.Errorf("config script failed: %s", errorMsg)
	}

	// Stream output ke log channel
	for _, line := range strings.Split(strings.TrimSpace(outBuf.String()), "\n") {
		if line != "" {
			select {
			case statusChan <- line:
			default:
			}
		}
	}

	return nil
}

// InstallRunner mengunduh dan menginstal GitHub Actions runner
func InstallRunner(config model.RunnerConfig, statusChan chan<- string) error {
	// Pastikan work directory ada
	if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}

	statusChan <- "Checking for latest runner version..."

	// Dapatkan versi terbaru
	version, err := getLatestRunnerVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}
	statusChan <- fmt.Sprintf("Latest version: %s", version)

	// Dapatkan URL download
	url, ext, err := getDownloadURL(version)
	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	// Download file
	archivePath := filepath.Join(config.WorkDir, "runner"+ext)
	if err := downloadFile(url, archivePath, statusChan); err != nil {
		return fmt.Errorf("failed to download runner: %w", err)
	}

	// Ekstrak archive
	if ext == ".zip" {
		err = extractZip(archivePath, config.WorkDir, statusChan)
	} else {
		err = extractTarGz(archivePath, config.WorkDir, statusChan)
	}
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Hapus archive
	os.Remove(archivePath)
	statusChan <- "Installation complete"

	// Jalankan config.sh untuk mendaftarkan runner ke GitHub
	if err := runConfigScript(config.WorkDir, config.RepoURL, config.Token, config.Name, config.Labels, statusChan); err != nil {
		return fmt.Errorf("failed to configure runner: %w", err)
	}

	statusChan <- "Runner configured successfully"
	return nil
}

// CheckRunnerInstalled memeriksa apakah runner sudah terinstal di work directory
func CheckRunnerInstalled(workDir string) bool {
	if runtime.GOOS == "windows" {
		_, err := os.Stat(filepath.Join(workDir, "run.cmd"))
		return err == nil
	}

	_, err := os.Stat(filepath.Join(workDir, "run.sh"))
	return err == nil
}

// CheckRunnerConfigured memeriksa apakah runner sudah dikonfigurasi (terdaftar ke GitHub)
func CheckRunnerConfigured(workDir string) bool {
	_, err := os.Stat(filepath.Join(workDir, ".runner"))
	return err == nil
}
