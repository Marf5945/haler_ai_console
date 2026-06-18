#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
build_helper="${root_dir}/scripts/wails_build_darwin.sh"
app_name="HaLer AI Console"
open_app=1
clean_args=()
passthrough_args=()

usage() {
  cat <<'USAGE'
HaLer AI Console macOS installer

Usage:
  bash install_macos.sh [--clean] [--setup-only] [--no-open] [--help]

What it does:
  1. Checks macOS developer tools.
  2. Installs missing pinned tools only after asking you first.
  3. Verifies official Go and Node.js installers with SHA256.
  4. Builds the macOS app with Wails.
  5. Opens the built app unless --no-open is used.

Pinned toolchain:
  Go:      1.26.4, official go.dev pkg, SHA256 checked per CPU arch.
  Node.js: 24.16.0, official nodejs.org pkg, SHA256 checked.
  Wails:   v2.12.0, installed with Go module checksum verification.

Examples:
  bash install_macos.sh
  bash install_macos.sh --clean
  bash install_macos.sh --setup-only
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --clean)
      clean_args+=("--clean")
      shift
      ;;
    --setup-only)
      passthrough_args+=("--setup-only")
      open_app=0
      shift
      ;;
    --no-open)
      open_app=0
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "[ERROR] Unknown argument: $1"
      echo
      usage
      exit 1
      ;;
  esac
done

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "[ERROR] This installer is for macOS only."
  exit 1
fi

if [[ ! -f "${build_helper}" ]]; then
  echo "[ERROR] Missing build helper: ${build_helper}"
  echo "        Run this script from the cloned repository root."
  exit 1
fi

echo
echo "HaLer AI Console macOS installer"
echo "================================"
echo
echo "Project: ${root_dir}"
echo
echo "Security lock:"
echo "  - Go and Node.js versions are pinned and official installer SHA256 hashes are checked."
echo "  - Frontend packages are installed with npm ci from package-lock.json."
echo "  - Wails is pinned to v2.12.0 and installed through Go module checksum verification."
echo
echo "You may be asked for your Mac password if a signed official .pkg installer is needed."
echo

bash "${build_helper}" "${clean_args[@]}" "${passthrough_args[@]}"

app_path="${root_dir}/build/bin/${app_name}.app"
if [[ "${open_app}" -eq 1 ]]; then
  if [[ -d "${app_path}" ]]; then
    echo
    echo "Opening ${app_name}..."
    open -n "${app_path}"
  else
    echo
    echo "[WARN] Build finished, but the app bundle was not found at:"
    echo "       ${app_path}"
  fi
fi

echo
echo "Done."
