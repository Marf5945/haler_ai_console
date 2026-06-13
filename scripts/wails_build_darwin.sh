#!/usr/bin/env bash
set -euo pipefail

wails_bin="${WAILS_BIN:-$HOME/go/bin/wails}"
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"${wails_bin}" build -ldflags "-extldflags=-Wl,-no_warn_duplicate_libraries" "$@"

model_src="${root_dir}/assets/models/yolox_button_s.mlmodelc"
model_dst="${root_dir}/build/bin/ai-console.app/Contents/Resources/assets/models"
if [[ -d "${model_src}" && -d "${root_dir}/build/bin/ai-console.app" ]]; then
  mkdir -p "${model_dst}"
  rm -rf "${model_dst}/yolox_button_s.mlmodelc"
  cp -R "${model_src}" "${model_dst}/"
  if command -v codesign >/dev/null 2>&1; then
    codesign --force --deep --sign - "${root_dir}/build/bin/ai-console.app" >/dev/null
  fi
fi
