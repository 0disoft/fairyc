"""Compare authored runtime selection with the Go contract, including a compiler header."""
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
    return dict(path=path.resolve().relative_to(ROOT).as_posix(),
                bytes=path.stat().st_size, sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('compiler', 'candidate', 'builder', 'packet', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    packet = args.packet.resolve()
    if not 0 < packet.stat().st_size <= 100663296:
        parser.error('compiler packet must be within 96 MiB')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    fixture = output / 'runtime-cases.bin'
    sources = [ROOT / path for path in (
        'internal/candidatebuild/build.go',
        'internal/candidatebuild/runtime_selection_oracle_test.go',
        'internal/cstdlibcontract/contract.go', 'internal/candidatehost/runtime.go',
        'examples/lexer_candidate/src/native_runtime_contract.fy',
        'examples/lexer_candidate/src/native_runtime.fy',
        'examples/lexer_candidate/tests/native_runtime.fy',
    )] + [Path(__file__).resolve()]
    before = [identity(path) for path in sources + [packet]]
    env = os.environ.copy()
    env['FAIRYC_RUNTIME_ORACLE'] = str(fixture)
    env['FAIRYC_RUNTIME_PACKET'] = str(packet)
    oracle = subprocess.run(['go', 'test', './internal/candidatebuild', '-run',
                             '^TestCandidateRuntimeSelectionOracle$', '-count=1', '-v'],
                            cwd=ROOT, env=env, capture_output=True, timeout=90)
    (output / 'go-oracle.stdout').write_bytes(oracle.stdout)
    (output / 'go-oracle.stderr').write_bytes(oracle.stderr)
    assert oracle.returncode == 0, oracle.stdout + oracle.stderr
    data = fixture.read_bytes()
    assert data.startswith(b'FYCRUNTIME1\n')
    count = struct.unpack_from('<I', data, 12)[0]
    assert 1000 < count < 8192
    fixture_before = identity(fixture)
    tests = ['matches_go_symbols_closures_order_and_diagnostics',
             'selection_is_inert_and_bounds_diagnostics']
    argv = [sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'),
            '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()),
            '--builder', str(args.builder.resolve()), '--module', 'native_runtime',
            '--isolated-native', '--fixture', str(fixture), '--output', str(output / 'execution')]
    for test in tests:
        argv.extend(['--test', test])
    result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=600)
    (output / 'execution.stdout').write_bytes(result.stdout)
    (output / 'execution.stderr').write_bytes(result.stderr)
    assert result.returncode == 0, result.stdout + result.stderr
    assert before == [identity(path) for path in sources + [packet]], 'selection inputs changed'
    assert fixture_before == identity(fixture), 'Go fixture changed during execution'
    execution = output / 'execution/receipt.json'
    assert json.loads(execution.read_text(encoding='utf-8'))['passed']
    receipt = dict(schema='fairyc.candidate-runtime-selection/v1', passed=True,
                   sources=before[:-1], compilerPacket=before[-1], goOracleCases=count,
                   fixture=fixture_before, execution=identity(execution), authoredTests=len(tests),
                   profiles=['Go-native', 'development', 'release'],
                   runtimeCBytesPackaged=False, cliIntegrated=False,
                   fullCliFixedPointRefreshed=False, linuxExecuted=False,
                   publicApiChanged=False, sdkAbiChanged=False, databaseChanged=False,
                   ciChanged=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print(f'Runtime selection passed: {count} Go cases in Go native and both candidate C profiles')


if __name__ == '__main__':
    main()
