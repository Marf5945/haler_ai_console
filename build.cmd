@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "ROOT=%~dp0"
cd /d "%ROOT%" || exit /b 1

echo.
echo AI Console Windows build helper
echo ===============================
echo.

if /i "%OS%"=="Windows_NT" (
  set "HOST_OS=windows"
) else (
  echo [ERROR] This .cmd helper is for Windows. On macOS, use scripts/wails_build_darwin.sh or run wails build.
  exit /b 1
)

set "HOST_ARCH=%PROCESSOR_ARCHITECTURE%"
if /i "%PROCESSOR_ARCHITEW6432%"=="AMD64" set "HOST_ARCH=AMD64"
if /i "%PROCESSOR_ARCHITEW6432%"=="ARM64" set "HOST_ARCH=ARM64"

if /i "%HOST_ARCH%"=="AMD64" (
  set "WAILS_PLATFORM=windows/amd64"
) else if /i "%HOST_ARCH%"=="ARM64" (
  set "WAILS_PLATFORM=windows/arm64"
) else (
  set "WAILS_PLATFORM=windows/amd64"
)

echo Host: %HOST_OS% %HOST_ARCH%
echo Wails target: %WAILS_PLATFORM%
echo.

if exist "%USERPROFILE%\go\bin" set "PATH=%USERPROFILE%\go\bin;%PATH%"
if exist "C:\Program Files\Go\bin" set "PATH=C:\Program Files\Go\bin;%PATH%"
if exist "C:\Program Files\Git\cmd" set "PATH=C:\Program Files\Git\cmd;%PATH%"

call :ensure_command winget "winget" "winget is required to auto-install missing Windows dependencies." "" || exit /b 1
call :ensure_command git "Git" "Git is required to manage and build this repository." "winget install --id Git.Git -e" || exit /b 1
call :ensure_command go "Go" "Go 1.23 or newer is required to build the backend." "winget install --id GoLang.Go -e" || exit /b 1
if exist "%USERPROFILE%\go\bin" set "PATH=%USERPROFILE%\go\bin;%PATH%"
if exist "C:\Program Files\Go\bin" set "PATH=C:\Program Files\Go\bin;%PATH%"
call :ensure_command node "Node.js" "Node.js LTS is required to build the frontend." "winget install --id OpenJS.NodeJS.LTS -e" || exit /b 1
call :ensure_command npm "npm" "npm is required to install frontend dependencies." "winget install --id OpenJS.NodeJS.LTS -e" || exit /b 1
call :ensure_webview2 || exit /b 1
call :ensure_command wails "Wails CLI" "Wails CLI is required to package the desktop app." "go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0" || exit /b 1
if exist "%USERPROFILE%\go\bin" set "PATH=%USERPROFILE%\go\bin;%PATH%"

echo Tool versions:
go version
node --version
call npm --version
call wails version
echo.

if /i "%~1"=="--clean" (
  echo [CLEAN] Removing generated frontend dependency/build folders...
  if exist "frontend\node_modules" rmdir /s /q "frontend\node_modules"
  if exist "frontend\dist" rmdir /s /q "frontend\dist"
)

if exist "frontend\node_modules\fsevents\fsevents.node" (
  echo [WARN] macOS-only fsevents.node was found in frontend\node_modules.
  echo        node_modules cannot be reused across macOS and Windows.
  choice /C YN /M "Delete frontend\node_modules and reinstall for Windows"
  if errorlevel 2 (
    echo [ERROR] Please remove frontend\node_modules manually, then rerun build.cmd.
    exit /b 1
  )
  rmdir /s /q "frontend\node_modules"
)

if not exist "frontend\package.json" (
  echo [ERROR] Missing frontend\package.json. Run this script from the repository root.
  exit /b 1
)

pushd frontend || exit /b 1
echo Installing frontend dependencies...
if exist "package-lock.json" (
  if exist "node_modules" (
    call npm install --audit=false --fund=false
  ) else (
    call npm ci --audit=false --fund=false
  )
) else (
  call npm install --audit=false --fund=false
)
if errorlevel 1 (
  popd
  echo [ERROR] npm dependency install failed.
  exit /b 1
)
popd

