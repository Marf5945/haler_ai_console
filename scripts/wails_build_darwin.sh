#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
build_args=()
wails_bin="${WAILS_BIN:-}"

refresh_path() {
  export PATH="$HOME/go/bin:$PATH"
  if [[ -x /opt/homebrew/bin/brew ]]; then
    eval "$(/opt/homebrew/bin/brew shellenv)"
  elif [[ -x /usr/local/bin/brew ]]; then
    eval "$(/usr/local/bin/brew shellenv)"
  fi
}

confirm_install() {
  local label="$1"
  local answer
  read -r -p "Install ${label} now? [y/N] " answer
  case "${answer}" in
    [Yy]|[Yy][Ee][Ss]) return 0 ;;
    *) return 1 ;;
  esac
}

ensure_brew() {
  if command -v brew >/dev/null 2>&1; then
    return 0
  fi
  echo "[WARN] Homebrew is missing."
  echo "       Homebrew is required to auto-install missing macOS dependencies."
  if ! confirm_install "Homebrew"; then
    echo "[ERROR] Homebrew is required for automatic dependency installation."
    return 1
  fi
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  refresh_path
  command -v brew >/dev/null 2>&1
}

ensure_brew_package() {
  local command_name="$1"
  local label="$2"
  local package_name="$3"
  local reason="$4"

  if command -v "${command_name}" >/dev/null 2>&1; then
    return 0
  fi

  echo "[WARN] ${label} is missing."
  echo "       ${reason}"
  ensure_brew || return 1
  if ! confirm_install "${label}"; then
    echo "[ERROR] ${label} is required. Install it and rerun this script."
    return 1
  fi
  brew install "${package_name}"
  command -v "${command_name}" >/dev/null 2>&1
}

ensure_xcode_clt() {
  if xcode-select -p >/dev/null 2>&1; then
    return 0
  fi
  echo "[WARN] Xcode Command Line Tools are missing."
  echo "       They are required by Wails and some native builds on macOS."
  if ! confirm_install "Xcode Command Line Tools"; then
    echo "[ERROR] Xcode Command Line Tools are required. Install them and rerun this script."
    return 1
  fi
  xcode-select --install || true
  echo "[ERROR] Finish the Xcode Command Line Tools installation, then rerun this script."
  return 1
}

ensure_wails() {
  if [[ -n "${wails_bin}" && -x "${wails_bin}" ]]; then
    return 0
  fi
  if command -v wails >/dev/null 2>&1; then
    wails_bin="$(command -v wails)"
    return 0
  fi
  echo "[WARN] Wails CLI is missing."
  echo "       Wails CLI is required to package the desktop app."
  if ! confirm_install "Wails CLI"; then
    echo "[ERROR] Wails CLI is required. Install it and rerun this script."
    return 1
  fi
  go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
  refresh_path
  if command -v wails >/dev/null 2>&1; then
    wails_bin="$(command -v wails)"
    return 0
  fi
  return 1
}

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "[ERROR] This script is for macOS only."
  exit 1
fi

refresh_path

echo
echo "AI Console macOS build helper"
echo "============================="
echo

while [[ $# -gt 0 ]]; do
  case "$1" in
    --clean)
      echo "[CLEAN] Removing generated frontend dependency/build folders..."
      rm -rf "${root_dir}/frontend/node_modules" "${root_dir}/frontend/dist"
      shift
      ;;
    *)
      build_args+=("$1")
      shift
      ;;
  esac
done

ensure_xcode_clt
ensure_brew_package git "Git" "git" "Git is required to manage and build this repository."
ensure_brew_package go "Go" "go" "Go 1.23 or newer is required to build the backend."
refresh_path
ensure_brew_package node "Node.js" "node" "Node.js LTS is required to build the frontend."
if ! command -v npm >/dev/null 2>&1; then
  echo "[WARN] npm is missing."
  echo "       npm is required to install frontend dependencies."
  ensure_brew || exit 1
  if ! confirm_install "Node.js (includes npm)"; then
    echo "[ERROR] npm is required. Reinstall Node.js and rerun this script."
    exit 1
  fi
  brew install node
fi
ensure_wails

echo "Tool versions:"
go version
node --version
npm --version
"${wails_bin}" version
echo

if [[ ! -f "${root_dir}/frontend/package.json" ]]; then
  echo "[ERROR] Missing frontend/package.json. Run this script from the repository root."
  exit 1
fi

pushd "${root_dir}/frontend" >/dev/null
echo "Installing frontend dependencies..."
if [[ -f package-lock.json ]]; then
  if [[ -d node_modules ]]; then
    npm install --audit=false --fund=false
  else
    npm ci --audit=false --fund=false
  fi
else
  npm install --audit=false --fund=false
fi
popd >/dev/null

echo
echo "Running wails doctor..."
"${wails_bin}" doctor

echo
echo "Building AI Console for macOS..."
"${wails_bin}" build -ldflags "-extldflags=-Wl,-no_warn_duplicate_libraries" "${build_args[@]}"

model_src="${root_dir}/assets/models/yolox_button_s.mlmodelc"
model_dst="${root_dir}/build/bin/ai-console.app/Contents/Resources/assets/models"
if [[ -d "${model_src}" && -d "${root_dir}/build/bin/ai-console.app" ]]; then
  mkdir -p "${model_dst}"
  rm -rf "${model_dst}/yolox_button_s.mlmodelc"
  cp -R "${model_src}" "${model_dst}/"
  if command -v codesign >/dev/null 2>&1; then
    codesign --force --deep --sign - "${root_dir}/build/bin/ai-console.app" >/dev/null
  fi
else
  echo "[WARN] assets/models/yolox_button_s.mlmodelc not found; Visual Learning CoreML model will be unavailable."
fi

echo
echo "Build complete."
if [[ -d "${root_dir}/build/bin/ai-console.app" ]]; then
  echo "Output: ${root_dir}/build/bin/ai-console.app"
else
  echo "Output folder: ${root_dir}/build/bin"
fi
