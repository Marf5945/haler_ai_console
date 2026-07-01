#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="${SCRIPT_DIR}/.venv"

python3 -m venv "${VENV_DIR}"
"${VENV_DIR}/bin/python" -m pip install --upgrade pip
"${VENV_DIR}/bin/python" -m pip install --only-binary=:all: -r "${SCRIPT_DIR}/requirements-wd14.txt"
"${VENV_DIR}/bin/python" -m pip_audit

cat <<EOF

WD14 environment ready.

Activate:
  source "${VENV_DIR}/bin/activate"

Run:
  "${VENV_DIR}/bin/python" "${SCRIPT_DIR}/caption_tagger.py" /path/to/images --token my_style_token --wd14 --review
EOF
