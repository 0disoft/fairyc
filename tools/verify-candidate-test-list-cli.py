"""Verify the stable test-list CLI contract against the Go reference compiler.

Promotes the cached test-list probe suite: 89 'fairyc test --list ...' invocations run on the
Go reference and on every candidate image named by the build receipt, and both engines must
agree byte for byte on exit code, stdout and stderr. The build receipt authenticates the exact
candidate sources and images, so a receipt from a different source snapshot is rejected before
any comparison. Semantic and native execution stay recorded as observations: while the
candidate rejects a mode the documented rejection is asserted, and once the candidate
implements it the same invocation must match the reference exactly. A timeout is a failure,
never a pass.
"""
import argparse
import hashlib
import json
import shutil
import subprocess
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CASE_COUNT = 89
MANIFEST = ('schema = "fairyc.manifest/v1"\n[package]\nuid = "probe.test-list"\n'
            'name = "test-list-probe"\nversion = "0.1.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n')
DEFAULT_SOURCE = 'module main;\nfn i32 main(){return 0;}\n'
UNIMPLEMENTED = {'semantic': 'interpreter is not implemented',
                 'native': 'native test execution is not implemented'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True,
                        help='candidate build receipt with passed sources and images')
    parser.add_argument('--reference', type=Path, required=True,
                        help='Go reference fairy compiler image')
    parser.add_argument('--output', type=Path, required=True,
                        help='new directory under .cache for fixtures, observations and the receipt')
    parser.add_argument('--profile', action='append', choices=['development', 'release'],
                        help='verify only this candidate profile (default: both)')
    parser.add_argument('--timeout-seconds', type=float, default=90,
                        help='per-invocation timeout; a timeout is a failure, never a pass')
    args = parser.parse_args()
    assert __debug__, 'verification requires Python without -O'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists(), output
    digest = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
    build_path = args.build_receipt.resolve()
    build_sha = digest(build_path)
    build = json.loads(build_path.read_text(encoding='utf-8'))
    assert build['passed'] and not build['goFrontendInvoked'] and not build.get('commandsOnly') and not build.get('compilerOnly')
    assert 'sources' in build and 'images' in build, 'build receipt must carry repo-relative sources and profile images'
    source_inputs = build['sources'] + [
        dict(path=Path(__file__).relative_to(ROOT).as_posix(), sha256=digest(Path(__file__)))]
    for row in source_inputs:
        assert digest(ROOT / row['path']) == row['sha256'], ('stale build receipt source', row['path'])
    reference = args.reference.resolve()
    reference_sha = digest(reference)
    images = [row for row in build['images'] if not args.profile or row['profile'] in args.profile]
    assert images, 'no candidate image matches the requested profile'
    for row in images:
        assert row['profile'] in ('development', 'release'), row
        assert Path(row['path']).exists(), ('missing candidate image', row['path'])
    output.mkdir(parents=True)
    rows = []
    admissions = []
    locks = {}

    def run(image, argv, cwd):
        started = time.monotonic()
        try:
            result = subprocess.run([str(image), *map(str, argv)], cwd=cwd, capture_output=True,
                                    timeout=args.timeout_seconds)
        except subprocess.TimeoutExpired as error:
            return dict(exit=None, stdout=(error.stdout or b'').decode('utf-8', errors='replace'),
                        stderr=(error.stderr or b'').decode('utf-8', errors='replace'),
                        seconds=time.monotonic() - started, timedOut=True)
        return dict(exit=result.returncode, stdout=result.stdout.decode('utf-8', errors='replace'),
                    stderr=result.stderr.decode('utf-8', errors='replace'),
                    seconds=time.monotonic() - started)

    def fixture(profile, name, source=None, tests=None, example=None):
        homes = {}
        for engine in ('go', 'candidate'):
            home = output / profile / name / engine
            if home.exists():
                # Delete only copies this tool previously generated inside its own output root.
                assert home.resolve().is_relative_to(output)
                shutil.rmtree(home)
            if example:
                shutil.copytree(ROOT / 'examples' / example, home,
                                ignore=shutil.ignore_patterns('build', 'fairy.lock'))
            else:
                (home / 'src').mkdir(parents=True)
                (home / 'fairy.toml').write_text(MANIFEST, encoding='utf-8')
                (home / 'src/main.fy').write_text(source or DEFAULT_SOURCE, encoding='utf-8')
            for relative, text in (tests or {}).items():
                target = home / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(text.encode('utf-8'))
            admitted = run(reference, ['check'], home)
            assert admitted['exit'] == 0, (name, engine, admitted)
            admissions.append(dict(profile=profile, fixture=name, engine=engine, result=admitted))
            locks[str(home)] = digest(home / 'fairy.lock')
            homes[engine] = home
        return homes

    def compare(profile, candidate, name, homes, argv, expected_exit=0):
        go = run(reference, ['test', *argv], homes['go'])
        other = run(candidate, ['test', *argv], homes['candidate'])
        for home in homes.values():
            assert digest(home / 'fairy.lock') == locks[str(home)], ('production lock mutated', home, argv)
        exact = all(go[key] == other[key] for key in ('exit', 'stdout', 'stderr'))
        row = dict(profile=profile, id=name, argv=['test', *argv], expectedExit=expected_exit,
                   go=go, candidate=other, exactMatch=exact)
        rows.append(row)
        (output / 'observations.json').write_text(json.dumps(rows, indent=2) + '\n', encoding='utf-8')
        assert go['exit'] == expected_exit, ('reference exit', row)
        assert other['exit'] == expected_exit, ('candidate exit', row)
        assert exact, ('candidate mismatch', row)
        print('passed:', profile, name, flush=True)
        return row

    def suite(profile, candidate):
        start = len(rows)
        workspace = fixture(profile, 'workspace', example='workspace_quickstart')
        for name, argv in [
            ('root-text', ['--list']), ('root-json', ['--list', '--json']),
            ('root-locations', ['--list', '--json', '--locations']),
            ('root-located-text', ['--list', '--locations']),
            ('workspace-text', ['--list', '--workspace']),
            ('workspace-json', ['--list', '--json', '--workspace']),
            ('workspace-locations', ['--list', '--json', '--workspace', '--locations']),
            ('member', ['--list', '--json', '--package', 'example.workspace-quickstart.answer']),
            ('member-locations', ['--list', '--json', '--package', 'example.workspace-quickstart.answer', '--locations']),
            ('workspace-file', ['--list', '--workspace', '--file', 'libs/answer/tests/answer.fy', '--json', '--locations']),
            ('workspace-dir', ['--list', '--workspace', '--dir', 'libs/answer/tests', '--json']),
            ('root-file', ['--list', '--file', 'tests/answer.fy']), ('root-dir', ['--list', '--dir', 'tests']),
        ]:
            compare(profile, candidate, name, workspace, [*argv, '--locked'])
        for name, argv in [
            ('duplicate-list', ['--list', '--list']), ('duplicate-json', ['--list', '--json', '--json']),
            ('duplicate-locations', ['--list', '--locations', '--locations']),
            ('duplicate-locked', ['--list', '--locked', '--locked']),
            ('list-exact', ['--list', '--exact', 'tests::answer::dependency_answer_is_42']),
            ('list-native', ['--list', '--native']), ('list-release', ['--list', '--release']),
            ('list-stdin', ['--list', '--stdin']), ('list-input', ['--list', '--stdin-file', 'input.bin']),
            ('list-timeout', ['--list', '--timeout-seconds', '1']), ('list-capture', ['--list', '--capture-output']),
            ('workspace-package', ['--list', '--workspace', '--package', 'p.q']),
            ('file-exact', ['--list', '--file', 'tests/answer.fy', '--exact', 'a']),
            ('dir-file', ['--list', '--dir', 'tests', '--file', 'tests/answer.fy']),
            ('dir-exact', ['--list', '--dir', 'tests', '--exact', 'a']),
            ('duplicate-file', ['--list', '--file', 'a', '--file', 'a']),
            ('duplicate-dir', ['--list', '--dir', 'a', '--dir', 'a']),
            ('duplicate-exact', ['--list', '--exact', 'a', '--exact', 'a']),
            ('duplicate-package', ['--list', '--package', 'a.b', '--package', 'c.d']),
            ('missing-package', ['--list', '--package']), ('missing-file', ['--list', '--file']),
            ('missing-dir', ['--list', '--dir']), ('missing-exact', ['--list', '--exact']),
            ('missing-input', ['--list', '--stdin-file']), ('missing-timeout', ['--list', '--timeout-seconds']),
            ('flag-instead-value', ['--list', '--package', '--locked']),
            ('bad-timeout', ['--list', '--timeout-seconds', '01']), ('unknown-flag', ['--list', '--bad']),
            ('unknown-package', ['--list', '--package', 'missing.package']),
            ('unknown-file', ['--list', '--file', 'tests/missing.fy']),
            ('unknown-dir', ['--list', '--dir', 'tests/missing']),
            ('noncanonical-file', ['--list', '--file', 'tests\\answer.fy']),
            ('missing-order', ['--list', '--file', 'z', '--file', 'a']), ('delimiter', ['--list', '--', 'a']),
        ]:
            compare(profile, candidate, name, workspace, argv, 2)
        for index, value in enumerate(['<>&"', '\a\b\f\n\r\t\v\x01\x1f\x7f', '\u00a0\u00ad', '한글😀',
                                       '\u200b\u2028\u2029', '\U0001000c\U000e0001']):
            compare(profile, candidate, 'quoted-file-' + str(index), workspace, ['--list', '--file', value], 2)
            compare(profile, candidate, 'quoted-duplicate-' + str(index), workspace,
                    ['--list', '--file', value, '--file', value], 2)
        for name, argv, code in [
            ('empty-package', ['--list', '--package', ''], 0),
            ('empty-workspace-package', ['--list', '--workspace', '--package', ''], 0),
            ('empty-input', ['--list', '--stdin-file', ''], 0),
            ('empty-file', ['--list', '--file', ''], 2), ('empty-dir', ['--list', '--dir', ''], 2),
            ('duplicate-timeout', ['--list', '--timeout-seconds', '1', '--timeout-seconds', '2'], 2),
            ('long-timeout', ['--list', '--timeout-seconds', '999999999999999999999999'], 2),
            ('empty-delimiter', ['--list', '--'], 2),
        ]:
            compare(profile, candidate, name, workspace, argv, code)
        empty = fixture(profile, 'empty')
        for argv in [['--list'], ['--list', '--json'], ['--list', '--json', '--locations'],
                     ['--list', '--json', '--workspace', '--locations']]:
            compare(profile, candidate, 'empty-' + str(len(argv)), empty, [*argv, '--locked'])
        trap = fixture(profile, 'trap', tests={
            'tests/trap.fy': 'module tests::trap;\n@test\nfn void stops(){assert(false);}\n'})
        compare(profile, candidate, 'trap-list', trap, ['--list', '--json', '--locations', '--locked'])
        nested = fixture(profile, 'nested', tests={
            'tests/zero.fy': 'module tests::zero;\nfn void helper(){}\n',
            'tests/unit/nested/b.fy': 'module tests::unit::nested::b;\r\n// 한글 😀\r\n@test\r\nfn void zulu(){assert(true);}\r\n@test\r\nfn void alpha(){}\r\n',
            'tests/unit_other/a.fy': 'module tests::unit_other::a;\n@test fn void other(){}\n',
            'tests/a.fy': 'module tests::a;\n@test fn void beta(){}\n'})
        for name, argv in [
            ('nested-locations', ['--list', '--locations', '--json']),
            ('nested-dir', ['--list', '--dir', 'tests/unit', '--locations', '--json']),
            ('nested-file', ['--list', '--file', 'tests/unit/nested/b.fy', '--locations', '--json']),
            ('repeated-dirs', ['--list', '--dir', 'tests/unit', '--dir', 'tests/unit/nested', '--json']),
            ('repeated-files', ['--list', '--file', 'tests/a.fy', '--file', 'tests/unit/nested/b.fy', '--json']),
            ('empty-test-file', ['--list', '--file', 'tests/zero.fy', '--json']),
            ('empty-module-dir', ['--list', '--dir', 'tests', '--json']),
        ]:
            compare(profile, candidate, name, nested, [*argv, '--locked'])
        for name, declaration in [
            ('arguments', '@test("extra")\nfn void bad(){}'), ('duplicate', '@test\n@test\nfn void bad(){}'),
            ('public', '@test\npub fn void bad(){}'), ('unsafe', '@test\nunsafe fn void bad(){}'),
            ('parameters', '@test\nfn void bad(i32 value){}'), ('result', '@test\nfn i32 bad(){return 0;}'),
            ('record', '@test\nstruct Bad{i32 n;}'),
        ]:
            homes = fixture(profile, 'invalid-' + name, tests={
                'tests/invalid.fy': 'module tests::invalid;\n' + declaration + '\n'})
            compare(profile, candidate, 'invalid-' + name, homes, ['--list', '--json', '--locations', '--locked'], 3)
        sdk = fixture(profile, 'sdk-tests-only', tests={
            'tests/sdk.fy': 'module tests::sdk;\nimport std::io as io;\n@test\n@effects(console)\nfn void writes(){discard io::write("not executed");}\n'})
        compare(profile, candidate, 'sdk-tests-only', sdk, ['--list', '--json', '--locations', '--locked'])
        vector = fixture(profile, 'vector', example='std_vec_quickstart')
        compare(profile, candidate, 'vector', vector, ['--list', '--json', '--locations', '--locked'])
        bad = fixture(profile, 'semantic', tests={
            'tests/good.fy': 'module tests::good;\n@test fn void good(){}\n',
            'tests/bad.fy': 'module tests::bad;\n@test fn void bad(){let i32 n=true;}\n'})
        compare(profile, candidate, 'semantic-outside-file-filter', bad, ['--list', '--file', 'tests/good.fy', '--locked'], 3)
        assert len(rows) - start == CASE_COUNT, ('case suite changed', len(rows) - start)
        return empty

    def execution_mode(profile, candidate, empty, mode):
        argv = ['--locked'] if mode == 'semantic' else ['--native', '--locked']
        other = run(candidate, ['test', *argv], empty['candidate'])
        if UNIMPLEMENTED[mode] in other['stderr']:
            assert other['exit'] not in (0, None) and not other['stdout'], (mode, other)
            return dict(profile=profile, mode=mode, implemented=False, candidate=other)
        reference_row = run(reference, ['test', *argv], empty['go'])
        assert all(other[key] == reference_row[key] for key in ('exit', 'stdout', 'stderr')), (mode, other, reference_row)
        return dict(profile=profile, mode=mode, implemented=True, reference=reference_row, candidate=other)

    modes = []
    for selected in images:
        candidate = Path(selected['path'])
        assert digest(candidate) == selected['sha256'], ('candidate image changed', candidate)
        if 'driverSHA256' in selected:
            assert digest(candidate.parent / 'fairyc-candidate-build.exe') == selected['driverSHA256']
        empty = suite(selected['profile'], candidate)
        modes.append(execution_mode(selected['profile'], candidate, empty, 'semantic'))
        modes.append(execution_mode(selected['profile'], candidate, empty, 'native'))
    assert digest(build_path) == build_sha, 'build receipt changed during verification'
    assert digest(reference) == reference_sha, 'reference changed during verification'
    for selected in images:
        assert digest(Path(selected['path'])) == selected['sha256'], ('image changed', selected['path'])
    for row in source_inputs:
        assert digest(ROOT / row['path']) == row['sha256'], ('source changed', row['path'])
    receipt = dict(schema='fairyc.candidate-test-list-cli/v0', passed=True, cases=len(rows),
                   caseCountPerProfile=CASE_COUNT, exactMatches=sum(row['exactMatch'] for row in rows),
                   profiles=[row['profile'] for row in images],
                   buildReceipt=dict(path=str(build_path.relative_to(ROOT)), sha256=build_sha),
                   sourceInputs=source_inputs, reference=dict(path=str(reference), sha256=reference_sha),
                   images=images, admissions=admissions, executionModes=modes, rows=rows,
                   productionLocksUnchanged=True, trackedSourceEdits=False,
                   publicGoCliChanged=False, ciChanged=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print('test-list CLI passed:', len(rows), 'exact comparison(s)', flush=True)


if __name__ == '__main__':
    main()
