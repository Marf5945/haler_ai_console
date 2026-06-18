#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
build_args=()
wails_bin="${WAILS_BIN:-}"
setup_only=0
app_name="HaLer AI Console"
go_version="1.26.4"
node_version="24.16.0"
wails_version="v2.12.0"
go_darwin_amd64_sha256="47b07b6e7515ec724f6d5015d7d5339e2b6467a9667d4029c8b7077b83f3fafe"
go_darwin_arm64_sha256="9d35ecdcc142f3f2b9010b495ee0051e64ccd7bcf340d3c1258fe2ceb1026c87"
node_pkg_sha256="65843aafbab48999c9d5f072746836965340c9ef2fbf17a377d3f919dcb0cb7a"

refresh_path() {
  local cache_root="${TMPDIR:-/tmp}/haler-ai-console-build-cache"
  export GOCACHE="${cache_root}/go-build"
  export GOMODCACHE="${cache_root}/go-mod"
  if [[ -x /opt/homebrew/bin/brew ]]; then
    eval "$(/opt/homebrew/bin/brew shellenv)"
  elif [[ -x /usr/local/bin/brew ]]; then
    eval "$(/usr/local/bin/brew shellenv)"
  fi
  export PATH="$HOME/go/bin:/usr/local/go/bin:/usr/local/bin:$PATH"
  mkdir -p "${GOCACHE}" "${GOMODCACHE}"
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

install_pkg_with_hash() {
  local url="$1"
  local expected_sha="$2"
  local label="$3"
  local file_name
  local tmp_file
  local actual_sha

  file_name="$(basename "${url}")"
  tmp_file="${TMPDIR:-/tmp}/${file_name}"
  echo "Downloading ${label}..."
  curl -fsSL "${url}" -o "${tmp_file}"
  actual_sha="$(shasum -a 256 "${tmp_file}" | awk '{print $1}')"
  if [[ "${actual_sha}" != "${expected_sha}" ]]; then
    echo "[ERROR] SHA256 verification failed for ${label}."
    echo "        Expected: ${expected_sha}"
    echo "        Actual:   ${actual_sha}"
    return 1
  fi
  echo "Installing ${label}..."
  sudo installer -pkg "${tmp_file}" -target /
}

ensure_git() {
  if command -v git >/dev/null 2>&1; then
    return 0
  fi
  echo "[WARN] Git is missing."
  echo "       Git is usually provided by Xcode Command Line Tools on macOS."
  ensure_xcode_clt || return 1
  command -v git >/dev/null 2>&1
}

ensure_go() {
  local found_version=""
  local arch
  local pkg_arch
  local expected_sha

  if command -v go >/dev/null 2>&1; then
    found_version="$(go version | awk '{print $3}')"
    if [[ "${found_version}" == "go${go_version}" ]]; then
      return 0
    fi
  fi

  if [[ -n "${found_version}" ]]; then
    echo "[WARN] Found Go ${found_version}, but this build helper is pinned to go${go_version}."
  else
    echo "[WARN] Go is missing."
  fi
  echo "       Installing from the official Go download URL with SHA256 verification."
  if ! confirm_install "Go ${go_version}"; then
    echo "[ERROR] Go ${go_version} is required. Install it and rerun this script."
    return 1
  fi

  arch="$(uname -m)"
  case "${arch}" in
    arm64)
      pkg_arch="arm64"
      expected_sha="${go_darwin_arm64_sha256}"
      ;;
    x86_64)
      pkg_arch="amd64"
      expected_sha="${go_darwin_amd64_sha256}"
      ;;
    *)
      echo "[ERROR] Unsupported macOS architecture: ${arch}"
      return 1
      ;;
  esac

  install_pkg_with_hash "https://go.dev/dl/go${go_version}.darwin-${pkg_arch}.pkg" "${expected_sha}" "Go ${go_version}" || return 1
  refresh_path
  found_version="$(go version | awk '{print $3}')"
  if [[ "${found_version}" == "go${go_version}" ]]; then
    return 0
  fi
  echo "[ERROR] Go ${go_version} was installed, but PATH still resolves ${found_version}."
  echo "        Current go: $(command -v go)"
  echo "        Try opening a new Terminal, or run: export PATH=\"/usr/local/go/bin:\$PATH\""
  return 1
}

ensure_node() {
  local found_version=""

  if command -v node >/dev/null 2>&1; then
    found_version="$(node --version)"
    if [[ "${found_version}" == "v${node_version}" ]] && command -v npm >/dev/null 2>&1; then
      return 0
    fi
  fi

  if [[ -n "${found_version}" ]]; then
    echo "[WARN] Found Node.js ${found_version}, but this build helper is pinned to v${node_version}."
  else
    echo "[WARN] Node.js is missing."
  fi
  echo "       Installing from the official Node.js LTS download URL with SHA256 verification."
  if ! confirm_install "Node.js ${node_version}"; then
    echo "[ERROR] Node.js ${node_version} is required. Install it and rerun this script."
    return 1
  fi

  install_pkg_with_hash "https://nodejs.org/dist/v${node_version}/node-v${node_version}.pkg" "${node_pkg_sha256}" "Node.js ${node_version}" || return 1
  refresh_path
  found_version="$(node --version 2>/dev/null || true)"
  if [[ "${found_version}" == "v${node_version}" ]] && command -v npm >/dev/null 2>&1; then
    return 0
  fi
  echo "[ERROR] Node.js ${node_version} was installed, but PATH still resolves ${found_version:-no node}."
  echo "        Current node: $(command -v node || echo not found)"
  echo "        Try opening a new Terminal, or run: export PATH=\"/usr/local/bin:\$PATH\""
  return 1
}

