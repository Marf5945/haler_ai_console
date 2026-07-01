#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON="${SCRIPT_DIR}/.venv/bin/python"

if [[ ! -x "${PYTHON}" ]]; then
  echo "WD14 env not found. Creating it now..."
  bash "${SCRIPT_DIR}/setup_wd14_env.sh"
fi

exec "${PYTHON}" "${SCRIPT_DIR}/tagger_gui.py"
