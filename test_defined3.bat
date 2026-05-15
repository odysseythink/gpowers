@echo on
setlocal
set "GIT_BASH="
set GIT_BASH
echo ---
if defined GIT_BASH (
    echo INSIDE
) else (
    echo OUTSIDE
)
