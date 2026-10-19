"""Compare authored MSVC discovery against the live Go resolver on Windows."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent


def identity(path):
    return dict(path=str(path.resolve()), sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('compiler', 'candidate', 'builder', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    fixture = output / 'toolchain.txt'
    sdk = Path(str(fixture) + '.sdk')
    for version in ('10.1.0', '10.2.0', '10.9.0'):
        for leaf in ('ucrt', 'shared', 'um'):
            (sdk / 'Include' / version / leaf).mkdir(parents=True, exist_ok=True)
        for leaf in ('ucrt', 'um'):
            if version != '10.9.0' or leaf != 'um':
                (sdk / 'Lib' / version / leaf / 'x64').mkdir(parents=True, exist_ok=True)
    env = os.environ.copy()
    env['FAIRYC_MSVC_ORACLE'] = str(fixture)
    oracle = subprocess.run(['go', 'test', './internal/cdriver', '-run',
                             '^TestCandidateMSVCToolchainOracle$', '-count=1', '-v'],
                            cwd=ROOT, env=env, capture_output=True, timeout=60)
    (output / 'oracle.stdout').write_bytes(oracle.stdout)
    (output / 'oracle.stderr').write_bytes(oracle.stderr)
    assert oracle.returncode == 0, oracle.stdout + oracle.stderr
    before = identity(fixture)
    argv = [sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'),
            '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()),
            '--builder', str(args.builder.resolve()), '--module', 'windows_msvc_toolchain',
            '--candidate-only', '--fixture', str(fixture), '--output', str(output / 'execution')]
    for test in ('matches_real_go_toolchain_and_child_environment',
                 'selects_latest_complete_sdk_and_rejects_missing_sdk',
                 'validates_components_and_replaces_environment_without_mutation'):
        argv.extend(['--test', test])
    result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=300)
    (output / 'execution.stdout').write_bytes(result.stdout)
    (output / 'execution.stderr').write_bytes(result.stderr)
    assert result.returncode == 0, result.stdout + result.stderr
    assert before == identity(fixture), 'Go oracle changed during execution'
    execution = output / 'execution/receipt.json'
    assert json.loads(execution.read_text(encoding='utf-8'))['passed']
    receipt = dict(schema='fairyc.candidate-msvc-toolchain/v1', passed=True,
                   goOracle=before, execution=identity(execution), authoredTests=3,
                   profiles=['development', 'release'], frontendGoInvokedByResolver=False,
                   parentEnvironmentModified=False, cliIntegrated=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print('MSVC discovery passed: real Go paths/environment and three authored tests in both C profiles')


if __name__ == '__main__':
    main()
