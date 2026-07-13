param(
    [int]$Rate = 25,
    [string]$Duration = "30s",
    [string]$Name = "single-replica"
)

$ErrorActionPreference = "Stop"
$results = Join-Path $PSScriptRoot "results"
New-Item -ItemType Directory -Force -Path $results | Out-Null
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$bin = Join-Path $results "$Name-$timestamp.bin"
$report = Join-Path $results "$Name-$timestamp.txt"

Get-Content (Join-Path $PSScriptRoot "ingest-target.json") |
    vegeta attack -rate="$Rate/s" -duration=$Duration |
    Tee-Object -FilePath $bin |
    vegeta report | Tee-Object -FilePath $report

Write-Host "Saved raw results to $bin and report to $report"
