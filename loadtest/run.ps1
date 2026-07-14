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

$targets = Join-Path $PSScriptRoot "ingest-target.http"
$body = Join-Path $PSScriptRoot "ingest-body.json"

& vegeta attack -targets $targets -body $body -rate "$Rate/s" -duration $Duration -output $bin
& vegeta report $bin | Tee-Object -FilePath $report

Write-Host "Saved raw results to $bin and report to $report"
