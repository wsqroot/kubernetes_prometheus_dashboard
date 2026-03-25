$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$supervisorPidFile = Join-Path $root "backend-supervisor.pid"
$serverPidFile = Join-Path $root "backend-server.pid"

function Stop-ManagedProcess {
  param(
    [string]$PidFile,
    [string]$Name
  )

  if (-not (Test-Path $PidFile)) {
    Write-Host "$Name is not running."
    return
  }

  $pidValue = Get-Content $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1
  if (-not $pidValue) {
    Remove-Item -Path $PidFile -ErrorAction SilentlyContinue
    Write-Host "$Name pid file was empty."
    return
  }

  try {
    Stop-Process -Id ([int]$pidValue) -Force -ErrorAction Stop
    Write-Host "Stopped $Name. PID=$pidValue"
  } catch {
    Write-Host "$Name was not running. PID=$pidValue"
  } finally {
    Remove-Item -Path $PidFile -ErrorAction SilentlyContinue
  }
}

Stop-ManagedProcess -PidFile $serverPidFile -Name "backend server"
Stop-ManagedProcess -PidFile $supervisorPidFile -Name "backend keepalive"
