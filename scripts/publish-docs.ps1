# Publish through the organization-owned App, without persisting an access token.
[CmdletBinding()]
param(
    [string]$PublisherPath = (Join-Path (Split-Path $PSScriptRoot -Parent) '..\.publisher\publisher.ps1')
)
$ErrorActionPreference = 'Stop'
$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$publisher = [IO.Path]::GetFullPath($PublisherPath)
if (-not (Test-Path -LiteralPath $publisher -PathType Leaf)) {
    throw 'Provide -PublisherPath pointing to the KanataLabs Publisher utility.'
}
function Invoke-GitChecked {
    param([string[]]$Arguments)
    $output = & git @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Git operation failed: $($Arguments[0])" }
    $output
}
$remote = Invoke-GitChecked -Arguments @('-C', $repoRoot, 'remote', 'get-url', 'origin')
if ($remote -ne 'https://github.com/KanataLabs/fleetsh.git') {
    throw 'Only the public KanataLabs/fleetsh origin is supported.'
}
$status = Invoke-GitChecked -Arguments @('-C', $repoRoot, 'status', '--porcelain')
if ($status) { throw 'Commit or set aside local changes before publishing.' }
$sourceCommit = Invoke-GitChecked -Arguments @('-C', $repoRoot, 'rev-parse', 'HEAD')
$branch = Invoke-GitChecked -Arguments @('-C', $repoRoot, 'branch', '--show-current')
if ($branch -ne 'main') { throw 'Publish from the main branch.' }

$staging = Join-Path ([IO.Path]::GetTempPath()) ('fleetsh-docs-' + [Guid]::NewGuid().ToString('N'))
$helperPath = $publisher.Replace('\', '/')
$helper = '!pwsh -NoProfile -File "' + $helperPath + '" credential'
$gitAuth = @('-c', 'credential.helper=', '-c', ('credential.helper=' + $helper), '-c', 'credential.useHttpPath=true')
try {
    Invoke-GitChecked -Arguments ($gitAuth + @('clone', '--single-branch', '--branch', 'gh-pages', $remote, $staging)) | Out-Null
    Invoke-GitChecked -Arguments @('-C', $staging, 'rm', '-r', '--ignore-unmatch', '.') | Out-Null
    $trackedDocs = Invoke-GitChecked -Arguments @('-C', $repoRoot, 'ls-files', 'docs')
    foreach ($relative in $trackedDocs) {
        if (-not $relative.StartsWith('docs/')) { throw 'Unexpected source path.' }
        $destination = [IO.Path]::GetFullPath((Join-Path $staging $relative.Substring(5)))
        $destinationRoot = [IO.Path]::GetFullPath($staging) + [IO.Path]::DirectorySeparatorChar
        if (-not $destination.StartsWith($destinationRoot, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Refusing a destination outside the publication checkout.'
        }
        [IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($destination)) | Out-Null
        Copy-Item -LiteralPath (Join-Path $repoRoot $relative) -Destination $destination -Force
    }
    Invoke-GitChecked -Arguments @('-C', $staging, 'add', '--all') | Out-Null
    & git -C $staging diff --cached --quiet
    if ($LASTEXITCODE -eq 0) {
        Write-Output 'Documentation is already synchronized.'
        return
    }
    if ($LASTEXITCODE -ne 1) { throw 'Unable to compare documentation changes.' }
    Invoke-GitChecked -Arguments @('-C', $staging, 'config', 'user.name', 'Kanata Labs') | Out-Null
    Invoke-GitChecked -Arguments @('-C', $staging, 'config', 'user.email', 'contact@kanatalabs.com') | Out-Null
    Invoke-GitChecked -Arguments @('-C', $staging, 'commit', '-m', "docs: publish from $sourceCommit") | Out-Null
    Invoke-GitChecked -Arguments ($gitAuth + @('-C', $staging, 'push', 'origin', 'HEAD:gh-pages')) | Out-Null
    Write-Output "Published documentation from $sourceCommit with the KanataLabs Publisher App."
} finally {
    # Delete only the exact temporary checkout created by this invocation.
    $tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\', '/') + [IO.Path]::DirectorySeparatorChar
    $resolvedStaging = [IO.Path]::GetFullPath($staging)
    if (-not $resolvedStaging.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase) -or
        [IO.Path]::GetFileName($resolvedStaging) -notmatch '^fleetsh-docs-[a-f0-9]{32}$') {
        throw 'Refusing cleanup outside the verified temporary checkout.'
    }
    if (Test-Path -LiteralPath $resolvedStaging) {
        Remove-Item -LiteralPath $resolvedStaging -Recurse -Force
    }
}
