#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

exec bash "${script_dir}/wails_build_darwin.sh" --setup-only "$@"
