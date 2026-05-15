@echo off
:: gpowers polyglot wrapper: invoked by Claude Code on Windows.
:: Delegates to PowerShell sibling .ps1 with the same basename as $1.
setlocal
set HOOK_NAME=%~1
if "%HOOK_NAME%"=="" (
  echo run-hook.cmd: missing hook name argument 1>&2
  exit /b 2
)
set PLATFORM=%~2
if "%PLATFORM%"=="" set PLATFORM=claude-code
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0%HOOK_NAME%.ps1" "%PLATFORM%"
exit /b %ERRORLEVEL%
