"""Execute authored private Fairy tests with the Go frontend and candidate C."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
PROJECT = ROOT / 'examples/lexer_candidate'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8', newline='\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--builder', type=Path, required=True)
    parser.add_argument('--module', required=True)
    parser.add_argument('--test', action='append', required=True)
    parser.add_argument('--selected-tests-only', action='store_true', help='verify only explicitly selected authored tests instead of requiring the entire module')
    parser.add_argument('--frontend-timeout', type=int, default=60, help='verification-only candidate frontend allowance, 1..900 seconds')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--fixture', type=Path, help='hash and forward one binary oracle fixture to every execution')
    parser.add_argument('--candidate-only', action='store_true', help='execute private host-ABI tests only through candidate C; Go native cannot link these private adapters')
    parser.add_argument('--unchanged-paths', type=Path, help='JSON list of repository-relative paths whose bytes and modification times must remain unchanged')
    args = parser.parse_args()
    if not 1 <= args.frontend_timeout <= 900:
        parser.error('frontend timeout must be in 1..900 seconds')
    if not __debug__:
        parser.error('verification requires Python without -O')
    if not re.fullmatch(r'[a-z][a-z0-9_]*', args.module):
        parser.error('module must be a private module leaf')
    if len(set(args.test)) != len(args.test) or any(not re.fullmatch(r'[a-z][a-z0-9_]*', name) for name in args.test):
        parser.error('test names must be unique identifiers')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    test_source = PROJECT / f'tests/{args.module}.fy'
    authored = test_source.read_text(encoding='utf-8')
    test_names = re.findall(r'@test\s+(?:@effects\([^\n]+\)\s+)?fn void (\w+)\(', authored)
    if not set(args.test).issubset(test_names):
        parser.error('selected test is not authored in the module')
    if not args.selected_tests_only and set(test_names) != set(args.test):
        parser.error('select all authored tests in the module')
    original_module = f'module tests::{args.module};'
    assert authored.startswith(original_module)
    authored = re.sub(r'(?m)^(\s*)@test\b\s*', r'\1', authored.replace(original_module, 'module main;', 1))
    authored += '\n@effects(filesystem,console) fn i32 main(){' + ''.join(name + '();' for name in args.test) + 'return 0;}\n'
    main_source = output / 'main.fy'
    main_source.write_text(authored, encoding='utf-8', newline='\n')
    modules = {}
    for base in (PROJECT / 'src', ROOT / 'internal/sdkstdlib/sources'):
        for path in base.rglob('*.fy'):
            name = re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))[1]
            assert name not in modules
            modules[name] = path
    modules['main'] = main_source
    seen = set()
    pending = ['main']
    while pending:
        name = pending.pop()
        if name in seen:
            continue
        seen.add(name)
        pending.extend(re.findall(r'^import\s+([\w:]+)', modules[name].read_text(encoding='utf-8'), re.M))
    paths = [main_source] + [modules[name] for name in sorted(seen - {'main'})]
    source_inputs = [dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in paths[1:] + [test_source, Path(__file__)]]
    snapshots = []
    if args.unchanged_paths:
        listed = args.unchanged_paths.resolve()
        source_inputs.append(dict(path=listed.relative_to(ROOT).as_posix(), sha256=digest(listed)))
        targets = json.loads(listed.read_text(encoding='utf-8'))
        if not isinstance(targets, list) or len(targets) > 128:
            parser.error('unchanged paths must be a bounded list')
        for name in targets:
            if not isinstance(name, str) or Path(name).is_absolute() or '..' in Path(name).parts:
                parser.error('unchanged paths must stay within the repository')
            path = ROOT / name
            def snapshot(path=path):
                if not path.exists():
                    return None
                stat = path.stat()
                return dict(directory=path.is_dir(), size=stat.st_size, modifiedNS=stat.st_mtime_ns, sha256=None if path.is_dir() else digest(path))
            snapshots.append((name, snapshot, snapshot()))
    arguments = []
    if args.fixture:
        fixture = args.fixture.resolve()
        source_inputs.append(dict(path=fixture.relative_to(ROOT).as_posix(), sha256=digest(fixture)))
        arguments = [str(fixture)]
    images = {name: dict(path=str(path.resolve()), sha256=digest(path)) for name, path in [('compiler', args.compiler), ('candidate', args.candidate), ('builder', args.builder)]}
    lock = PROJECT / 'fairy.lock'
    lock_sha256 = digest(lock)
    invocations = []

    def run(argv, cwd=ROOT, timeout=60):
        start = time.monotonic()
        result = subprocess.run(list(map(str, argv)), cwd=cwd, capture_output=True, timeout=timeout)
        row = dict(argv=list(map(str, argv)), cwd=str(cwd), seconds=time.monotonic() - start, exit=result.returncode, stdoutBytes=len(result.stdout), stdoutSHA256=hashlib.sha256(result.stdout).hexdigest(), stderr=result.stderr.decode('utf-8', errors='replace'))
        invocations.append(row)
        save(output / 'invocations.json', invocations)
        for name, snapshot, before in snapshots:
            assert snapshot() == before, 'readonly path changed: ' + name
        if len(argv)>1 and str(argv[1])=='test':
            (output / 'native.json').write_bytes(result.stdout)
        assert result.returncode == 0 and not result.stderr, row
        return result

    expected = {f'tests::{args.module}::{name}' for name in args.test}
    selection = [arg for name in args.test for arg in ('--exact', f'tests::{args.module}::{name}')]
    report = None
    if not args.candidate_only:
        native = run([args.compiler.resolve(), 'test', '--native', '--locked', '--json', '--capture-output', *selection, *(['--', *arguments] if arguments else [])], cwd=PROJECT, timeout=600)
        report = json.loads(native.stdout)
        save(output / 'native.json', report)
        assert report['passed'] == len(expected) and report['failed'] == 0
        assert report['engine'] == 'native' and report['lock'] == 'verified'
        assert {row['identity'] for row in report['tests']} == expected
    manifest = output / 'sources.list'
    manifest.write_text(''.join(p.relative_to(ROOT).as_posix() + '\n' for p in paths), encoding='utf-8', newline='\n')
    packet = run([args.candidate.resolve(), '--c-project-file', manifest], timeout=args.frontend_timeout)
    packet_path = output / 'candidate-packet.bin'
    packet_path.write_bytes(packet.stdout)
    assert packet.stdout.startswith(b'FYCPARTS1\n')
    executions = []
    for profile, flags in [('development', []), ('release', ['--release'])]:
        image = output / (profile + '.exe')
        run([args.builder.resolve(), '--packet', packet_path, '--packet-sha256', digest(packet_path), '--output', image, *flags], timeout=600)
        executed = run([image, *arguments])
        assert not executed.stdout
        executions.append(dict(profile=profile, path=str(image), sha256=digest(image), exit=executed.returncode))
    assert digest(lock) == lock_sha256, 'lock changed during verification'
    assert all(digest(ROOT / row['path']) == row['sha256'] for row in source_inputs), 'sources changed during verification'
    assert all(digest(Path(row['path'])) == row['sha256'] for row in images.values()), 'tool changed during verification'
    receipt = dict(schema='fairyc.candidate-private-tests/v1', passed=True, module=args.module, sourceInputs=source_inputs, tools=images, native=report, nativeTests=0 if args.candidate_only else len(expected), authoredTestsExecuted=len(expected), candidateOnly=args.candidate_only, lockSHA256=lock_sha256, candidateInputFiles=len(paths), candidatePacketSHA256=digest(packet_path), candidateExecutions=executions, invocations=invocations, publicApiChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    receipt['selection'] = dict(mode='explicit-subset' if args.selected_tests_only else 'all-authored', authoredTests=len(test_names), selectedTests=len(expected))
    receipt['frontendTimeoutSeconds'] = args.frontend_timeout
    receipt['productTimeoutChanged'] = False
    if snapshots:
        receipt['unchangedPaths'] = [dict(path=name, snapshot=before) for name, _, before in snapshots]
    save(output / 'receipt.json', receipt)
    print(args.module, 'passed:', len(expected), 'authored tests in both candidate C profiles' + ('' if args.candidate_only else ' and Go-native'), flush=True)


if __name__ == '__main__':
    main()
