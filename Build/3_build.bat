@echo off
SETLOCAL

:: go.modが存在しない場合にのみ実行
IF NOT EXIST go.mod (
    go mod init orbit
)

:: 依存関係の整理
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
