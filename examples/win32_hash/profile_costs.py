"""Attribute cached-file read, Fairy SHA and actual native UI costs in an owned build."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import time


def checked(args, cwd=None):
    result = subprocess.run([str(v) for v in args], cwd=cwd, capture_output=True, timeout=300)
    if result.returncode:
        raise RuntimeError(f'{args[0]} exited {result.returncode}: {result.stdout!r} {result.stderr!r}')
    return result


def profile(compiler, source, mib):
    example = Path(__file__).resolve().parent
    rows = []
    with tempfile.TemporaryDirectory(prefix='fairy-hash-profile-') as directory:
        root = Path(directory)
        for name in ('src', 'native'):
            shutil.copytree(source/name, root/name)
        for name in ('fairy.toml', 'fairy.windows.json'):
            shutil.copy2(source/name, root/name)
        shutil.copy2(example/'profiling/main.fy', root/'src/main.fy')
        shutil.copy2(example/'profiling/clock.c', root/'native/profile_clock.c')
        config = json.loads((root/'fairy.windows.json').read_text(encoding='utf8'))
        config['sources'].append('native/profile_clock.c')
        (root/'fairy.windows.json').write_text(json.dumps(config), encoding='utf8')
        checked([compiler, 'check'], root)
        checked([compiler, 'build', '--locked', '--release'], root)
        image = root/'build/win32-hash.exe'
        fixture, block, reference = root/'profile.bin', bytes(range(256))*4096, hashlib.sha256()
        with fixture.open('wb') as stream:
            for _ in range(mib):
                stream.write(block)
                reference.update(block)
        for mode in ('read', 'memory', 'cli', 'gui'):
            started = time.perf_counter()
            result = checked([image, mode, fixture])
            elapsed = time.perf_counter()-started
            if result.stderr:
                raise RuntimeError(f'{mode}: unexpected stderr {result.stderr!r}')
            row = json.loads(result.stdout)
            if not row.get('ok') or row['bytes'] != mib*1048576 or row['sha256'] != ('' if mode == 'read' else reference.hexdigest()):
                raise RuntimeError(f'{mode}: component reference mismatch: {row}')
            frequency = row['frequency']
            for field in ('total', 'read', 'hash', 'poll', 'progress', 'final'):
                row[field+'Seconds'] = row[field+'Ticks']/frequency
            measured = sum(row[field+'Ticks'] for field in ('read', 'hash', 'poll', 'progress', 'final'))
            if measured > row['totalTicks']:
                raise RuntimeError('overlapping component measurements')
            row.update(mode=mode, processSeconds=elapsed, otherSeconds=(row['totalTicks']-measured)/frequency,
                       MiBPerSecond=mib/row['totalSeconds'])
            rows.append(row)
    return dict(ok=True, verification='fairy-hash-component-costs', measurements=rows,
                compilerSHA256=hashlib.sha256(compiler.read_bytes()).hexdigest(),
                methodology='instrumented temporary release; cached repeating file; QPC wall intervals; '
                            'read-only, reused-memory SHA, file+SHA, file+SHA+actual HWND poll/100ms progress; '
                            'component totals exclude startup/dialogs/close; production primitives, separate entry point',
                processExited=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    parser.add_argument('--source', type=Path, default=Path(__file__).parent)
    parser.add_argument('--mib', type=int, choices=range(16, 257), default=64)
    args = parser.parse_args()
    try:
        print(json.dumps(profile(args.compiler.resolve(), args.source.resolve(), args.mib)))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
