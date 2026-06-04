#Requires -Version 5.1
<#
.SYNOPSIS
  Cross-compile Linux binary on Windows (default: linux/amd64).

.EXAMPLE
  .\scripts\build-linux.ps1
  .\scripts\build-linux.ps1 -Arch arm64
#>
param(
    [ValidateSet('amd64', 'arm64')]
    [string] $Arch = 'amd64',

    [string] $OutDir = '',

    [switch] $SkipVerify
)

$ErrorActionPreference = 'Stop'
$BackendRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $BackendRoot

if (-not $OutDir) {
    $OutDir = Join-Path $BackendRoot "dist\linux-$Arch"
}
$OutFile = Join-Path $OutDir 'lumehub'

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

Write-Host ">> cross-compile linux/$Arch -> $OutFile" -ForegroundColor Cyan

$savedGOOS = $env:GOOS
$savedGOARCH = $env:GOARCH
$savedCGO = $env:CGO_ENABLED
try {
    $env:GOOS = 'linux'
    $env:GOARCH = $Arch
    $env:CGO_ENABLED = '0'

    go build -trimpath -ldflags '-s -w' -o $OutFile ./cmd/lumehub
    if ($LASTEXITCODE -ne 0) {
        throw 'go build failed'
    }

    if (-not $SkipVerify) {
        $head = [System.IO.File]::ReadAllBytes($OutFile)[0..3]
        if ($head[0] -ne 0x7F -or $head[1] -ne 0x45 -or $head[2] -ne 0x4C -or $head[3] -ne 0x46) {
            throw 'Output is not ELF; check GOOS/GOARCH'
        }
        Write-Host '>> ELF verified' -ForegroundColor Green
    }

    $size = (Get-Item $OutFile).Length
    Write-Host ">> done ($([math]::Round($size / 1MB, 2)) MB): $OutFile" -ForegroundColor Green
}
finally {
    if ($null -eq $savedGOOS) { Remove-Item Env:GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $savedGOOS }
    if ($null -eq $savedGOARCH) { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $savedGOARCH }
    if ($null -eq $savedCGO) { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue } else { $env:CGO_ENABLED = $savedCGO }
}
