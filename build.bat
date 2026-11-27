@echo off
setlocal enabledelayedexpansion
chcp 65001 >nul
cd /d "%~dp0"

set "MODE=%~1"
set "AUTO_RUN=%~2"
set "INTERACTIVE=0"

if "%MODE%"=="" goto :menu
goto :dispatch

:menu
set "INTERACTIVE=1"
echo ======================================================
echo   T-DECK QUICK BUILD (Windows x64)
echo ======================================================
echo   [1] Build nhanh Go backend (bo qua Frontend)
echo   [2] Build day du (Frontend + Go -^> TDeck.exe)
echo   [3] Clean build day du (npm install + FE + -clean)
echo   [4] Chi build Frontend (tsc ^&^& vite build)
echo   [5] Dong goi Windows Installer (TDeck-amd64-installer.exe)
echo   [6] Dong goi tat ca ban Release (Windows + Linux)
echo   [0] Thoat
echo ======================================================
set /p "CHOICE=Chon che do [1-6, mac dinh 2]: "
if "%CHOICE%"=="" set "CHOICE=2"
if "%CHOICE%"=="0" exit /b 0
if "%CHOICE%"=="1" set "MODE=fast"
if "%CHOICE%"=="2" set "MODE=full"
if "%CHOICE%"=="3" set "MODE=clean"
if "%CHOICE%"=="4" set "MODE=fe"
if "%CHOICE%"=="5" set "MODE=installer"
if "%CHOICE%"=="6" set "MODE=release"

:dispatch
if /i "%MODE%"=="fast" goto :build_fast
if /i "%MODE%"=="full" goto :build_full
if /i "%MODE%"=="clean" goto :build_clean
if /i "%MODE%"=="fe" goto :build_fe_only
if /i "%MODE%"=="installer" goto :build_installer
if /i "%MODE%"=="release" goto :build_release

echo [ERROR] Che do khong hop le: "%MODE%"
echo Su dung: build.bat [fast^|full^|clean^|fe^|installer^|release] [run]
if "%INTERACTIVE%"=="1" pause
exit /b 1

:build_fe_only
call :do_build_fe 0
if errorlevel 1 goto :fail
echo.
echo [OK] Build Frontend thanh cong (frontend\dist).
goto :done

:build_fast
if not exist "frontend\dist\index.html" (
    echo [INFO] Chua co frontend\dist, tu dong build Frontend truoc...
    call :do_build_fe 0
    if errorlevel 1 goto :fail
)
call :stop_tdeck
echo.
echo [BUILD] Dang build nhanh Go backend (skip Frontend)...
wails build -s -platform windows/amd64
if errorlevel 1 goto :fail
goto :after_exe

:build_full
call :do_build_fe 0
if errorlevel 1 goto :fail
call :stop_tdeck
echo.
echo [BUILD] Dang dong goi TDeck.exe...
wails build -s -platform windows/amd64
if errorlevel 1 goto :fail
goto :after_exe

:build_clean
call :do_build_fe 1
if errorlevel 1 goto :fail
call :stop_tdeck
echo.
echo [BUILD] Dang clean build TDeck.exe...
wails build -clean -s -platform windows/amd64
if errorlevel 1 goto :fail
goto :after_exe

:build_installer
call :do_build_fe 0
if errorlevel 1 goto :fail
call :stop_tdeck
call :ensure_nsis
if errorlevel 1 goto :fail
echo.
echo [BUILD] Dang dong goi Windows Installer (NSIS)...
wails build -s -nsis -platform windows/amd64
if errorlevel 1 goto :fail
echo.
echo [OK] Dong goi Installer thanh cong: "%~dp0build\bin\TDeck-amd64-installer.exe"
goto :done