ensure_wails() {
  if [[ -n "${wails_bin}" && -x "${wails_bin}" ]]; then
    if "${wails_bin}" version | grep -F "${wails_version}" >/dev/null 2>&1; then
      return 0
    fi
    echo "[WARN] Wails CLI at ${wails_bin} is not ${wails_version}."
  fi
  if command -v wails >/dev/null 2>&1; then
    wails_bin="$(command -v wails)"
    if "${wails_bin}" version | grep -F "${wails_version}" >/dev/null 2>&1; then
      return 0
    fi
    echo "[WARN] Found Wails CLI, but this build helper is pinned to ${wails_version}."
  else
    echo "[WARN] Wails CLI is missing."
  fi
  echo "       Wails CLI is required to package the desktop app."
  if ! confirm_install "Wails CLI ${wails_version}"; then
    echo "[ERROR] Wails CLI ${wails_version} is required. Install it and rerun this script."
    return 1
  fi
  if [[ "${GOSUMDB:-sum.golang.org}" == "off" ]]; then
    echo "[ERROR] GOSUMDB=off disables Go module checksum verification."
    echo "        Re-run with Go's default checksum database enabled."
    return 1
  fi
  GOSUMDB="${GOSUMDB:-sum.golang.org}" go install "github.com/wailsapp/wails/v2/cmd/wails@${wails_version}"
  refresh_path
  if command -v wails >/dev/null 2>&1; then
    wails_bin="$(command -v wails)"
    "${wails_bin}" version | grep -F "${wails_version}" >/dev/null 2>&1
    return $?
  fi
  return 1
}

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "[ERROR] This script is for macOS only."
  exit 1
fi

refresh_path

echo
echo "HaLer AI Console macOS build helper"
echo "==================================="
echo

while [[ $# -gt 0 ]]; do
  case "$1" in
    --clean)
      echo "[CLEAN] Removing generated frontend dependency/build folders..."
      rm -rf "${root_dir}/frontend/node_modules" "${root_dir}/frontend/dist"
      shift
      ;;
    --setup-only)
      setup_only=1
      shift
      ;;
    *)
      build_args+=("$1")
      shift
      ;;
  esac
done

ensure_xcode_clt
ensure_git
ensure_go
refresh_path
ensure_node
ensure_wails

echo "Tool versions:"
go version
node --version
npm --version
"${wails_bin}" version
echo

if [[ "${setup_only}" -eq 1 ]]; then
  echo "Setup complete. Run bash scripts/wails_build_darwin.sh to build the app."
  exit 0
fi

if [[ ! -f "${root_dir}/frontend/package.json" ]]; then
  echo "[ERROR] Missing frontend/package.json. Run this script from the repository root."
  exit 1
fi

if [[ -d "${root_dir}/build/cache" ]]; then
  echo "Removing legacy in-repo Go cache..."
  rm -rf "${root_dir}/build/cache"
fi

if [[ -d "${root_dir}/frontend/node_modules" ]]; then
  echo "Removing frontend/node_modules before Wails binding generation..."
  rm -rf "${root_dir}/frontend/node_modules"
fi

echo
echo "Running wails doctor..."
"${wails_bin}" doctor

echo
echo "Building ${app_name} for macOS..."
if [[ "${#build_args[@]}" -gt 0 ]]; then
  "${wails_bin}" build -ldflags "-extldflags=-Wl,-no_warn_duplicate_libraries" "${build_args[@]}"
else
  "${wails_bin}" build -ldflags "-extldflags=-Wl,-no_warn_duplicate_libraries"
fi

model_src="${root_dir}/assets/models/yolox_button_s.mlmodelc"
model_dst="${root_dir}/build/bin/${app_name}.app/Contents/Resources/assets/models"
if [[ -d "${model_src}" && -d "${root_dir}/build/bin/${app_name}.app" ]]; then
  mkdir -p "${model_dst}"
  rm -rf "${model_dst}/yolox_button_s.mlmodelc"
  cp -R "${model_src}" "${model_dst}/"
  if command -v codesign >/dev/null 2>&1; then
    codesign --force --deep --sign - "${root_dir}/build/bin/${app_name}.app" >/dev/null
  fi
else
  echo "[WARN] assets/models/yolox_button_s.mlmodelc not found; Visual Learning CoreML model will be unavailable."
fi

echo
echo "Build complete."
if [[ -d "${root_dir}/build/bin/${app_name}.app" ]]; then
  echo "Output: ${root_dir}/build/bin/${app_name}.app"
else
  echo "Output folder: ${root_dir}/build/bin"
fi
