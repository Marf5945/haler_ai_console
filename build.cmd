@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "ROOT=%~dp0"
cd /d "%ROOT%" || exit /b 1

set "PINNED_GO_VERSION=1.26.4"
set "PINNED_NODE_VERSION=24.16.0"
set "PINNED_WAILS_VERSION=v2.12.0"
set "GO_AMD64_SHA256=55902c036634c7ab3159cf259af692abc86989aaefcc7f75bef888f3263031c4"
set "GO_ARM64_SHA256=b87863733cd87624387ee61307a5ebaf405351bf4035a3aa7744c26a785a3d3e"
set "NODE_X64_SHA256=43749d78a28ff11a36cb279407bc13e79bcfb8670e7926e469018d31c2ec6453"
set "NODE_ARM64_SHA256=beac2056574ebc523d5feaad7cdc434cb1d752eba076db7ebb4b62bc13ec70b9"
set "DOWNLOAD_DIR=%TEMP%\ai-console-build-tools"
set "APP_DISPLAY_NAME=HaLer AI Console"
set "BUILD_INSTALLER=0"
set "CLEAN_BUILD=0"
set "WAILS_EXE=wails"
set "BUILD_CACHE_ROOT=%TEMP%\haler-ai-console-build-cache"
set "GOCACHE=%BUILD_CACHE_ROOT%\go-build"
set "GOMODCACHE=%BUILD_CACHE_ROOT%\go-mod"

:parse_args
if "%~1"=="" goto args_done
if /i "%~1"=="--clean" (
  set "CLEAN_BUILD=1"
  shift
  goto parse_args
)
if /i "%~1"=="--installer" (
  set "BUILD_INSTALLER=1"
  shift
  goto parse_args
)
if /i "%~1"=="--release" (
  set "BUILD_INSTALLER=1"
  shift
  goto parse_args
)
echo [ERROR] Unknown argument: %~1
echo Usage: build.cmd [--clean] [--installer] [--release]
exit /b 1
:args_done

echo.
echo HaLer AI Console Windows build helper
echo =====================================
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
if "%BUILD_INSTALLER%"=="1" echo Package: NSIS installer
echo.

call :refresh_path
call :warn_missing_git
call :ensure_go || exit /b 1
call :ensure_node || exit /b 1
call :ensure_webview2 || exit /b 1
call :ensure_wails || exit /b 1
if "%BUILD_INSTALLER%"=="1" call :ensure_nsis || exit /b 1
call :refresh_path

echo Tool versions:
go version
node --version
call npm --version
call "%WAILS_EXE%" version
echo.

if "%CLEAN_BUILD%"=="1" (
  echo [CLEAN] Removing generated frontend dependency/build folders...
  if exist "frontend\node_modules" rmdir /s /q "frontend\node_modules"
  if exist "frontend\dist" rmdir /s /q "frontend\dist"
)

if not exist "frontend\package.json" (
  echo [ERROR] Missing frontend\package.json. Run this script from the repository root.
  exit /b 1
)

if not exist "%GOCACHE%" mkdir "%GOCACHE%"
if not exist "%GOMODCACHE%" mkdir "%GOMODCACHE%"

if exist "build\cache" (
  echo Removing legacy in-repo Go cache...
  rmdir /s /q "build\cache"
)

if exist "frontend\node_modules" (
  echo Removing frontend\node_modules before Wails binding generation...
  rmdir /s /q "frontend\node_modules"
)

echo.
echo Running wails doctor...
call "%WAILS_EXE%" doctor
if errorlevel 1 (
  echo [ERROR] wails doctor reported a problem. Fix the issue above, then rerun build.cmd.
  exit /b 1
)

echo.
echo Building HaLer AI Console for %WAILS_PLATFORM%...
if "%BUILD_INSTALLER%"=="1" (
  call "%WAILS_EXE%" build -platform %WAILS_PLATFORM% -nsis -webview2 download
) else (
  call "%WAILS_EXE%" build -platform %WAILS_PLATFORM%
)
if errorlevel 1 (
  echo [ERROR] Wails build failed.
  exit /b 1
)

