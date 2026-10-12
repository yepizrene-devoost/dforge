# installer-contract.ps1 — contract test for scripts/install.ps1.
#
# Shadows Invoke-WebRequest so the installer runs against a local fixture
# "release" (no network): fixture files are served by URL basename, every
# requested URL is recorded, and the /releases/latest probe returns a fixture
# Location header. The cases mirror tests/installer-contract.sh: exact asset
# names per architecture, a good archive installs, and a corrupted archive, a
# missing checksums.txt or an unsupported architecture fail hard without
# touching the installed binary. Runs under Windows PowerShell 5.1 and pwsh.

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = Split-Path -Parent (Split-Path -Parent $PSCommandPath)
$installer = Join-Path $root 'scripts/install.ps1'
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("dforge-installer-test-" + [Guid]::NewGuid().ToString('N'))
$fixture = Join-Path $temp 'fixture'
$install = Join-Path $temp 'install'
$repoUrl = 'https://github.com/yepizrene-devoost/dforge'
$assetName = 'dforge_Windows_x86_64.zip'
$armAssetName = 'dforge_Windows_arm64.zip'
$global:DforgeInstallerUris = @()
$global:DforgeInstallerBasicParsing = @()
$global:DforgeInstallerFixtureDir = $fixture
$global:DforgeInstallerLatestTag = 'v1.2.3'

function Invoke-WebRequest {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [string]$OutFile,
        [int]$MaximumRedirection,
        [switch]$UseBasicParsing
    )
    $global:DforgeInstallerUris += $Uri
    $global:DforgeInstallerBasicParsing += $UseBasicParsing.IsPresent

    if ([string]::IsNullOrWhiteSpace($OutFile)) {
        # Latest-release redirect probe: hand back the Location header the
        # installer reads, without following the redirect.
        $location = "$repoUrl/releases/tag/$($global:DforgeInstallerLatestTag)"
        return [pscustomobject]@{ Headers = @{ Location = $location } }
    }

    $name = ($Uri -split '/')[-1]
    $source = Join-Path $global:DforgeInstallerFixtureDir $name
    if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
        throw "Unexpected fixture download: $Uri"
    }
    Copy-Item -LiteralPath $source -Destination $OutFile
}

function Reset-Requests {
    $global:DforgeInstallerUris = @()
    $global:DforgeInstallerBasicParsing = @()
}

function Assert-Uris {
    param(
        [Parameter(Mandatory = $true)][string[]]$Expected,
        [Parameter(Mandatory = $true)][string]$Reason
    )
    $actual = @($global:DforgeInstallerUris)
    if ($actual.Count -ne $Expected.Count) {
        throw "$Reason (observed $($actual.Count) requests: $($actual -join ', '))"
    }
    for ($i = 0; $i -lt $Expected.Count; $i++) {
        if ($actual[$i] -ne $Expected[$i]) {
            throw "$Reason (request $($i + 1) was '$($actual[$i])', expected '$($Expected[$i])')"
        }
    }
}

function Assert-BasicParsingMatchesEngine {
    $expect = $PSVersionTable.PSVersion.Major -lt 6
    $unexpected = @($global:DforgeInstallerBasicParsing | Where-Object { $_ -ne $expect })
    if ($unexpected.Count -ne 0) {
        throw "Invoke-WebRequest UseBasicParsing did not match PowerShell $($PSVersionTable.PSVersion.Major) requirements."
    }
}

function Assert-InstalledBinary {
    param(
        [Parameter(Mandatory = $true)][string]$Expected,
        [Parameter(Mandatory = $true)][string]$Reason
    )
    $target = Join-Path $install 'dforge.exe'
    if ((Get-Content -LiteralPath $target -Raw) -ne $Expected) {
        throw $Reason
    }
}

function Invoke-InstallerExpectFailure {
    param([Parameter(Mandatory = $true)][string]$Reason)
    try {
        & $installer *> $null
        throw "Installer unexpectedly accepted $Reason."
    }
    catch {
        if ($_.Exception.Message -eq "Installer unexpectedly accepted $Reason.") { throw }
    }
}

# New-FixtureArchive <name> <content>: one-binary zip plus a checksums.txt
# naming that asset, GoReleaser's "<sha256>  <name>" shape.
function New-FixtureArchive {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Content
    )
    $payload = Join-Path $temp 'payload'
    Remove-Item -LiteralPath $payload -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Path $payload | Out-Null
    Set-Content -LiteralPath (Join-Path $payload 'dforge.exe') -Value $Content -NoNewline
    $archive = Join-Path $fixture $Name
    Remove-Item -LiteralPath $archive -Force -ErrorAction SilentlyContinue
    Compress-Archive -LiteralPath (Join-Path $payload 'dforge.exe') -DestinationPath $archive
    $hash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    Set-Content -LiteralPath (Join-Path $fixture 'checksums.txt') -Value "$hash  $Name"
}

