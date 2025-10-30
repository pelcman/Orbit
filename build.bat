@echo off
echo ========================================
echo   Building Orbit - ComfyUI Launcher
echo ========================================
echo.

REM Go to the project directory
cd /d "%~dp0"

REM Add MinGW to PATH if it exists
set MINGW_PATH=%~dp0Misc\mingw64\bin
if exist "%MINGW_PATH%" (
    echo Adding MinGW-w64 to PATH...
    set PATH=%MINGW_PATH%;%PATH%
)

REM Enable CGO for Fyne
set CGO_ENABLED=1

REM Check if go is installed
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Error: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

echo Current Go version:
go version
echo.

REM Check if gcc is available
where gcc >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Error: GCC (MinGW-w64) is not found in PATH
    echo Please install MinGW-w64 or check Misc\mingw64\bin
    pause
    exit /b 1
)

echo Current GCC version:
gcc --version | findstr gcc
echo.

REM Clean previous build
if exist "orbit.exe" (
    echo Cleaning previous build...
    del /f /q "orbit.exe"
)

REM Initialize go modules
echo Initializing Go modules...
go mod tidy
if %ERRORLEVEL% NEQ 0 (
    echo Error: Failed to initialize Go modules
    pause
    exit /b 1
)

REM Build the application
echo Building Orbit...
go build -ldflags="-H windowsgui" -o orbit.exe main.go
if %ERRORLEVEL% NEQ 0 (
    echo Error: Build failed
    pause
    exit /b 1
)

echo.
echo ========================================
echo   Build successful!
echo   Output: orbit.exe
echo ========================================
echo.

REM Create necessary directories
if not exist "packages" mkdir packages
if not exist "Img" mkdir Img

echo Ready to run. Execute orbit.exe to launch.
pause
