# Verifies companion recorded checksums against Yeetcraft canonical artifacts.
# Does not copy schemas. No Go/npm runtime dependency.
$ErrorActionPreference = "Stop"
$CompanionRoot = Split-Path -Parent $PSScriptRoot
$Recorded = Join-Path $CompanionRoot "testdata\contract\v1\CANONICAL_CHECKSUMS.sha256"
$YeetcraftRoot = Resolve-Path (Join-Path $CompanionRoot "..\Yeetcraft") -ErrorAction SilentlyContinue
if (-not $YeetcraftRoot) {
    $YeetcraftRoot = Resolve-Path (Join-Path $CompanionRoot "..\yeetcraft") -ErrorAction SilentlyContinue
}

Write-Host "Recorded checksums: $Recorded"
if (-not (Test-Path $Recorded)) {
    Write-Error "Missing testdata/contract/v1/CANONICAL_CHECKSUMS.sha256"
}

$contractCopies = @(
    Get-ChildItem -Path $CompanionRoot -Recurse -Filter "ingest-batch-*.schema.json" -ErrorAction SilentlyContinue
    Get-ChildItem -Path $CompanionRoot -Recurse -Filter "error.schema.json" -ErrorAction SilentlyContinue
) | Where-Object { $_.FullName -notmatch '\\testdata\\contract\\v1\\derived\\' }

if ($contractCopies.Count -gt 0) {
    Write-Error "Forbidden canonical schema copy in companion repo: $($contractCopies.FullName -join ', ')"
}

$lines = Get-Content -LiteralPath $Recorded | Where-Object { $_ -and $_ -notmatch '^#' }
if ($lines.Count -lt 1) {
    Write-Error "Recorded checksum file has no digest lines"
}

if (-not $YeetcraftRoot) {
    Write-Host "Sibling Yeetcraft checkout not found. Recorded checksums kept; live compare deferred to Phase 2/3 CI."
    Write-Host "verify-canonical-checksums: recorded file present, no schema fork detected."
    exit 0
}

$canonicalChecksums = Join-Path $YeetcraftRoot "contracts\companion\v1\CHECKSUMS.sha256"
if (-not (Test-Path $canonicalChecksums)) {
    Write-Error "Sibling Yeetcraft is missing contracts/companion/v1/CHECKSUMS.sha256"
}

$recordedText = [IO.File]::ReadAllText($Recorded).Replace("`r`n", "`n")
$canonicalText = [IO.File]::ReadAllText($canonicalChecksums).Replace("`r`n", "`n")
if ($recordedText -ne $canonicalText) {
    Write-Error "CANONICAL_CHECKSUMS.sha256 differs from Yeetcraft CHECKSUMS.sha256. Copy the canonical file after Validate."
}

$v1 = Join-Path $YeetcraftRoot "contracts\companion\v1"
foreach ($line in $lines) {
    if ($line -notmatch '^([0-9a-f]{64})  (.+)$') {
        Write-Error "Malformed checksum line: $line"
    }
    $expected = $Matches[1]
    $rel = $Matches[2] -replace '/', '\'
    $path = Join-Path $v1 $rel
    if (-not (Test-Path $path)) {
        Write-Error "Canonical artifact missing: $rel"
    }
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $path).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        Write-Error "Checksum drift for $rel (recorded $expected, disk $actual)"
    }
}

function Get-HeadingSlugs([string]$MarkdownPath) {
    $counts = @{}
    $slugs = New-Object "System.Collections.Generic.HashSet[string]"
    Get-Content -LiteralPath $MarkdownPath | ForEach-Object {
        if ($_ -notmatch '^(#{1,6})\s+(.+)$') { return }
        $text = $Matches[2].Trim()
        $text = $text -replace '`', ''
        $text = $text -replace '\*\*', ''
        $text = $text.ToLowerInvariant()
        $text = [regex]::Replace($text, '[^\p{L}\p{N} \-]', '')
        $text = [regex]::Replace($text, '\s+', '-')
        $text = $text.Trim('-')
        if (-not $counts.ContainsKey($text)) { $counts[$text] = 0 }
        else {
            $counts[$text]++
            $text = "$text-$($counts[$text])"
        }
        [void]$slugs.Add($text)
    }
    return $slugs
}

function Test-RelativeMarkdownLinks([string]$MarkdownPath) {
    $dir = Split-Path -Parent $MarkdownPath
    $content = Get-Content -Raw -LiteralPath $MarkdownPath
    foreach ($match in [regex]::Matches($content, '\[(?:[^\]]+)\]\(([^)]+)\)')) {
        $href = $match.Groups[1].Value.Trim()
        if ($href -match '^(https?:|mailto:|#)') { continue }
        $pathPart = $href
        $frag = $null
        if ($href.Contains("#")) {
            $split = $href.Split("#", 2)
            $pathPart = $split[0]
            $frag = $split[1]
        }
        if ([string]::IsNullOrWhiteSpace($pathPart)) { continue }
        $resolved = [IO.Path]::GetFullPath((Join-Path $dir ($pathPart -replace '/', [IO.Path]::DirectorySeparatorChar)))
        $candidates = @(
            $resolved,
            ($resolved -replace '(?i)\\yeetcraft\\', '\Yeetcraft\'),
            ($resolved -replace '(?i)\\Yeetcraft\\', '\yeetcraft\')
        ) | Select-Object -Unique
        $hit = $candidates | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
        if (-not $hit) {
            Write-Error "Broken relative link in $MarkdownPath : $href"
        }
        if ($frag -and $hit.ToLowerInvariant().EndsWith(".md")) {
            $slugs = Get-HeadingSlugs $hit
            if (-not $slugs.Contains($frag)) {
                Write-Error "Missing heading fragment in $MarkdownPath : $href"
            }
        }
    }
}

Write-Host "Relative links (Validate docs + file maps)..."
@(
    (Join-Path $CompanionRoot "docs\CONTRACT_V1_DERIVED_FIXTURES.md"),
    (Join-Path $CompanionRoot "docs\CONTRACT_V1_WP1_REVIEW.md"),
    (Join-Path $CompanionRoot "docs\PHASE_2_FILE_MAP.md"),
    (Join-Path $CompanionRoot "docs\YEETCRAFT_INTEGRATION.md"),
    (Join-Path $CompanionRoot "testdata\contract\v1\README.md")
) | ForEach-Object {
    Write-Host "  links: $_"
    Test-RelativeMarkdownLinks $_
}

Write-Host "verify-canonical-checksums passed against sibling Yeetcraft ($($lines.Count) artifacts)."
Write-Host "Deferred CI: companion go-test derived copies; GitHub Actions for this script."
