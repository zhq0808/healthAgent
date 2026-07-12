@echo off
REM 双击或在 cmd 里执行 dev 即可，绕过 PowerShell 执行策略限制。
REM 透传参数：dev -Down / dev -SkipDocker
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0dev.ps1" %*
