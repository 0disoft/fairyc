"""Compare authored PE/unwind reports with Go using mutations and real MSVC images."""
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
    for name in ('compiler', 'candidate', 'builder', 'images', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    fixture = output / 'cases.bin'
    images = sorted(args.images.resolve().glob('*.exe'))
    assert len(images) == 4, 'exactly four previously verified MSVC images required'
    inputs = [ROOT / path for path in (
        'internal/pecoff/executable.go', 'internal/pecoff/unwind.go',
        'internal/pecoff/executable_test.go', 'internal/pecoff/unwind_test.go',
        'internal/pecoff/handler_link_test.go', 'internal/pecoff/candidate_pe_oracle_test.go',
        'examples/lexer_candidate/src/native_pe.fy',
        'examples/lexer_candidate/src/native_pe_unwind.fy',
        'examples/lexer_candidate/src/native_msvc_artifacts.fy',
        'examples/lexer_candidate/tests/native_pe.fy',
    )] + [Path(__file__).resolve(), args.candidate.resolve(), *images]
    before = [identity(path) for path in inputs]
    env = os.environ.copy()
    env['FAIRYC_PE_ORACLE'] = str(fixture)
    env['FAIRYC_PE_IMAGES'] = str(args.images.resolve())
    env['FAIRYC_PE_COMPILER'] = str(args.candidate.resolve())
    oracle = subprocess.run(['go', 'test', './internal/pecoff', '-run', '^TestCandidatePEOracle$',
                             '-count=1', '-v'], cwd=ROOT, env=env, capture_output=True, timeout=90)
    (output / 'oracle.stdout').write_bytes(oracle.stdout)
    (output / 'oracle.stderr').write_bytes(oracle.stderr)
    assert oracle.returncode == 0, oracle.stdout + oracle.stderr
    with fixture.open('rb') as stream:
        assert stream.read(len(b'FYCPEVERIFY1\n')) == b'FYCPEVERIFY1\n'
        count = struct.unpack('<I', stream.read(4))[0]
    fixture_before = identity(fixture)
    argv = [sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'),
            '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()),
            '--builder', str(args.builder.resolve()), '--module', 'native_pe',
            '--test', 'matches_go_pe_and_unwind_reports', '--isolated-native',
            '--fixture', str(fixture), '--output', str(output / 'execution')]
    result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=600)
    (output / 'execution.stdout').write_bytes(result.stdout)
    (output / 'execution.stderr').write_bytes(result.stderr)
    assert result.returncode == 0, result.stdout + result.stderr
    assert before == [identity(path) for path in inputs], 'PE verification input changed'
    assert fixture_before == identity(fixture), 'PE oracle changed during verification'
    execution = output / 'execution/receipt.json'
    assert json.loads(execution.read_text(encoding='utf-8'))['passed']
    receipt = dict(schema='fairyc.candidate-pe-unwind/v1', passed=True, inputs=before,
                   goOracleCases=count, fixture=fixture_before, execution=identity(execution),
                   profiles=['Go-native', 'development', 'release'],
                   authoredBuilderIntegrated=True, realImages=5,
                   compared='PE identity/sections/directories, unwind counts and function intervals, acceptance',
                   diagnosticTextParity=False, fullCoffParity=False, cliIntegrated=False,
                   fullCliFixedPointRefreshed=False, linuxExecuted=False,
                   publicApiChanged=False, sdkAbiChanged=False, databaseChanged=False, ciChanged=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8')
    print(f'PE/unwind passed: {count} Go report comparisons in three execution paths')


if __name__ == '__main__':
    main()