try {
    New-Item -ItemType Directory -Path $fixture, $install -Force | Out-Null
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    $env:LOCALAPPDATA = Join-Path $temp 'local-app-data'
    $env:DFORGE_INSTALL_DIR = $install
    $env:DFORGE_VERSION = 'v1.2.3'
    $env:DFORGE_SKIP_PATH_UPDATE = '1'

    # --- Good archive installs and replaces the old binary (x86_64) ----------
    New-FixtureArchive -Name $assetName -Content 'new binary'
    Set-Content -LiteralPath (Join-Path $install 'dforge.exe') -Value 'old binary' -NoNewline
    Reset-Requests
    & $installer *> $null
    Assert-InstalledBinary -Expected 'new binary' -Reason 'Installed binary did not safely replace the old binary.'
    Assert-Uris -Expected @(
        "$repoUrl/releases/download/v1.2.3/$assetName",
        "$repoUrl/releases/download/v1.2.3/checksums.txt"
    ) -Reason 'Pinned install did not request the exact x86_64 asset and checksums.txt.'
    Assert-BasicParsingMatchesEngine
    Write-Host 'ok - x86_64 asset installs and replaces the old binary'

    # --- Asset name construction for arm64 -----------------------------------
    $env:PROCESSOR_ARCHITECTURE = 'ARM64'
    New-FixtureArchive -Name $armAssetName -Content 'arm binary'
    Reset-Requests
    & $installer *> $null
    Assert-InstalledBinary -Expected 'arm binary' -Reason 'ARM64 install produced the wrong payload.'
    Assert-Uris -Expected @(
        "$repoUrl/releases/download/v1.2.3/$armAssetName",
        "$repoUrl/releases/download/v1.2.3/checksums.txt"
    ) -Reason 'ARM64 install did not request the exact arm64 asset.'
    Write-Host 'ok - arm64 asset name construction'
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'

    # --- Corrupted archive must fail before extraction ------------------------
    New-FixtureArchive -Name $assetName -Content 'new binary'
    $archivePath = Join-Path $fixture $assetName
    $bytes = [System.IO.File]::ReadAllBytes($archivePath)
    [System.IO.File]::WriteAllBytes($archivePath, [byte[]]($bytes + 0x00))
    Set-Content -LiteralPath (Join-Path $install 'dforge.exe') -Value 'old binary' -NoNewline
    Reset-Requests
    Invoke-InstallerExpectFailure -Reason 'a corrupted archive'
    Assert-InstalledBinary -Expected 'old binary' -Reason 'Checksum mismatch changed the installed binary.'
    Write-Host 'ok - corrupted archive rejected, install untouched'

    # --- Missing checksums.txt must fail --------------------------------------
    New-FixtureArchive -Name $assetName -Content 'new binary'
    Remove-Item -LiteralPath (Join-Path $fixture 'checksums.txt') -Force
    Reset-Requests
    Invoke-InstallerExpectFailure -Reason 'a missing checksums.txt'
    Assert-InstalledBinary -Expected 'old binary' -Reason 'Missing checksums.txt changed the installed binary.'
    Write-Host 'ok - missing checksums.txt rejected'

    # --- Unsupported architecture must fail before download -------------------
    Reset-Requests
    $env:PROCESSOR_ARCHITECTURE = 'x86'
    Invoke-InstallerExpectFailure -Reason 'an unsupported architecture'
    if ($global:DforgeInstallerUris.Count -ne 0) {
        throw 'Installer downloaded before rejecting the architecture.'
    }
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    Write-Host 'ok - unsupported architecture rejected'

    # --- Latest-release resolution (no pinned version) ------------------------
    New-FixtureArchive -Name $assetName -Content 'latest binary'
    Remove-Item Env:DFORGE_VERSION -ErrorAction SilentlyContinue
    Reset-Requests
    $latestOutput = (& $installer 6>&1 | Out-String)
    Assert-InstalledBinary -Expected 'latest binary' -Reason 'Latest-release install produced the wrong payload.'
    Assert-Uris -Expected @(
        "$repoUrl/releases/latest",
        "$repoUrl/releases/download/v1.2.3/$assetName",
        "$repoUrl/releases/download/v1.2.3/checksums.txt"
    ) -Reason 'Latest-release install did not resolve the tag and download the exact assets.'
    Assert-BasicParsingMatchesEngine
    if ($latestOutput -notmatch 'to your user PATH') {
        throw 'Installer did not print the PATH hint.'
    }
    Write-Host 'ok - latest release resolution and PATH hint'

    # --- Static release-contract checks ---------------------------------------
    $config = Get-Content -LiteralPath (Join-Path $root '.goreleaser.yaml') -Raw
    if ($config -notmatch [regex]::Escape('scripts/install.ps1')) {
        throw 'GoReleaser does not publish scripts/install.ps1.'
    }
    if ($config -notmatch [regex]::Escape('name_template: checksums.txt')) {
        throw 'GoReleaser does not publish checksums.txt.'
    }
    Write-Host 'ok - static release contract'

    Write-Host 'PowerShell installer contract passed'
}
finally {
    Remove-Variable -Name DforgeInstallerUris -Scope Global -ErrorAction SilentlyContinue
    Remove-Variable -Name DforgeInstallerBasicParsing -Scope Global -ErrorAction SilentlyContinue
    Remove-Variable -Name DforgeInstallerFixtureDir -Scope Global -ErrorAction SilentlyContinue
    Remove-Variable -Name DforgeInstallerLatestTag -Scope Global -ErrorAction SilentlyContinue
    Remove-Item Env:DFORGE_INSTALL_DIR -ErrorAction SilentlyContinue
    Remove-Item Env:DFORGE_VERSION -ErrorAction SilentlyContinue
    Remove-Item Env:DFORGE_SKIP_PATH_UPDATE -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $temp -Recurse -Force -ErrorAction SilentlyContinue
}
