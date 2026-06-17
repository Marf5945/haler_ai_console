#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
release_dir="${root_dir}/build/release"
app_name="HaLer AI Console"
app_path="${root_dir}/build/bin/${app_name}.app"
wails_bin="${WAILS_BIN:-}"

refresh_path() {
  export PATH="$HOME/go/bin:$PATH"
  local cache_root="${TMPDIR:-/tmp}/haler-ai-console-build-cache"
  export GOCACHE="${cache_root}/go-build"
  export GOMODCACHE="${cache_root}/go-mod"
  if [[ -x /opt/homebrew/bin/brew ]]; then
    eval "$(/opt/homebrew/bin/brew shellenv)"
  elif [[ -x /usr/local/bin/brew ]]; then
    eval "$(/usr/local/bin/brew shellenv)"
  fi
  mkdir -p "${GOCACHE}" "${GOMODCACHE}"
}

copy_optional_model() {
  local model_src="${root_dir}/assets/models/yolox_button_s.mlmodelc"
  local model_dst="${app_path}/Contents/Resources/assets/models"

  if [[ -d "${model_src}" && -d "${app_path}" ]]; then
    mkdir -p "${model_dst}"
    rm -rf "${model_dst}/yolox_button_s.mlmodelc"
    cp -R "${model_src}" "${model_dst}/"
  else
    echo "[WARN] assets/models/yolox_button_s.mlmodelc not found; Visual Learning CoreML model will be unavailable."
  fi
}

sign_app() {
  if command -v codesign >/dev/null 2>&1 && [[ -d "${app_path}" ]]; then
    codesign --force --deep --sign - "${app_path}" >/dev/null
  fi
}

zip_app() {
  local arch="$1"
  local zip_path="${release_dir}/HaLer-AI-Console-macOS-${arch}.zip"

  rm -f "${zip_path}"
  if command -v ditto >/dev/null 2>&1; then
    ditto -c -k --sequesterRsrc --keepParent "${app_path}" "${zip_path}"
  else
    (cd "${root_dir}/build/bin" && zip -qr "${zip_path}" "${app_name}.app")
  fi
  shasum -a 256 "${zip_path}" > "${zip_path}.sha256"
  echo "Release asset: ${zip_path}"
}

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "[ERROR] This release script is for macOS only."
  exit 1
fi

refresh_path
bash "${root_dir}/scripts/wails_build_darwin.sh" --setup-only
refresh_path

if [[ -n "${wails_bin}" && -x "${wails_bin}" ]]; then
  :
elif command -v wails >/dev/null 2>&1; then
  wails_bin="$(command -v wails)"
else
  echo "[ERROR] Wails CLI not found after setup."
  exit 1
fi

mkdir -p "${release_dir}"

if [[ -d "${root_dir}/build/cache" ]]; then
  echo "Removing legacy in-repo Go cache..."
  rm -rf "${root_dir}/build/cache"
fi

for arch in arm64 amd64; do
  platform="darwin/${arch}"
  echo
  echo "Building ${app_name} for ${platform}..."
  if [[ -d "${root_dir}/frontend/node_modules" ]]; then
    echo "Removing frontend/node_modules before Wails binding generation..."
    rm -rf "${root_dir}/frontend/node_modules"
  fi
  rm -rf "${app_path}"
  "${wails_bin}" build -platform "${platform}" -ldflags "-extldflags=-Wl,-no_warn_duplicate_libraries"
  copy_optional_model
  sign_app
  zip_app "${arch}"
done

echo
echo "macOS release packaging complete."
echo "Output folder: ${release_dir}"
