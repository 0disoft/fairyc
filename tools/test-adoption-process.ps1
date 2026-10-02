#requires -Version 7.0
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'adoption-process.ps1')
. (Join-Path $PSScriptRoot 'adoption-output-audit.ps1')
$root = Join-Path $PSScriptRoot ('../.cache/adoption-process-tests/'+[Guid]::NewGuid().ToString('N')+'/space 한')
$root = [IO.Path]::GetFullPath($root)
[void][IO.Directory]::CreateDirectory($root)
$child = Join-Path $root 'child.ps1'
$utf8 = [Text.UTF8Encoding]::new($false, $true)
$source = @'
param([Parameter(ValueFromRemainingArguments=$true)][string[]]$Values)
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
if ($Values[0] -eq 'timeout') {
    [IO.File]::WriteAllText($Values[1], $PID.ToString())
    Start-Sleep -Seconds 30
    exit 0
}
[Console]::Out.Write(($Values | ConvertTo-Json -Compress) + ('x'*70000))
[Console]::Error.Write("한글`n" + ('y'*70000))
exit 7
'@
[IO.File]::WriteAllText($child, $source+"`n", $utf8)
$exe = (Get-Process -Id $PID).Path
$values = @('', 'a b', '"quoted"', '한글', 'a;$()&|', ('e'+[char]0x301))
$argv = @('-NoLogo','-NoProfile','-File',$child) + $values
$expected = @{exitCode=7; stdout=($values | ConvertTo-Json -Compress)+('x'*70000); stderr="한글`n"+('y'*70000)}
foreach ($layout in @('nested','flat')) {
    $rawPath = Join-Path $root ($layout+'.jsonl')
    $actual = Invoke-AdoptionProcess -Exe $exe -Argv $argv -Cwd $root -Kind program -RawPath $rawPath -Layout $layout
    Assert-AdoptionObservation $actual $expected $layout
    $records = @([IO.File]::ReadAllLines($rawPath, $utf8) | ForEach-Object { $_ | ConvertFrom-Json })
    if ($records.Count -ne 1) { throw 'Raw inventory mismatch' }
    $raw = $records[0]
    $recorded = if ($layout -eq 'nested') { $raw.actual } else { $raw }
    Assert-AdoptionObservation $recorded $expected ('raw/'+$layout)
    if ($raw.kind -cne 'program' -or ![StringComparer]::Ordinal.Equals($raw.cwd,$root) -or
        ![StringComparer]::Ordinal.Equals($raw.executable,$exe) -or $raw.argv.Count -ne $argv.Count) { throw 'Raw boundary mismatch' }
    for ($i=0; $i -lt $argv.Count; $i++) { if (![StringComparer]::Ordinal.Equals($raw.argv[$i],$argv[$i])) { throw 'Raw argv mismatch' } }
    if (($layout -eq 'nested' -and $raw.PSObject.Properties.Name -contains 'exitCode') -or
        ($layout -eq 'flat' -and $raw.PSObject.Properties.Name -contains 'actual')) { throw 'Legacy raw layout changed' }
}
foreach ($bad in @(@{argv=@($null)}, @{argv=@(123)})) {
    $caught=$false
    try { Invoke-AdoptionProcess -Exe 'must-not-start' -Argv $bad.argv -Cwd $root -Kind program -RawPath (Join-Path $root 'bad.jsonl') }
    catch { if ($_.Exception.Message -cne 'Arguments must be non-null strings') { throw }; $caught=$true }
    if (!$caught -or (Test-Path -LiteralPath (Join-Path $root 'bad.jsonl'))) { throw 'Malformed argv was accepted' }
}
$pidPath=Join-Path $root 'owned.pid'; $timeoutLog=Join-Path $root 'timeout.jsonl'; $caught=$false
try { Invoke-AdoptionProcess -Exe $exe -Argv @('-NoLogo','-NoProfile','-File',$child,'timeout',$pidPath) -Cwd $root -Kind program -RawPath $timeoutLog -TimeoutMs 5000 }
catch { if ($_.Exception.Message -cne 'Owned child exceeded 5000 milliseconds') { throw }; $caught=$true }
if (!$caught -or (Test-Path -LiteralPath $timeoutLog)) { throw 'Timeout produced a success record' }
$ownedPid=[int][IO.File]::ReadAllText($pidPath)
try { $survivor=[Diagnostics.Process]::GetProcessById($ownedPid) }
catch [ArgumentException] { $survivor=$null }
if ($survivor) { try { if (!$survivor.HasExited) { throw 'Timed-out owned child survived' } } finally { $survivor.Dispose() } }
Write-Output 'PASS 5 process-boundary cases; 3 PowerShell fixture children, zero Fairy/compiler executions'
