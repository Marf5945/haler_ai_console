# sync_agents_md.ps1 — 雙向同步 CLAUDE.md 與 AGENTS.md
# 規則:內容不同時,以 mtime 較新者為準,覆寫較舊者。
# 用法:
#   單次同步(適合放進 build.cmd / pre-commit):
#     powershell -ExecutionPolicy Bypass -File scripts\sync_agents_md.ps1 -Once
#   常駐監看(每 2 秒輪詢):
#     powershell -ExecutionPolicy Bypass -File scripts\sync_agents_md.ps1
#   背景常駐:
#     Start-Process powershell -WindowStyle Hidden -ArgumentList "-ExecutionPolicy Bypass -File `"$PSScriptRoot\sync_agents_md.ps1`""

param(
    [switch]$Once,
    [int]$IntervalSec = 2
)

$repoRoot = Split-Path -Parent $PSScriptRoot
$fileA = Join-Path $repoRoot 'CLAUDE.md'
$fileB = Join-Path $repoRoot 'AGENTS.md'

function Get-ContentHash([string]$path) {
    if (-not (Test-Path $path)) { return $null }
    return (Get-FileHash -Path $path -Algorithm SHA256).Hash
}

function Sync-Pair {
    $existsA = Test-Path $fileA
    $existsB = Test-Path $fileB

    if (-not $existsA -and -not $existsB) {
        Write-Host "[sync] 兩檔皆不存在,無事可做" ; return
    }
    # 只有一邊存在 → 複製過去
    if ($existsA -and -not $existsB) {
        Copy-Item $fileA $fileB
        Write-Host "[sync] AGENTS.md 不存在,已從 CLAUDE.md 建立"
        return
    }
    if ($existsB -and -not $existsA) {
        Copy-Item $fileB $fileA
        Write-Host "[sync] CLAUDE.md 不存在,已從 AGENTS.md 建立"
        return
    }

    if ((Get-ContentHash $fileA) -eq (Get-ContentHash $fileB)) { return }

    # 內容不同 → 較新者為準
    $mtimeA = (Get-Item $fileA).LastWriteTimeUtc
    $mtimeB = (Get-Item $fileB).LastWriteTimeUtc
    if ($mtimeA -ge $mtimeB) {
        Copy-Item $fileA $fileB -Force
        Write-Host "[sync] $(Get-Date -Format 'HH:mm:ss') CLAUDE.md 較新 → 已覆寫 AGENTS.md"
    }
    else {
        Copy-Item $fileB $fileA -Force
        Write-Host "[sync] $(Get-Date -Format 'HH:mm:ss') AGENTS.md 較新 → 已覆寫 CLAUDE.md"
    }
}

if ($Once) {
    Sync-Pair
    exit 0
}

Write-Host "[sync] 監看中:CLAUDE.md <-> AGENTS.md(每 $IntervalSec 秒,Ctrl+C 結束)"
Sync-Pair
while ($true) {
    Start-Sleep -Seconds $IntervalSec
    try { Sync-Pair } catch { Write-Warning "[sync] $_" }
}
