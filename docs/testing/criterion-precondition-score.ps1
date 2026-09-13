param(
    [Parameter(Mandatory = $true)]
    [string]$EvaluationPath
)

$ErrorActionPreference = 'Stop'
$result = Get-Content -LiteralPath $EvaluationPath -Raw | ConvertFrom-Json
$reviews = @($result.reviews)

function Get-Review([string]$packetID) {
    $matches = @($reviews | Where-Object { $_.packet_id -eq $packetID })
    if ($matches.Count -ne 1) {
        throw "Expected exactly one review for $packetID"
    }
    return $matches[0]
}

function Has-Citation($review, [string]$artifact) {
    return @($review.citations | Where-Object { $_.artifact -eq $artifact -and -not [string]::IsNullOrWhiteSpace($_.fact) }).Count -gt 0
}

$earlier = Get-Review 'read5-record-20260913-0224'
$later = Get-Review 'read5-record-20260913-0302'
$checks = [ordered]@{
    earlier_incomplete = $earlier.criterion_status -eq 'incomplete'
    earlier_scope = $earlier.evidence_scope -eq 'historical-decision'
    earlier_cites_validation = Has-Citation $earlier '.superpowers/case-study/coordination/remedy-validator/read5-validation.json'
    earlier_cites_results = Has-Citation $earlier '.superpowers/case-study/coordination/remedy-validator/read5-startup-output.txt'
    later_complete = $later.criterion_status -eq 'complete'
    later_scope = $later.evidence_scope -eq 'later-prospective-addendum'
    later_cites_ownership = Has-Citation $later '.superpowers/case-study/coordination/repeat-directory-ownership.json'
    later_cites_identity = Has-Citation $later '.superpowers/case-study/coordination/effectiveness-repeat/ownership-session-result.json'
    later_cites_results = Has-Citation $later '.superpowers/case-study/coordination/effectiveness-repeat/startup-result.json'
    later_cites_guide_verification = Has-Citation $later '.superpowers/case-study/coordination/final-validator/read5-prospective-artifact-check.json'
    history_preserved = $result.history.original_acceptance_rewritten -eq $false
    reasons_present = -not [string]::IsNullOrWhiteSpace($earlier.reason) -and -not [string]::IsNullOrWhiteSpace($later.reason) -and -not [string]::IsNullOrWhiteSpace($result.history.reason)
    limits_present = -not [string]::IsNullOrWhiteSpace($result.limits)
}

$failed = @($checks.GetEnumerator() | Where-Object { -not $_.Value } | ForEach-Object { $_.Key })
[ordered]@{ passed = $failed.Count -eq 0; checks = $checks; failed = $failed } | ConvertTo-Json -Depth 5
if ($failed.Count -ne 0) { exit 1 }
