"""Compare private Fairy manifest results with the executable Go manifest contract."""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8', newline='\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['compiler', 'candidate', 'builder', 'reference', 'output']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--reuse-execution', type=Path, help='verify and reuse native/C observations only with identical inputs')
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    corpus_path = ROOT / 'spec/fixtures/project-manifest/candidate-manifest-documents-v1.json'
    corpus = json.loads(corpus_path.read_bytes())
    assert corpus['schema'] == 'fairyc.candidate-manifest-documents-corpus/v1'
    cases = corpus['cases']
    canonical = []
    for case in cases:
        source = base64.b64decode(case['source'], validate=True)
        if 'sourcePath' in case:
            path = (ROOT / case['sourcePath']).resolve()
            path.relative_to(ROOT)
            assert path.read_bytes() == source, 'canonical fixture drifted'
            canonical.append(path)
    identities = [dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in [corpus_path, ROOT / 'cmd/fairyc-candidate-toml-document-reference/main.go', ROOT / 'internal/projectmanifest/manifest.go', ROOT / 'go.mod', ROOT / 'go.sum', Path(__file__), *canonical]]
    oracle = args.reference.resolve()
    oracle_sha256 = digest(oracle)
    request = output / 'reference-request.json'
    report_path = output / 'reference.json'
    binary = output / 'reference.bin'
    save(request, dict(schema='fairyc.candidate-toml-documents-request/v1', cases=cases))
    argv = [str(oracle), str(request), str(report_path), str(binary)]
    result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=30)
    assert result.returncode == 0 and not result.stdout and not result.stderr, result.stderr
    report = json.loads(report_path.read_bytes())
    assert report['schema'] == 'fairyc.candidate-toml-documents-reference/v1' and len(report['cases']) == len(cases)
    assert [row['name'] for row in report['cases']] == [case['name'] for case in cases]
    if args.reuse_execution:
        execution_path = args.reuse_execution.resolve()
    else:
        execution_path = output / 'execution/receipt.json'
        argv = [sys.executable, '-X', 'utf8', str(ROOT / 'tools/verify-candidate-private-tests.py'), '--compiler', str(args.compiler.resolve()), '--candidate', str(args.candidate.resolve()), '--builder', str(args.builder.resolve()), '--module', 'project_manifest', '--test', 'keeps_owned_manifest_values_and_missing_workspace', '--test', 'matches_go_manifest_documents', '--fixture', str(binary), '--output', str(execution_path.parent)]
        result = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=600)
        assert result.returncode == 0 and not result.stderr, result.stderr
    execution = json.loads(execution_path.read_bytes())
    assert execution['passed'] and execution['module'] == 'project_manifest' and execution['nativeTests'] == 2
    assert execution['native']['passed'] == 2 and execution['native']['failed'] == 0
    assert {row['identity'] for row in execution['native']['tests']} == {'tests::project_manifest::keeps_owned_manifest_values_and_missing_workspace', 'tests::project_manifest::matches_go_manifest_documents'}
    assert all(digest(ROOT / row['path']) == row['sha256'] for row in execution['sourceInputs']), 'execution source or fixture changed'
    assert digest(ROOT / 'examples/lexer_candidate/fairy.lock') == execution['lockSHA256']
    assert all(digest(Path(row['path'])) == row['sha256'] for row in execution['tools'].values())
    for name in ['compiler', 'candidate', 'builder']:
        assert digest(getattr(args, name)) == execution['tools'][name]['sha256'], name
    assert len(execution['candidateExecutions']) == 2 and {row['profile'] for row in execution['candidateExecutions']} == {'development', 'release'}
    assert all(row['exit'] == 0 and digest(Path(row['path'])) == row['sha256'] for row in execution['candidateExecutions'])
    fixture_inputs = [row for row in execution['sourceInputs'] if row['path'].endswith('/reference.bin')]
    assert len(fixture_inputs) == 1 and fixture_inputs[0]['sha256'] == digest(binary), 'oracle bytes differ from executed fixture'
    assert digest(oracle) == oracle_sha256
    assert all(digest(ROOT / row['path']) == row['sha256'] for row in identities)
    receipt = dict(schema='fairyc.candidate-manifest/v1', passed=True, scope='Private closed/required field and decoded value validation on the admitted TOML syntax corpus', sourceInputs=execution['sourceInputs'] + identities, tools=execution['tools'], reference=dict(path=str(oracle), sha256=oracle_sha256, reportSHA256=digest(report_path)), binaryFixtureSHA256=digest(binary), binaryFixtureBytes=binary.stat().st_size, cases=len(cases), accepted=sum(row['manifestAccepted'] for row in report['cases']), nativeTests=2, native=execution['native'], candidateExecutions=execution['candidateExecutions'], candidatePacketSHA256=execution['candidatePacketSHA256'], nativeInvocations=execution['invocations'], lockSHA256=execution['lockSHA256'], executionReused=bool(args.reuse_execution), executionReceiptSHA256=digest(execution_path), manifestParsingComplete=False, projectDiscoveryComplete=False, fullTOMLParity=False, publicApiChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    save(output / 'receipt.json', receipt)
    print(f'Manifest corpus passed: {len(cases)} Go acceptance/reason/value comparisons; native 2 and both candidate C profiles')


if __name__ == '__main__':
    main()
