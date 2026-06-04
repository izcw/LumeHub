#Requires -Version 5.1
<#
.SYNOPSIS
  Upload Linux binary and www/ to remote server (keeps data/).

.EXAMPLE
  Copy deploy.example.env to deploy.env first.
  .\scripts\deploy.ps1
  .\scripts\deploy.ps1 -Build
  .\scripts\deploy.ps1 -BuildFrontend
#>
param(
    [ValidateSet('amd64', 'arm64')]
    [string] $Arch = 'amd64',

    [string] $ConfigFile = '',

    [switch] $Build,

    [switch] $BuildFrontend,

    [switch] $SkipWww
)

$ErrorActionPreference = 'Stop'
$BackendRoot = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$RepoRoot = Split-Path -Parent $BackendRoot
Set-Location $BackendRoot

function Read-DeployEnv {
    param([string] $Path)
    $vars = @{}
    Get-Content -LiteralPath $Path -Encoding UTF8 | ForEach-Object {
        $line = $_.Trim()
        if (-not $line -or $line.StartsWith('#')) { return }
        $eq = $line.IndexOf('=')
        if ($eq -lt 1) { return }
        $key = $line.Substring(0, $eq).Trim()
        $val = $line.Substring($eq + 1).Trim()
        $vars[$key] = $val
    }
    return $vars
}

function Invoke-Ssh {
    param(
        [string[]] $BaseArgs,
        [string] $RemoteCommand
    )
    $args = @($BaseArgs)
    if ($RemoteCommand) {
        $args += $RemoteCommand
    }
    & ssh @args
    if ($LASTEXITCODE -ne 0) {
        throw "ssh failed (exit $LASTEXITCODE)"
    }
}

function Invoke-Scp {
    param(
        [string[]] $BaseArgs,
        [string[]] $Sources,
        [string] $Dest
    )
    $args = @($BaseArgs) + $Sources + @($Dest)
    & scp @args
    if ($LASTEXITCODE -ne 0) {
        throw "scp failed (exit $LASTEXITCODE)"
    }
}

if (-not $ConfigFile) {
    $ConfigFile = Join-Path $BackendRoot 'deploy.env'
}
if (-not (Test-Path -LiteralPath $ConfigFile)) {
    throw "Missing $ConfigFile. Copy deploy.example.env to deploy.env and set DEPLOY_TARGET, REMOTE_PATH."
}

$cfg = Read-DeployEnv -Path $ConfigFile
$target = $cfg['DEPLOY_TARGET']
$remotePath = $cfg['REMOTE_PATH']
$sshPort = $cfg['SSH_PORT']
$sshKey = $cfg['SSH_KEY']
$postCmd = $cfg['REMOTE_POST_CMD']

if (-not $target) { throw 'deploy.env: DEPLOY_TARGET is required' }
if (-not $remotePath) { throw 'deploy.env: REMOTE_PATH is required' }

if ($BuildFrontend) {
    Write-Host '>> pnpm build (Frontend -> Backend/www)' -ForegroundColor Cyan
    Push-Location (Join-Path $RepoRoot 'Frontend')
    try {
        pnpm build
        if ($LASTEXITCODE -ne 0) { throw 'pnpm build failed' }
    } finally {
        Pop-Location
    }
}

if ($Build -or $BuildFrontend) {
    & (Join-Path $BackendRoot 'scripts\build-linux.ps1') -Arch $Arch
}

$binary = Join-Path $BackendRoot "dist\linux-$Arch\lumehub"
if (-not (Test-Path -LiteralPath $binary)) {
    throw "Binary not found: $binary. Run .\scripts\build-linux.ps1 or use -Build."
}

$wwwDir = Join-Path $BackendRoot 'www'
if (-not $SkipWww -and -not (Test-Path -LiteralPath (Join-Path $wwwDir 'index.html'))) {
    throw 'www/index.html missing. Run pnpm build in Frontend, or use -BuildFrontend.'
}

$sshBase = @()
$scpBase = @()
if ($sshPort) {
    $sshBase += @('-p', $sshPort)
    $scpBase += @('-P', $sshPort)
}
if ($sshKey) {
    $sshBase += @('-i', $sshKey)
    $scpBase += @('-i', $sshKey)
}

$remote = "${target}:${remotePath}"
Write-Host ">> upload lumehub -> $remote/" -ForegroundColor Cyan
Invoke-Scp -BaseArgs $scpBase -Sources @($binary) -Dest "${remote}/lumehub"

if (-not $SkipWww) {
    Write-Host ">> upload www/ -> $remote/www/" -ForegroundColor Cyan
    Invoke-Ssh -BaseArgs $sshBase -RemoteCommand "${target} mkdir -p ${remotePath}/www"
    Invoke-Scp -BaseArgs ($scpBase + @('-r')) -Sources @(
        (Join-Path $wwwDir '*')
    ) -Dest "${remote}/www/"
}

Write-Host '>> chmod +x on server' -ForegroundColor Cyan
Invoke-Ssh -BaseArgs $sshBase -RemoteCommand "${target} chmod +x ${remotePath}/lumehub"

if ($postCmd) {
    Write-Host ">> remote: $postCmd" -ForegroundColor Cyan
    Invoke-Ssh -BaseArgs $sshBase -RemoteCommand "${target} $postCmd"
}

Write-Host '>> done (data/ untouched; restart Supervisor on server)' -ForegroundColor Green
