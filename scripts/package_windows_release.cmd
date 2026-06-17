@echo off
setlocal EnableExtensions

set "SCRIPT_DIR=%~dp0"
set "PROJECT=%SCRIPT_DIR%.."
set "HOST_ARCH=%PROCESSOR_ARCHITECTURE%"
if /i "%PROCESSOR_ARCHITEW6432%"=="AMD64" set "HOST_ARCH=AMD64"
if /i "%PROCESSOR_ARCHITEW6432%"=="ARM64" set "HOST_ARCH=ARM64"
if /i "%HOST_ARCH%"=="ARM64" (
  set "RELEASE_ARCH=arm64"
) else (
  set "RELEASE_ARCH=amd64"
)

cd /d "%PROJECT%" || exit /b 1

echo Packaging HaLer AI Console Windows installer
echo ===========================================
echo.

call build.cmd --installer %*
if errorlevel 1 exit /b %ERRORLEVEL%

if not exist "build\release" mkdir "build\release"

set "INSTALLER="
for /f "delims=" %%F in ('dir /b /a-d /o-d "build\bin\*installer*.exe" 2^>nul') do (
  set "INSTALLER=build\bin\%%F"
  goto installer_found
)

:installer_found
if "%INSTALLER%"=="" (
  echo [ERROR] No NSIS installer was found under build\bin.
  exit /b 1
)

set "RELEASE_ASSET=build\release\HaLer-AI-Console-Windows-%RELEASE_ARCH%-installer.exe"
copy /Y "%INSTALLER%" "%RELEASE_ASSET%" >nul
if errorlevel 1 exit /b 1

certutil -hashfile "%RELEASE_ASSET%" SHA256
echo.
echo Release asset: %CD%\%RELEASE_ASSET%
exit /b 0
