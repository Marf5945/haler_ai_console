# HaLer for AI Console

HaLer for AI Console is a Wails desktop app with a Go backend and a React/Vite frontend.

## 功能概覽（繁中）

HaLer for AI Console 是一個本機優先的 AI 工作台，將多種 LLM 介面、文件資料、任務流程與安全治理集中在同一個桌面應用中。

- **多模型 / 多介面整合**：支援 CLI adapter、API adapter、本機模型與 Ollama 相關流程，並可為不同 adapter 保存模型選擇與健康狀態。
- **任務 DAG 與自動流程**：可將任務拆成 DAG run、追蹤節點狀態、保留 debug trace，並在高風險步驟前以 review card 暫停等待確認。
- **Bounded Replan**：執行失敗時可在低風險、同目標、read-only 條件下嘗試重新規劃尚未開始的後續步驟；Go 端負責裁決，並保留 audit log。
- **Skill 與 Go Program Authoring**：可掃描、建立、保存與執行 skill；也能引導產生受限權限的 Go program，經驗證與 review 後再納入工作流。
- **文件、引用與搜尋**：支援拖放匯入文件、建立本機文件庫、TF-IDF / Ollama embedding 檢索、Reference prompt context、URL provenance、local search 與可設定的 web search provider。
- **內建資料工具**：提供 CSV / JSON / Office 文件讀寫能力，並包含零第三方依賴的 xlsx 產生路徑與電料 BOM 產出技能範例。
- **Visual Learning**：包含螢幕/影像元素偵測、OCR、按鈕候選、元素字典、動作候選與 dry-run 信心計算；YOLOX 模型為選配，缺少時會回退到 OpenCV shape/text 偵測。
- **Remote Bridge**：可設定 Telegram、Discord、LINE、Teams、QQ 或 custom webhook channel，支援遠端送出、審核回覆、分段 dispatch 與 audit。
- **安全治理**：包含 OS-backed credential store、source trust allowlist、LLM context 過濾、Controlled Trust、draft sandbox、package import review、W3A media provenance 與模型污染風險檢查。
- **個人化工作區**：支援 persona / avatar、主對話與 subagent 匯出、排程任務、Status Rail、語音設定與多語系 UI。

## Feature Overview (English)

HaLer for AI Console is a local-first AI workbench that brings LLM adapters, documents, task automation, and safety controls into one desktop app.

- **Multi-model and multi-adapter console**: supports CLI adapters, API adapters, local models, and Ollama-oriented flows, with per-adapter model choices and runtime health state.
- **Task DAG automation**: breaks work into DAG runs, tracks node status, keeps debug traces, and pauses for review cards before high-risk steps continue.
- **Bounded Replan**: when execution fails, the app can try to rewrite only the not-yet-started tail of a task under low-risk, same-goal, read-only constraints; Go owns the final decision and writes audit logs.
- **Skills and Go Program Authoring**: scans, builds, saves, and executes skills; guided Go program generation is validated, permission-scoped, and review-gated before entering workflows.
- **Documents, references, and search**: imports dropped documents, maintains a local document store, supports TF-IDF / Ollama embedding retrieval, builds reference prompt context, records URL provenance, and offers local search plus configurable web search providers.
- **Built-in data tools**: includes CSV / JSON / Office document readers and writers, a zero-third-party xlsx generation path, and an electrical-material BOM skill example.
- **Visual Learning**: provides screen/image element detection, OCR, button candidates, an element dictionary, action candidates, and dry-run confidence scoring; the optional YOLOX model falls back to OpenCV shape/text detection when unavailable.
- **Remote Bridge**: configures Telegram, Discord, LINE, Teams, QQ, or custom webhook channels for remote submission, review replies, segmented dispatch, and audit trails.
- **Safety and governance**: includes OS-backed credential storage, source-trust allowlists, LLM context filtering, Controlled Trust, draft sandboxing, package import review, W3A media provenance, and model-pollution risk checks.
- **Personal workspace**: supports personas and avatars, main-chat and subagent export, scheduled jobs, Status Rail, voice settings, and multilingual UI.

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
