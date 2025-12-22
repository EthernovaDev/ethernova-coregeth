@echo off
setlocal
powershell -ExecutionPolicy Bypass -File "%~dp0apply-upgrade-mainnet.ps1"
exit /b %ERRORLEVEL%
