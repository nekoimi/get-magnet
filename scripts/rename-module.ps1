[CmdletBinding()]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidateNotNullOrEmpty()]
    [string]$NewModule
)

$ErrorActionPreference = 'Stop'

if (
    $NewModule.StartsWith('/') -or
    $NewModule.EndsWith('/') -or
    $NewModule.Contains('//') -or
    $NewModule -match '[\s\\]'
) {
    throw "Invalid module path: $NewModule"
}

$projectRoot = Split-Path -Parent $PSScriptRoot
$goModPath = Join-Path $projectRoot 'go.mod'

if (-not (Test-Path -LiteralPath $goModPath -PathType Leaf)) {
    throw "go.mod not found: $goModPath"
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'The go command is required but was not found.'
}

$goModContent = [System.IO.File]::ReadAllText($goModPath)
$moduleMatch = [regex]::Match($goModContent, '(?m)^module\s+(\S+)\s*$')
if (-not $moduleMatch.Success) {
    throw "No module directive found in $goModPath"
}

$oldModule = $moduleMatch.Groups[1].Value
if ($oldModule -eq $NewModule) {
    Write-Host "Module path is already $NewModule"
    exit 0
}

# Let the Go tool validate the new path and update only the module directive.
& go mod edit "-module=$NewModule" $goModPath
if ($LASTEXITCODE -ne 0) {
    throw "go mod edit failed with exit code $LASTEXITCODE"
}

$utf8WithoutBom = [System.Text.UTF8Encoding]::new($false)
$changedFiles = 0

Get-ChildItem -LiteralPath $projectRoot -Recurse -File -Filter '*.go' |
    Where-Object { $_.FullName -notmatch '[\\/](?:\.git|vendor)[\\/]' } |
    ForEach-Object {
        $content = [System.IO.File]::ReadAllText($_.FullName)
        if (-not $content.Contains($oldModule)) {
            return
        }

        $updatedContent = $content.Replace($oldModule, $NewModule)
        [System.IO.File]::WriteAllText($_.FullName, $updatedContent, $utf8WithoutBom)
        $changedFiles++
    }

$dockerfilePath = Join-Path $projectRoot 'Dockerfile'
if (Test-Path -LiteralPath $dockerfilePath -PathType Leaf) {
    $content = [System.IO.File]::ReadAllText($dockerfilePath)
    if ($content.Contains($oldModule)) {
        $updatedContent = $content.Replace($oldModule, $NewModule)
        [System.IO.File]::WriteAllText($dockerfilePath, $updatedContent, $utf8WithoutBom)
        $changedFiles++
    }
}

Write-Host "Module path changed: $oldModule -> $NewModule"
Write-Host "Updated $changedFiles source file(s)."