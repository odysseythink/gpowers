@echo off
setlocal EnableDelayedExpansion

:: ================================================================
:: gpowers Windows Installer
:: ================================================================
:: This script is a Windows-friendly entry point for installing
:: gpowers. It auto-detects Git Bash or WSL, then delegates to the
:: Unix install script. After install, it fixes Kimi skill links
:: on Windows (junctions instead of symlinks, which Kimi cannot
:: follow on Windows).
::
:: Usage:
::   install.bat [--platforms=kimi] [--location=PATH] [--dry-run]
::
:: If no bash environment is found, you will be prompted to install
:: Git for Windows (recommended) or WSL.
:: ================================================================

echo ========================================
echo gpowers Windows Installer
echo ========================================
echo.

:: --- Parse arguments ---
set "INSTALL_ARGS=%*"
set "DO_KIMI=0"
set "DO_DRY_RUN=0"

:parse_loop
if "%~1"=="" goto :done_parse
if /I "%~1"=="--platforms" (
    if /I "%~2"=="kimi" set "DO_KIMI=1"
    if /I "%~2"=="all" set "DO_KIMI=1"
)
if /I "%~1"=="--platforms=kimi" set "DO_KIMI=1"
if /I "%~1"=="--platforms=all" set "DO_KIMI=1"
if /I "%~1"=="--dry-run" set "DO_DRY_RUN=1"
shift
goto :parse_loop
:done_parse

:: --- Step 1: Find a working Git Bash ---
set "GIT_BASH="

:: Check PATH entries for real Git Bash (avoid WSL stub)
for /f "delims=" %%a in ('where bash 2^>nul') do (
    set "CANDIDATE=%%a"
    :: WSL stub fails on simple echo; real bash succeeds
    "%%a" -c "echo OK" >nul 2>&1
    if !errorlevel! equ 0 (
        for /f "delims=" %%u in ('"%%a" -c "uname -o" 2^>nul') do (
            if "%%u"=="Msys" set "GIT_BASH=%%a"
        )
    )
)

:: Check common installation paths
if not defined GIT_BASH (
    if exist "C:\Program Files\Git\bin\bash.exe" set "GIT_BASH=C:\Program Files\Git\bin\bash.exe"
)
if not defined GIT_BASH (
    if exist "C:\Program Files (x86)\Git\bin\bash.exe" set "GIT_BASH=C:\Program Files (x86)\Git\bin\bash.exe"
)

:: Normalize install location so bash and batch agree (batch always uses USERPROFILE)
set "GPOWERS_WIN_LOC=%USERPROFILE%\.gpowers"
set "GPOWERS_BASH_LOC=%GPOWERS_WIN_LOC:\=/%"

:: --- Step 2: If Git Bash found, run install ---
if defined GIT_BASH (
    echo [OK] Found Git Bash: %GIT_BASH%
    echo.

    :: Enable native Windows symlinks in Git Bash
    :: (requires Developer Mode or admin privileges)
    set "MSYS=winsymlinks:nativestrict"

    echo Running: bash install %INSTALL_ARGS% --location=%GPOWERS_BASH_LOC%
    echo.
    "%GIT_BASH%" install %INSTALL_ARGS% --location=%GPOWERS_BASH_LOC%
    set "INSTALL_EXIT=%errorlevel%"

    if "%DO_DRY_RUN%"=="1" goto :success

    :: Fix Kimi links on Windows (symlinks -^> junctions/copies)
    if "%DO_KIMI%"=="1" (
        echo.
        echo [Windows] Ensuring Kimi skills are accessible...
        call :fix_kimi_links
    )

    if "%INSTALL_EXIT%"=="0" goto :success
    goto :fail
)

:: --- Step 3: Fallback to WSL ---
where wsl >nul 2>&1
if %errorlevel% equ 0 (
    wsl echo test >nul 2>&1
    if !errorlevel! equ 0 (
        echo [OK] Found WSL.
        echo.
        echo Running: wsl bash install %INSTALL_ARGS%
        echo.
        wsl bash install %INSTALL_ARGS%
        set "INSTALL_EXIT=%errorlevel%"

        if "%DO_DRY_RUN%"=="1" goto :success
        if "%DO_KIMI%"=="1" (
            echo.
            echo [Windows] Ensuring Kimi skills are accessible...
            call :fix_kimi_links
        )

        if "%INSTALL_EXIT%"=="0" goto :success
        goto :fail
    )
)

:: --- Step 4: No bash found ---
echo [ERROR] No suitable bash environment found.
echo.
echo gpowers is built on bash scripts. On Windows you need one of:
echo.
echo  Option 1 - Git for Windows  ^(recommended, lighter^):
echo    1. Download from: https://git-scm.com/download/win
echo    2. Install with default settings ^(ensure "Git Bash" is added to PATH^)
echo    3. Reopen this terminal and run:
echo         install.bat --platforms=kimi
echo.
echo  Option 2 - Windows Subsystem for Linux ^(WSL^):
echo    1. Open PowerShell as Administrator
echo    2. Run: wsl --install
echo    3. Restart your computer
echo    4. Reopen this terminal and run:
echo         install.bat --platforms=kimi
echo.
pause
exit /b 1

:: ================================================================
:: Subroutine: Fix Kimi skill links on Windows
:: ================================================================
:: Kimi on Windows cannot follow symlinks. We replace them with
:: junctions (directory hardlinks) which work without admin rights.
:: If junction fails, fall back to copying.
:: ================================================================
:fix_kimi_links
set "KIMI_SKILLS=%USERPROFILE%\.kimi\skills"
set "GPOWERS_ADAPTERS=%GPOWERS_WIN_LOC%\platforms\kimi\adapters"

if not exist "%GPOWERS_ADAPTERS%" (
    echo [WARN] Adapters not found: %GPOWERS_ADAPTERS%
    echo [WARN] Skipping Kimi link fix. If install failed, check the output above.
    exit /b 0
)

if not exist "%KIMI_SKILLS%" mkdir "%KIMI_SKILLS%"

for /f "delims=" %%d in ('dir /b "%GPOWERS_ADAPTERS%"') do (
    set "TARGET=%KIMI_SKILLS%\%%d"
    set "SOURCE=%GPOWERS_ADAPTERS%\%%d"

    :: Remove existing link/junction/file
    if exist "!TARGET!\*" (
        rmdir /s /q "!TARGET!" 2>nul
    ) else (
        del /f /q "!TARGET!" 2>nul
    )

    :: Try junction first (no admin needed, works across same-volume dirs)
    mklink /J "!TARGET!" "!SOURCE!" >nul 2>&1
    if !errorlevel! equ 0 (
        echo [OK] Junction: %%d
    ) else (
        :: Fallback: deep copy
        xcopy /E /I /Q /Y "!SOURCE!" "!TARGET!" >nul 2>&1
        if !errorlevel! equ 0 (
            echo [OK] Copied:  %%d
        ) else (
            echo [ERR] Failed:  %%d
        )
    )
)
exit /b 0

:: ================================================================
:: Exit handlers
:: ================================================================
:success
echo.
echo ========================================
echo Installation complete!
echo ========================================
echo.
pause
exit /b 0

:fail
echo.
echo ========================================
echo Installation failed. Check output above.
echo ========================================
echo.
pause
exit /b 1
