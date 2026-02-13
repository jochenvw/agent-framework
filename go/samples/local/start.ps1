<#
start.ps1
Launches the Dev Tunnel and agent HTTP server together.

The tunnel setup runs first (create, port, access), then hosts in the background.
The agent runs in the foreground so you see all [http] and [agent] logs.
Press Ctrl+C to stop.

Tunnel ID is saved to .devtunnel file for reuse. Delete the file to create a new tunnel.

Usage:
  .\start.ps1                          # defaults: phi-4-mini, port 8080, reuses saved tunnel
  .\start.ps1 -Model phi-4-mini -Port 9090
  .\start.ps1 -TunnelId my-tunnel      # use specific tunnel (saves it)
#>

param(
  [string]$Model = "phi-4-mini",
  [int]$Port = 8080,
  [string]$TunnelId = "",
  [ValidateSet("anonymous","tenant","token")]
  [string]$AccessMode = "anonymous",
  [string]$Expiration = "30d"
)

$ErrorActionPreference = "Stop"

# ── Ensure devtunnel CLI is available ────────────────────────────────
if (-not (Get-Command devtunnel -ErrorAction SilentlyContinue)) {
  throw "devtunnel CLI not found. Install it and ensure it's on PATH."
}

# Ensure logged in
$loginStatus = & devtunnel user show 2>$null
if ($LASTEXITCODE -ne 0) {
  Write-Host "[start] devtunnel not logged in. Launching device-code login..."
  & devtunnel user login -d | Out-Host
  if ($LASTEXITCODE -ne 0) { throw "devtunnel login failed." }
}

# ── Create (or reuse) tunnel ────────────────────────────────────────
$tunnelFile = Join-Path $PSScriptRoot ".devtunnel"

# Priority: -TunnelId param > .devtunnel file > create new
if ([string]::IsNullOrWhiteSpace($TunnelId) -and (Test-Path $tunnelFile)) {
  $TunnelId = (Get-Content $tunnelFile -Raw).Trim()
  if (-not [string]::IsNullOrWhiteSpace($TunnelId)) {
    Write-Host "[start] reusing saved tunnel: $TunnelId (from .devtunnel)"
  }
}

if ([string]::IsNullOrWhiteSpace($TunnelId)) {
  Write-Host "[start] creating dev tunnel (expiration=$Expiration)..."
  & devtunnel create --expiration $Expiration | Out-Host
  if ($LASTEXITCODE -ne 0) { throw "devtunnel create failed." }

  $show = & devtunnel show | Out-String
  if ($show -match 'Tunnel\s*ID\s*:\s*(\S+)') {
    $TunnelId = $Matches[1]
  } elseif ($show -match 'TunnelId:\s*(\S+)') {
    $TunnelId = $Matches[1]
  } elseif ($show -match 'tunnel:\s*(\S+)') {
    $TunnelId = $Matches[1]
  } else {
    throw "Could not parse Tunnel ID from devtunnel show output.`n$show"
  }
}

# Save tunnel ID for next run
$TunnelId | Out-File -FilePath $tunnelFile -Encoding utf8 -NoNewline
Write-Host "[start] tunnel ID: $TunnelId (saved to .devtunnel)"

# Add port and access (ignore errors if already configured from a previous run)
Write-Host "[start] configuring port $Port..."
try { & devtunnel port create $TunnelId -p $Port --protocol http 2>&1 | Out-Null } catch {}

switch ($AccessMode) {
  "anonymous" {
    Write-Host "[start] enabling anonymous access"
    try { & devtunnel access create $TunnelId --anonymous 2>&1 | Out-Null } catch {}
  }
  "tenant" {
    Write-Host "[start] enabling tenant access"
    try { & devtunnel access create $TunnelId --tenant 2>&1 | Out-Null } catch {}
  }
}

# Derive the public tunnel URL from devtunnel show output.
# devtunnel show lists per-port connect URLs like:
#   8080   http   https://wzjnn4nf-8080.eun1.devtunnels.ms
# These already route to the correct port — no :port suffix needed.
$showDetails = & devtunnel show $TunnelId | Out-String

# Try to find the port-specific connect URL first (most reliable)
if ($showDetails -match "(?m)$Port\s+\S+\s+(https://\S+)") {
  $tunnelURL = $Matches[1].TrimEnd('/')
} elseif ($showDetails -match '(https://[a-zA-Z0-9._-]+\.devtunnels\.[a-z]+)') {
  # Fallback: use the base tunnel URL with port suffix
  $baseFqdn = $Matches[1].TrimEnd('/')
  $tunnelURL = "${baseFqdn}:${Port}"
} else {
  # Last resort: construct from tunnel ID
  $tunnelURL = "https://${TunnelId}.devtunnels.ms:${Port}"
}

Write-Host ""
Write-Host "[start] tunnel URL: $tunnelURL"
Write-Host ""

# ── Host tunnel in background ───────────────────────────────────────
Write-Host "[start] starting tunnel host in background..."
$tunnelJob = Start-Job -ScriptBlock {
  param($id)
  & devtunnel host $id 2>&1
} -ArgumentList $TunnelId

Start-Sleep -Seconds 2
if ($tunnelJob.State -ne "Running") {
  Write-Host "[start] ERROR: tunnel host failed. Output:"
  Receive-Job $tunnelJob | Write-Host
  Remove-Job $tunnelJob -Force
  exit 1
}

# ── Run agent in foreground (all logs visible) ──────────────────────
Write-Host "[start] launching agent on port $Port (model=$Model, GPU)..."
Write-Host ""

$env:DEVTUNNEL_URL = $tunnelURL

try {
  & go run . --model $Model --gpu --serve --port $Port
} finally {
  Write-Host ""
  Write-Host "[start] stopping tunnel..."
  Stop-Job $tunnelJob -ErrorAction SilentlyContinue
  Remove-Job $tunnelJob -Force -ErrorAction SilentlyContinue
  Write-Host "[start] done. (tunnel $TunnelId preserved for reuse, delete .devtunnel to reset)"
}
