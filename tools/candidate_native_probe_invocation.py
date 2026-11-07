"""Private native-probe invocation logging, including diagnostic failure output."""
import hashlib
import json
import subprocess
import time


def _output_bytes(value):
    return value.encode('utf-8') if isinstance(value, str) else value or b''


def run_invocation(argv, timeout, *, cwd, output, calls, runner=None, clock=None):
    runner = subprocess.run if runner is None else runner
    clock = time.monotonic if clock is None else clock
    argv = list(map(str, argv))

    def record(stdout, stderr, returncode, seconds, error=None):
        row = dict(argv=argv, seconds=seconds, exit=returncode,
                   stdoutBytes=len(stdout), stdoutSHA256=hashlib.sha256(stdout).hexdigest(),
                   stderr=stderr.decode('utf-8', errors='replace'))
        if error is not None:
            row['error'] = dict(type=type(error).__name__, message=str(error))
            if isinstance(error, subprocess.TimeoutExpired):
                row['timedOut'] = True
                row['timeoutSeconds'] = error.timeout
        calls.append(row)
        if error is not None or returncode != 0 or stderr:
            # Raw files preserve bytes that cannot be represented in the JSON log.
            prefix = f'invocation-{len(calls):03d}'
            for stream, data in (('stdout', stdout), ('stderr', stderr)):
                name = f'{prefix}.{stream}.bin'
                row[f'{stream}Path'] = name
                (output / name).write_bytes(data)
        (output / 'invocations.json').write_text(
            json.dumps(calls, indent=2) + '\n', encoding='utf-8', newline='\n')

    started = clock()
    try:
        result = runner(argv, cwd=cwd, capture_output=True, timeout=timeout)
    except Exception as error:
        try:
            record(_output_bytes(getattr(error, 'stdout', None)),
                   _output_bytes(getattr(error, 'stderr', None)),
                   getattr(error, 'returncode', None), clock() - started, error)
        except Exception as diagnostic_error:
            # A diagnostic write failure must not replace the subprocess exception.
            if hasattr(error, 'add_note'):
                error.add_note(f'Invocation diagnostics could not be saved: {diagnostic_error}')
        raise
    record(result.stdout, result.stderr, result.returncode, clock() - started)
    assert result.returncode == 0 and not result.stderr, calls[-1]
    return result.stdout
