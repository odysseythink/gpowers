@echo off
setlocal enabledelayedexpansion

if "%~1"=="" (
    set "TARGET=all"
) else (
    set "TARGET=%~1"
)

:: Version info
for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%i"
if not defined VERSION set "VERSION=dev"
for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set "COMMIT=%%i"
if not defined COMMIT set "COMMIT=none"

:: Get UTC date via wmic (works on all Windows)
for /f %%a in ('wmic os get localdatetime /value ^| find "="') do set "RAWDT=%%a"
set "RAWDT=!RAWDT:~14,14!"
if defined RAWDT (
    set "DATE=!RAWDT:~0,4!-!RAWDT:~4,2!-!RAWDT:~6,2!T!RAWDT:~8,2!:!RAWDT:~10,2!:!RAWDT:~12,2!Z"
) else (
    set "DATE=unknown"
)

set "LDFLAGS=-s -w -X main.version=%VERSION% -X main.commit=%COMMIT% -X main.date=%DATE%"
set "BIN=build"

:: Targets
if /i "%TARGET%"=="help" goto _help
if /i "%TARGET%"=="h" goto _help
if /i "%TARGET%"=="all" goto _all
if /i "%TARGET%"=="build" goto _build
if /i "%TARGET%"=="build-install" goto _build_install
if /i "%TARGET%"=="build-rewrite" goto _build_rewrite
if /i "%TARGET%"=="build-browse" goto _build_browse
if /i "%TARGET%"=="build-terminal" goto _build_terminal
if /i "%TARGET%"=="test" goto _test
if /i "%TARGET%"=="tidy" goto _tidy
if /i "%TARGET%"=="lint" goto _lint
if /i "%TARGET%"=="clean" goto _clean

echo Unknown target: %TARGET%
exit /b 1

:_help
echo gpowers Makefile (Windows CMD)
echo.
echo Usage: make [target]
echo.
echo Targets:
echo   all, build        Build all binaries
echo   build-install     Build install.exe
echo   build-rewrite     Build _gpowers-rewrite-browser.exe
echo   build-browse      Build browse.exe
echo   build-terminal    Build terminal-agent.exe
echo   test              Run go test
echo   tidy              Run go mod tidy
echo   lint              Run go vet
echo   clean             Remove build artifacts
echo   help              Show this help
echo.
exit /b 0

:_all
:_build
call :_build_install || exit /b %ERRORLEVEL%
call :_build_rewrite || exit /b %ERRORLEVEL%
call :_build_browse || exit /b %ERRORLEVEL%
call :_build_terminal || exit /b %ERRORLEVEL%
exit /b 0

:_build_install
if not exist "%BIN%" mkdir "%BIN%"
echo [debug] LDFLAGS=%LDFLAGS%
echo [debug] go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN%\install.exe" ./cmd/install
echo [build] install.exe
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN%\install.exe" ./cmd/install
if errorlevel 1 exit /b %ERRORLEVEL%
exit /b 0

:_build_rewrite
if not exist "%BIN%" mkdir "%BIN%"
echo [build] _gpowers-rewrite-browser.exe
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN%\_gpowers-rewrite-browser.exe" ./cmd/rewrite-browser
if errorlevel 1 exit /b %ERRORLEVEL%
exit /b 0

:_build_browse
if not exist "%BIN%" mkdir "%BIN%"
echo [build] browse.exe (root module)
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN%\browse.exe" ./cmd/browse
if errorlevel 1 exit /b %ERRORLEVEL%
echo [build] browse.exe (browse-go module)
pushd tools\skills\browse-go
go build -trimpath -ldflags "%LDFLAGS%" -o "..\..\..\%BIN%\browse.exe" .
if errorlevel 1 (
    popd
    exit /b %ERRORLEVEL%
)
popd
exit /b 0

:_build_terminal
if not exist "%BIN%" mkdir "%BIN%"
echo [build] terminal-agent.exe
go build -trimpath -ldflags "%LDFLAGS%" -o "%BIN%\terminal-agent.exe" ./cmd/terminal-agent
if errorlevel 1 exit /b %ERRORLEVEL%
exit /b 0

:_test
echo [test] root module
go test ./...
set "ROOT_CODE=%ERRORLEVEL%"
echo [test] browse-go module
pushd tools\skills\browse-go
go test ./...
set "BROWSE_CODE=%ERRORLEVEL%"
popd
if %ROOT_CODE% neq 0 exit /b %ROOT_CODE%
if %BROWSE_CODE% neq 0 exit /b %BROWSE_CODE%
exit /b 0

:_tidy
echo [tidy] root module
go mod tidy
if errorlevel 1 exit /b %ERRORLEVEL%
echo [tidy] browse-go module
pushd tools\skills\browse-go
go mod tidy
if errorlevel 1 (
    popd
    exit /b %ERRORLEVEL%
)
popd
exit /b 0

:_lint
echo [lint] root module
go vet ./...
if errorlevel 1 exit /b %ERRORLEVEL%
echo [lint] browse-go module
pushd tools\skills\browse-go
go vet ./...
if errorlevel 1 (
    popd
    exit /b %ERRORLEVEL%
)
popd
exit /b 0

:_clean
echo [clean] removing build artifacts
if exist "%BIN%" rmdir /s /q "%BIN%"
if exist "dist" rmdir /s /q "dist"
exit /b 0
