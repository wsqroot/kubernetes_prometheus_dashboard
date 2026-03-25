$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$supervisorScript = Join-Path $root "frontend-keepalive.ps1"
$pidFile = Join-Path $root "frontend-supervisor.pid"

function Get-RunningSupervisorId {
  if (-not (Test-Path $pidFile)) {
    return $null
  }

  $supervisorPid = Get-Content $pidFile -ErrorAction SilentlyContinue | Select-Object -First 1
  if (-not $supervisorPid) {
    return $null
  }

  try {
    $process = Get-Process -Id ([int]$supervisorPid) -ErrorAction Stop
    return $process.Id
  } catch {
    Remove-Item -Path $pidFile -ErrorAction SilentlyContinue
    return $null
  }
}

$runningId = Get-RunningSupervisorId
if ($runningId) {
  Write-Host "Frontend keepalive is already running. PID=$runningId"
  exit 0
}

Write-Host "Starting frontend keepalive..."
Start-Process -FilePath "powershell.exe" `
  -ArgumentList "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", $supervisorScript `
  -WorkingDirectory $root | Out-Null

Start-Sleep -Seconds 2
$runningId = Get-RunningSupervisorId
if (-not $runningId) {
  throw "Failed to start frontend keepalive"
}

Write-Host "Frontend keepalive started. PID=$runningId"
