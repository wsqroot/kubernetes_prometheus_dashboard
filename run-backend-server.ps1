$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$loginDir = Join-Path $root "login"
$binaryPath = Join-Path $loginDir "server.exe"
$configPath = Join-Path $loginDir "configs\config.yaml"
$stdoutLog = Join-Path $loginDir "server.stdout.log"
$stderrLog = Join-Path $loginDir "server.stderr.log"
$pidFile = Join-Path $root "backend-server.pid"

Set-Content -Path $pidFile -Value $PID
$env:LOGIN_CONFIG = $configPath

try {
  Push-Location $loginDir
  try {
    & $binaryPath 1>> $stdoutLog 2>> $stderrLog
  } finally {
    Pop-Location
  }
} finally {
  Remove-Item -Path $pidFile -ErrorAction SilentlyContinue
}
