"""Differential SHA checks for padding, read boundaries, carries and nonrepeating data."""
import argparse
import hashlib
import json
from pathlib import Path
import random
import subprocess
import tempfile


def verify(image):
    rng = random.Random(0xfa17)
    payloads = [('abc', b'abc'), ('million-a', b'a'*1000000),
                ('zero-read-boundary', bytes(65537)), ('ff-read-boundary', b'\xff'*65537),
                ('random-large', rng.randbytes(1048833))]
    for length in (0, 1, 2, 7, 31, 55, 56, 57, 63, 64, 65, 119, 120, 127, 128, 129, 65535, 65536, 65537):
        payloads.append((f'random-{length}', rng.randbytes(length)))
    with tempfile.TemporaryDirectory(prefix='fairy-sha-vectors-') as directory:
        for label, payload in payloads:
            source = Path(directory)/f'검증-{label}.bin'
            source.write_bytes(payload)
            result = subprocess.run([str(image), '--hash', str(source)], capture_output=True, timeout=30)
            expected = dict(algorithm='SHA-256', file=str(source), bytes=len(payload),
                            sha256=hashlib.sha256(payload).hexdigest())
            if result.returncode or result.stderr or json.loads(result.stdout) != expected:
                raise RuntimeError(f'SHA reference mismatch: {label}, exit={result.returncode}, stderr={result.stderr!r}')
    return dict(ok=True, verification='fairy-sha256-vectors', cases=len(payloads),
                imageSHA256=hashlib.sha256(image.read_bytes()).hexdigest(), processExited=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe', type=Path, default=Path(__file__).parent/'build/win32-hash.exe')
    args = parser.parse_args()
    try:
        print(json.dumps(verify(args.exe.resolve())))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
