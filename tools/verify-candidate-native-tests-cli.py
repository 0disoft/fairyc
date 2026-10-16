"""Compare candidate test --native with Go using real compiled child processes."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--profile', choices=['development', 'release'])
    parser.add_argument('--reference-receipt', type=Path,
                        help='reuse identical Go observations authenticated by tool, image and fixture hashes')
    args = parser.parse_args()
    assert __debug__
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    build = json.loads(args.build_receipt.read_text(encoding='utf-8'))
    build_sha = digest(args.build_receipt)
    verifier_sha = digest(Path(__file__))
    assert build['passed'] and not build['goFrontendInvoked'] and not build.get('commandsOnly')
    for source in build['sources']:
        assert digest(ROOT / source['path']) == source['sha256'], source['path']
    output.mkdir(parents=True)
    reference = args.reference.resolve()
    reference_sha = digest(reference)
    rows = []
    reference_cache = {}
    if args.reference_receipt:
        prior = json.loads(args.reference_receipt.read_text(encoding='utf-8'))
        assert prior['passed'] and prior['verifierSHA256'] == digest(Path(__file__))
        assert prior['reference']['sha256'] == digest(reference)
        for row in prior['comparisons']:
            key = (row['id'], tuple(row['argv']), row['stdinSHA256'], row['fixtureSHA256'])
            reference_cache[key] = row['go']

    def run(image, argv, cwd, stdin=b''):
        start = time.monotonic()
        result = subprocess.run([str(image), *argv], cwd=cwd, input=stdin,
                                capture_output=True, timeout=120)
        return dict(exit=result.returncode, stdout=result.stdout.decode('utf-8'),
                    stderr=result.stderr.decode('utf-8'), seconds=time.monotonic() - start)

    for image in build['images']:
        if args.profile and image['profile'] != args.profile:
            continue
        candidate = Path(image['path'])
        assert digest(candidate) == image['sha256']
        assert digest(candidate.with_name('fairyc-candidate-build.exe')) == image['driverSHA256']
        profile = image['profile']
        homes = {}

        def fixture(name, example=None, tests=None):
            home = output / profile / name
            if example:
                shutil.copytree(ROOT / 'examples' / example, home,
                                ignore=shutil.ignore_patterns('build', 'fairy.lock'))
            else:
                (home / 'src').mkdir(parents=True)
                (home / 'fairy.toml').write_text(
                    'schema = "fairyc.manifest/v1"\n[package]\nuid = "probe.native-tests"\n'
                    'name = "native-tests"\nversion = "0.1.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n',
                    encoding='utf-8')
                (home / 'src/main.fy').write_text('module main;\nfn i32 main(){return 0;}\n', encoding='utf-8')
            for relative, text in (tests or {}).items():
                target = home / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text(text, encoding='utf-8')
            admitted = run(reference, ['check'], home)
            assert admitted['exit'] == 0, admitted
            homes[name] = (home, digest(home / 'fairy.lock'))
            return name

        def compare(name, home_name, argv, code=0, stdin=b'', exact=True):
            home, lock = homes[home_name]
            fixture_inputs = [(p.relative_to(home).as_posix(), digest(p)) for p in sorted(home.rglob('*'))
                              if p.is_file() and p.name != 'fairy.lock' and 'build' not in p.relative_to(home).parts]
            fixture_sha = hashlib.sha256(json.dumps(fixture_inputs).encode()).hexdigest()
            stdin_sha = hashlib.sha256(stdin).hexdigest()
            key = (name, tuple(argv), stdin_sha, fixture_sha)
            reused = key in reference_cache
            if not reused:
                reference_cache[key] = run(reference, ['test', '--locked', *argv], home, stdin)
            go = reference_cache[key]
            other = run(candidate, ['test', '--locked', *argv], home, stdin)
            matched = all(go[key] == other[key] for key in ('exit', 'stdout', 'stderr'))
            row = dict(profile=profile, id=name, argv=argv, go=go, candidate=other,
                       exactMatch=matched, expectedExit=code, referenceReused=reused,
                       stdinSHA256=stdin_sha, fixtureSHA256=fixture_sha)
            rows.append(row)
            (output / 'observations.json').write_text(json.dumps(rows, indent=2) + '\n', encoding='utf-8')
            assert digest(home / 'fairy.lock') == lock, 'locked execution changed lock'
            assert go['exit'] == code and other['exit'] == code, row
            if exact:
                assert matched, row
            else:
                assert not go['stdout'] and not other['stdout'] and other['stderr'], row
            print('passed:', profile, name, flush=True)

        mixed = fixture('mixed', tests={'tests/mixed.fy': 'module tests::mixed;\n'
                        '@test fn void a_fails(){assert(false);}\n'
                        '@test fn void z_passes(){assert(true);}\n'})
        for label, extra in [('text', []), ('json', ['--json']), ('locations', ['--locations', '--json']),
                             ('located-text', ['--locations']), ('release', ['--release', '--json']),
                             ('capture', ['--capture-output', '--json', '--locations'])]:
            compare('mixed-' + label, mixed, ['--native', *extra], 5)
        compare('exact-pass', mixed, ['--native', '--exact', 'tests::mixed::z_passes'])
        compare('exact-fail', mixed, ['--native', '--exact', 'tests::mixed::a_fails'], 5)
        compare('missing-exact', mixed, ['--native', '--exact', 'tests::mixed::missing'], 2)
        compare('file-selection', mixed, ['--native', '--file', 'tests/mixed.fy', '--json'], 5)
        compare('dir-selection', mixed, ['--native', '--dir', 'tests', '--json'], 5)
        compare('stdin-ambiguous', mixed, ['--native', '--stdin'], 2)
        empty = fixture('empty')
        compare('empty-json', empty, ['--native', '--json'])
        compare('empty-release', empty, ['--native', '--release'])
        eof = fixture('eof', tests={'tests/eof.fy': 'module tests::eof;\nimport std::io as io;\n'
                      '@test @effects(console) fn void empty(){var u8[1] b=[7];'
                      'let bool eof=switch(io::read_stdin(ref mut b)){case .ok(n):{n==0}case .err(_):{false}};assert(eof);}\n'})
        compare('default-eof', eof, ['--native'], stdin=b'caller input must remain unused')
        traps = fixture('traps', tests={'tests/traps.fy': 'module tests::traps;\nimport std::process as process;\n'
                        '@test fn void a_bounds(){let i32[1] a=[42];discard a[process::arg_count()];}\n'
                        '@test fn void b_overflow(){var i32 n=2147483647;n+=1;}\n'
                        '@test fn void c_division(){let u32 n=42;discard n/0;}\n'
                        '@test fn void d_signed_division(){let i32 n=-2147483648;discard n/-1;}\n'
                        '@test fn void e_shift(){var u8 n=1;n>>=8;}\n'
                        '@test fn void z_after_traps(){assert(true);}\n'})
        compare('trap-categories', traps, ['--native', '--json', '--capture-output'], 5)
        workspace = fixture('workspace', 'workspace_quickstart')
        compare('workspace-locations', workspace, ['--native', '--workspace', '--json', '--locations'])
        compare('package', workspace, ['--native', '--package', 'example.workspace-quickstart.answer', '--json'])
        vector = fixture('vector', 'std_vec_quickstart')
        compare('resource-sdk', vector, ['--native', '--json'])
        arguments = fixture('args', 'native_test_args_quickstart')
        compare('args', arguments, ['--native', '--json', '--', '--locked', '', '한글λ'])
        streams = fixture('output', 'native_test_output_quickstart')
        compare('binary-output', streams, ['--native', '--json', '--capture-output'], 5)
        compare('unrequested-output', streams, ['--native'], 2)
        inputs = fixture('stdin', 'native_test_stdin_quickstart')
        compare('stdin-binary', inputs, ['--native', '--stdin', '--exact', 'tests::stdin::caller_input_is_forwarded'],
                stdin=b'Fairy\x00\n')
        compare('stdin-replay-each', inputs, ['--native', '--stdin-file', 'tests/fixtures/input.txt', '--json'])
        compare('stdin-missing', inputs, ['--native', '--stdin-file', 'missing.bin'], 7)
        compare('stdin-parent', inputs, ['--native', '--stdin-file', '../escape.bin'], 7)
        (homes[inputs][0] / 'large.bin').write_bytes(b'x' * 65537)
        compare('stdin-oversize', inputs, ['--native', '--stdin-file', 'large.bin'], 7)
        deadline = fixture('timeout', 'native_test_timeout_quickstart')
        compare('deadline', deadline, ['--native', '--timeout-seconds', '1'], 6, exact=False)
        compare('deadline-nonhanging', deadline, ['--native', '--timeout-seconds', '1', '--', 'skip'])

    assert rows, 'no candidate profile selected'
    assert digest(args.build_receipt) == build_sha and digest(reference) == reference_sha
    assert digest(Path(__file__)) == verifier_sha
    for source in build['sources']:
        assert digest(ROOT / source['path']) == source['sha256'], source['path']
    for image in build['images']:
        assert digest(Path(image['path'])) == image['sha256']
        assert digest(Path(image['path']).with_name('fairyc-candidate-build.exe')) == image['driverSHA256']
    receipt = dict(schema='fairyc.candidate-native-tests-cli/v1', passed=True,
                   verifierSHA256=digest(Path(__file__)),
                   buildReceipt=str(args.build_receipt.resolve()), buildReceiptSHA256=digest(args.build_receipt),
                   reference=dict(path=str(reference), sha256=digest(reference)), comparisons=rows,
                   exactComparisons=sum(row['exactMatch'] for row in rows),
                   limitations=['Windows native candidate only; semantic interpreter remains unimplemented',
                                'Deadline infrastructure diagnostics compared by exit and empty report; platform wording may differ'])
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print('passed:', len(rows), 'native test comparisons', flush=True)


if __name__ == '__main__':
    main()
