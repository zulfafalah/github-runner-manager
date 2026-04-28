# GitHub Runner Manager

A desktop GUI application for managing multiple GitHub Actions self-hosted runners, built with Go and [Fyne](https://fyne.io/).

## Features

- Add, configure, and remove multiple GitHub Actions runners
- Start and stop runners with a single click
- Real-time log streaming per runner
- Persistent configuration saved to disk
- Cross-platform support (Linux, macOS, Windows)

## Requirements

- Go 1.23+
- GitHub Actions runner binary installed on the machine
- A GitHub repository registration token

## Installation

```bash
git clone https://github.com/your-username/github-runner-manager.git
cd github-runner-manager
go build -o github-runner-manager .
./github-runner-manager
```

## Configuration

Configuration is stored automatically at:

| Platform | Path |
|----------|------|
| Linux    | `~/.config/github-runner-manager/config.json` |
| macOS    | `~/Library/Application Support/github-runner-manager/config.json` |
| Windows  | `%APPDATA%\github-runner-manager\config.json` |

## Usage

1. Launch the application.
2. Click **Add Runner** to register a new runner by providing:
   - **Name** – display name for the runner
   - **Repository URL** – e.g. `https://github.com/owner/repo`
   - **Token** – GitHub Actions registration token
   - **Work Directory** – local path where the runner will be installed
   - **Labels** – optional custom labels
3. Select a runner from the list to view its logs and status.
4. Use the **Start** / **Stop** buttons to control the runner process.

## Project Structure

```
├── main.go          # Entry point
├── model/           # Data models (RunnerConfig, RunnerState)
├── runner/          # Runner logic (install, start, stop, config persistence)
└── ui/              # Fyne GUI (app, runner list, detail panel, log panel, toolbar)
```

## License

MIT
