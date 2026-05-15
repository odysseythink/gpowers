@echo on
setlocal EnableDelayedExpansion

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
    "%%a" -c "echo OK" >nul 2>&1
    if !errorlevel! equ 0 (
        for /f "delims=" %%u in ('"%%a" -c "uname -o" 2^>nul') do (
            if "%%u"=="Msys" set "GIT_BASH=%%a"
        )
    )
)

echo After where loop, GIT_BASH=%GIT_BASH%

:: Check common installation paths
if not defined GIT_BASH (
    if exist "C:\Program Files\Git\bin\bash.exe" set "GIT_BASH=C:\Program Files\Git\bin\bash.exe"
)
if not defined GIT_BASH (
    if exist "C:\Program Files (x86)\Git\bin\bash.exe" set "GIT_BASH=C:\Program Files (x86)\Git\bin\bash.exe"
)

echo After fixed paths, GIT_BASH=%GIT_BASH%

:: --- Step 2: If Git Bash found, run install ---
if defined GIT_BASH (
    echo [OK] Found Git Bash: %GIT_BASH%
    echo.
    "%GIT_BASH%" install %INSTALL_ARGS%
    set "INSTALL_EXIT=%errorlevel%"
    if "%DO_DRY_RUN%"=="1" goto :success
    if "%DO_KIMI%"=="1" (
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
            echo [Windows] Ensuring Kimi skills are accessible...
            call :fix_kimi_links
        )
        if "%INSTALL_EXIT%"=="0" goto :success
        goto :fail
    )
)

:: --- Step 4: No bash found ---
echo [ERROR] No suitable bash environment found.
pause
exit /b 1

:fix_kimi_links
echo fix_kimi_links called
exit /b 0

:success
echo Installation complete!
exit /b 0

:fail
echo Installation failed.
exit /b 1
