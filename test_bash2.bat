@echo off
setlocal EnableDelayedExpansion
for /f "delims=" %%a in ('where bash') do (
    echo Found: %%a
    "%%a" -c "echo OK"
    if !errorlevel! equ 0 (
        for /f "delims=" %%u in ('"%%a" -c "uname -o" 2^>nul') do (
            echo uname output: %%u
        )
    )
)
