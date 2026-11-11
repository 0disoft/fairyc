"""Compare the actual Fairy hash/save pipeline with Python's SHA-256 reference."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile


def verify(image):
    rows = []
    with tempfile.TemporaryDirectory(prefix='fairy-hash-한글-') as directory:
        root = Path(directory)
        def run(source, target=None, error=None):
            args = [str(image), '--hash', str(source)] + ([str(target)] if target else [])
            result = subprocess.run(args, capture_output=True, timeout=30)
            if result.stderr:
                raise RuntimeError('unexpected stderr: ' + repr(result.stderr))
            reply = json.loads(result.stdout)
            if error is not None:
                if result.returncode != 1 or reply != dict(ok=False, errorCode=error):
                    raise RuntimeError('failure policy mismatch: ' + repr((result.returncode, reply)))
            else:
                data = source.read_bytes()
                expected = dict(algorithm='SHA-256', file=str(source), bytes=len(data),
                                sha256=hashlib.sha256(data).hexdigest())
                if result.returncode or reply != expected:
                    raise RuntimeError('SHA-256 reference mismatch: ' + repr((result.returncode, reply, expected)))
                if target and (json.loads(target.read_bytes()) != expected or target.read_bytes() != result.stdout):
                    raise RuntimeError('saved UTF-8 report differs from actual Fairy output')
            rows.append(dict(file=source.name, saved=target is not None, error=error))
        for size in (0, 3, 55, 56, 64, 129, 1048833):
            source = root / f'입력-Ω-{size}.bin'
            source.write_bytes(b'abc' if size == 3 else bytes(range(256)) * (size // 256) + bytes(range(size % 256)))
            target = root / f'결과-{size}.json'
            run(source, target)
            target.write_text('previous report', encoding='utf-8')
            run(source, target)  # Prepared publication also replaces an existing report.
        source = root / '원본.bin'
        source.write_bytes(b'source must survive')
        before = source.read_bytes()
        run(source, source, error=7)
        alias = root / '하드링크.bin'
        os.link(source, alias)
        run(source, alias, error=7)
        if source.read_bytes() != before or alias.read_bytes() != before:
            raise RuntimeError('source/alias protection damaged input')
        run(root / 'missing.bin', error=2)
        target = root / 'absent-parent/result.json'
        run(source, target, error=6)
        # An unrelated preexisting temporary name must never be truncated.
        target = root / '충돌.json'
        collision = Path(str(target) + '.fairy-hash-0.tmp')
        collision.write_bytes(b'unrelated temporary content')
        run(source, target)
        if collision.read_bytes() != b'unrelated temporary content':
            raise RuntimeError('exclusive temporary creation was bypassed')
        collision.unlink()
        if list(root.glob('*.tmp')):
            raise RuntimeError('owned temporary file leaked')
    return dict(ok=True, verification='fairy-file-sha256', cases=len(rows), observations=rows, processExited=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe', type=Path, default=Path(__file__).parent / 'build/win32-hash.exe')
    args = parser.parse_args()
    try:
        print(json.dumps(verify(args.exe.resolve()), ensure_ascii=True))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
