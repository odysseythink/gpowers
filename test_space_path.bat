@echo off
setlocal EnableDelayedExpansion
set "TEST_BASH=C:\Program Files\Git\usr\bin\bash.exe"
if defined TEST_BASH (
    echo TEST_BASH is defined: [%TEST_BASH%]
    for /f "delims=" %%u in ('"%TEST_BASH%" -c "uname -o" 2^>nul') do (
        echo Output: %%u
    )
) else (
    echo TEST_BASH is not defined
)
