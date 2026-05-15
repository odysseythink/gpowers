@echo on
setlocal
set "GIT_BASH="
echo before check: [%GIT_BASH%]
if defined GIT_BASH (
    echo INSIDE if defined
) else (
    echo OUTSIDE if defined
)
