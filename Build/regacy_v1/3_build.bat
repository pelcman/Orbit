@echo off
SETLOCAL

go mod init myapp

go get github.com/srwiley/oksvg
go get github.com/srwiley/rasterx
go get github.com/fredbi/uri
go get fyne.io/fyne/v2
go get gopkg.in/ini.v1

go mod tidy

echo Setting up environment variables...
set "OLD_PATH=%PATH%"
set PATH=%~dp0..\Misc\mingw64\bin;%PATH%
set CGO_ENABLED=1

echo Building the project...
go build

echo Restoring original PATH environment variable...
set PATH=%OLD_PATH%

ENDLOCAL
echo Build process completed.
pause
