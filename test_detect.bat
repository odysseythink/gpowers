@echo off
setlocal EnableDelayedExpansion
set "GIT_BASH="
for /f "delims=" %%a in ('where bash 2^>nul') do (
    echo Candidate: %%a
    "%%a" -c "echo OK"
    echo Errorlevel after echo OK: !errorlevel!
    if !errorlevel! equ 0 (
        for /f "delims=" %%u in ('"%%a" -c "uname -o" 2^>nul') do (
            echo uname output: %%u
            if "%%u"=="Msys" set "GIT_BASH=%%a"
        )
        echo Errorlevel after for: !errorlevel!
        echo GIT_BASH after loop: !GIT_BASH!
    )
)
echo Final GIT_BASH: %GIT_BASH%
