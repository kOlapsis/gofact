# Installe gofact sous Windows : binaire de la dernière release dans
# %LOCALAPPDATA%\gofact, puis enregistrement des clients MCP du poste.
$ErrorActionPreference = "Stop"
$repo = "kolapsis/gofact"
$arch = if ([Environment]::Is64BitOperatingSystem -and $env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$tag = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
$url = "https://github.com/$repo/releases/download/$tag/gofact_$($tag.TrimStart('v'))_windows_$arch.zip"
$dir = Join-Path $env:LOCALAPPDATA "gofact"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$zip = Join-Path $env:TEMP "gofact.zip"
Invoke-WebRequest $url -OutFile $zip
Expand-Archive $zip -DestinationPath $dir -Force
Remove-Item $zip
Write-Host "✓ gofact $tag installé dans $dir"
& (Join-Path $dir "gofact.exe") install
Write-Host "Pour terminer : $dir\gofact.exe install -yes"
