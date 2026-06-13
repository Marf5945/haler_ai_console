#!/usr/bin/env bash
# 把「產出電料Bom」安裝到正在執行的 App 資料夾。
# App 執行時讀的是 os.UserConfigDir()/ai-console，不是這個 repo。
# macOS: ~/Library/Application Support/ai-console
set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
APPDATA="${HOME}/Library/Application Support/ai-console"   # macOS

SKILL_SRC="${REPO}/data/skills/dianliao.bom"
SKILL_DST="${APPDATA}/data/skills/dianliao.bom"

CAT_SRC="${REPO}/_install_dianliao_bom/go_program_authoring/dianliao_bom"
CAT_DST="${APPDATA}/data/projects/default/data/go_program_authoring/dianliao_bom"

echo "App 資料夾: ${APPDATA}"
mkdir -p "$(dirname "$SKILL_DST")" "$(dirname "$CAT_DST")"

# 1) skill 套件（chat 路由用）
rm -rf "$SKILL_DST"
cp -R "$SKILL_SRC" "$SKILL_DST"
echo "✓ 已安裝 skill → ${SKILL_DST}"

# 2) 自動流程目錄項（工具 > 自動流程 顯示用）
rm -rf "$CAT_DST"
cp -R "$CAT_SRC" "$CAT_DST"
echo "✓ 已安裝自動流程項 → ${CAT_DST}"

echo
echo "完成。重開 App（或切換 工具 > 自動流程 分頁，12 秒內會自動刷新）即可看到「產出電料Bom」。"
