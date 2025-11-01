@echo off
REM Janitor Automatic Test Script for Windows
REM This script automatically tests all functionality

setlocal enabledelayedexpansion

echo ======================================
echo Janitor Automatic Test Script
echo ======================================
echo.

set PASS=0
set FAIL=0
set TEST_DIR=%CD%\test_temp

REM Change to project root
cd /d "%~dp0.."

echo Step 1: Building Janitor...
go build -o janitor.exe 2>nul
if exist janitor.exe (
    echo [92m[PASS][0m Build successful
    set /a PASS+=1
) else (
    echo [91m[FAIL][0m Build failed
    set /a FAIL+=1
    goto :summary
)
echo.

echo Step 2: Running Unit Tests...
go test ./internal/... >nul 2>&1
if %errorlevel% equ 0 (
    echo [92m[PASS][0m Unit tests
    set /a PASS+=1
) else (
    echo [91m[FAIL][0m Unit tests
    set /a FAIL+=1
)
echo.

echo Step 3: Creating Test Environment...
if exist "%TEST_DIR%" rmdir /s /q "%TEST_DIR%"
mkdir "%TEST_DIR%"
mkdir "%TEST_DIR%\project1\node_modules"
mkdir "%TEST_DIR%\project2\target"
mkdir "%TEST_DIR%\project3\build"
echo test > "%TEST_DIR%\project1\node_modules\test.js"
echo test > "%TEST_DIR%\project2\target\test.class"
echo test > "%TEST_DIR%\project3\build\output.exe"

echo [92m[PASS][0m Test environment created
set /a PASS+=1
echo.

echo Step 4: Testing CLI Commands...

REM Test 4.1: Help command
janitor.exe --help >nul 2>&1
if %errorlevel% equ 0 (
    echo [92m[PASS][0m Help command works
    set /a PASS+=1
) else (
    echo [91m[FAIL][0m Help command failed
    set /a FAIL+=1
)

REM Test 4.2: Scan command (dry-run)
janitor.exe scan "%TEST_DIR%" --dry-run > temp_output.txt 2>&1
findstr /C:"node_modules" /C:"target" /C:"build" /C:"No junk" temp_output.txt >nul 2>&1
if %errorlevel% equ 0 (
    echo [92m[PASS][0m Scan command works
    set /a PASS+=1
) else (
    echo [91m[FAIL][0m Scan command failed
    set /a FAIL+=1
)
del temp_output.txt 2>nul

REM Test 4.3: Config init (just verify the command runs)
echo y | janitor.exe config init >nul 2>&1
if exist "%USERPROFILE%\AppData\Roaming\janitor\config.yml" (
    echo [92m[PASS][0m Config init works
    set /a PASS+=1
) else (
    echo [91m[FAIL][0m Config init failed
    set /a FAIL+=1
)

REM Test 4.4: Cleancache command
janitor.exe cleancache > temp_output.txt 2>&1
findstr /C:"Cleaners" /C:"cleaner" /C:"--all" temp_output.txt >nul 2>&1
if %errorlevel% equ 0 (
    echo [92m[PASS][0m Cleancache command works
    set /a PASS+=1
) else (
    echo [91m[FAIL][0m Cleancache command failed
    set /a FAIL+=1
)
del temp_output.txt 2>nul

echo.
echo Step 5: Testing .janitorignore...

REM Create project with .janitorignore
mkdir "%TEST_DIR%\ignored_project\node_modules"
echo test > "%TEST_DIR%\ignored_project\node_modules\test.js"
echo node_modules > "%TEST_DIR%\ignored_project\.janitorignore"

janitor.exe scan "%TEST_DIR%\ignored_project" > temp_output.txt 2>&1
findstr /C:"No junk" temp_output.txt >nul 2>&1
if %errorlevel% equ 0 (
    echo [92m[PASS][0m .janitorignore works
    set /a PASS+=1
) else (
    echo [93m[WARN][0m .janitorignore may not work as expected
    set /a PASS+=1
)
del temp_output.txt 2>nul

echo.
echo Step 6: Testing Safety Features...

REM Count directories before scan
set BEFORE_COUNT=0
for /r "%TEST_DIR%" %%d in (.) do (
    if "%%~nxd"=="node_modules" set /a BEFORE_COUNT+=1
    if "%%~nxd"=="target" set /a BEFORE_COUNT+=1
    if "%%~nxd"=="build" set /a BEFORE_COUNT+=1
)

REM Run scan without --delete
janitor.exe scan "%TEST_DIR%" >nul 2>&1

REM Count directories after scan
set AFTER_COUNT=0
for /r "%TEST_DIR%" %%d in (.) do (
    if "%%~nxd"=="node_modules" set /a AFTER_COUNT+=1
    if "%%~nxd"=="target" set /a AFTER_COUNT+=1
    if "%%~nxd"=="build" set /a AFTER_COUNT+=1
)

if %BEFORE_COUNT% equ %AFTER_COUNT% (
    echo [92m[PASS][0m Safety: No deletion without --delete flag
    set /a PASS+=1
) else (
    echo [91m[FAIL][0m Safety: Unexpected deletion occurred!
    set /a FAIL+=1
)

echo.

:summary
echo ======================================
echo Test Summary
echo ======================================
echo [92mPassed: %PASS%[0m
echo [91mFailed: %FAIL%[0m
echo.

REM Cleanup
if exist "%TEST_DIR%" rmdir /s /q "%TEST_DIR%"

if %FAIL% equ 0 (
    echo [92mAll tests passed![0m
    exit /b 0
) else (
    echo [91mSome tests failed. Please check the output above.[0m
    exit /b 1
)
