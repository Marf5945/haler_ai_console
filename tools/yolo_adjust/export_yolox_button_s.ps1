param(
    [string]$YoloxRoot,
    [string]$ExpFile,
    [string]$Checkpoint,
    [string]$Output,
    [string]$Python,
    [switch]$NoOnnxSim
)

$ErrorActionPreference = "Stop"

$ScriptDir = $PSScriptRoot
$ProjectRoot = (Resolve-Path (Join-Path $ScriptDir "..\..")).Path

if (-not $YoloxRoot) {
    $YoloxRoot = Join-Path $ScriptDir "external\YOLOX"
}
if (-not $ExpFile) {
    $ExpFile = Join-Path $YoloxRoot "exps\example\custom\yolox_button_s.py"
}
if (-not $Checkpoint) {
    $Checkpoint = Join-Path $YoloxRoot "YOLOX_outputs\yolox_button_s\best_ckpt.pth"
}
if (-not $Output) {
    $Output = Join-Path $ProjectRoot "assets\models\yolox_button_s.onnx"
}
if (-not $Python) {
    $Python = Join-Path $ScriptDir ".venv-yolox\Scripts\python.exe"
}

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
    $compatExporter = Join-Path $ScriptDir "export_yolox_onnx_compat.py"
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
$manifestPath = Join-Path $ProjectRoot "adapter\visual_learning\model_hashes.json"
$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
if ($null -eq $manifest.models) {
    $manifest | Add-Member -MemberType NoteProperty -Name "models" -Value ([pscustomobject]@{})
}
$manifest.models | Add-Member -MemberType NoteProperty -Name "yolox_button_s.onnx" -Value "sha256:$hash" -Force
$manifest | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $manifestPath -Encoding UTF8

Write-Host "Exported: $Output"
Write-Host "SHA256: sha256:$hash"
Write-Host "Updated:  $manifestPath"
