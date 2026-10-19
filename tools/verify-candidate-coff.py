"""Compare authored COFF metadata and project admission with the live Go inspectors."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import struct
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent


def identity(path):
    return dict(path=path.resolve().relative_to(ROOT).as_posix(), bytes=path.stat().st_size,
                sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('compiler', 'candidate', 'builder', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    objects = output / 'objects'
    fixture = output / 'cases.bin'
    inputs = [ROOT / p for p in (
        'internal/pecoff/object.go', 'internal/pecoff/big_object.go',
        'internal/cdriver/windows_project.go',
        'internal/pecoff/candidate_coff_oracle_test.go',
        'internal/cdriver/candidate_coff_oracle_test.go',
        'examples/lexer_candidate/src/native_coff.fy',
        'examples/lexer_candidate/src/native_coff_layout.fy',
        'examples/lexer_candidate/src/native_msvc_artifacts.fy',
        'examples/lexer_candidate/tests/native_coff.fy',
    )] + [Path(__file__).resolve()]
    before = [identity(path) for path in inputs]
    env = os.environ.copy()
    env['FAIRYC_COFF_OBJECTS'] = str(objects)
    env['FAIRYC_COFF_ORACLE'] = str(fixture)
    for package, test, label in (
        ('cdriver', 'TestCandidateCOFFBuildObjects', 'objects'),
        ('pecoff', 'TestCandidateCOFFOracle', 'strict'),
        ('cdriver', 'TestCandidateCOFFProjectOracle', 'project'),
    ):
        result = subprocess.run(['go', 'test', './internal/'+package, '-run', '^'+test+'$',
                                 '-count=1', '-v'], cwd=ROOT, env=env, capture_output=True, timeout=120)
        (output / (label+'.stdout')).write_bytes(result.stdout)
        (output / (label+'.stderr')).write_bytes(result.stderr)
        assert result.returncode == 0, result.stdout + result.stderr
    with fixture.open('rb') as stream:
        assert stream.read(len(b'FYCCOFFVERIFY1\n')) == b'FYCCOFFVERIFY1\n'
        count = struct.unpack('<I', stream.read(4))[0]
    artifacts = [identity(p) for p in sorted(objects.glob('*.obj'))]
    assert len(artifacts) == 12
    fixture_before = identity(fixture)
    result = subprocess.run([
        sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'),
        '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()),
        '--builder', str(args.builder.resolve()), '--module', 'native_coff',
        '--test', 'matches_go_coff_reports_and_project_admission', '--isolated-native',
        '--fixture', str(fixture), '--output', str(output / 'execution'),
    ], cwd=ROOT, capture_output=True, timeout=600)
    (output / 'execution.stdout').write_bytes(result.stdout)
    (output / 'execution.stderr').write_bytes(result.stderr)
    assert result.returncode == 0, result.stdout + result.stderr
    assert before == [identity(p) for p in inputs], 'COFF inputs changed during verification'
    assert fixture_before == identity(fixture), 'COFF oracle changed during execution'
    assert artifacts == [identity(p) for p in sorted(objects.glob('*.obj'))], 'real objects changed'
    execution = output / 'execution/receipt.json'
    assert json.loads(execution.read_text(encoding='utf-8'))['passed']
    receipt = dict(schema='fairyc.candidate-coff/v1', passed=True, inputs=before,
                   goOracleCases=count, fixture=fixture_before, execution=identity(execution),
                   realObjects=artifacts, profiles=['Go-native', 'development', 'release'],
                   compared='strict full COFF reports and actual Go project driver admission',
                   authoredBuilderIntegrated=True, diagnosticTextParity=False,
                   cliIntegrated=False, fullCliFixedPointRefreshed=False, linuxExecuted=False,
                   publicApiChanged=False, sdkAbiChanged=False, databaseChanged=False, ciChanged=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8')
    print(f'COFF passed: {count} Go report/admission comparisons with 12 real objects in three execution paths')


if __name__ == '__main__':
    main()
