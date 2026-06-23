#!/usr/bin/env bash
# =============================================================================
# fetch_fonts.sh — 下載「字體版型」所需的 OFL 開源字型
# -----------------------------------------------------------------------------
# 這些字型「沒有」被 commit 進 repo（檔案大、且授權只需散布時附帶）。
# 在 build 之前於本機跑一次，把字型放進 frontend/public/fonts/。
#
#   bash scripts/fetch_fonts.sh           # 下載全部
#   bash scripts/fetch_fonts.sh --core    # 只下載基礎字型（普通/等寬會用到）
#
# 所有字型皆為 SIL Open Font License 1.1（OFL）。來源與授權見 FONT_SPEC.md
# 與 THIRD_PARTY_NOTICES.md。輸出檔名必須與 frontend/src/fontFaces.js 一致。
# =============================================================================
set -uo pipefail

# 專案根目錄（本腳本位於 scripts/）
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="$ROOT/frontend/public/fonts"
mkdir -p "$DEST"

# GitHub release 版本（如失效，更新這兩個變數即可）
LXGW_VER="v1.520"
HUNINN_VER="v2.1"

GF="https://cdn.jsdelivr.net/gh/google/fonts@main/ofl"   # Google Fonts 鏡像（OFL）

# 格式： "輸出檔名|下載網址|授權|顯示名稱"
ALL=(
  # --- 基礎覆蓋字型（多版型共用；--core 也會下載這些） ---
  "inter.ttf|$GF/inter/Inter%5Bopsz,wght%5D.ttf|OFL-1.1|Inter (拉丁)"
  "noto-sans-tc.ttf|$GF/notosanstc/NotoSansTC%5Bwght%5D.ttf|OFL-1.1|Noto Sans TC (繁中)"
  "noto-sans-jp.ttf|$GF/notosansjp/NotoSansJP%5Bwght%5D.ttf|OFL-1.1|Noto Sans JP (日文)"
  "jetbrains-mono.ttf|$GF/jetbrainsmono/JetBrainsMono%5Bwght%5D.ttf|OFL-1.1|JetBrains Mono (等寬)"

  # --- 手寫 ---
  "caveat.ttf|$GF/caveat/Caveat%5Bwght%5D.ttf|OFL-1.1|Caveat (拉丁手寫)"
  "klee-one.ttf|$GF/kleeone/KleeOne-Regular.ttf|OFL-1.1|Klee One (日系鉛筆手寫)"

  # --- 書法 / 毛筆 ---
  "noto-serif-tc.ttf|$GF/notoseriftc/NotoSerifTC%5Bwght%5D.ttf|OFL-1.1|Noto Serif TC (繁中明體)"
  "dancing-script.ttf|$GF/dancingscript/DancingScript%5Bwght%5D.ttf|OFL-1.1|Dancing Script (拉丁書寫)"
  "yuji-syuku.ttf|$GF/yujisyuku/YujiSyuku-Regular.ttf|OFL-1.1|Yuji Syuku (日文毛筆楷書)"
  "lxgw-wenkai.ttf|https://github.com/lxgw/LxgwWenKai/releases/download/$LXGW_VER/LXGWWenKai-Regular.ttf|OFL-1.1|LXGW WenKai 霞鶩文楷 (繁中楷書)"

  # --- 圓潤 / 普普 ---
  "fredoka.ttf|$GF/fredoka/Fredoka%5Bwdth%2Cwght%5D.ttf|OFL-1.1|Fredoka (拉丁圓體)"
  "jf-openhuninn.ttf|https://github.com/justfont/open-huninn-font/releases/download/$HUNINN_VER/jf-openhuninn-2.1.ttf|OFL-1.1|jf open 粉圓 (繁中圓體)"

)

CORE=("inter.ttf" "noto-sans-tc.ttf" "noto-sans-jp.ttf" "jetbrains-mono.ttf")

want_core=0
[ "${1:-}" = "--core" ] && want_core=1

in_core() { for c in "${CORE[@]}"; do [ "$c" = "$1" ] && return 0; done; return 1; }

ok=0; fail=0; skipped=0
for row in "${ALL[@]}"; do
  IFS='|' read -r out url lic name <<< "$row"
  if [ "$want_core" = "1" ] && ! in_core "$out"; then continue; fi
  target="$DEST/$out"
  if [ -s "$target" ]; then
    echo "✓ 已存在  ${out}"; skipped=$((skipped+1)); continue
  fi
  echo "↓ 下載中  $name  ->  $out"
  if curl -fL --retry 3 --retry-delay 2 -m 180 -o "$target" "$url"; then
    sz=$(wc -c < "$target" | tr -d ' ')
    if [ "$sz" -lt 2000 ]; then
      echo "✗ 失敗（檔案過小，可能是錯誤頁）：${out}（${sz} bytes）"; rm -f "$target"; fail=$((fail+1))
    else
      echo "✓ 完成   ${out}（$((sz/1024)) KB）[${lic}]"; ok=$((ok+1))
    fi
  else
    echo "✗ 失敗   $out  <- $url"; rm -f "$target"; fail=$((fail+1))
  fi
done

echo
echo "──────────────────────────────────────────"
echo "完成：${ok}　已存在：${skipped}　失敗：${fail}"
echo "輸出目錄：${DEST}"
[ "$fail" -gt 0 ] && echo "提示：GitHub release 失效時，請更新本腳本頂端的 LXGW_VER / HUNINN_VER。"
echo "授權：全部為 SIL OFL 1.1，散布時請保留 OFL 授權（見 THIRD_PARTY_NOTICES.md）。"
exit 0
