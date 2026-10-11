"""Record scoped Fairy document checks and exact pinned Go decoder differences."""
import argparse
import ast
import base64
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8', newline='\n')


def authored_cases(source):
    # This extracts test input bytes and their authored acceptance assertions;
    # it implements no TOML parsing or manifest validation predicates.
    text = source.read_text(encoding='utf-8')
    literal = r'(b"(?:[^"\\]|\\.)*")'
    cases = []
    for prefix, pattern in [('manifest', r'\bmanifest\(' + literal + r'\)'), ('document', r'let ref u8\[\] source=' + literal + r';'), ('empty', r'document::parse\(' + literal + r'\)')]:
        for index, raw in enumerate(re.findall(pattern, text)):
            cases.append(dict(name=f'{prefix}-{index}', source=base64.b64encode(ast.literal_eval(raw)).decode(), candidateAccepted=True))
    for index, (raw, code) in enumerate(re.findall(r'\brejected\(' + literal + r',([124])\)', text)):
        cases.append(dict(name=f'rejected-{index}', source=base64.b64encode(ast.literal_eval(raw)).decode(), candidateAccepted=False, errorCode=int(code)))
    return cases


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--native-receipt', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    source = ROOT / 'examples/lexer_candidate/tests/project_toml_document.fy'
    corpus_path = ROOT / 'spec/fixtures/project-manifest/candidate-toml-documents-v1.json'
    corpus = json.loads(corpus_path.read_bytes())
    assert corpus['schema'] == 'fairyc.candidate-toml-documents-corpus/v1'
    cases = authored_cases(source)
    assert cases == corpus['cases'], 'authored inputs or assertions differ from corpus'
    native = json.loads(args.native_receipt.read_bytes())
    assert native['passed'] and native['module'] == 'project_toml_document' and native['nativeTests'] == 3
    assert native['native']['passed'] == 3 and native['native']['failed'] == 0
    assert len(native['candidateExecutions']) == 2 and all(row['exit'] == 0 for row in native['candidateExecutions'])
    assert all(digest(ROOT / row['path']) == row['sha256'] for row in native['sourceInputs']), 'native source inputs changed'
    assert digest(ROOT / 'examples/lexer_candidate/fairy.lock') == native['lockSHA256']
    assert all(digest(Path(row['path'])) == row['sha256'] for row in native['tools'].values()), 'native tool changed'
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    request = output / 'reference-request.json'
    report_path = output / 'reference.json'
    save(request, dict(schema='fairyc.candidate-toml-documents-request/v1', cases=cases))
    oracle = args.reference.resolve()
    oracle_sha256 = digest(oracle)
    result = subprocess.run([str(oracle), str(request), str(report_path)], capture_output=True, timeout=30)
    assert result.returncode == 0 and not result.stdout and not result.stderr, result.stderr
    report = json.loads(report_path.read_bytes())
    assert report['schema'] == 'fairyc.candidate-toml-documents-reference/v1' and len(report['cases']) == len(cases)
    policy = set()
    outside = set()
    differences = []
    for case, row in zip(cases, report['cases']):
        assert case['name'] == row['name']
        if case['name'].startswith('manifest-'):
            assert row['manifestAccepted'], row
        if case['candidateAccepted'] != row['accepted']:
            data = base64.b64decode(case['source'], validate=True)
            assert not case['candidateAccepted'] and row['accepted'] and data in policy | outside, (case, row)
            differences.append(dict(name=case['name'], source=case['source'], candidateAccepted=False, goAccepted=True, kind='table-definition-policy' if data in policy else 'outside-manifest-value-kinds'))
    assert len(differences) == len(policy) + len(outside), 'known difference set changed'
    assert digest(oracle) == oracle_sha256
    inputs = native['sourceInputs'] + [dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in [corpus_path, ROOT / 'cmd/fairyc-candidate-toml-document-reference/main.go', Path(__file__)]]
    receipt = dict(schema='fairyc.candidate-toml-document/v1', passed=True, scope='Manifest value-kind syntax and document graph; no complete TOML or manifest parser parity claim', sourceInputs=inputs, nativeTests=3, native=native['native'], nativeReceiptSHA256=digest(args.native_receipt), candidateExecutions=native['candidateExecutions'], candidatePacketSHA256=native['candidatePacketSHA256'], tools=native['tools'], lockSHA256=native['lockSHA256'], reference=dict(path=str(oracle), sha256=oracle_sha256, reportSHA256=digest(report_path)), cases=len(cases), goMatches=len(cases)-len(differences), differences=differences, nativeInvocations=native['invocations'], fullTOMLParity=False, manifestParsingComplete=False, projectDiscoveryComplete=False, publicApiChanged=False, dbChanged=False, ciChanged=False, pushed=False)
    save(output / 'receipt.json', receipt)
    print(f'Document scope passed: {len(cases)} authored inputs; {len(cases)-len(differences)} Go matches; {len(differences)} explicitly recorded differences; native 3 and candidate C profiles pass')


if __name__ == '__main__':
    main()
