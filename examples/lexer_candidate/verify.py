#!/usr/bin/env python3
"""Verify the file CLI against a fixture receipt produced by the Go lexer."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent
KINDS = ['invalid', 'eof', 'trivia', 'identifier', 'keyword', 'integer', 'float', 'string', 'byte-string', 'c-string', 'symbol']
TRIVIA = ['none', 'preamble', 'whitespace', 'line-comment', 'block-comment', 'documentation-line']
HEADER = 'kind\ttrivia\tstart\tend\traw_hex\tcooked_hex\n'
USAGE = 'usage: lexer-candidate --tokens-file <path>\n'
HELP = USAGE + '       lexer-candidate --help\nNo arguments: private binary stdin protocol.\n'


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def save(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n', encoding='utf-8', newline='\n')


def location(source, offset):
    breaks = list(re.finditer(rb'\r\n|\r|\n', source[:offset]))
    return len(breaks) + 1, offset - (breaks[-1].end() if breaks else 0) + 1


def reference_output(fixture):
    return HEADER + ''.join(f"{KINDS[t['kind']]}\t{TRIVIA[t['trivia']]}\t{t['start']}\t{t['end']}\t{t['rawHex']}\t{t['cookedHex']}\n"
                            for t in fixture['tokens'])


def reconstruct(stdout, source):
    assert stdout.startswith(HEADER)
    reconstructed, position, eof = bytearray(), 0, 0
    lines = stdout[len(HEADER):].splitlines()
    for index, line in enumerate(lines):
        kind, trivia, start, end, raw, cooked = line.split('\t')
        start, end = int(start), int(end)
        raw = bytes.fromhex(raw)
        assert start == position and 0 <= start <= end <= len(source)
        assert raw == source[start:end] and len(raw) == end - start
        if kind == 'eof':
            assert index == len(lines) - 1 and start == end == len(source)
            eof += 1
        else:
            assert end > start
            reconstructed.extend(raw)
        position = end
    assert eof == 1 and bytes(reconstructed) == source


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--receipt', type=Path, required=True)
    options = parser.parse_args()
    compiler = options.compiler.resolve(strict=True)
    reference_path = options.reference.resolve(strict=True)
    reference = json.loads(reference_path.read_text(encoding='utf-8'))
    assert reference['schema'] == 'fairyc.lexer-file-oracle/v1' and reference['fixtures']
    receipt = options.receipt.resolve()
    receipt.parent.mkdir(parents=True, exist_ok=True)
    common, profiles = [], []

    def run(argv, rows, name, stdin=b''):
        process = subprocess.run([str(x) for x in argv], cwd=ROOT, input=stdin,
                                 capture_output=True, timeout=60)
        row = dict(case=name, argv=[str(x) for x in argv], cwd=str(ROOT),
                   exit=process.returncode, stdout=process.stdout.decode('utf-8'),
                   stderr=process.stderr.decode('utf-8'))
        if stdin:
            row['stdinHex'] = stdin.hex()
        rows.append(row)
        return row

    for args in (['--version'], ['check'], ['check', '--locked']):
        assert run([compiler, *args], common, 'compiler ' + ' '.join(args))['exit'] == 0
    build = ROOT / 'build'
    build.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='file-cli-', dir=build) as temp:
        directory = Path(temp)
        for profile, flags in [('dev', []), ('release', ['--release'])]:
            rows = []
            assert run([compiler, 'build', *flags, '--locked'], rows, 'build')['exit'] == 0
            image = build / ('lexer-candidate.exe' if os.name == 'nt' else 'lexer-candidate')
            saved = receipt.parent / (profile + image.suffix)
            shutil.copy2(image, saved)
            unchanged = successes = diagnostics = limits = 0
            for index, fixture in enumerate(reference['fixtures']):
                source = bytes.fromhex(fixture['inputHex'])
                path = directory / ('한글 소스.fy' if index == 3 else f'{index}.fy')
                path.write_bytes(source)
                argument = str(path.relative_to(ROOT)) if index == 3 else str(path)
                before = sha(path)
                row = run([image, '--tokens-file', argument], rows, fixture['name'])
                row['input'] = dict(path=str(path), beforeSHA256=before, afterSHA256=sha(path), bytes=len(source))
                assert before == sha(path), row
                unchanged += 1
                expected_errors = fixture['diagnostics'] or []
                if fixture['name'] == 'output-limit':
                    assert len(reference_output(fixture).encode('utf-8')) > 1048576
                    assert (row['exit'], row['stdout'], row['stderr']) == (2, '', 'error: output limit (1048576 bytes)\n'), row
                    limits += 1
                elif expected_errors:
                    assert row['exit'] == 2 and row['stdout'] == '', row
                    expected = []
                    for error in expected_errors:
                        start, end = error['Span']['StartByte'], error['Span']['EndByte']
                        line, column = location(source, start)
                        expected.append(f"{argument}:{line}:{column}: error: {error['Code']} (bytes {start}..{end})")
                    # The owned Stream exposes recognition order; Go returns sorted
                    # diagnostics. Compare records as a multiset, preserving duplicates.
                    assert sorted(row['stderr'].splitlines()) == sorted(expected), row
                    diagnostics += 1
                else:
                    assert (row['exit'], row['stdout'], row['stderr']) == (0, reference_output(fixture), ''), row
                    reconstruct(row['stdout'], source)
                    successes += 1

            errors = [('missing-argument', ['--tokens-file'], USAGE),
                      ('unknown-option', ['--unknown'], USAGE),
                      ('extra-argument', ['--tokens-file', 'a', 'b'], USAGE),
                      ('missing-file', ['--tokens-file', str(directory / 'missing.fy')], 'error: input file read failed\n'),
                      ('directory', ['--tokens-file', str(directory)], 'error: input file read failed\n'),
                      ('empty-path', ['--tokens-file', ''], 'error: input file read failed\n'),
                      ('path-limit', ['--tokens-file', 'a' * 4096], 'error: file path limit (4095 UTF-8 bytes)\n')]
            over = directory / 'oversize.fy'
            over.write_bytes(b'a' * 32769)
            errors.append(('source-limit-exceeded', ['--tokens-file', str(over)], 'error: source limit (32768 bytes)\n'))
            for name, args, expected in errors:
                before = sha(over)
                row = run([image, *args], rows, name)
                assert (row['exit'], row['stdout'], row['stderr']) == (2, '', expected), row
                assert sha(over) == before
            row = run([image, '--help'], rows, 'help')
            assert (row['exit'], row['stdout'], row['stderr']) == (0, HELP, ''), row
            row = run([image], rows, 'legacy-empty-stdin')
            assert (row['exit'], row['stdout'], row['stderr']) == (0, '', ''), row
            # Empty source frame remains the exact legacy numeric observation.
            row = run([image], rows, 'legacy-empty-frame', b'\x00\x00\x00')
            assert (row['exit'], row['stdout'], row['stderr']) == (0, '1 0 0 0 0 0 ', ''), row
            row = run([image], rows, 'legacy-invalid-frame', b'\x04')
            assert row['exit'] == 8 and row['stdout'] == row['stderr'] == '', row
            raw = receipt.with_name(receipt.stem + '.' + profile + '.json')
            save(raw, dict(profile=profile, runs=rows))
            profiles.append(dict(profile=profile, passed=True, fileCases=len(reference['fixtures']),
                                 accepted=successes, diagnosticCases=diagnostics, limitCases=limits, unchangedFiles=unchanged,
                                 cliExecutions=len(rows) - 1, imagePath=str(saved), imageSHA256=sha(saved),
                                 raw=str(raw), rawSHA256=sha(raw)))
    sources = [ROOT / 'fairy.toml', ROOT / 'fairy.lock', ROOT / 'verify.py', *sorted((ROOT / 'src').glob('*.fy')), *sorted((ROOT / 'samples').glob('*.fy'))]
    save(receipt, dict(schema='fairyc.lexer-file-cli-verification/v1', compiler=dict(path=str(compiler), sha256=sha(compiler)),
                       reference=dict(path=str(reference_path), sha256=sha(reference_path)), common=common,
                       sources=[dict(path=str(p.relative_to(ROOT)), sha256=sha(p)) for p in sources], profiles=profiles))
    print('PASS:', sum(p['cliExecutions'] for p in profiles), 'CLI executions;', len(reference['fixtures']), 'Go reference files/profile')


if __name__ == '__main__':
    main()
