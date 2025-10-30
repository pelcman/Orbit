@echo off
setlocal
echo ========================================
echo   Building Orbit - ComfyUI Launcher
echo ========================================
echo.

pushd "%~sdp0"

REM Add Go to PATH manually if needed
if exist "C:\Program Files\Go\bin\go.exe" (
    set "PATH=C:\Program Files\Go\bin;%PATH%"
)
if exist "%USERPROFILE%\go\bin\go.exe" (
    set "PATH=%USERPROFILE%\go\bin;%PATH%"
)

set "MINGW_PATH=%~sdp0Misc\mingw64\bin"
set "PATH=%MINGW_PATH%;C:\Windows\System32;C:\Windows;%PATH%"
set CGO_ENABLED=1

echo Using path: %CD%
echo.

where go >nul 2>nul
if errorlevel 1 (
    echo Error: Go not found
    pause
    exit /b 1
)

where gcc >nul 2>nul
if errorlevel 1 (
    echo Error: GCC not found
    pause
    exit /b 1
)

echo Initializing Go modules...
go mod tidy
if errorlevel 1 (
    echo Error: go mod tidy failed
    pause
    exit /b 1
)

echo Building Orbit...
go build -o orbit.exe main.go
if errorlevel 1 (
    echo Error: Build failed
    pause
    exit /b 1
)

echo.
echo Build successful!
echo Output: orbit.exe
echo.

if not exist "packages" mkdir packages
if not exist "Img" mkdir Img

echo Ready to run. Execute orbit.exe to launch.
pause

popd
endlocal
