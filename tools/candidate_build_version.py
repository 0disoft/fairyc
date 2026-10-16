"""Freeze current source-build identity into a private candidate source module.

Use Go's own module/VCS version calculation (including tags and +dirty), rather
than the bootstrap seed's identity or a second implementation of that policy.
The Go executable is used only for --version, never for Fairy frontend work.
"""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
TEMPLATE = Path('examples/lexer_candidate/src/public_cli_data.fy')
VERSION = re.compile(r'pub fn ref str version\(\)\{return "(?:[^"\\]|\\.)*";\}')


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def render(template, text):
    lines = text.splitlines()
    if (len(lines) != 7 or lines[0] != 'Fairy C Stage 0' or
            any(not line.startswith(key + ': ') or not line[len(key) + 2:]
                for key, line in zip(('compiler', 'sdk', 'language', 'runtime', 'revision', 'modified'), lines[1:])) or
            lines[-1] not in ('modified: true', 'modified: false', 'modified: unknown') or
            not text.endswith('\n') or '\r' in text):
        raise ValueError('invalid source build version report')
    if len(VERSION.findall(template)) != 1:
        raise ValueError('expected exactly one candidate version function')
    return VERSION.sub(lambda _: 'pub fn ref str version(){return ' + json.dumps(text) + ';}', template)


def source_state(root=ROOT):
    # --no-optional-locks avoids refreshing the caller's index during inspection.
    def git(*args):
        result = subprocess.run(['git', '--no-optional-locks', *args], cwd=root,
                                capture_output=True, check=True, timeout=30)
        return result.stdout
    return (git('rev-parse', 'HEAD'), git('status', '--porcelain', '--untracked-files=normal'),
            git('for-each-ref', '--format=%(refname) %(objectname)', 'refs/tags'))


def generate(output, root=ROOT):
    """Build a fresh version oracle and replace only the module's version function."""
    root, output = root.resolve(), output.resolve()
    before = source_state(root)
    oracle = output / ('source-version.exe' if os.name == 'nt' else 'source-version')
    subprocess.run(['go', 'build', '-buildvcs=true', '-o', str(oracle), './cmd/fairyc'],
                   cwd=root, check=True, capture_output=True, timeout=180)
    result = subprocess.run([str(oracle), '--version'], cwd=output,
                            check=True, capture_output=True, timeout=30)
    if result.stderr:
        raise ValueError('source version oracle wrote stderr')
    text = result.stdout.decode('utf-8')
    template = root / TEMPLATE
    generated = output / 'public_cli_data.fy'
    generated.write_text(render(template.read_text(encoding='utf-8'), text), encoding='utf-8', newline='\n')
    if source_state(root) != before:
        raise ValueError('source build identity changed during version generation')
    metadata = dict(text=text, metadataOnlyGoInvoked=True, goFrontendInvoked=False,
                    oracle=dict(path=str(oracle), sha256=sha(oracle)),
                    template=dict(path=TEMPLATE.as_posix(), sha256=sha(template)),
                    generated=dict(path=generated.relative_to(root).as_posix(), sha256=sha(generated)))
    return generated, metadata


if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True,
                        help='ignored build directory inside the repository, normally .cache/...')
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    _, metadata = generate(args.output)
    print(json.dumps(metadata, indent=2))
