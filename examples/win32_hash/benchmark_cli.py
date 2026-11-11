"""Measure whole-process CLI hash time on one owned, cached local-file fixture."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import time


def measure(image, baseline, mib, reverse=False):
    rows = []
    with tempfile.TemporaryDirectory(prefix='fairy-hash-benchmark-') as directory:
        source = Path(directory) / 'large-입력.bin'
        chunk, reference = bytes(range(256))*4096, hashlib.sha256()
        with source.open('wb') as stream:
            for _ in range(mib):
                stream.write(chunk)
                reference.update(chunk)
        expected = dict(algorithm='SHA-256', file=str(source), bytes=mib*1048576,
                        sha256=reference.hexdigest())
        images = ([('baseline', baseline)] if baseline else []) + [('current', image)]
        for label, executable in reversed(images) if reverse else images:
            started = time.perf_counter()
            result = subprocess.run([str(executable), '--hash', str(source)], capture_output=True, timeout=300)
            elapsed = time.perf_counter()-started
            if result.returncode or result.stderr or json.loads(result.stdout) != expected:
                raise RuntimeError(f'{label}: CLI reference mismatch, exit={result.returncode}')
            rows.append(dict(label=label, image=str(executable), imageBytes=executable.stat().st_size,
                             seconds=elapsed, MiBPerSecond=mib/elapsed,
                             sha256=reference.hexdigest()))
    return dict(ok=True, bytes=mib*1048576, measurements=rows, processExited=True,
                methodology='single-run cached local file; whole process including startup; no GUI observer')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe', type=Path, default=Path(__file__).parent/'build/win32-hash.exe')
    parser.add_argument('--baseline', type=Path)
    parser.add_argument('--reverse', action='store_true', help='measure current before baseline to check order effects')
    parser.add_argument('--mib', type=int, choices=range(16, 1025), metavar='16..1024', default=64)
    args = parser.parse_args()
    try:
        print(json.dumps(measure(args.exe.resolve(), args.baseline.resolve() if args.baseline else None,
                                 args.mib, args.reverse), ensure_ascii=True))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