echo.
echo Running wails doctor...
call wails doctor
if errorlevel 1 (
  echo [ERROR] wails doctor reported a problem. Fix the issue above, then rerun build.cmd.
  exit /b 1
)

echo.
echo Building AI Console for %WAILS_PLATFORM%...
call wails build -platform %WAILS_PLATFORM%
if errorlevel 1 (
  echo [ERROR] Wails build failed.
  exit /b 1
)

if exist "assets\models\yolox_button_s.onnx" (
  echo.
  echo Copying YOLOX-S button model...
  if not exist "build\bin\assets\models" mkdir "build\bin\assets\models"
  copy /Y "assets\models\yolox_button_s.onnx" "build\bin\assets\models\yolox_button_s.onnx" >nul
  if errorlevel 1 (
    echo [ERROR] Failed to copy assets\models\yolox_button_s.onnx
    exit /b 1
  )
) else (
  echo [WARN] assets\models\yolox_button_s.onnx not found; Visual Learning YOLO will run degraded.
)

if exist "assets\runtimes\onnxruntime-directml\1.24.4\win-x64\onnxruntime.dll" (
  echo.
  echo Copying locked ONNX Runtime DirectML 1.24.4 runtime...
  if not exist "build\bin\assets\runtimes\onnxruntime-directml\1.24.4\win-x64" mkdir "build\bin\assets\runtimes\onnxruntime-directml\1.24.4\win-x64"
  xcopy /Y /Q "assets\runtimes\onnxruntime-directml\1.24.4\win-x64\*.dll" "build\bin\assets\runtimes\onnxruntime-directml\1.24.4\win-x64\" >nul
  if errorlevel 1 (
    echo [ERROR] Failed to copy locked ONNX Runtime DirectML runtime.
    exit /b 1
  )
) else (
  echo [WARN] Locked ONNX Runtime DirectML 1.24.4 runtime not found; Visual Learning YOLO will run OpenCV-only.
)

echo.
echo Build complete.
if exist "build\bin\ai-console.exe" (
  echo Output: %ROOT%build\bin\ai-console.exe
  certutil -hashfile "build\bin\ai-console.exe" SHA256
) else (
  echo Output folder: %ROOT%build\bin
)
exit /b 0

:ensure_command
where %~1 >nul 2>nul
if not errorlevel 1 exit /b 0
echo [WARN] %~2 is missing.
echo        %~3
if "%~4"=="" (
  echo [ERROR] Unable to auto-install %~2. Please install it manually, then rerun build.cmd.
  exit /b 1
)
call :prompt_install "%~2"
if errorlevel 1 (
  echo [ERROR] %~2 is required. Install it and rerun build.cmd.
  exit /b 1
)
echo Installing %~2...
cmd /c "%~4"
if errorlevel 1 (
  echo [ERROR] Failed to install %~2.
  exit /b 1
)
where %~1 >nul 2>nul
if errorlevel 1 (
  echo [ERROR] %~2 is still missing after installation.
  exit /b 1
)
exit /b 0

:ensure_webview2
call :has_webview2
if not errorlevel 1 exit /b 0
echo [WARN] Microsoft Edge WebView2 Runtime is missing.
echo        It is required to run the packaged Windows app.
call :prompt_install "Microsoft Edge WebView2 Runtime"
if errorlevel 1 (
  echo [ERROR] WebView2 Runtime is required. Install it and rerun build.cmd.
  exit /b 1
)
echo Installing Microsoft Edge WebView2 Runtime...
cmd /c "winget install --id Microsoft.EdgeWebView2Runtime -e"
if errorlevel 1 (
  echo [ERROR] Failed to install Microsoft Edge WebView2 Runtime.
  exit /b 1
)
call :has_webview2
if errorlevel 1 (
  echo [ERROR] WebView2 Runtime is still missing after installation.
  exit /b 1
)
exit /b 0

:has_webview2
reg query "HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv >nul 2>nul && exit /b 0
reg query "HKLM\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv >nul 2>nul && exit /b 0
reg query "HKCU\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv >nul 2>nul && exit /b 0
exit /b 1

:prompt_install
choice /C YN /M "Install %~1 now"
if errorlevel 2 exit /b 1
exit /b 0
