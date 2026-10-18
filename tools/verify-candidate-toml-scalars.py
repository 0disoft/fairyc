"""Compare Fairy scalar classification with pinned Go, retaining native/C evidence."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, data):
    path.write_text(json.dumps(data, indent=2) + '\n', encoding='utf-8', newline='\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['compiler', 'candidate', 'builder', 'reference', 'output']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--reuse-execution', type=Path)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    corpus_path = ROOT / 'spec/fixtures/project-manifest/candidate-toml-scalars-v1.json'
    corpus = json.loads(corpus_path.read_bytes())
    assert corpus['schema'] == 'fairyc.candidate-toml-scalars-corpus/v1'
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    request, fixture, report_path = [output / name for name in ['request.json', 'reference.bin', 'reference.json']]
    oracle = args.reference.resolve()
    oracle_digest = digest(oracle)
    save(request, dict(schema='fairyc.candidate-toml-scalars-request/v1', cases=corpus['cases']))
    result = subprocess.run([str(oracle), str(request), str(fixture), str(report_path)], capture_output=True, timeout=30)
    assert result.returncode == 0 and not result.stdout and not result.stderr, result.stderr
    report = json.loads(report_path.read_bytes())
    assert report['schema'] == 'fairyc.candidate-toml-scalars-reference/v1'
    assert [row['name'] for row in report['cases']] == [row['name'] for row in corpus['cases']]
    if args.reuse_execution:
        receipt_path = args.reuse_execution.resolve()
    else:
        receipt_path = output / 'execution/receipt.json'
        argv = [sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'), '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()), '--builder', str(args.builder.resolve()), '--module', 'project_toml_scalars', '--test', 'bounds_integer_float_calendar_and_token_consumption', '--test', 'matches_pinned_go_scalar_kinds', '--fixture', str(fixture), '--output', str(receipt_path.parent)]
        result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=600)
        assert result.returncode == 0 and not result.stderr, result.stderr
    execution = json.loads(receipt_path.read_bytes())
    assert execution['passed'] and execution['module'] == 'project_toml_scalars' and execution['nativeTests'] == 2
    assert execution['native']['passed'] == 2 and execution['native']['failed'] == 0
    assert all(digest(ROOT / row['path']) == row['sha256'] for row in execution['sourceInputs'])
    binary_inputs = [row for row in execution['sourceInputs'] if row['path'].endswith('/reference.bin')]
    assert len(binary_inputs) == 1 and binary_inputs[0]['sha256'] == digest(fixture)
    assert digest(ROOT / 'examples/lexer_candidate/fairy.lock') == execution['lockSHA256']
    for name in ['compiler', 'candidate', 'builder']:
        assert digest(getattr(args, name)) == execution['tools'][name]['sha256']
    assert len(execution['candidateExecutions']) == 2 and {row['profile'] for row in execution['candidateExecutions']} == {'development', 'release'}
    assert all(row['exit'] == 0 and digest(Path(row['path'])) == row['sha256'] for row in execution['candidateExecutions'])
    assert digest(oracle) == oracle_digest
    inputs = execution['sourceInputs'] + [dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in [corpus_path, ROOT / 'tools/candidate/fairyc-candidate-toml-scalar-reference/main.go', ROOT / 'go.mod', ROOT / 'go.sum', Path(__file__)]]
    receipt = dict(schema='fairyc.candidate-toml-scalars/v1', passed=True, scope='Integer/float/date-time syntax and range classification for manifest error parity; values stay typed raw atoms', sourceInputs=inputs, cases=len(report['cases']), kinds={str(kind):sum(row['kind'] == kind for row in report['cases']) for kind in [0,6,7,8]}, reference=dict(path=str(oracle), sha256=oracle_digest, reportSHA256=digest(report_path)), fixtureSHA256=digest(fixture), fixtureBytes=fixture.stat().st_size, execution=execution, executionReused=bool(args.reuse_execution), fullTOMLParity=False, publicApiChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    save(output / 'receipt.json', receipt)
    print(f'Scalar corpus passed: {len(report["cases"])} Go classifications; native 2 and both C profiles')


if __name__ == '__main__':
    main()
