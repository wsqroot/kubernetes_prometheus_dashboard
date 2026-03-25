$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$supervisorScript = Join-Path $root "backend-keepalive.ps1"
$pidFile = Join-Path $root "backend-supervisor.pid"

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
  Write-Host "Backend keepalive is already running. PID=$runningId"
  exit 0
}

Write-Host "Starting backend keepalive..."
Start-Process -FilePath "powershell.exe" `
  -ArgumentList "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", $supervisorScript `
  -WorkingDirectory $root | Out-Null

Start-Sleep -Seconds 3
$runningId = Get-RunningSupervisorId
if (-not $runningId) {
  throw "Failed to start backend keepalive"
}

Write-Host "Backend keepalive started. PID=$runningId"
