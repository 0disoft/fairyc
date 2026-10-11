"""Compare private Fairy table-context acceptance with the pinned Go decoder."""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import struct
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8', newline='\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('compiler', 'candidate', 'builder', 'reference', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--reuse-execution', type=Path)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    corpus_path = ROOT / 'spec/fixtures/project-manifest/candidate-toml-table-contexts-v1.json'
    corpus = json.loads(corpus_path.read_bytes())
    assert corpus['schema'] == 'fairyc.candidate-toml-table-contexts-corpus/v1'
    cases = corpus['cases']
    assert 0 < len(cases) <= 2048 and len({r['name'] for r in cases}) == len(cases)
    request = output / 'request.json'
    report_path = output / 'reference.json'
    binary = output / 'reference.bin'
    save(request, dict(schema='fairyc.candidate-toml-documents-request/v1', cases=cases))
    oracle = args.reference.resolve()
    oracle_sha = digest(oracle)
    result = subprocess.run([str(oracle), str(request), str(report_path)], capture_output=True, timeout=30)
    assert result.returncode == 0 and not result.stdout and not result.stderr, result.stderr
    report = json.loads(report_path.read_bytes())
    assert report['schema'] == 'fairyc.candidate-toml-documents-reference/v1'
    assert [r['name'] for r in report['cases']] == [r['name'] for r in cases]
    wire = bytearray(b'FYTBL001\n')
    for case, row in zip(cases, report['cases']):
        source = base64.b64decode(case['source'], validate=True)
        assert len(source) <= 65535 and isinstance(row['accepted'], bool)
        wire += struct.pack('<HB', len(source), row['accepted']) + source
    assert len(wire) <= 65536
    binary.write_bytes(wire)
    execution_path = args.reuse_execution.resolve() if args.reuse_execution else output / 'execution/receipt.json'
    if not args.reuse_execution:
        argv = [sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'), '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()), '--builder', str(args.builder.resolve()), '--module', 'project_toml_compatibility', '--test', 'matches_pinned_go_table_context_acceptance', '--fixture', str(binary), '--output', str(execution_path.parent)]
        result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=600)
        assert result.returncode == 0 and not result.stderr, result.stderr
    execution = json.loads(execution_path.read_bytes())
    assert execution['passed'] and execution['module'] == 'project_toml_compatibility'
    assert execution['nativeTests'] == 1 and execution['native']['passed'] == 1 and execution['native']['failed'] == 0
    assert {r['identity'] for r in execution['native']['tests']} == {'tests::project_toml_compatibility::matches_pinned_go_table_context_acceptance'}
    assert len(execution['candidateExecutions']) == 2 and all(r['exit'] == 0 for r in execution['candidateExecutions'])
    assert all(digest(ROOT / r['path']) == r['sha256'] for r in execution['sourceInputs'])
    assert digest(ROOT / 'examples/lexer_candidate/fairy.lock') == execution['lockSHA256']
    for name in ('compiler', 'candidate', 'builder'):
        assert digest(getattr(args, name).resolve()) == execution['tools'][name]['sha256']
    assert digest(oracle) == oracle_sha
    inputs = execution['sourceInputs'] + [dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in (corpus_path, ROOT / 'cmd/fairyc-candidate-toml-document-reference/main.go', ROOT / 'go.mod', ROOT / 'go.sum', Path(__file__))]
    receipt = dict(schema='fairyc.candidate-toml-contexts/v1', passed=True, sourceInputs=inputs, cases=len(cases), accepted=sum(r['accepted'] for r in report['cases']), acceptanceOnly=True, fullTOMLParity=False, manifestParsingComplete=False, projectDiscoveryComplete=False, reference=dict(sha256=oracle_sha, reportSHA256=digest(report_path)), nativeTests=1, native=execution['native'], candidateExecutions=execution['candidateExecutions'], tools=execution['tools'], lockSHA256=execution['lockSHA256'], executionReceiptSHA256=digest(execution_path), publicApiChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    save(output / 'receipt.json', receipt)
    print(f'Table contexts passed: {len(cases)} Go acceptance comparisons; native 1 and both candidate C profiles')


if __name__ == '__main__':
    main()
