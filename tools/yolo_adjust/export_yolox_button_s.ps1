param(
    [string]$YoloxRoot = "$env:USERPROFILE\Desktop\yolo_adjust\external\YOLOX",
    [string]$ExpFile = "$env:USERPROFILE\Desktop\yolo_adjust\external\YOLOX\exps\example\custom\yolox_button_s.py",
    [string]$Checkpoint = "$env:USERPROFILE\Desktop\yolo_adjust\external\YOLOX\YOLOX_outputs\yolox_button_s\best_ckpt.pth",
    [string]$Output = "$env:USERPROFILE\Desktop\ui_console\ui_console_wails_v_3.1.1\assets\models\yolox_button_s.onnx",
    [string]$Python = "$env:USERPROFILE\Desktop\yolo_adjust\.venv-yolox\Scripts\python.exe",
    [switch]$NoOnnxSim
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $YoloxRoot -PathType Container)) {
    throw "YOLOX root not found: $YoloxRoot"
}
if (-not (Test-Path -LiteralPath $ExpFile -PathType Leaf)) {
    throw "YOLOX exp file not found: $ExpFile"
}
if (-not (Test-Path -LiteralPath $Checkpoint -PathType Leaf)) {
    throw "YOLOX checkpoint not found: $Checkpoint"
}
if (-not (Test-Path -LiteralPath $Python -PathType Leaf)) {
    throw "Python executable not found: $Python"
}

$outputDir = Split-Path -Parent $Output
New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

Push-Location $YoloxRoot
try {
    $compatExporter = "$env:USERPROFILE\Desktop\ui_console\ui_console_wails_v_3.1.1\tools\yolo_adjust\export_yolox_onnx_compat.py"
    $args = @(
        $compatExporter,
        "--yolox-root", $YoloxRoot,
        "-f", $ExpFile,
        "-c", $Checkpoint,
        "--output-name", $Output,
        "--opset", "11"
    )
    if ($NoOnnxSim) {
        $args += "--no-onnxsim"
    }

    & $Python @args
    if ($LASTEXITCODE -ne 0) {
        throw "YOLOX ONNX export failed with exit code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}

if (-not (Test-Path -LiteralPath $Output -PathType Leaf)) {
    throw "YOLOX ONNX export did not create output: $Output"
}

$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $Output).Hash.ToLowerInvariant()
$manifestPath = "$env:USERPROFILE\Desktop\ui_console\ui_console_wails_v_3.1.1\adapter\visual_learning\model_hashes.json"
$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
if ($null -eq $manifest.models) {
    $manifest | Add-Member -MemberType NoteProperty -Name "models" -Value ([pscustomobject]@{})
}
$manifest.models | Add-Member -MemberType NoteProperty -Name "yolox_button_s.onnx" -Value "sha256:$hash" -Force
$manifest | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $manifestPath -Encoding UTF8

Write-Host "Exported: $Output"
Write-Host "SHA256: sha256:$hash"
Write-Host "Updated:  $manifestPath"
