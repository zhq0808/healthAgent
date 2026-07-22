#requires -Version 5.1
<#
.SYNOPSIS
    Starts the complete local development stack.

.DESCRIPTION
    Starts PostgreSQL and Redis with Docker Compose, then starts the Go API and
    Vite frontend in separate PowerShell windows. Re-running the script replaces
    the frontend and backend processes created by the previous run.

.PARAMETER Restart
    Restarts PostgreSQL, Redis, the Go API, and the Vite frontend.

.PARAMETER Down
    Stops the frontend, backend, and Docker Compose services.

.PARAMETER SkipDocker
    Leaves Docker Compose services unchanged.

.EXAMPLE
    .\dev.ps1
    .\dev.ps1 -Restart
    .\dev.ps1 -SkipDocker
    .\dev.ps1 -Down
#>
param(
    [switch]$Restart,
    [switch]$Down,
    [switch]$SkipDocker
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$stateDir = Join-Path $root '.dev'
$stateFile = Join-Path $stateDir 'processes.json'
Set-Location $root

function Write-Step($Message) { Write-Host "==> $Message" -ForegroundColor Cyan }
function Write-Ok($Message) { Write-Host "    $Message" -ForegroundColor Green }

function Test-DockerEngine {
    cmd.exe /d /c "docker info >nul 2>nul"
    return $LASTEXITCODE -eq 0
}

function Wait-DockerEngine {
    if (Test-DockerEngine) {
        Write-Ok 'Docker Engine is ready.'
        return
    }

    $dockerDesktop = 'C:\Program Files\Docker\Docker\Docker Desktop.exe'
    if (-not (Test-Path $dockerDesktop)) {
        $dockerDesktop = Join-Path $env:LOCALAPPDATA 'Docker\Docker Desktop.exe'
    }
    if (-not (Test-Path $dockerDesktop)) {
        throw 'Docker Desktop is not installed in a standard location.'
    }

    if (-not (Get-Process -Name 'Docker Desktop' -ErrorAction SilentlyContinue)) {
        Write-Step 'Starting Docker Desktop...'
        Start-Process $dockerDesktop | Out-Null
    } else {
        Write-Step 'Waiting for the Docker Engine to finish starting...'
    }

    $deadline = (Get-Date).AddSeconds(120)
    while (-not (Test-DockerEngine)) {
        if ((Get-Date) -gt $deadline) {
            throw 'Docker Engine was not ready within 120 seconds. Check Docker Desktop.'
        }
        Start-Sleep -Milliseconds 500
    }
    Write-Ok 'Docker Engine is ready.'
}

function Stop-ProcessTree([int]$RootProcessId) {
    $children = Get-CimInstance Win32_Process -Filter "ParentProcessId = $RootProcessId" -ErrorAction SilentlyContinue
    foreach ($child in $children) {
        Stop-ProcessTree -RootProcessId $child.ProcessId
    }

    Stop-Process -Id $RootProcessId -Force -ErrorAction SilentlyContinue
}

function Stop-DevProcesses {
    $processIds = @()
    if (Test-Path $stateFile) {
        try {
            $state = Get-Content $stateFile -Raw | ConvertFrom-Json
            $processIds += @($state.backend, $state.frontend)
        } catch {
            Write-Host '    Invalid process state file; falling back to port cleanup.' -ForegroundColor DarkYellow
        }
    }

    foreach ($port in @(8091, 5173)) {
        $listeners = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue
        $processIds += @($listeners | Select-Object -ExpandProperty OwningProcess)
    }

    $processIds |
        Where-Object { $_ -and $_ -ne $PID } |
        Sort-Object -Unique |
        ForEach-Object { Stop-ProcessTree -RootProcessId ([int]$_) }

    Remove-Item $stateFile -Force -ErrorAction SilentlyContinue
}

function Wait-ContainerHealthy([string]$ContainerName, [string]$DisplayName) {
    Write-Step "Waiting for $DisplayName..."
    $deadline = (Get-Date).AddSeconds(60)
    while ($true) {
        $status = docker inspect -f '{{.State.Health.Status}}' $ContainerName 2>$null
        if ($status -eq 'healthy') {
            Write-Ok "$DisplayName is ready."
            return
        }
        if ((Get-Date) -gt $deadline) {
            throw "$DisplayName was not healthy within 60 seconds. Check docker compose logs."
        }
        Start-Sleep -Milliseconds 500
    }
}

function Wait-HttpReady([string]$Url, [string]$DisplayName) {
    Write-Step "Waiting for $DisplayName..."
    $deadline = (Get-Date).AddSeconds(45)
    while ($true) {
        try {
            $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 2
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) {
                Write-Ok "$DisplayName is ready."
                return
            }
        } catch {
            if ((Get-Date) -gt $deadline) {
                throw "$DisplayName was not ready within 45 seconds. Check its PowerShell window."
            }
        }
        Start-Sleep -Milliseconds 500
    }
}

if ($Down) {
    Write-Step 'Stopping Go API and Vite frontend...'
    Stop-DevProcesses
    Write-Ok 'Application processes stopped.'

    if (-not $SkipDocker) {
        Write-Step 'Stopping PostgreSQL and Redis...'
        docker compose down | Out-Host
        Write-Ok 'Docker services stopped.'
    }
    return
}

Write-Step 'Cleaning up previous Go API and Vite frontend processes...'
Stop-DevProcesses
Write-Ok 'Application ports are available.'

if (-not $SkipDocker) {
    Wait-DockerEngine

    if ($Restart) {
        Write-Step 'Restarting PostgreSQL and Redis...'
        docker compose down | Out-Host
    }

    Write-Step 'Starting PostgreSQL and Redis...'
    docker compose up -d | Out-Host
    Wait-ContainerHealthy 'interview-postgres' 'PostgreSQL'
    Wait-ContainerHealthy 'interview-redis' 'Redis'
}

Write-Step 'Starting Go API at http://127.0.0.1:8091 ...'
$backendProcess = Start-Process powershell -PassThru -ArgumentList @(
    '-NoExit', '-NoProfile', '-Command',
    "`$Host.UI.RawUI.WindowTitle = 'Zhijing - Go API'; Set-Location '$root'; Write-Host '[API] go run .' -ForegroundColor Yellow; go run ."
)

Write-Step 'Starting Vite frontend at http://127.0.0.1:5173 ...'
$front = Join-Path $root 'front'
$frontendProcess = Start-Process powershell -PassThru -ArgumentList @(
    '-NoExit', '-NoProfile', '-Command',
    "`$Host.UI.RawUI.WindowTitle = 'Zhijing - Vite'; Set-Location '$front'; Write-Host '[Web] npm run dev' -ForegroundColor Yellow; npm run dev"
)

New-Item -ItemType Directory -Path $stateDir -Force | Out-Null
@{
    backend = $backendProcess.Id
    frontend = $frontendProcess.Id
} | ConvertTo-Json | Set-Content -Path $stateFile -Encoding UTF8

Wait-HttpReady 'http://127.0.0.1:8091/health' 'Go API'
Wait-HttpReady 'http://127.0.0.1:5173/' 'Vite frontend'

Write-Host ''
Write-Ok 'All services are running:'
Write-Host '  Frontend: http://127.0.0.1:5173/'
Write-Host '  Backend:  http://127.0.0.1:8091/'
Write-Host ''
Write-Host 'Start: .\dev.cmd    Full restart: .\dev.cmd -Restart    Stop: .\dev.cmd -Down' -ForegroundColor DarkGray