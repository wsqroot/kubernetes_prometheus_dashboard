$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$staticDir = Join-Path $root "static"
$pythonPath = "C:\Python311\python.exe"
$stdoutLog = Join-Path $root "frontend.stdout.log"
$stderrLog = Join-Path $root "frontend.stderr.log"
$supervisorLog = Join-Path $root "frontend-supervisor.log"
$supervisorPidFile = Join-Path $root "frontend-supervisor.pid"
$serverPidFile = Join-Path $root "frontend-server.pid"

if (-not (Test-Path $pythonPath)) {
  $pythonPath = (Get-Command python -ErrorAction Stop).Source
}

Set-Content -Path $supervisorPidFile -Value $PID

function Write-SupervisorLog {
  param([string]$Message)
  $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
  Add-Content -Path $supervisorLog -Value "[$timestamp] $Message"
}

function Get-FrontendPid {
  if (Test-Path $serverPidFile) {
    $frontendPid = Get-Content $serverPidFile -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($frontendPid) {
      try {
        $process = Get-Process -Id ([int]$frontendPid) -ErrorAction Stop
        return $process.Id
      } catch {
        Remove-Item -Path $serverPidFile -ErrorAction SilentlyContinue
      }
    }
  }

  $lines = netstat -ano -p tcp | Select-String ":8000"
  foreach ($line in $lines) {
    $text = ($line.ToString() -replace "\s+", " ").Trim()
    if ($text -match "LISTENING (\d+)$") {
      return [int]$Matches[1]
    }
  }
  return $null
}

function Test-FrontendHealthy {
  try {
    $response = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:8000/" -TimeoutSec 3
    return $response.StatusCode -eq 200 -and $response.Content -like "*<div id=`"app`"></div>*"
  } catch {
    return $false
  }
}

function Start-FrontendServer {
  Write-SupervisorLog "Starting frontend server on port 8000"
  Remove-Item -Path $serverPidFile -ErrorAction SilentlyContinue
  $process = Start-Process -FilePath $pythonPath `
    -ArgumentList "-m", "http.server", "8000", "--directory", $staticDir `
    -WorkingDirectory $staticDir `
    -RedirectStandardOutput $stdoutLog `
    -RedirectStandardError $stderrLog `
    -PassThru
  Set-Content -Path $serverPidFile -Value $process.Id
}

function Stop-FrontendServer {
  $frontendPid = Get-FrontendPid
  if ($frontendPid) {
    try {
      Stop-Process -Id $frontendPid -Force -ErrorAction Stop
      Write-SupervisorLog "Stopped frontend process PID=$frontendPid"
    } catch {
      Write-SupervisorLog "Failed to stop frontend PID=${frontendPid}: $($_.Exception.Message)"
    }
  }
}

Write-SupervisorLog "Frontend keepalive started. Supervisor PID=$PID"

try {
  while ($true) {
    if (-not (Test-FrontendHealthy)) {
      Stop-FrontendServer
      Start-Sleep -Seconds 1
      Start-FrontendServer
      Start-Sleep -Seconds 2
    }
    Start-Sleep -Seconds 5
  }
} finally {
  Stop-FrontendServer
  Remove-Item -Path $supervisorPidFile -ErrorAction SilentlyContinue
  Write-SupervisorLog "Frontend keepalive stopped"
}
