$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$staticDir = Join-Path $root "static"
$pythonPath = "C:\Python311\python.exe"
$stdoutLog = Join-Path $root "frontend.stdout.log"
$stderrLog = Join-Path $root "frontend.stderr.log"
$pidFile = Join-Path $root "frontend-server.pid"

if (-not (Test-Path $pythonPath)) {
  $pythonPath = (Get-Command python -ErrorAction Stop).Source
}

Set-Content -Path $pidFile -Value $PID

try {
  Push-Location $staticDir
  try {
    & $pythonPath -m http.server 8000 --directory $staticDir 1>> $stdoutLog 2>> $stderrLog
  } finally {
    Pop-Location
  }
} finally {
  Remove-Item -Path $pidFile -ErrorAction SilentlyContinue
}
