"""Compare the real Fairy check process with the owning Go CLI on isolated inputs."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import time

def digest(data):
    return hashlib.sha256(data).hexdigest()


def snapshot(root):
    return {p.relative_to(root).as_posix(): dict(sha256=digest(p.read_bytes()), modifiedNS=p.stat().st_mtime_ns)
            for p in sorted(root.rglob('*')) if p.is_file()}


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8', newline='\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image', type=Path, required=True)
    parser.add_argument('--fixtures', type=Path, required=True, help='compiler-core oracle fixture directory')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--label', required=True)
    parser.add_argument('--reference', type=Path, help='observations.json from the reference label')
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    if not args.label.replace('-', '').isalnum():
        parser.error('label must be a portable single directory component')
    image = args.image.resolve()
    fixtures = args.fixtures.resolve()
    output = args.output.resolve() / args.label
    output.mkdir(parents=True, exist_ok=False)
    specs = [(name, name, ['check', '--locked'], '', '', '', True)
             for name in ['Current', 'Signature', 'RootBody', 'DependencyBody', 'Private', 'Nominal', 'Moved', 'SDK']]
    specs += [
        ('Unchanged', 'Current', ['check'], '', '', '', True),
        ('Updated', 'Current', ['check'], 'delete-lock', '', '', True),
        ('Recanonicalized', 'Current', ['check'], 'noncanonical-lock', '', '', True),
        ('SemanticUpdated', 'RootBody', ['check'], 'delete-lock', '', '', True),
        ('Lexical', 'Current', ['check'], 'source', 'src/main.fy', '\n`\n', True),
        ('Syntax', 'Current', ['check'], 'source', 'src/main.fy', '\nfn i32 broken(){return 1}\n', True),
        ('MissingLock', 'Current', ['check', '--locked'], 'delete-lock', '', '', False),
        ('StaleLock', 'Current', ['check', '--locked'], 'source', 'src/main.fy', '\nfn i32 added(){return 1;}\n', True),
        ('InvalidLock', 'Current', ['check', '--locked'], 'invalid-lock', '', '', False),
        ('MissingManifest', '', ['check'], '', '', '', False),
        ('UnknownFlag', '', ['check', '--bad'], '', '', '', True),
        ('RepeatedLocked', '', ['check', '--locked', '--locked'], '', '', '', True),
        ('Positional', '', ['check', 'src/main.fy'], '', '', '', True),
    ]
    expected = {} if args.reference is None else {r['name']: r for r in json.loads(args.reference.read_text(encoding='utf-8'))['cases']}
    rows = []
    for name, base, argv, change, file, text, exact in specs:
        cwd = output / name
        if base:
            shutil.copytree(fixtures / base, cwd)
        else:
            cwd.mkdir()
        lock = cwd / 'fairy.lock'
        if change == 'delete-lock':
            lock.unlink()
        elif change == 'noncanonical-lock':
            lock.write_bytes(lock.read_bytes() + b'\n')
        elif change == 'invalid-lock':
            lock.write_bytes(b'{bad lock}\n')
        elif change == 'source':
            with (cwd / file).open('ab') as f:
                f.write(text.encode('utf-8'))
        before = snapshot(cwd)
        input_digest = digest(json.dumps({p: v['sha256'] for p, v in before.items()}, sort_keys=True).encode('utf-8'))
        started = time.monotonic()
        result = subprocess.run([str(image), *argv], cwd=cwd, capture_output=True, timeout=90)
        after = snapshot(cwd)
        changed = sorted(p for p in before.keys() | after.keys() if before.get(p) != after.get(p))
        assert all(p == 'fairy.lock' for p in changed), (name, changed)
        if '--locked' in argv or name in ['Lexical', 'Syntax', 'MissingManifest', 'UnknownFlag', 'RepeatedLocked', 'Positional']:
            assert not changed, (name, 'read-only command mutated input', changed)
        comparison = 'complete' if exact else 'package-prefix'
        if name in ['UnknownFlag', 'RepeatedLocked', 'Positional']:
            comparison = 'invocation-first-line'
        row = dict(name=name, argv=argv, exit=result.returncode, stdout=result.stdout.decode('utf-8'),
                   stderr=result.stderr.decode('utf-8'), seconds=time.monotonic()-started,
                   diagnosticComparison=comparison, inputSHA256=input_digest, changedPaths=changed,
                   lockSHA256=digest(lock.read_bytes()) if lock.exists() else None)
        if expected:
            oracle = expected[name]
            diagnostic_match = (row['stderr'] == oracle['stderr'] if comparison == 'complete'
                                else row['stderr'].splitlines()[:1] == oracle['stderr'].splitlines()[:1] if comparison == 'invocation-first-line'
                                else row['stderr'].startswith('fairyc: package workspace failure: '))
            row['matched'] = (row['exit'] == oracle['exit'] and row['stdout'] == oracle['stdout']
                              and row['inputSHA256'] == oracle['inputSHA256']
                              and row['lockSHA256'] == oracle['lockSHA256']
                              and diagnostic_match)
        rows.append(row)
        save(output / 'observations.json', dict(schema='fairyc.candidate-check-process/v1', image=dict(path=str(image), sha256=digest(image.read_bytes())), cases=rows))
        print(name, row['exit'], 'matched=' + str(row.get('matched', 'reference')), flush=True)
    if expected:
        mismatches = [r for r in rows if not r['matched']]
        assert not mismatches, json.dumps(mismatches, indent=2)
    print('check CLI:', len(rows), 'cases completed', flush=True)


if __name__ == '__main__':
    main()
