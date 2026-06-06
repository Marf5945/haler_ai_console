# AI Console

AI Console is a Wails desktop app with a Go backend and a React/Vite frontend.

## For Users

Download the packaged app from GitHub Releases when available.

Users do not need Go, Wails, Node.js, or Git just to run a packaged release. Those tools are only required for developers who want to build from source.

Runtime requirements:

- macOS: a recent supported macOS version.
- Windows: Microsoft Edge WebView2 Runtime.
- Optional: CLI tools used by adapters, such as Codex, Claude, Gemini, or Ollama.

## For Developers

Install these tools before building from source:

- Git
- Go 1.23 or newer
- Node.js LTS, including npm
- Wails CLI v2

Windows also needs:

- Microsoft Edge WebView2 Runtime
- A C/C++ build environment if `wails doctor` reports one is missing

macOS also needs:

- Xcode Command Line Tools

## Install Tools

### macOS

Using Homebrew:

```bash
brew install git go node
xcode-select --install
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Make sure Go's bin directory is in your shell path:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Add that line to `~/.zshrc` if needed, then restart the terminal.

### Windows

Using PowerShell:

```powershell
winget install --id Git.Git -e
winget install --id GoLang.Go -e
winget install --id OpenJS.NodeJS.LTS -e
winget install --id Microsoft.EdgeWebView2Runtime -e
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Make sure this directory is in your `PATH`, then restart PowerShell:

```text
%USERPROFILE%\go\bin
```

## Check Environment

Run:

```bash
go version
node --version
npm --version
git --version
wails doctor
```

Fix anything reported by `wails doctor` before building.

## Build From Source

### Windows

From Command Prompt or PowerShell:

```cmd
build.cmd
```

For a clean Windows rebuild, especially after copying the project from macOS:

```cmd
build.cmd --clean
```

The helper checks the current Windows architecture, verifies Go/Node/npm/Wails, installs frontend dependencies, runs `wails doctor`, and then runs `wails build` for the current Windows target.

### macOS

```bash
git clone https://github.com/<owner>/<repo>.git
cd <repo>
cd frontend
npm install
cd ..
wails build
```

Build output is generated under:

```text
build/bin
```

## Optional: YOLOX Button Detection Model

The fine-tuned YOLOX-S button detection weights (`yolox_button_s.onnx`, ~34 MB)
are distributed via GitHub Releases instead of the repository. Without the model,
Visual Learning automatically falls back to OpenCV shape/text detection.

To enable full button detection:

1. Download `yolox_button_s.onnx` from the latest GitHub Release.
2. Place it at `assets/models/yolox_button_s.onnx`.
3. The app verifies the file against the SHA256 manifest
   (`adapter/visual_learning/model_hashes.json`) at load time; a mismatched or
   tampered file is rejected.

## Development

Run the app in development mode:

```bash
wails dev
```

Run frontend tests:

```bash
npm test --prefix frontend
```

## Repository Notes

Do not commit generated output or dependency folders:

- `build/bin`
- `frontend/dist`
- `frontend/node_modules`
- `*.exe`
- `.DS_Store`
- `._*`
- `assets/runtimes`
- `assets/models/*.onnx`

Do not commit device-local secrets or runtime history:

- `data/secrets`
- `data/status_rail`
- `*.enc`
- `*.dpapi`
- `.env*`

Credential storage is platform-specific by design. Windows protects the local
credential master key with DPAPI (`windows_dpapi`), while macOS protects it with
Keychain (`macos_keychain`). API keys, bot tokens, and channel secrets are stored
only through the encrypted credential store and must not be copied into source,
portable exports, logs, or documentation.

Do not copy `frontend/node_modules` between operating systems. Native packages such as `fsevents` are macOS-only, while packages such as `esbuild` install platform-specific binaries for the current OS. Keep `package.json` and `package-lock.json` in Git, then run the install/build commands on each target system.

Build outputs and platform-specific inference payloads should be rebuilt or
downloaded on the target OS. Windows DirectML runtime DLLs and ONNX model files
are optional local artifacts; they should not be committed for a macOS migration.
Without the optional YOLOX button ONNX model, Visual Learning degrades to the
OpenCV shape/text fallback instead of failing startup. Keep the small model hash
manifest in source so a locally supplied model can still be verified.

If the project was copied from macOS to Windows, remove AppleDouble metadata files before building:

```powershell
Get-ChildItem -Recurse -Force -Filter '._*' | Remove-Item -Force
Get-ChildItem -Recurse -Force -Filter '.DS_Store' | Remove-Item -Force
```

## Important Packaging Note

Do not package Go, Wails, Git, or Node.js inside the desktop app. They are developer build tools, not runtime dependencies.

For GitHub, document the required tools in this README and publish packaged apps through Releases.
