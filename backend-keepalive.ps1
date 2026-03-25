$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$loginDir = Join-Path $root "login"
$binaryPath = Join-Path $loginDir "server.exe"
$configPath = Join-Path $loginDir "configs\config.yaml"
$stdoutLog = Join-Path $loginDir "server.stdout.log"
$stderrLog = Join-Path $loginDir "server.stderr.log"
$supervisorLog = Join-Path $root "backend-supervisor.log"
$supervisorPidFile = Join-Path $root "backend-supervisor.pid"
$serverPidFile = Join-Path $root "backend-server.pid"
$goCache = Join-Path $root ".gocache"

Set-Content -Path $supervisorPidFile -Value $PID

function Write-SupervisorLog {
  param([string]$Message)
  $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
  Add-Content -Path $supervisorLog -Value "[$timestamp] $Message"
}

function Get-BackendPid {
  if (Test-Path $serverPidFile) {
    $backendPid = Get-Content $serverPidFile -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($backendPid) {
      try {
        $process = Get-Process -Id ([int]$backendPid) -ErrorAction Stop
        return $process.Id
      } catch {
        Remove-Item -Path $serverPidFile -ErrorAction SilentlyContinue
      }
    }
  }

  $lines = netstat -ano -p tcp | Select-String ":8080"
  foreach ($line in $lines) {
    $text = ($line.ToString() -replace "\s+", " ").Trim()
    if ($text -match "LISTENING (\d+)$") {
      return [int]$Matches[1]
    }
  }
  return $null
}

function Stop-BackendServer {
  $backendPid = Get-BackendPid
  if ($backendPid) {
    try {
      Stop-Process -Id $backendPid -Force -ErrorAction Stop
      Write-SupervisorLog "Stopped backend process PID=$backendPid"
    } catch {
      Write-SupervisorLog "Failed to stop backend PID=${backendPid}: $($_.Exception.Message)"
    }
  }
}

function Build-Backend {
  Write-SupervisorLog "Building backend binary"
  Push-Location $loginDir
  try {
    $env:GOCACHE = $goCache
    go build -o server.exe ./cmd/server | Out-Null
  } finally {
    Pop-Location
  }
}

function Start-BackendServer {
  Write-SupervisorLog "Starting backend server on port 8080"
  Remove-Item -Path $serverPidFile -ErrorAction SilentlyContinue
  $env:LOGIN_CONFIG = $configPath
  $process = Start-Process -FilePath $binaryPath `
    -WorkingDirectory $loginDir `
    -RedirectStandardOutput $stdoutLog `
    -RedirectStandardError $stderrLog `
    -PassThru
  Set-Content -Path $serverPidFile -Value $process.Id
}

function Test-BackendHealthy {
  try {
    $response = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:8080/healthz" -TimeoutSec 3
    return $response.StatusCode -eq 200
  } catch {
    return $false
  }
}

Write-SupervisorLog "Backend keepalive started. Supervisor PID=$PID"

try {
  while ($true) {
    if (-not (Test-BackendHealthy)) {
      Stop-BackendServer
      Build-Backend
      Start-Sleep -Seconds 1
      Start-BackendServer
      Start-Sleep -Seconds 3
    }
    Start-Sleep -Seconds 5
  }
} finally {
  Stop-BackendServer
  Remove-Item -Path $supervisorPidFile -ErrorAction SilentlyContinue
  Write-SupervisorLog "Backend keepalive stopped"
}
