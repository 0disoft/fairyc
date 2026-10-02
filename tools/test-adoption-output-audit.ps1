#requires -Version 7.0
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'adoption-output-audit.ps1')
$composed = 'é'; $decomposed = 'e' + [char]0x301
if (!($composed -ceq $decomposed) -or [StringComparer]::Ordinal.Equals($composed, $decomposed)) { throw 'Expected cultural/ordinal distinction' }
$tests = @(
    @{name='empty'; actual=@{exitCode=0;stdout='';stderr=''}; expected=@{exitCode=0;stdout='';stderr=''}; pass=$true},
    @{name='exact-unicode-and-nul'; actual=@{exitCode=0;stdout=('한'+[char]0+$decomposed);stderr=$composed}; expected=@{exitCode=0;stdout=('한'+[char]0+$decomposed);stderr=$composed}; pass=$true},
    @{name='stdout-composed'; actual=@{exitCode=0;stdout=$composed;stderr=''}; expected=@{exitCode=0;stdout=$decomposed;stderr=''}; pass=$false},
    @{name='stdout-decomposed'; actual=@{exitCode=0;stdout=$decomposed;stderr=''}; expected=@{exitCode=0;stdout=$composed;stderr=''}; pass=$false},
    @{name='stderr-composed'; actual=@{exitCode=1;stdout='';stderr=$composed}; expected=@{exitCode=1;stdout='';stderr=$decomposed}; pass=$false},
    @{name='case'; actual=@{exitCode=0;stdout='A';stderr=''}; expected=@{exitCode=0;stdout='a';stderr=''}; pass=$false},
    @{name='newline'; actual=@{exitCode=0;stdout="a`r`n";stderr=''}; expected=@{exitCode=0;stdout="a`n";stderr=''}; pass=$false},
    @{name='exit'; actual=@{exitCode=1;stdout='';stderr=''}; expected=@{exitCode=0;stdout='';stderr=''}; pass=$false},
    @{name='null-output'; actual=@{exitCode=0;stdout=$null;stderr=''}; expected=@{exitCode=0;stdout='';stderr=''}; pass=$false},
    @{name='missing-exit'; actual=@{stdout='';stderr=''}; expected=@{exitCode=0;stdout='';stderr=''}; pass=$false}
)
foreach ($test in $tests) {
    $caught = $false
    try { Assert-AdoptionOutputs @($test) }
    catch { if (!$_.Exception.Message.StartsWith('Ordinal output mismatch:', [StringComparison]::Ordinal) -and !$_.Exception.Message.StartsWith('Malformed output observation:', [StringComparison]::Ordinal)) { throw }; $caught=$true }
    if ($caught -eq $test.pass) { throw "Unexpected assertion result: $($test.name)" }
}
Write-Output 'PASS 10 output-gate cases, including cultural Unicode aliases; no native executions'
