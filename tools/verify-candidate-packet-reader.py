"""Compare the authored FYCPARTS1 reader with Go, including one real C packet."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent


def identity(path):
    return dict(path=str(path.resolve()), bytes=path.stat().st_size,
                sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('compiler', 'candidate', 'builder', 'packet', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    packet = args.packet.resolve()
    if not 0 < packet.stat().st_size <= 100663296:
        parser.error('the real compiler packet must be within 96 MiB')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    fixture = output / 'go-packet-cases.bin'
    inputs = [ROOT / 'internal/candidatebuild/packet.go',
              ROOT / 'internal/candidatebuild/packet_decode_oracle_test.go',
              ROOT / 'examples/lexer_candidate/src/native_packet_decode.fy',
              ROOT / 'examples/lexer_candidate/tests/native_packet_decode.fy',
              Path(__file__).resolve(), packet]
    before = [identity(path) for path in inputs]
    env = os.environ.copy()
    env['FAIRYC_PACKET_DECODE_FIXTURE'] = str(fixture)
    env['FAIRYC_PACKET_DECODE_FULL_PACKET'] = str(packet)
    oracle = subprocess.run(['go', 'test', './internal/candidatebuild', '-run',
                             '^TestPacketDecodeOracle$', '-count=1', '-v'],
                            cwd=ROOT, env=env, capture_output=True, timeout=60)
    (output / 'go-oracle.stdout').write_bytes(oracle.stdout)
    (output / 'go-oracle.stderr').write_bytes(oracle.stderr)
    assert oracle.returncode == 0, oracle.stdout + oracle.stderr
    cases = json.loads(Path(str(fixture) + '.json').read_text(encoding='utf-8'))
    assert len(cases) > 300 and any(row['name'] == 'full-compiler-packet' for row in cases)
    fixtures = [fixture, Path(str(fixture) + '.packet'), Path(str(fixture) + '.invalid'),
                Path(str(fixture) + '.json')]
    fixture_identities = [identity(path) for path in fixtures]
    assert fixture_identities[1]['sha256'] == identity(packet)['sha256']
    tests = ['matches_go_admission_and_part_lengths',
             'owns_the_decoded_bytes_and_exposes_exact_parts',
             'rejects_packet_over_the_total_bound_before_scanning',
             'reads_fresh_packet_and_rejects_nonpacket_and_missing_files']
    argv = [sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'),
            '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()),
            '--builder', str(args.builder.resolve()), '--module', 'native_packet_decode',
            '--isolated-native', '--fixture', str(fixture), '--output', str(output / 'execution')]
    for test in tests:
        argv.extend(['--test', test])
    result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=600)
    (output / 'execution.stdout').write_bytes(result.stdout)
    (output / 'execution.stderr').write_bytes(result.stderr)
    assert result.returncode == 0, result.stdout + result.stderr
    assert before == [identity(path) for path in inputs], 'reader inputs changed during verification'
    assert fixture_identities == [identity(path) for path in fixtures], 'oracle fixture changed during execution'
    execution = json.loads((output / 'execution/receipt.json').read_text(encoding='utf-8'))
    assert execution['passed'] and execution['nativeTests'] == len(tests)
    receipt = dict(schema='fairyc.candidate-packet-reader/v1', passed=True,
                   sources=before, goOracleCases=len(cases),
                   accepted=sum(row['accepted'] for row in cases),
                   rejected=sum(not row['accepted'] for row in cases),
                   fixtures=fixture_identities, realPacket=identity(packet),
                   authoredTests=len(tests), profiles=['Go-native', 'development', 'release'],
                   execution=identity(output / 'execution/receipt.json'),
                   fullCLIIntegrated=False, goBuildBridgeRemoved=False,
                   publicApiChanged=False, sdkAbiChanged=False, ciChanged=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print(f'packet reader passed: {len(cases)} Go cases, {len(tests)} tests in three engines')


if __name__ == '__main__':
    main()