if "%BUILD_INSTALLER%"=="1" (
  dir /b "build\bin\*installer*.exe" >nul 2>nul
  if errorlevel 1 (
    echo [ERROR] Wails build completed, but no NSIS installer was produced.
    echo         Make sure makensis.exe is installed and available on PATH, then rerun build.cmd --installer.
    exit /b 1
  )
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
if "%BUILD_INSTALLER%"=="1" (
  echo Installer output candidates:
  dir /b "build\bin\*installer*.exe" 2>nul
  for %%F in ("build\bin\*installer*.exe") do if exist "%%~fF" certutil -hashfile "%%~fF" SHA256
) else if exist "build\bin\%APP_DISPLAY_NAME%.exe" (
  echo Output: %ROOT%build\bin\%APP_DISPLAY_NAME%.exe
  certutil -hashfile "build\bin\%APP_DISPLAY_NAME%.exe" SHA256
) else (
  echo Output folder: %ROOT%build\bin
)
exit /b 0

:refresh_path
if exist "C:\Program Files\Go\bin" set "PATH=C:\Program Files\Go\bin;%PATH%"
if exist "%USERPROFILE%\go\bin" set "PATH=%USERPROFILE%\go\bin;%PATH%"
if exist "%USERPROFILE%\go\bin\wails.exe" set "WAILS_EXE=%USERPROFILE%\go\bin\wails.exe"
if exist "C:\Program Files\nodejs" set "PATH=C:\Program Files\nodejs;%PATH%"
if exist "C:\Program Files\Git\cmd" set "PATH=C:\Program Files\Git\cmd;%PATH%"
if exist "C:\Program Files (x86)\NSIS" set "PATH=C:\Program Files (x86)\NSIS;%PATH%"
if exist "C:\Program Files\NSIS" set "PATH=C:\Program Files\NSIS;%PATH%"
exit /b 0

:warn_missing_git
where git >nul 2>nul
if not errorlevel 1 exit /b 0
echo [WARN] Git is missing.
echo        Git is recommended for source control, but this build can continue from an already cloned repository.
exit /b 0

:ensure_go
call :refresh_path
set "FOUND_GO="
for /f "tokens=3" %%V in ('go version 2^>nul') do set "FOUND_GO=%%V"
if "!FOUND_GO!"=="go%PINNED_GO_VERSION%" exit /b 0
if defined FOUND_GO (
  echo [WARN] Found Go !FOUND_GO!, but this build helper is pinned to go%PINNED_GO_VERSION%.
) else (
  echo [WARN] Go is missing.
)
echo        Installing from the official Go download URL with SHA256 verification.
call :prompt_install "Go %PINNED_GO_VERSION%"
if errorlevel 1 (
  echo [ERROR] Go %PINNED_GO_VERSION% is required. Install it and rerun build.cmd.
  exit /b 1
)
if /i "%WAILS_PLATFORM%"=="windows/arm64" (
  call :install_msi_with_hash "https://go.dev/dl/go%PINNED_GO_VERSION%.windows-arm64.msi" "%DOWNLOAD_DIR%\go%PINNED_GO_VERSION%.windows-arm64.msi" "%GO_ARM64_SHA256%" "Go %PINNED_GO_VERSION%" || exit /b 1
) else (
  call :install_msi_with_hash "https://go.dev/dl/go%PINNED_GO_VERSION%.windows-amd64.msi" "%DOWNLOAD_DIR%\go%PINNED_GO_VERSION%.windows-amd64.msi" "%GO_AMD64_SHA256%" "Go %PINNED_GO_VERSION%" || exit /b 1
)
call :refresh_path
set "FOUND_GO="
for /f "tokens=3" %%V in ('go version 2^>nul') do set "FOUND_GO=%%V"
if "!FOUND_GO!"=="go%PINNED_GO_VERSION%" exit /b 0
echo [ERROR] Go %PINNED_GO_VERSION% is still missing after installation.
echo         Restart this terminal if Windows has not refreshed PATH yet.
exit /b 1

:ensure_node
call :refresh_path
set "FOUND_NODE="
for /f "delims=" %%V in ('node --version 2^>nul') do set "FOUND_NODE=%%V"
if "!FOUND_NODE!"=="v%PINNED_NODE_VERSION%" (
  where npm >nul 2>nul
  if not errorlevel 1 exit /b 0
)
if defined FOUND_NODE (
  echo [WARN] Found Node.js !FOUND_NODE!, but this build helper is pinned to v%PINNED_NODE_VERSION%.
) else (
  echo [WARN] Node.js is missing.
)
echo        Installing from the official Node.js LTS download URL with SHA256 verification.
call :prompt_install "Node.js %PINNED_NODE_VERSION%"
if errorlevel 1 (
  echo [ERROR] Node.js %PINNED_NODE_VERSION% is required. Install it and rerun build.cmd.
  exit /b 1
)
if /i "%WAILS_PLATFORM%"=="windows/arm64" (
  call :install_msi_with_hash "https://nodejs.org/dist/v%PINNED_NODE_VERSION%/node-v%PINNED_NODE_VERSION%-arm64.msi" "%DOWNLOAD_DIR%\node-v%PINNED_NODE_VERSION%-arm64.msi" "%NODE_ARM64_SHA256%" "Node.js %PINNED_NODE_VERSION%" || exit /b 1
) else (
  call :install_msi_with_hash "https://nodejs.org/dist/v%PINNED_NODE_VERSION%/node-v%PINNED_NODE_VERSION%-x64.msi" "%DOWNLOAD_DIR%\node-v%PINNED_NODE_VERSION%-x64.msi" "%NODE_X64_SHA256%" "Node.js %PINNED_NODE_VERSION%" || exit /b 1
)
call :refresh_path
set "FOUND_NODE="
for /f "delims=" %%V in ('node --version 2^>nul') do set "FOUND_NODE=%%V"
if "!FOUND_NODE!"=="v%PINNED_NODE_VERSION%" (
  where npm >nul 2>nul
  if not errorlevel 1 exit /b 0
)
echo [ERROR] Node.js %PINNED_NODE_VERSION% or npm is still missing after installation.
echo         Restart this terminal if Windows has not refreshed PATH yet.
exit /b 1

:ensure_wails
call :refresh_path
set "FOUND_WAILS="
for /f "delims=" %%V in ('"%WAILS_EXE%" version 2^>nul') do if not defined FOUND_WAILS set "FOUND_WAILS=%%V"
if defined FOUND_WAILS (
  echo !FOUND_WAILS! | findstr /C:"%PINNED_WAILS_VERSION%" >nul
  if not errorlevel 1 exit /b 0
  echo [WARN] Found Wails CLI !FOUND_WAILS!, but this build helper is pinned to %PINNED_WAILS_VERSION%.
) else (
  echo [WARN] Wails CLI is missing.
)
call :prompt_install "Wails CLI %PINNED_WAILS_VERSION%"
if errorlevel 1 (
  echo [ERROR] Wails CLI %PINNED_WAILS_VERSION% is required. Install it and rerun build.cmd.
  exit /b 1
)
echo Installing Wails CLI %PINNED_WAILS_VERSION%...
go install github.com/wailsapp/wails/v2/cmd/wails@%PINNED_WAILS_VERSION%
if errorlevel 1 (
  echo [ERROR] Failed to install Wails CLI %PINNED_WAILS_VERSION%.
  exit /b 1
)
call :refresh_path
set "FOUND_WAILS="
for /f "delims=" %%V in ('"%WAILS_EXE%" version 2^>nul') do if not defined FOUND_WAILS set "FOUND_WAILS=%%V"
echo !FOUND_WAILS! | findstr /C:"%PINNED_WAILS_VERSION%" >nul
if not errorlevel 1 exit /b 0
echo [ERROR] Wails CLI %PINNED_WAILS_VERSION% is still missing after installation.
echo         Expected executable: %WAILS_EXE%
exit /b 1

:install_msi_with_hash
if not exist "%DOWNLOAD_DIR%" mkdir "%DOWNLOAD_DIR%"
echo Downloading %~4...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri '%~1' -OutFile '%~2'"
if errorlevel 1 (
  echo [ERROR] Failed to download %~4.
  exit /b 1
)
certutil -hashfile "%~2" SHA256 | findstr /I /C:"%~3" >nul
if errorlevel 1 (
  echo [ERROR] SHA256 verification failed for %~4.
  exit /b 1
)
echo Installing %~4...
start /wait msiexec.exe /i "%~2" /qn /norestart
if errorlevel 1 (
  echo [ERROR] Failed to install %~4.
  exit /b 1
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
call :ensure_command winget "winget" "winget is required to auto-install Microsoft Edge WebView2 Runtime." "" || exit /b 1
cmd /c "winget install --id Microsoft.EdgeWebView2Runtime -e --source winget"
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

:ensure_nsis
call :refresh_path
where makensis >nul 2>nul
if not errorlevel 1 exit /b 0
echo [WARN] NSIS installer compiler is missing.
echo        It is required only when building the Windows installer with --installer.
call :prompt_install "NSIS installer compiler"
if errorlevel 1 (
  echo [ERROR] NSIS is required to produce the Windows installer.
  exit /b 1
)
echo Installing NSIS...
call :ensure_command winget "winget" "winget is required to auto-install NSIS." "" || exit /b 1
cmd /c "winget install --id NSIS.NSIS -e --source winget"
if errorlevel 1 (
  echo [ERROR] Failed to install NSIS.
  exit /b 1
)
call :refresh_path
where makensis >nul 2>nul
if errorlevel 1 (
  echo [ERROR] makensis.exe is still missing after NSIS installation.
  echo         Restart this terminal if Windows has not refreshed PATH yet.
  exit /b 1
)
exit /b 0

:prompt_install
choice /C YN /M "Install %~1 now"
if errorlevel 2 exit /b 1
exit /b 0
