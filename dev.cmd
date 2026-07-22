@echo off
REM Double-click this file or run it from cmd. Arguments are passed to dev.ps1.
REM Supported arguments: -Restart, -Down, -SkipDocker
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0dev.ps1" %*