:build_release
call :stop_tdeck
call :ensure_nsis
if errorlevel 1 goto :fail
echo.
echo [BUILD] Dang dong goi tat ca ban Release (Windows + Linux)...
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\package-releases.ps1"
if errorlevel 1 goto :fail
echo.
echo [OK] Dong goi tat ca ban Release thanh cong: "%~dp0build\release"
goto :done

:ensure_nsis
if exist "%USERPROFILE%\.local\nsis\makensis.exe" set "PATH=%USERPROFILE%\.local\nsis;%PATH%"
if exist "C:\Program Files (x86)\NSIS\makensis.exe" set "PATH=C:\Program Files (x86)\NSIS;%PATH%"
if exist "C:\Program Files\NSIS\makensis.exe" set "PATH=C:\Program Files\NSIS;%PATH%"
where makensis.exe >nul 2>&1
if not errorlevel 1 exit /b 0
echo [INFO] Chua co NSIS, dang tai ban portable ve %USERPROFILE%\.local\nsis...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$d = Join-Path $env:USERPROFILE '.local\nsis'; $z = Join-Path $env:TEMP 'nsis-3.10.zip'; $t = Join-Path $env:TEMP 'nsis-3.10-ext'; & curl.exe -fL 'https://downloads.sourceforge.net/project/nsis/NSIS%203/3.10/nsis-3.10.zip' -o $z; if (-not (Test-Path $z) -or ((Get-Item $z).Length -lt 1000000)) { throw 'Download NSIS failed' }; Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $t; Expand-Archive -Path $z -DestinationPath $t -Force; $in = Get-ChildItem $t -Directory | Select-Object -First 1; if (-not $in -or -not (Test-Path (Join-Path $in.FullName 'makensis.exe'))) { throw 'Invalid NSIS archive' }; New-Item -ItemType Directory -Force -Path $d | Out-Null; Copy-Item -Path (Join-Path $in.FullName '*') -Destination $d -Recurse -Force; Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $t, $z"
if errorlevel 1 (
    echo [ERROR] Khong the tai NSIS. Vui long cai NSIS thu cong roi thu lai.
    exit /b 1
)
set "PATH=%USERPROFILE%\.local\nsis;%PATH%"
exit /b 0

:do_build_fe
set "DO_INSTALL=%~1"
pushd "frontend" || exit /b 1
if "%DO_INSTALL%"=="1" goto :fe_install
if not exist "node_modules" goto :fe_install
goto :fe_compile

:fe_install
echo.
echo [FE] Dang chay npm install...
call npm install
if errorlevel 1 (
    popd
    exit /b 1
)

:fe_compile
echo.
echo [FE] Dang build Frontend (tsc ^&^& vite build)...
call npm run build
set "FE_ERR=%errorlevel%"
popd
exit /b %FE_ERR%

:stop_tdeck
tasklist /FI "IMAGENAME eq TDeck.exe" 2>nul | find /I "TDeck.exe" >nul
if not errorlevel 1 (
    echo [INFO] Dang tat TDeck.exe dang chay de ghi de file...
    taskkill /IM TDeck.exe /F >nul 2>&1
    ping 127.0.0.1 -n 2 >nul
)
exit /b 0

:after_exe
echo.
echo [OK] Build thanh cong: "%~dp0build\bin\TDeck.exe"
if /i "%AUTO_RUN%"=="run" (
    start "" "%~dp0build\bin\TDeck.exe"
    goto :done
)
if "%INTERACTIVE%"=="1" goto :ask_run
goto :done

:ask_run
set /p "RUN_NOW=Khoi chay TDeck.exe ngay bay gio? [Y/n]: "
if "%RUN_NOW%"=="" set "RUN_NOW=Y"
if /i "%RUN_NOW%"=="Y" start "" "%~dp0build\bin\TDeck.exe"
goto :done

:fail
echo.
echo [FAIL] Build that bai (exit code %errorlevel%).
if "%INTERACTIVE%"=="1" pause
exit /b 1

:done
if "%INTERACTIVE%"=="1" pause
exit /b 0
