<#
register-foundry-control-plane.ps1
Best-effort “registration helper” script.

Important:
As of the current Microsoft documentation, registering a *custom agent* in Foundry Control Plane is performed via the Foundry (new) portal wizard
(Operate -> Register agent). The doc does not provide a supported public REST/CLI for this registration. :contentReference[oaicite:1]{index=1}

So this script:
- Validates your public agent base URL responds (basic reachability)
- Prints the exact fields you’ll paste into the Foundry registration wizard
- Opens the Foundry portal to start registration

It does NOT claim to automate the registration via an undocumented API.
#>

param(
  [Parameter(Mandatory=$true)]
  [string]$AgentBaseUrl,

  # Optional: If your agent requires a header (e.g., devtunnel token or your own auth), provide it as "HeaderName: value"
  [string[]]$Headers = @(),

  # If your agent uses an agent-card JSON (A2A), set this; otherwise leave default and Foundry will use /.well-known/agent-card.json
  [string]$AgentCardPath = "/.well-known/agent-card.json",

  [ValidateSet("HTTP","A2A")]
  [string]$Protocol = "HTTP",

  # Optional OpenTelemetry Agent ID if you emit gen_ai.agents.id spans
  [string]$OpenTelemetryAgentId = "",

  # Optional admin portal URL you want Foundry to store
  [string]$AdminPortalUrl = ""
)

function Normalize-BaseUrl([string]$u) {
  $u = $u.Trim()
  if ($u.EndsWith("/")) { $u = $u.Substring(0, $u.Length - 1) }
  return $u
}

$AgentBaseUrl = Normalize-BaseUrl $AgentBaseUrl

# Basic reachability test: try /v1/models then /health then root
$testPaths = @("/v1/models","/health","/")

$hdr = @{}
foreach ($h in $Headers) {
  if ($h -match '^\s*([^:]+)\s*:\s*(.+)\s*$') {
    $hdr[$Matches[1]] = $Matches[2]
  }
}

Write-Host "[foundry] AgentBaseUrl: $AgentBaseUrl"
Write-Host "[foundry] Protocol: $Protocol"
Write-Host ""

$reachable = $false
foreach ($p in $testPaths) {
  $url = "$AgentBaseUrl$p"
  try {
    Write-Host "[foundry] Probing $url"
    $resp = Invoke-WebRequest -Uri $url -Headers $hdr -Method GET -TimeoutSec 10 -ErrorAction Stop
    Write-Host "[foundry]   OK: HTTP $($resp.StatusCode)"
    $reachable = $true
    break
  } catch {
    Write-Host "[foundry]   FAIL: $($_.Exception.Message)"
  }
}

Write-Host ""
if (-not $reachable) {
  Write-Host "[foundry] WARNING: Could not confirm reachability. Registration may still work if the endpoint requires a different path or auth."
}

# Print the exact wizard fields (per doc)
Write-Host ""
Write-Host "==================== Paste into Foundry Control Plane registration wizard ===================="
Write-Host "Agent URL: $AgentBaseUrl"
Write-Host "Protocol: $Protocol"
Write-Host "A2A agent card URL (optional): $AgentCardPath"
if (-not [string]::IsNullOrWhiteSpace($OpenTelemetryAgentId)) {
  Write-Host "OpenTelemetry Agent ID (optional): $OpenTelemetryAgentId"
} else {
  Write-Host "OpenTelemetry Agent ID (optional): (leave blank unless you emit gen_ai.agents.id)"
}
if (-not [string]::IsNullOrWhiteSpace($AdminPortalUrl)) {
  Write-Host "Admin portal URL (optional): $AdminPortalUrl"
} else {
  Write-Host "Admin portal URL (optional): (optional)"
}
Write-Host "=============================================================================================="
Write-Host ""

# Open the Foundry portal (new portal is required per doc)
# The doc: sign in to https://ai.azure.com, ensure New Foundry toggle is on, then Operate -> Register agent. :contentReference[oaicite:2]{index=2}
Write-Host "[foundry] Opening Foundry portal. Use: Operate -> Register agent."
Start-Process "https://ai.azure.com/"
