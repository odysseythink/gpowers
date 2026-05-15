@echo off
setlocal
set "VAR="
if defined VAR (
    echo VAR is defined after set to empty
) else (
    echo VAR is NOT defined after set to empty
)
set "VAR=""
if defined VAR (
    echo VAR is defined after set to quotes
) else (
    echo VAR is NOT defined after set to quotes
)
set "VAR=value"
if defined VAR (
    echo VAR is defined after set to value
) else (
    echo VAR is NOT defined after set to value
)
endlocal
