param(
    [string] $Version = "",
    [string] $ModulePath = "github.com/mohamedhabibwork/abp-cli",
    [string] $InstallDir = "",
    [string] $CliName = "abp-cli"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is required to install $CliName. Install Go from https://go.dev/doc/install, then run this script again."
}

if ($InstallDir) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $env:GOBIN = (Resolve-Path $InstallDir).Path
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$goModPath = Join-Path $scriptDir "go.mod"
$localModulePath = ""

if (Test-Path $goModPath) {
    foreach ($line in Get-Content $goModPath) {
        if ($line -match "^\s*module\s+(\S+)") {
            $localModulePath = $Matches[1]
            break
        }
    }
}

if ($Version) {
    $installTarget = "$ModulePath@$Version"
    Write-Host "Installing $CliName from $installTarget..."
    go install $installTarget
}
elseif (($localModulePath -eq $ModulePath) -and (Test-Path (Join-Path $scriptDir "main.go"))) {
    Write-Host "Installing $CliName from $scriptDir..."
    Push-Location $scriptDir
    try {
        go install .
    }
    finally {
        Pop-Location
    }
}
else {
    $installTarget = "$ModulePath@latest"
    Write-Host "Installing $CliName from $installTarget..."
    go install $installTarget
}

$binDir = if ($env:GOBIN) { $env:GOBIN } else { Join-Path (go env GOPATH) "bin" }
$exeName = if ($IsWindows -or $env:OS -eq "Windows_NT") { "$CliName.exe" } else { $CliName }
$binPath = Join-Path $binDir $exeName

if (-not (Test-Path $binPath)) {
    throw "Install finished, but $binPath was not found. Check GOBIN or GOPATH with: go env GOBIN GOPATH"
}

Write-Host ""
Write-Host "$CliName installed:"
Write-Host "  $binPath"
Write-Host ""

$pathParts = $env:PATH -split [IO.Path]::PathSeparator
if ($pathParts -notcontains $binDir) {
    Write-Host "Add this directory to PATH:"
    if ($IsWindows -or $env:OS -eq "Windows_NT") {
        Write-Host "  setx PATH `"$binDir;%PATH%`""
    }
    else {
        Write-Host "  `$env:PATH = `"$binDir$([IO.Path]::PathSeparator)`$env:PATH`""
    }
    Write-Host ""
}

& $binPath --help | Out-Null
Write-Host "Verified: $CliName --help"
