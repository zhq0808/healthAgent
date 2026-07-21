#requires -Version 5.1
<#
.SYNOPSIS
    一键启动本地开发环境：PostgreSQL + Redis（docker）、Go 后端、前端 Vite。

.DESCRIPTION
    - 自动启动 docker-compose 里的 postgres/redis，并等待 PostgreSQL 健康后再拉起后端，
      避免后端连不上数据库直接退出。
    - 后端和前端各自开一个新 PowerShell 窗口运行，日志独立、可单独 Ctrl+C 停止。
    - 前端用 front 目录下的本地 vite（npm run dev），避免 npx 误装全局 vite。

.PARAMETER Down
    停止并移除 docker 容器（数据保留在 volume），不启动前后端。

.PARAMETER SkipDocker
    跳过 docker 步骤（容器已在运行时用，省去等待）。

.EXAMPLE
    .\dev.ps1            # 启动全部
    .\dev.ps1 -SkipDocker
    .\dev.ps1 -Down      # 停止容器
#>
param(
    [switch]$Down,
    [switch]$SkipDocker
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
Set-Location $root

function Write-Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "    $msg" -ForegroundColor Green }

# 停止容器
if ($Down) {
    Write-Step '停止 PostgreSQL / Redis 容器...'
    docker compose down | Out-Host
    Write-Ok '已停止。'
    return
}

# 校验 docker 可用
if (-not $SkipDocker) {
    try {
        docker info *> $null
    } catch {
        throw 'Docker 未运行，请先启动 Docker Desktop。'
    }

    # 1. 启动依赖容器（已在运行会原地保持，不重启）
    Write-Step '启动 PostgreSQL / Redis 容器...'
    docker compose up -d | Out-Host

    # 2. 等待 postgres 健康，最多 60 秒
    Write-Step '等待 PostgreSQL 就绪...'
    $deadline = (Get-Date).AddSeconds(60)
    while ($true) {
        $status = (docker inspect -f '{{.State.Health.Status}}' interview-postgres 2>$null)
        if ($status -eq 'healthy') { break }
        if ((Get-Date) -gt $deadline) {
            throw 'PostgreSQL 60 秒内未就绪，请检查：docker compose logs postgres'
        }
        Start-Sleep -Seconds 2
    }
    Write-Ok 'PostgreSQL 已就绪。'
}

# 3. 新窗口启动 Go 后端（:8091）
Write-Step '启动 Go 后端 (http://127.0.0.1:8091) ...'
Start-Process powershell -ArgumentList @(
    '-NoExit', '-NoProfile', '-Command',
    "Set-Location '$root'; Write-Host '[后端] go run .' -ForegroundColor Yellow; go run ."
)

# 4. 新窗口启动前端 Vite（:5173），用本地 vite
Write-Step '启动前端 Vite (http://127.0.0.1:5173) ...'
$front = Join-Path $root 'front'
Start-Process powershell -ArgumentList @(
    '-NoExit', '-NoProfile', '-Command',
    "Set-Location '$front'; Write-Host '[前端] npm run dev' -ForegroundColor Yellow; npm run dev"
)

Write-Host ''
Write-Ok '全部启动完成（前后端各在独立窗口运行）：'
Write-Host '  前端: http://127.0.0.1:5173/'
Write-Host '  后端: http://127.0.0.1:8091/'
Write-Host ''
Write-Host '停止：在对应窗口按 Ctrl+C；停止数据库容器用  .\dev.ps1 -Down' -ForegroundColor DarkGray
