@echo off
setlocal EnableExtensions

set "ROOT=%~dp0"
set "APP_EXE=%ROOT%build\bin\HaLer AI Console.exe"

if not exist "%APP_EXE%" (
  echo [ERROR] App executable not found.
  echo Expected: %APP_EXE%
  echo.
  echo Run build.cmd first, then run app.cmd again.
  exit /b 1
)

start "" "%APP_EXE%"
