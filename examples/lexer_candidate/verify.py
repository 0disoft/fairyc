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
from verify_names import verify_names
from verify_types import verify_types
from verify_modules import verify_modules
from verify_hir import verify_hir
from verify_mir import verify_mir

ROOT = Path(__file__).resolve().parent
KINDS = ['invalid', 'eof', 'trivia', 'identifier', 'keyword', 'integer', 'float', 'string', 'byte-string', 'c-string', 'symbol']
TRIVIA = ['none', 'preamble', 'whitespace', 'line-comment', 'block-comment', 'documentation-line']
HEADER = 'kind\ttrivia\tstart\tend\traw_hex\tcooked_hex\n'
USAGE = 'usage: lexer-candidate --tokens-file <path>\n       lexer-candidate --declarations-file <path>\n       lexer-candidate --check-names-file <path>\n       lexer-candidate --bindings-file <path>\n       lexer-candidate --check-types-file <path>\n       lexer-candidate --check-modules-file <manifest>\n       lexer-candidate --hir-file <path>\n       lexer-candidate --mir-file <path>\n       lexer-candidate --c-file <path>\n'
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


DECLARATION_HEADER = 'kind\tname\tstart\tend\tname_start\tname_end\n'


def verify_declarations(image, reference, directory, rows, run):
    counts = dict(files=0, accepted=0, syntaxErrors=0, lexicalErrors=0, limits=0, unchangedFiles=0)
    for index, fixture in enumerate(reference['fixtures']):
        source = bytes.fromhex(fixture['inputHex'])
        path = directory / ('한글 선언.fy' if index == 1 else f'declarations-{index}.fy')
        path.write_bytes(source)
        argument = str(path.relative_to(ROOT)) if index == 1 else str(path)
        before = sha(path)
        row = run([image, '--declarations-file', argument], rows, 'declarations-' + fixture['name'])
        row['input'] = dict(path=str(path), beforeSHA256=before, afterSHA256=sha(path), bytes=len(source))
        assert before == sha(path), row
        counts['files'] += 1
        counts['unchangedFiles'] += 1
        outcome = fixture['outcome']
        if outcome == 'pass':
            expected = DECLARATION_HEADER + ''.join(f"{d['kind']}\t{d['name']}\t{d['start']}\t{d['end']}\t{d['nameStart']}\t{d['nameEnd']}\n" for d in fixture['records'])
            assert (row['exit'], row['stdout'], row['stderr']) == (0, expected, ''), dict(row, expected=expected)
            for d in fixture['records']:
                assert source[d['nameStart']:d['nameEnd']].decode('ascii') == d['name']
                assert 0 <= d['start'] < d['end'] <= len(source)
            counts['accepted'] += 1
        elif outcome == 'syntax':
            assert row['exit'] == 2 and row['stdout'] == '', row
            expected = []
            for problem in fixture['goldens']:
                start, end = problem['start'], problem['end']
                line, column = location(source, start)
                expected.append(f"{argument}:{line}:{column}: error: {problem['message']} (bytes {start}..{end})")
            assert row['stderr'].splitlines() == expected, dict(row, expected=expected)
            counts['syntaxErrors'] += 1
        elif outcome == 'lexical':
            assert row['exit'] == 2 and row['stdout'] == '', row
            expected = []
            for error in fixture['lexical']:
                start, end = error['Span']['StartByte'], error['Span']['EndByte']
                line, column = location(source, start)
                expected.append(f"{argument}:{line}:{column}: error: {error['Code']} (bytes {start}..{end})")
            assert sorted(row['stderr'].splitlines()) == sorted(expected), row
            counts['lexicalErrors'] += 1
        elif outcome == 'limit':
            assert (row['exit'], row['stdout'], row['stderr']) == (2, '', 'error: parser limit\n'), row
            counts['limits'] += 1
        else:
            raise AssertionError(fixture)
    return counts


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--receipt', type=Path, required=True)
    parser.add_argument('--declarations-reference', type=Path)
    parser.add_argument('--names-reference', type=Path)
    parser.add_argument('--types-reference', type=Path)
    parser.add_argument('--modules-reference', type=Path)
    parser.add_argument('--hir-reference', type=Path)
    parser.add_argument('--mir-reference', type=Path)
    options = parser.parse_args()
    compiler = options.compiler.resolve(strict=True)
    reference_path = options.reference.resolve(strict=True)
    reference = json.loads(reference_path.read_text(encoding='utf-8'))
    assert reference['schema'] == 'fairyc.lexer-file-oracle/v1' and reference['fixtures']
    declarations_path = options.declarations_reference.resolve(strict=True) if options.declarations_reference else None
    declarations_reference = json.loads(declarations_path.read_text(encoding='utf-8')) if declarations_path else None
    if declarations_reference:
        assert declarations_reference['schema'] == 'fairyc.declarations-file-oracle/v1'
    types_path = options.types_reference.resolve(strict=True) if options.types_reference else None
    types_reference = json.loads(types_path.read_text(encoding='utf-8')) if types_path else None
    if types_reference:
        assert types_reference['schema'] == 'fairyc.types-file-oracle/v1' and types_reference['fixtures']
    names_path = options.names_reference.resolve(strict=True) if options.names_reference else None
    names_reference = json.loads(names_path.read_text(encoding='utf-8')) if names_path else None
    if names_reference:
        assert names_reference['schema'] == 'fairyc.names-file-oracle/v1' and names_reference['fixtures']
    modules_path = options.modules_reference.resolve(strict=True) if options.modules_reference else None
    modules_reference = json.loads(modules_path.read_text(encoding='utf-8')) if modules_path else None
    if modules_reference:
        assert modules_reference['schema'] == 'fairyc.modules-file-oracle/v1' and modules_reference['fixtures']
    hir_path = options.hir_reference.resolve(strict=True) if options.hir_reference else None
    hir_reference = json.loads(hir_path.read_text(encoding='utf-8')) if hir_path else None
    if hir_reference:
        assert hir_reference['schema'] == 'fairyc.hir-file-oracle/v1' and hir_reference['fixtures']
    mir_path = options.mir_reference.resolve(strict=True) if options.mir_reference else None
    mir_reference = json.loads(mir_path.read_text(encoding='utf-8')) if mir_path else None
    if mir_reference:
        assert mir_reference['schema'] == 'fairyc.mir-file-oracle/v1' and mir_reference['fixtures']
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

            declaration_counts = verify_declarations(image, declarations_reference, directory, rows, run) if declarations_reference else None

            name_counts = verify_names(image, names_reference, directory, rows, run, sha, location) if names_reference else None

            type_counts = verify_types(image, types_reference, directory, rows, run, sha, location) if types_reference else None
            module_counts = verify_modules(image, modules_reference, directory, rows, run, sha, location) if modules_reference else None
            hir_counts = verify_hir(image, hir_reference, directory, rows, run, sha, location) if hir_reference else None
            mir_counts = verify_mir(image, mir_reference, directory, rows, run, sha, location) if mir_reference else None

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
            if declarations_reference:
                for name, args, expected in errors:
                    declaration_args = ['--declarations-file' if x == '--tokens-file' else x for x in args]
                    row = run([image, *declaration_args], rows, 'declarations-' + name)
                    assert (row['exit'], row['stdout'], row['stderr']) == (2, '', expected), row
            if names_reference:
                for mode in ['--check-names-file', '--bindings-file']:
                    for name, args, expected in errors:
                        name_args = [mode if x == '--tokens-file' else x for x in args]
                        row = run([image, *name_args], rows, 'names-' + name)
                        assert (row['exit'], row['stdout'], row['stderr']) == (2, '', expected), row
                    for source, suffix in [(b'module main', ':1:12: error: expected ; (bytes 11..11)\n'),
                                           (b'module main;\xff', ':1:13: error: E0001 (bytes 12..13)\n')]:
                        path = directory / 'names-malformed.fy'
                        path.write_bytes(source)
                        before = sha(path)
                        row = run([image, mode, path], rows, 'names-malformed')
                        row['input'] = dict(path=str(path),beforeSHA256=before,afterSHA256=sha(path),bytes=len(source))
                        assert before == sha(path)
                        assert (row['exit'], row['stdout'], row['stderr']) == (2, '', str(path) + suffix), row
            if types_reference:
                for name, args, expected in errors:
                    type_args = ['--check-types-file' if x == '--tokens-file' else x for x in args]
                    row = run([image, *type_args], rows, 'types-' + name)
                    assert (row['exit'], row['stdout'], row['stderr']) == (2, '', expected), row
                for source, suffix in [(b'module main', ':1:12: error: expected ; (bytes 11..11)\n'),
                                       (b'module main;\xff', ':1:13: error: E0001 (bytes 12..13)\n')]:
                    path = directory / 'types-malformed.fy'
                    path.write_bytes(source)
                    before = sha(path)
                    row = run([image, '--check-types-file', path], rows, 'types-malformed')
                    assert before == sha(path)
                    assert (row['exit'], row['stdout'], row['stderr']) == (2, '', str(path) + suffix), row
            if hir_reference:
                for name,args,expected in errors:
                    hir_args=['--hir-file' if x=='--tokens-file' else x for x in args]
                    row=run([image,*hir_args],rows,'hir-'+name)
                    assert (row['exit'],row['stdout'],row['stderr'])==(2,'',expected),row
                for source,suffix in [(b'module main',':1:12: error: expected ; (bytes 11..11)\n'),(b'module main;\xff',':1:13: error: E0001 (bytes 12..13)\n')]:
                    path=directory/'hir-malformed.fy';path.write_bytes(source);before=sha(path)
                    row=run([image,'--hir-file',path],rows,'hir-malformed')
                    assert sha(path)==before and (row['exit'],row['stdout'],row['stderr'])==(2,'',str(path)+suffix),row
            if mir_reference:
                for name, args, expected in errors:
                    mir_args = ['--mir-file' if x == '--tokens-file' else x for x in args]
                    row = run([image, *mir_args], rows, 'mir-' + name)
                    assert (row['exit'], row['stdout'], row['stderr']) == (2, '', expected), row
                for source, suffix in [(b'module main', ':1:12: error: expected ; (bytes 11..11)\n'),
                                       (b'module main;\xff', ':1:13: error: E0001 (bytes 12..13)\n')]:
                    path = directory / 'mir-malformed.fy'; path.write_bytes(source); before = sha(path)
                    row = run([image, '--mir-file', path], rows, 'mir-malformed')
                    assert sha(path) == before and (row['exit'], row['stdout'], row['stderr']) == (2, '', str(path) + suffix), row
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
                                 declarations=declaration_counts, names=name_counts, types=type_counts, modules=module_counts, hir=hir_counts, mir=mir_counts, cliExecutions=len(rows) - 1, imagePath=str(saved), imageSHA256=sha(saved),
                                 raw=str(raw), rawSHA256=sha(raw)))
    sources = [ROOT / 'fairy.toml', ROOT / 'fairy.lock', ROOT / 'verify.py', ROOT / 'verify_names.py', ROOT / 'verify_types.py', ROOT / 'verify_modules.py', ROOT / 'verify_hir.py', ROOT / 'verify_mir.py', ROOT / 'verify_c.py', ROOT / 'verify_nominal.py', *sorted((ROOT / 'src').glob('*.fy')), *sorted((ROOT / 'samples').glob('*.fy')), ROOT / 'samples/modules.txt', *sorted((ROOT / 'tests').glob('*.fy'))]
    save(receipt, dict(schema='fairyc.lexer-file-cli-verification/v1', compiler=dict(path=str(compiler), sha256=sha(compiler)),
                       reference=dict(path=str(reference_path), sha256=sha(reference_path)),
                       declarationsReference=dict(path=str(declarations_path), sha256=sha(declarations_path)) if declarations_path else None, common=common,
                       namesReference=dict(path=str(names_path), sha256=sha(names_path)) if names_path else None,
                       typesReference=dict(path=str(types_path), sha256=sha(types_path)) if types_path else None,
                       modulesReference=dict(path=str(modules_path), sha256=sha(modules_path)) if modules_path else None,
                       hirReference=dict(path=str(hir_path), sha256=sha(hir_path)) if hir_path else None,
                       mirReference=dict(path=str(mir_path), sha256=sha(mir_path)) if mir_path else None,
                       sources=[dict(path=str(p.relative_to(ROOT)), sha256=sha(p)) for p in sources], profiles=profiles))
    print('PASS:', sum(p['cliExecutions'] for p in profiles), 'CLI executions;', len(reference['fixtures']), 'Go token reference files/profile; declarations:', declarations_reference and len(declarations_reference['fixtures']), '; names:', names_reference and len(names_reference['fixtures']), '; types:', types_reference and len(types_reference['fixtures']))


if __name__ == '__main__':
    main()
