param([string]$Platform = "claude-code")
$ErrorActionPreference = "Stop"

if (-not $env:GPOWERS_HOME) {
  Write-Error "GPOWERS_HOME must be set"
  exit 1
}

$skillFile = Join-Path $env:GPOWERS_HOME "core/skills/using-gpowers/SKILL.md"
if (-not (Test-Path $skillFile)) {
  Write-Error "gpowers: using-gpowers skill not found at $skillFile"
  exit 1
}

# Strip YAML frontmatter
$lines = Get-Content $skillFile
$fmCount = 0
$body = @()
foreach ($line in $lines) {
  if ($line -eq "---") { $fmCount++; continue }
  if ($fmCount -ge 2) { $body += $line }
}
$content = $body -join "`n"

switch ($Platform) {
  { @("claude-code","codex","opencode","copilot") -contains $_ } {
    "<EXTREMELY_IMPORTANT>`n$content`n</EXTREMELY_IMPORTANT>"
  }
  default { $content }
}
