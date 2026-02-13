<#
devtunnel.ps1
Creates (or reuses) a Dev Tunnel and hosts a local HTTP port so Foundry Control Plane can reach your agent.

Requires: devtunnel CLI (Microsoft Dev Tunnels)
Docs: devtunnel create/port/access/host/show/token :contentReference[oaicite:0]{index=0}
#>

param(
  [Parameter(Mandatory=$true)]
  [int]$LocalPort,

  # A stable tunnel ID you want (optional). If omitted, we’ll create one and read the ID from "devtunnel show".
  [string]$TunnelId = "",

  # "anonymous" is easiest for cloud services (no token); "tenant" requires Entra auth; "token" issues a connect token.
  [ValidateSet("anonymous","tenant","token")]
  [string]$AccessMode = "anonymous",

  # Port protocol as understood by devtunnel ("http","https","auto")
  [ValidateSet("http","https","auto")]
  [string]$Protocol = "http",

  # Expiration for persistent tunnels (e.g., 30d, 2d, 4h)
  [string]$Expiration = "30d",

  # If AccessMode=token, scopes for token issuance (connect is typical)
  [string[]]$TokenScopes = @("connect")
)

function Require-Cmd($name) {
  if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
    throw "Required command not found: $name. Install and ensure it's on PATH."
  }
}

Require-Cmd devtunnel

# Ensure logged in
$loginStatus = & devtunnel user show 2>$null
if ($LASTEXITCODE -ne 0) {
  Write-Host "[devtunnel] Not logged in. Launching device-code login..."
  & devtunnel user login -d | Out-Host
  if ($LASTEXITCODE -ne 0) { throw "devtunnel login failed." }
}

# Create (or use) tunnel
if ([string]::IsNullOrWhiteSpace($TunnelId)) {
  Write-Host "[devtunnel] Creating persistent tunnel (expiration=$Expiration)..."
  & devtunnel create --expiration $Expiration | Out-Host
  if ($LASTEXITCODE -ne 0) { throw "devtunnel create failed." }

  $show = & devtunnel show | Out-String
  # Try several known output formats: "Tunnel ID : xxx", "TunnelId: xxx", "tunnel: xxx"
  if ($show -match 'Tunnel\s*ID\s*:\s*(\S+)') {
    $TunnelId = $Matches[1]
  } elseif ($show -match 'TunnelId:\s*(\S+)') {
    $TunnelId = $Matches[1]
  } elseif ($show -match 'tunnel:\s*(\S+)') {
    $TunnelId = $Matches[1]
  } else {
    throw "Could not parse Tunnel ID from devtunnel show output.`n$show"
  }
} else {
  Write-Host "[devtunnel] Using tunnel: $TunnelId"
}

Write-Host "[devtunnel] Ensuring port $LocalPort exists (protocol=$Protocol)..."
# Create port (ignore error if already exists)
& devtunnel port create $TunnelId -p $LocalPort --protocol $Protocol 2>$null | Out-Host

# Configure access
switch ($AccessMode) {
  "anonymous" {
    Write-Host "[devtunnel] Enabling anonymous access (WARNING: internet reachable if URL is known)."
    & devtunnel access create $TunnelId --anonymous | Out-Host
  }
  "tenant" {
    Write-Host "[devtunnel] Enabling tenant access."
    & devtunnel access create $TunnelId --tenant | Out-Host
  }
  "token" {
    Write-Host "[devtunnel] Leaving access restricted; will issue a token for clients."
  }
}

# Show the public URL (web-forwarding URI includes tunnel+port)
$show2 = & devtunnel show $TunnelId | Out-String
Write-Host ""
Write-Host "[devtunnel] Tunnel details:"
Write-Host $show2

# Optionally issue a token (useful for non-anonymous scenarios)
if ($AccessMode -eq "token") {
  $scopesJoined = $TokenScopes -join ","
  Write-Host "[devtunnel] Issuing access token (scopes=$scopesJoined)..."
  $token = & devtunnel token $TunnelId --scopes $TokenScopes 2>$null
  if ($LASTEXITCODE -ne 0) { throw "devtunnel token failed." }
  Write-Host ""
  Write-Host "[devtunnel] CONNECT TOKEN (store securely):"
  Write-Host $token
}

Write-Host ""
Write-Host "[devtunnel] Hosting tunnel now. Keep this window open."
Write-Host "[devtunnel] Your agent should be listening locally on http://127.0.0.1:$LocalPort (or equivalent)."
& devtunnel host $TunnelId | Out-Host
