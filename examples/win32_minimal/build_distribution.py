"""Build this example without its private CLI, smoke scenarios or diagnostic probes."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

from size_report import inspect


def build(compiler, output, development=False):
    root = Path(__file__).resolve().parent
    # Keep the ordinary workspace/config/image intact. Both builds use exactly
    # the same Fairy source and native implementation, with explicit wrappers.
    with tempfile.TemporaryDirectory(prefix='fairyc-win32-distribution-') as directory:
        stage = Path(directory)
        for name in ('fairy.toml', 'fairy.lock', 'fairy.windows.json'):
            shutil.copyfile(root / name, stage / name)
        for name in ('src', 'native'):
            shutil.copytree(root / name, stage / name)
        config = json.loads((stage / 'fairy.windows.json').read_text(encoding='utf-8-sig'))
        if config['sources'] != ['native/window.c', 'native/ime.c']:
            raise ValueError('distribution wrapper selection needs updating for the native source list')
        config['sources'] = ['native/window_distribution.c', 'native/ime_distribution.c']
        (stage / 'fairy.windows.json').write_text(json.dumps(config, indent=2) + '\n', encoding='utf-8')
        args = [compiler, 'build', '--locked'] + ([] if development else ['--release'])
        result = subprocess.run(args, cwd=stage, capture_output=True, text=True, timeout=120)
        if result.returncode:
            raise RuntimeError(f'compiler exited {result.returncode}: {result.stdout}{result.stderr}')
        image = stage / 'build/win32-minimal.exe'
        report = inspect(image)
        if report['subsystem'] != 2:
            raise RuntimeError('distribution image is not a GUI executable')
        # Publish only a successfully built/inspected image, even across drives.
        output.parent.mkdir(parents=True, exist_ok=True)
        pending = None
        try:
            with tempfile.NamedTemporaryFile(dir=output.parent, suffix='.tmp', delete=False) as stream:
                pending = Path(stream.name)
                stream.write(image.read_bytes())
            os.replace(pending, output)
        finally:
            if pending is not None:
                pending.unlink(missing_ok=True)
    report['image'] = str(output)
    return dict(ok=True, profile='development' if development else 'release', diagnostics=False,
                compiler=str(compiler), measurement=report)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', default='fairyc', help='Fairy CLI executable (Go or authored)')
    parser.add_argument('--output', type=Path, default=Path(__file__).parent / 'build/win32-minimal-distribution.exe')
    parser.add_argument('--development', action='store_true')
    args = parser.parse_args()
    compiler = shutil.which(args.compiler)
    if compiler is None:
        parser.error('Fairy CLI executable not found')
    try:
        print(json.dumps(build(str(Path(compiler).resolve()), args.output.resolve(), args.development), indent=2))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
