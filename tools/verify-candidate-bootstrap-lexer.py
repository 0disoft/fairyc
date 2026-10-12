"""Compare consecutive Fairy compiler token streams with the current Go oracle."""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import struct
import subprocess
import time


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--fixtures', type=Path, required=True)
    parser.add_argument('--stage1', type=Path, required=True)
    parser.add_argument('--stage2', type=Path)
    parser.add_argument('--seed', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    fixtures_identity, reference_identity = digest(args.fixtures), digest(args.reference)
    args.output.mkdir(parents=True, exist_ok=True)
    req = json.loads(args.fixtures.read_bytes())
    assert req['schema'] == 'fairyc.candidate-lexer-reference-request/v1'
    assert 0 < len(req['cases']) <= 2048
    expected_path = args.output / 'go-reference.json'
    result = subprocess.run([str(args.reference.resolve()), str(args.fixtures.resolve()), str(expected_path.resolve())], capture_output=True, timeout=60)
    assert result.returncode == 0 and not result.stdout and not result.stderr, result.stderr
    oracle = json.loads(expected_path.read_bytes())
    assert oracle['schema'] == 'fairyc.candidate-lexer-reference/v1'
    assert [c['name'] for c in req['cases']] == [c['name'] for c in oracle['cases']]
    stream = bytearray()
    for case in req['cases']:
        data = base64.b64decode(case['source'], validate=True)
        assert len(data) <= 32768
        mode = int(case['previous']) + (2 if 'limits' in case else 0)
        stream.extend(struct.pack('<BH', mode, len(data)))
        if 'limits' in case:
            limits = case['limits']
            stream.extend(struct.pack('<IIII', *[limits[k] for k in ('MaxBytes', 'MaxTokens', 'MaxDiagnostics', 'MaxWorkUnits')]))
        stream.extend(data)
    want = b''.join(base64.b64decode(c['stdout'], validate=True) for c in oracle['cases'])
    assert len(want) <= 16 << 20
    programs = [('stage1', args.stage1)]
    if args.stage2:
        programs.append(('stage2', args.stage2))
    if args.seed:
        programs.insert(0, ('input-seed', args.seed))
    malformed = [b'\x04', b'\0', b'\0\0', b'\0\1\0', b'\2\0\0', b'\2\0\0\1', b'\0\1\x80']
    rows = []
    for label, exe in programs:
        image_identity = digest(exe)
        start = time.monotonic()
        run = subprocess.run([str(exe.resolve())], input=stream, capture_output=True, timeout=60)
        assert run.returncode == 0 and not run.stderr, (label, run.returncode, run.stderr)
        if run.stdout != want:
            at = next((i for i, pair in enumerate(zip(run.stdout, want)) if pair[0] != pair[1]), min(len(run.stdout), len(want)))
            raise AssertionError((label, 'canonical token stream mismatch', at, run.stdout[at:at+100], want[at:at+100]))
        failures = []
        for i, data in enumerate(malformed):
            bad = subprocess.run([str(exe.resolve())], input=data, capture_output=True, timeout=10)
            expected = 4 if i == len(malformed)-1 else 8
            assert bad.returncode == expected and not bad.stdout and not bad.stderr, (label, i, bad.returncode, bad.stdout, bad.stderr)
            failures.append(dict(case=i, exit=bad.returncode))
        assert digest(exe) == image_identity, 'compiler image changed during verification'
        rows.append(dict(stage=label, imageSHA256=image_identity, seconds=time.monotonic()-start, cases=len(req['cases']), stdoutBytes=len(run.stdout), stdoutSHA256=hashlib.sha256(run.stdout).hexdigest(), malformedFrames=failures))
    assert digest(args.fixtures) == fixtures_identity and digest(args.reference) == reference_identity, 'oracle inputs changed during verification'
    receipt = dict(schema='fairyc.candidate-bootstrap-lexer-differential/v1', passed=True, fixturesSHA256=fixtures_identity, referenceImageSHA256=reference_identity, referenceReportSHA256=digest(expected_path), inputSHA256=hashlib.sha256(stream).hexdigest(), inputBytes=len(stream), cases=len(req['cases']), oracle='Current Go lexer canonical tokens, cooked bytes, diagnostics and spans', observations=rows, scope='Lexer and private framing only; parser, typing and executable behavior require separate differential evidence')
    (args.output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8', newline='\n')
    print('Canonical lexer differential passed:', len(req['cases']), 'cases per compiler')


if __name__ == '__main__':
    main()
