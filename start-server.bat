@echo off
echo ========================================
echo   Tipitaka.app CST v1.1 - Starting...
echo ========================================
cd /d "%~dp0server"
start "" /B tipitaka_app.exe
echo Server started at http://localhost:8402
echo Opening browser...
timeout /t 2 /nobreak >nul
start http://localhost:8402
