param(
    [string]$Version = "1.0.0",
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$ReleaseDir = Join-Path $RepoRoot "build\release"
$BinDir = Join-Path $RepoRoot "build\bin"

New-Item -ItemType Directory -Force -Path $ReleaseDir | Out-Null

function ConvertTo-LFBytes {
    param([string]$FilePath)
    $raw = [System.IO.File]::ReadAllText($FilePath, [System.Text.UTF8Encoding]::new($false))
    $lf = $raw -replace "`r`n", "`n"
    return [System.Text.UTF8Encoding]::new($false).GetBytes($lf)
}

function Write-LFFile {
    param([string]$Src, [string]$Dst)
    $bytes = ConvertTo-LFBytes -FilePath $Src
    [System.IO.File]::WriteAllBytes($Dst, $bytes)
}

function Find-Makensis {
    $candidates = @(
        "$env:USERPROFILE\.local\nsis\makensis.exe",
        "C:\Program Files (x86)\NSIS\makensis.exe",
        "C:\Program Files\NSIS\makensis.exe"
    )
    foreach ($c in $candidates) {
        if (Test-Path $c) { return $c }
    }
    $cmd = Get-Command makensis.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return $null
}

Push-Location $RepoRoot
try {
    if (-not $SkipBuild) {
        Write-Host "[1/5] Building Frontend (tsc && vite build)..." -ForegroundColor Cyan
        Push-Location (Join-Path $RepoRoot "frontend")
        try {
            npm run build
            if ($LASTEXITCODE -ne 0) { throw "Frontend build failed" }
        } finally {
            Pop-Location
        }

        Write-Host "[2/5] Building Windows binaries (TDeck.exe & tdeck-sync.exe)..." -ForegroundColor Cyan
        Get-Process -Name "TDeck" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        wails build -s -platform windows/amd64
        if ($LASTEXITCODE -ne 0) { throw "wails build failed" }

        go build -trimpath -ldflags "-s -w" -o (Join-Path $BinDir "tdeck-sync.exe") ./cmd/tdeck-sync
        if ($LASTEXITCODE -ne 0) { throw "tdeck-sync windows build failed" }
    }

    $makensis = Find-Makensis
    if ($makensis) {
        Write-Host "[3/5] Packaging Windows NSIS Installer..." -ForegroundColor Cyan
        $nsisDir = Split-Path $makensis
        if ($env:PATH -notlike "*$nsisDir*") {
            $env:PATH = "$nsisDir;$env:PATH"
        }
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        wails build -s -nsis -skipbindings -platform windows/amd64
        if ($LASTEXITCODE -ne 0) { throw "wails NSIS installer build failed" }

        $syncExe = Join-Path $BinDir "tdeck-sync.exe"
        if (Test-Path $syncExe) {
            Push-Location (Join-Path $RepoRoot "build\windows\installer")
            try {
                & $makensis "-DARG_WAILS_AMD64_BINARY=..\..\bin\TDeck.exe" "-DARG_TDECK_SYNC_BINARY=..\..\bin\tdeck-sync.exe" "project.nsi"
                if ($LASTEXITCODE -ne 0) { throw "makensis rebuild with tdeck-sync.exe failed" }
            } finally {
                Pop-Location
            }
        }

        $nsisOut = Join-Path $BinDir "TDeck-amd64-installer.exe"
        if (Test-Path $nsisOut) {
            Copy-Item -Path $nsisOut -Destination (Join-Path $ReleaseDir "TDeck-windows-amd64-installer.exe") -Force
        }
    } else {
        Write-Warning "makensis.exe not found; skipping Windows NSIS installer packaging."
    }

    Write-Host "[4/5] Packaging Windows Portable ZIP..." -ForegroundColor Cyan
    $winStage = Join-Path $env:TEMP ("tdeck-win-stage-" + [Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $winStage | Out-Null
    try {
        Copy-Item (Join-Path $BinDir "TDeck.exe") $winStage
        if (Test-Path (Join-Path $BinDir "tdeck-sync.exe")) {
            Copy-Item (Join-Path $BinDir "tdeck-sync.exe") $winStage
        }
        Copy-Item (Join-Path $RepoRoot ".env.example") $winStage
        Copy-Item (Join-Path $RepoRoot "README.md") $winStage
        Copy-Item (Join-Path $RepoRoot "CHANGELOG.md") $winStage

        $winZip = Join-Path $ReleaseDir "TDeck-windows-amd64-portable.zip"
        Remove-Item -Force -ErrorAction SilentlyContinue $winZip
        Compress-Archive -Path "$winStage\*" -DestinationPath $winZip -CompressionLevel Optimal -Force
    } finally {
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $winStage
    }

    Write-Host "[5/5] Cross-compiling Linux CLI & Packaging Linux Releases (.tar.gz + .run)..." -ForegroundColor Cyan
    $linuxStage = Join-Path $env:TEMP ("tdeck-linux-stage-" + [Guid]::NewGuid().ToString("N"))
    $linuxPayloadName = "t-deck-linux-amd64"
    $linuxPayloadDir = Join-Path $linuxStage $linuxPayloadName
    New-Item -ItemType Directory -Force -Path $linuxPayloadDir | Out-Null
    try {
        $prevGOOS = $env:GOOS
        $prevGOARCH = $env:GOARCH
        $prevCGO = $env:CGO_ENABLED
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $env:CGO_ENABLED = "0"
        go build -trimpath -ldflags "-s -w" -o (Join-Path $linuxPayloadDir "tdeck-sync") ./cmd/tdeck-sync
        $goExit = $LASTEXITCODE
        $env:GOOS = $prevGOOS
        $env:GOARCH = $prevGOARCH
        $env:CGO_ENABLED = $prevCGO
        if ($goExit -ne 0) { throw "tdeck-sync linux/amd64 cross-compile failed" }

        $linuxGuiBin = Join-Path $BinDir "TDeck-linux-amd64"
        if (Test-Path $linuxGuiBin) {
            Copy-Item $linuxGuiBin (Join-Path $linuxPayloadDir "TDeck")
        }

        Write-LFFile -Src (Join-Path $RepoRoot "build\linux\install.sh") -Dst (Join-Path $linuxPayloadDir "install.sh")
        Write-LFFile -Src (Join-Path $RepoRoot "build\linux\uninstall.sh") -Dst (Join-Path $linuxPayloadDir "uninstall.sh")
        Write-LFFile -Src (Join-Path $RepoRoot "build\linux\t-deck.desktop") -Dst (Join-Path $linuxPayloadDir "t-deck.desktop")
        Write-LFFile -Src (Join-Path $RepoRoot ".env.example") -Dst (Join-Path $linuxPayloadDir ".env.example")
        Write-LFFile -Src (Join-Path $RepoRoot "README.md") -Dst (Join-Path $linuxPayloadDir "README.md")
        Write-LFFile -Src (Join-Path $RepoRoot "CHANGELOG.md") -Dst (Join-Path $linuxPayloadDir "CHANGELOG.md")
        Copy-Item (Join-Path $RepoRoot "build\appicon.png") (Join-Path $linuxPayloadDir "t-deck.png")

        $linuxTarGz = Join-Path $ReleaseDir "TDeck-linux-amd64.tar.gz"
        Remove-Item -Force -ErrorAction SilentlyContinue $linuxTarGz
        Push-Location $linuxStage
        try {
            tar -czf $linuxTarGz $linuxPayloadName
            if ($LASTEXITCODE -ne 0) { throw "tar failed creating Linux archive" }
        } finally {
            Pop-Location
        }

        # Build self-extracting .run installer for Linux
        $stubHeader = @'
#!/usr/bin/env sh
set -eu
TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t tdeck-install)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM
ARCHIVE_LINE=$(awk '/^__ARCHIVE_BELOW__$/ {print NR + 1; exit 0;}' "$0")
tail -n +"$ARCHIVE_LINE" "$0" | tar -xz -C "$TMP_DIR"
chmod +x "$TMP_DIR/t-deck-linux-amd64/install.sh" "$TMP_DIR/t-deck-linux-amd64/uninstall.sh"
if [ -f "$TMP_DIR/t-deck-linux-amd64/tdeck-sync" ]; then chmod +x "$TMP_DIR/t-deck-linux-amd64/tdeck-sync"; fi
if [ -f "$TMP_DIR/t-deck-linux-amd64/TDeck" ]; then chmod +x "$TMP_DIR/t-deck-linux-amd64/TDeck"; fi
exec bash "$TMP_DIR/t-deck-linux-amd64/install.sh" "$@"
__ARCHIVE_BELOW__
'@
        $stubLF = ($stubHeader -replace "`r`n", "`n").TrimEnd("`n") + "`n"
        $stubBytes = [System.Text.UTF8Encoding]::new($false).GetBytes($stubLF)
        $tarBytes = [System.IO.File]::ReadAllBytes($linuxTarGz)

        $linuxRun = Join-Path $ReleaseDir "TDeck-linux-amd64-installer.run"
        $fs = [System.IO.File]::Open($linuxRun, [System.IO.FileMode]::Create, [System.IO.FileAccess]::Write)
        try {
            $fs.Write($stubBytes, 0, $stubBytes.Length)
            $fs.Write($tarBytes, 0, $tarBytes.Length)
        } finally {
            $fs.Close()
        }
    } finally {
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $linuxStage
    }

    Write-Host "Generating SHA256SUMS.txt..." -ForegroundColor Cyan
    $sumLines = Get-ChildItem $ReleaseDir -File | Where-Object { $_.Name -ne "SHA256SUMS.txt" } | Sort-Object Name | ForEach-Object {
        $hash = (Get-FileHash -Path $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $($_.Name)"
    }
    [System.IO.File]::WriteAllText((Join-Path $ReleaseDir "SHA256SUMS.txt"), ($sumLines -join "`n") + "`n", [System.Text.UTF8Encoding]::new($false))

    Write-Host "`nRelease artifacts ready in ${ReleaseDir}:" -ForegroundColor Green
    Get-ChildItem $ReleaseDir -File | Select-Object Name, Length | Format-Table -AutoSize
} finally {
    Pop-Location
}
