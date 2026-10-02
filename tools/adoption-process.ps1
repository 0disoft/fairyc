# Shared ordinary execution boundary; controlled KV checkpoints keep their own driver.
function Invoke-AdoptionProcess {
    param(
        [string]$Exe, [AllowEmptyCollection()][object[]]$Argv, [string]$Cwd,
        [string]$Kind, [string]$RawPath,
        [ValidateSet('nested','flat')][string]$Layout = 'nested',
        [ValidateRange(1,60000)][int]$TimeoutMs = 60000
    )
    $utf8 = [Text.UTF8Encoding]::new($false, $true)
    $start = [Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $Exe; $start.WorkingDirectory = $Cwd
    $start.UseShellExecute = $false; $start.CreateNoWindow = $true
    $start.RedirectStandardOutput = $true; $start.RedirectStandardError = $true
    $start.RedirectStandardInput = $true
    $start.StandardOutputEncoding = $utf8; $start.StandardErrorEncoding = $utf8
    foreach ($argument in $Argv) {
        if ($null -eq $argument -or $argument -isnot [string]) { throw 'Arguments must be non-null strings' }
        $start.ArgumentList.Add($argument)
    }
    $process = [Diagnostics.Process]::new(); $process.StartInfo = $start
    $started = $false
    try {
        $started = $process.Start()
        if (!$started) { throw 'Owned child did not start' }
        $process.StandardInput.Close()
        $stdout = $process.StandardOutput.ReadToEndAsync(); $stderr = $process.StandardError.ReadToEndAsync()
        if (!$process.WaitForExit($TimeoutMs)) { throw "Owned child exceeded $TimeoutMs milliseconds" }
        if (![Threading.Tasks.Task]::WaitAll([Threading.Tasks.Task[]]@($stdout,$stderr), 10000)) { throw 'Owned child output drain exceeded 10 seconds' }
        $actual = [pscustomobject]@{exitCode=$process.ExitCode; stdout=$stdout.GetAwaiter().GetResult(); stderr=$stderr.GetAwaiter().GetResult()}
        $raw = @{kind=$Kind; executable=$Exe; argv=@($Argv); cwd=$Cwd}
        if ($Layout -eq 'nested') { $raw.actual=$actual }
        else { $raw.exitCode=$actual.exitCode; $raw.stdout=$actual.stdout; $raw.stderr=$actual.stderr }
        [IO.File]::AppendAllText($RawPath, ($raw | ConvertTo-Json -Depth 6 -Compress) + "`n", $utf8)
        return $actual
    } finally {
        try {
            if ($started -and !$process.HasExited) {
                $process.Kill($true)
                if (!$process.WaitForExit(10000)) { throw 'Owned child did not exit after termination' }
            }
        } finally { $process.Dispose() }
    }
}
