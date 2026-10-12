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
    parser.add_argument('--isolated-native', action='store_true', help='run Go native tests on exact copies of the selected module import closure; retain the original workspace source-check separately')
    parser.add_argument('--stdin-fixture', type=Path, help='exact binary input for each candidate C execution')
    parser.add_argument('--stdout-fixture', type=Path, help='expected complete candidate stdout, empty by default')
    parser.add_argument('--stderr-fixture', type=Path, help='expected complete candidate stderr, empty by default')
    parser.add_argument('--unchanged-paths', type=Path, help='JSON list of repository-relative paths whose bytes and modification times must remain unchanged')
    args = parser.parse_args()
    if not 1 <= args.frontend_timeout <= 900:
        parser.error('frontend timeout must be in 1..900 seconds')
    if args.isolated_native and args.candidate_only:
        parser.error('isolated-native requires Go native execution')
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
    effects = ['filesystem', 'console']
    for declaration, name in re.findall(r'@test\s+(?:@effects\(([^\n)]+)\)\s+)?fn void (\w+)\(', authored):
        if name in args.test:
            for effect in declaration.split(','):
                effect = effect.strip()
                if effect and effect not in effects:
                    effects.append(effect)
    original_module = f'module tests::{args.module};'
    assert authored.startswith(original_module)
    authored = re.sub(r'(?m)^(\s*)@test\b\s*', r'\1', authored.replace(original_module, 'module main;', 1))
    authored += '\n@effects(' + ','.join(effects) + ') fn i32 main(){' + ''.join(name + '();' for name in args.test) + 'return 0;}\n'
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
    streams = {}
    for name, path in [('stdin',args.stdin_fixture),('stdout',args.stdout_fixture),('stderr',args.stderr_fixture)]:
        if path:
            path = path.resolve()
            if path.stat().st_size > 1048576:
                parser.error('stream fixtures must be at most 1 MiB')
            streams[name] = path.read_bytes()
            source_inputs.append(dict(path=path.relative_to(ROOT).as_posix(),sha256=digest(path)))
    lock = PROJECT / 'fairy.lock'
    lock_sha256 = digest(lock)
    invocations = []

    def run(argv, cwd=ROOT, timeout=60, stdin=None, stderr=b''):
        start = time.monotonic()
        result = subprocess.run(list(map(str, argv)), cwd=cwd, capture_output=True, timeout=timeout, input=stdin)
        row = dict(argv=list(map(str, argv)), cwd=str(cwd), seconds=time.monotonic() - start, exit=result.returncode, stdoutBytes=len(result.stdout), stdoutSHA256=hashlib.sha256(result.stdout).hexdigest(), stderr=result.stderr.decode('utf-8', errors='replace'))
        invocations.append(row)
        save(output / 'invocations.json', invocations)
        for name, snapshot, before in snapshots:
            assert snapshot() == before, 'readonly path changed: ' + name
        if len(argv)>1 and str(argv[1])=='test':
            (output / 'native.json').write_bytes(result.stdout)
        assert result.returncode == 0 and result.stderr == stderr, row
        return result

    expected = {f'tests::{args.module}::{name}' for name in args.test}
    selection = [arg for name in args.test for arg in ('--exact', f'tests::{args.module}::{name}')]
    report = None
    native_isolation = None
    if not args.candidate_only:
        native_project = PROJECT
        copied_inputs = []
        if args.isolated_native:
            originals = [test_source] + [path for path in paths[1:] if path.is_relative_to(PROJECT / 'src')]
            manifest_bytes = (PROJECT / 'fairy.toml').read_bytes()
            source_inputs.append(dict(path=(PROJECT / 'fairy.toml').relative_to(ROOT).as_posix(), sha256=hashlib.sha256(manifest_bytes).hexdigest()))
            fingerprint = hashlib.sha256(manifest_bytes + b''.join(str(path.relative_to(PROJECT)).encode() + bytes.fromhex(digest(path)) for path in originals)).hexdigest()
            native_project = output / ('native-project-' + fingerprint[:20])
            native_project.mkdir(parents=True, exist_ok=True)
            native_manifest = native_project / 'fairy.toml'
            native_manifest.write_bytes(manifest_bytes)
            copied_inputs.append((native_manifest, hashlib.sha256(manifest_bytes).hexdigest()))
            expected_sources = set()
            for original in originals:
                destination = native_project / original.relative_to(PROJECT)
                destination.parent.mkdir(parents=True, exist_ok=True)
                destination.write_bytes(original.read_bytes())
                copied_inputs.append((destination, digest(original)))
                expected_sources.add(destination)
            entry = native_project / 'src' / 'main.fy'
            assert entry not in expected_sources
            entry.write_bytes(b'module main;fn i32 main(){return 0;}\n')
            expected_sources.add(entry)
            copied_inputs.append((entry, digest(entry)))
            assert set(native_project.rglob('*.fy')) == expected_sources, 'isolated native sources contain unexpected files'
            run([args.compiler.resolve(), 'check'], cwd=native_project, timeout=600)
            native_isolation = dict(path=str(native_project), localSourceFiles=len(expected_sources)-1, inputFingerprint=fingerprint, copiedInputs=[dict(path=str(path), sha256=sha) for path, sha in copied_inputs])
        native = run([args.compiler.resolve(), 'test', '--native', '--locked', '--json', '--capture-output', *selection, *(['--', *arguments] if arguments else [])], cwd=native_project, timeout=600)
        report = json.loads(native.stdout)
        save(output / 'native.json', report)
        assert report['passed'] == len(expected) and report['failed'] == 0
        assert report['engine'] == 'native' and report['lock'] == 'verified'
        assert {row['identity'] for row in report['tests']} == expected
        assert all(digest(path) == sha for path, sha in copied_inputs), 'isolated native input changed during execution'
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
        executed = run([image, *arguments],stdin=streams.get('stdin'),stderr=streams.get('stderr',b''))
        assert executed.stdout == streams.get('stdout',b''), invocations[-1]
        executions.append(dict(profile=profile, path=str(image), sha256=digest(image), exit=executed.returncode))
    assert digest(lock) == lock_sha256, 'lock changed during verification'
    assert all(digest(ROOT / row['path']) == row['sha256'] for row in source_inputs), 'sources changed during verification'
    assert all(digest(Path(row['path'])) == row['sha256'] for row in images.values()), 'tool changed during verification'
    receipt = dict(schema='fairyc.candidate-private-tests/v1', passed=True, module=args.module, sourceInputs=source_inputs, tools=images, native=report, nativeTests=0 if args.candidate_only else len(expected), authoredTestsExecuted=len(expected), candidateOnly=args.candidate_only, lockSHA256=lock_sha256, candidateInputFiles=len(paths), candidatePacketSHA256=digest(packet_path), candidateExecutions=executions, invocations=invocations, publicApiChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    receipt['selection'] = dict(mode='explicit-subset' if args.selected_tests_only else 'all-authored', authoredTests=len(test_names), selectedTests=len(expected))
    receipt['frontendTimeoutSeconds'] = args.frontend_timeout
    receipt['productTimeoutChanged'] = False
    receipt['wrapperEffects'] = effects
    if native_isolation:
        receipt['nativeIsolation'] = native_isolation
    if streams:
        receipt['streamVerification'] = {name:dict(bytes=len(data),sha256=hashlib.sha256(data).hexdigest()) for name,data in streams.items()}
    if snapshots:
        receipt['unchangedPaths'] = [dict(path=name, snapshot=before) for name, _, before in snapshots]
    save(output / 'receipt.json', receipt)
    print(args.module, 'passed:', len(expected), 'authored tests in both candidate C profiles' + ('' if args.candidate_only else ' and Go-native'), flush=True)


if __name__ == '__main__':
    main()
