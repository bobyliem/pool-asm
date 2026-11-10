@echo off
title ASM Workshop Server (Port 8080)
cd /d "%~dp0"

echo ===================================================
echo   Menyalakan ASM Workshop Server (Port 8080)
echo ===================================================
echo.

if not exist "asm.exe" (
    echo File asm.exe belum ada, melakukan compile...
    go build -o asm.exe .
)

echo Menjalankan asm.exe...
echo Server dapat diakses di: http://localhost:8080/prj-csv
echo Tekan Ctrl+C untuk menghentikan server.
echo ---------------------------------------------------
echo.

asm.exe

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [ERROR] Server berhenti dengan error code %ERRORLEVEL%.
    pause
)
