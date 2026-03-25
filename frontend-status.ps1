$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$supervisorPidFile = Join-Path $root "frontend-supervisor.pid"
$serverPidFile = Join-Path $root "frontend-server.pid"

function Get-ManagedProcessStatus {
  param(
    [string]$PidFile,
    [string]$Name
  )

  if (-not (Test-Path $PidFile)) {
    return "${Name}: stopped"
  }

  $pidValue = Get-Content $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1
  if (-not $pidValue) {
    Remove-Item -Path $PidFile -ErrorAction SilentlyContinue
    return "${Name}: stopped"
  }

  try {
    $process = Get-Process -Id ([int]$pidValue) -ErrorAction Stop
    return "${Name}: running (PID=$($process.Id))"
  } catch {
    Remove-Item -Path $PidFile -ErrorAction SilentlyContinue
    return "${Name}: stopped"
  }
}

Write-Host (Get-ManagedProcessStatus -PidFile $supervisorPidFile -Name "frontend keepalive")
Write-Host (Get-ManagedProcessStatus -PidFile $serverPidFile -Name "frontend server")
try {
  $response = Invoke-WebRequest -UseBasicParsing "http://127.0.0.1:8000/" -TimeoutSec 3
  Write-Host "frontend: ok ($($response.StatusCode))"
} catch {
  Write-Host "frontend: unreachable"
}
