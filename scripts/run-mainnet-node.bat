@echo off
setlocal
powershell -ExecutionPolicy Bypass -File "%~dp0run-mainnet-node.ps1" %*
exit /b %ERRORLEVEL%
