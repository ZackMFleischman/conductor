[CmdletBinding()]
param(
    [Parameter()]
    [string] $GoExecutable = 'go',

    [Parameter()]
    [string] $OutputDirectory = 'dist'
)

$ErrorActionPreference = 'Stop'
$sourceDirectory = (Get-Location).Path
$outputPath = if ([System.IO.Path]::IsPathFullyQualified($OutputDirectory)) {
    [System.IO.Path]::GetFullPath($OutputDirectory)
} else {
    [System.IO.Path]::GetFullPath((Join-Path $sourceDirectory $OutputDirectory))
}

$gitCommit = (& git -C $sourceDirectory rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($gitCommit)) {
    throw 'Unable to determine the git commit for the build source.'
}
$gitStatus = (& git -C $sourceDirectory status --porcelain --untracked-files=normal)
if ($LASTEXITCODE -ne 0) {
    throw 'Unable to determine whether the build source is dirty.'
}
$gitDirty = if ($gitStatus) { 'true' } else { 'false' }

$savedEnvironment = @{}
foreach ($name in @('GOOS', 'GOARCH', 'CGO_ENABLED')) {
    $savedEnvironment[$name] = [System.Environment]::GetEnvironmentVariable($name, 'Process')
}

try {
    New-Item -ItemType Directory -Force -Path $outputPath | Out-Null

    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    & $GoExecutable build -trimpath -o (Join-Path $outputPath 'conductor.exe') '.\cmd\conductor'
    if ($LASTEXITCODE -ne 0) {
        throw "Windows amd64 build failed with exit code $LASTEXITCODE."
    }

    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    & $GoExecutable build -trimpath -o (Join-Path $outputPath 'conductor-linux-amd64') '.\cmd\conductor'
    if ($LASTEXITCODE -ne 0) {
        throw "Linux amd64 build failed with exit code $LASTEXITCODE."
    }

    $checksumLines = foreach ($artifactName in @('conductor.exe', 'conductor-linux-amd64')) {
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $outputPath $artifactName)).Hash.ToLowerInvariant()
        "$hash  $artifactName"
    }
    [System.IO.File]::WriteAllLines((Join-Path $outputPath 'SHA256SUMS'), $checksumLines, [System.Text.UTF8Encoding]::new($false))

    $metadata = @(
        "git_commit=$gitCommit"
        "git_dirty=$gitDirty"
        "generated_utc=$([DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ'))"
        'targets=windows/amd64,linux/amd64'
        'cgo_enabled=0'
    )
    [System.IO.File]::WriteAllLines((Join-Path $outputPath 'build-metadata.txt'), $metadata, [System.Text.UTF8Encoding]::new($false))
} finally {
    foreach ($name in @('GOOS', 'GOARCH', 'CGO_ENABLED')) {
        [System.Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name], 'Process')
    }
}
