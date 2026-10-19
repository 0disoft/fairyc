"""Verify authored runtime packaging against complete Go C bytes and asset tampering."""
import argparse
import hashlib
import importlib.util
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
    for name in ('compiler', 'candidate', 'builder', 'packet', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    spec = importlib.util.spec_from_file_location('runtime_assets', ROOT / 'tools/package-candidate-runtime-assets.py')
    packer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(packer)
    data, pin = packer.pack_and_pin()
    assert packer.PIN.read_text(encoding='utf-8') == pin, 'stale asset pin'
    fixture = output / 'cases.bin'
    asset = Path(str(fixture)+'.assets')
    asset.write_bytes(data)
    sources = [ROOT / path for path in packer.SOURCES + [
        'examples/lexer_candidate/src/native_runtime_assets.fy',
        'examples/lexer_candidate/src/native_runtime_asset_pin.fy',
        'examples/lexer_candidate/tests/native_runtime_assets.fy',
        'examples/lexer_candidate/src/native_runtime.fy',
        'examples/lexer_candidate/src/native_runtime_contract.fy',
        'internal/candidatebuild/runtime_selection_oracle_test.go',
        'internal/candidatebuild/runtime_packaging_oracle_test.go',
        'internal/candidatebuild/build.go', 'internal/cruntime/runtime.go',
        'tools/package-candidate-runtime-assets.py',
    ]] + [Path(__file__).resolve(), args.packet.resolve()]
    before = [identity(path) for path in sources]
    env = os.environ.copy()
    env['FAIRYC_RUNTIME_PACKAGING_ORACLE'] = str(fixture)
    env['FAIRYC_RUNTIME_PACKET'] = str(args.packet.resolve())
    oracle = subprocess.run(['go', 'test', './internal/candidatebuild', '-run',
                             '^TestCandidateRuntimePackagingOracle$', '-count=1', '-v'],
                            cwd=ROOT, env=env, capture_output=True, timeout=90)
    (output / 'oracle.stdout').write_bytes(oracle.stdout)
    (output / 'oracle.stderr').write_bytes(oracle.stderr)
    assert oracle.returncode == 0, oracle.stdout + oracle.stderr
    with fixture.open('rb') as stream:
        assert stream.read(len(b'FYCRTPACKTEST1\n')) == b'FYCRTPACKTEST1\n'
        count = struct.unpack('<I', stream.read(4))[0]
    assert 200 < count < 1024
    fixtures = [identity(path) for path in (fixture, asset)]
    argv = [sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'),
            '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()),
            '--builder', str(args.builder.resolve()), '--module', 'native_runtime_assets',
            '--isolated-native', '--fixture', str(fixture), '--output', str(output / 'execution')]
    for test in ('matches_complete_go_runtime_bytes_and_admission',
                 'rejects_tampered_reordered_truncated_and_extended_assets'):
        argv.extend(['--test', test])
    result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=600)
    (output / 'execution.stdout').write_bytes(result.stdout)
    (output / 'execution.stderr').write_bytes(result.stderr)
    assert result.returncode == 0, result.stdout + result.stderr
    assert before == [identity(path) for path in sources], 'packaging source changed'
    assert fixtures == [identity(path) for path in (fixture, asset)], 'packaging fixture changed'
    execution = output / 'execution/receipt.json'
    assert json.loads(execution.read_text(encoding='utf-8'))['passed']
    receipt = dict(schema='fairyc.candidate-runtime-packaging/v1', passed=True,
                   sources=before, fixtures=fixtures, goOracleCases=count,
                   comparison='complete runtime C bytes and exact admission diagnostics',
                   rawAssets=len(packer.SOURCES), assetPack=identity(asset), execution=identity(execution),
                   profiles=['Go-native', 'development', 'release'], authoredTests=2,
                   runtimeGoInvokedByAuthoredPackager=False, assetDelivery='explicit pinned distribution pack',
                   cliIntegrated=False, fullCliFixedPointRefreshed=False, linuxExecuted=False,
                   publicApiChanged=False, sdkAbiChanged=False, databaseChanged=False, ciChanged=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8')
    print(f'Runtime packaging passed: {count} exact Go byte comparisons in three execution paths')


if __name__ == '__main__':
    main()
