@echo off
set /p ProjectName="Enter Project Name: "
if "%ProjectName%"=="" goto :eof

set "BaseDir=%~dp0Conf\Proj\%ProjectName%\common"
mkdir "%BaseDir%"

echo Creating directories and JSON files for %ProjectName%...
if not exist "%BaseDir%" mkdir "%BaseDir%"

echo {} > "%BaseDir%\Maya_env.json"
echo {} > "%BaseDir%\Blender_env.json"
echo {} > "%BaseDir%\AfterEffects_env.json"
echo {} > "%BaseDir%\Photoshop_env.json"

echo All files have been created in the common directory.
pause
