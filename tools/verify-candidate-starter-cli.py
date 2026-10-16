"""Compare public starter creation, exact file bytes, and existing-target preservation."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def snapshot(root):
    rows = []
    for path in sorted(root.rglob('*')):
        name = path.relative_to(root).as_posix()
        if path.is_symlink():
            target = Path(os.readlink(path))
            if not target.is_absolute():
                target = path.parent / target
            rows.append(dict(path=name, kind='link', target=os.path.relpath(target, root)))
        elif path.is_dir():
            rows.append(dict(path=name, kind='directory'))
        else:
            rows.append(dict(path=name, kind='file', sha256=sha(path), bytes=path.stat().st_size))
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--profile', action='append', choices=['development', 'release'])
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    build = json.loads(args.build_receipt.read_bytes())
    assert build['passed'] and not build['goFrontendInvoked'] and not build.get('commandsOnly')
    inputs = build['sources'] + [dict(path=Path(__file__).relative_to(ROOT).as_posix(), sha256=sha(Path(__file__)))]
    assert all(sha(ROOT / row['path']) == row['sha256'] for row in inputs)
    reference = args.reference.resolve()
    reference_sha = sha(reference)
    images = [row for row in build['images'] if not args.profile or row['profile'] in args.profile]
    assert images
    cases = [
        ('fresh', ['new', 'hello', '--uid', 'example.hello']),
        ('repeated', ['new', 'hello', '--uid', 'example.hello']),
        ('unicode-parent', ['new', 'hello', '--uid', 'example.hello']),
        ('existing-file', ['new', 'hello', '--uid', 'example.hello']),
        ('existing-directory', ['new', 'hello', '--uid', 'example.hello']),
        ('live-directory-link', ['new', 'hello', '--uid', 'example.hello']),
        ('dangling-directory-link', ['new', 'hello', '--uid', 'example.hello']),
        ('dangling-file-link', ['new', 'hello', '--uid', 'example.hello']),
        ('missing-options', ['new']),
        ('bad-name', ['new', '../escape', '--uid', 'example.hello']),
        ('reserved-name', ['new', 'CON', '--uid', 'example.hello']),
        ('bad-uid', ['new', 'hello', '--uid', '']),
        ('duplicate-uid', ['new', 'hello', '--uid', 'example.hello', '--uid', 'example.other']),
        ('help', ['new', '--help']),
    ]
    rows = []
    for selected in images:
        candidate = Path(selected['path'])
        assert sha(candidate) == selected['sha256']
        for name, argv in cases:
            observations = []
            for engine, image in [('go', reference), ('candidate', candidate)]:
                root = output / selected['profile'] / name / engine
                if name == 'unicode-parent':
                    root /= '프로젝트 한글'
                root.mkdir(parents=True)
                target = root / 'hello'
                if name == 'existing-file':
                    target.write_bytes(b'foreign-file')
                elif name == 'existing-directory':
                    target.mkdir()
                    (target / 'keep').write_bytes(b'foreign-file')
                elif name in ['live-directory-link', 'dangling-directory-link']:
                    (root / 'referent').mkdir()
                    target.symlink_to('referent', target_is_directory=True)
                    if name == 'dangling-directory-link':
                        (root / 'referent').rmdir()
                elif name == 'dangling-file-link':
                    target.symlink_to('missing', target_is_directory=False)
                if name == 'repeated':
                    first = subprocess.run([str(image), *argv], cwd=root, capture_output=True, timeout=30)
                    assert first.returncode == 0 and not first.stderr
                before = snapshot(root)
                result = subprocess.run([str(image), *argv], cwd=root, capture_output=True, timeout=30)
                after = snapshot(root)
                if name.startswith('existing-') or name.endswith('-link') or name == 'repeated':
                    assert before == after, (selected['profile'], name, engine, 'existing entry changed')
                    assert result.returncode != 0
                elif name in ['fresh', 'unicode-parent']:
                    assert result.returncode == 0 and not result.stderr
                    files = {row['path'] for row in after if row['kind'] == 'file'}
                    assert files == {'hello/.gitignore', 'hello/README.md', 'hello/fairy.toml',
                                     'hello/src/main.fy', 'hello/tests/main.fy'}
                elif name != 'help':
                    assert result.returncode != 0 and before == after
                observations.append(dict(exit=result.returncode, stdout=result.stdout.decode('utf-8'),
                                         stderr=result.stderr.decode('utf-8'), entries=after))
            exact = observations[0] == observations[1]
            rows.append(dict(profile=selected['profile'], case=name, argv=argv,
                             go=observations[0], candidate=observations[1], exact=exact))
            (output / 'observations.json').write_text(json.dumps(rows, indent=2)+'\n', encoding='utf-8')
            assert exact, rows[-1]
            print('passed:', selected['profile'], name, flush=True)
    assert sha(reference) == reference_sha
    assert all(sha(ROOT / row['path']) == row['sha256'] for row in inputs)
    assert all(sha(Path(row['path'])) == row['sha256'] for row in images)
    receipt = dict(schema='fairyc.candidate-starter-cli/v1', passed=True, sourceInputs=inputs,
                   reference=dict(path=str(reference), sha256=reference_sha), images=images,
                   observations=rows, sdkAbiChanged=False, ciChanged=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8')
    print('public starter CLI passed:', len(rows), 'exact comparisons', flush=True)


if __name__ == '__main__':
    main()
