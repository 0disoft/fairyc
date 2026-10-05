"""Compare the Fairy type CLI with the Go semantic oracle, as test data only."""
import re


def verify_types(image, reference, directory, rows, run, sha, location):
    counts = dict(files=0, accepted=0, errors=0, unsupported=0, limits=0, unchangedFiles=0)
    codes = {'type mismatch': {'E1006', 'E1009'}, 'integer literal out of range': {'E1013'},
             'expected bool operand': {'E1007'}, 'assignment target is immutable': {'E1008'},
             'call argument count mismatch': {'E1010'}, 'unresolved name': {'E0101'}, 'loop control outside loop': {'E1012'}}
    for index, fixture in enumerate(reference['fixtures']):
        source = bytes.fromhex(fixture['inputHex'])
        path = directory / ('한글 타입.fy' if index == 12 else f'types-{index}.fy')
        path.write_bytes(source)
        before = sha(path)
        row = run([image, '--check-types-file', path], rows, 'types-' + fixture['name'])
        row['input'] = dict(path=str(path), beforeSHA256=before, afterSHA256=sha(path), bytes=len(source))
        assert before == sha(path), row
        counts['files'] += 1
        counts['unchangedFiles'] += 1
        outcome = fixture['outcome']
        if outcome == 'pass':
            assert (row['exit'], row['stdout'], row['stderr']) == (0, 'types ok\n', ''), row
            counts['accepted'] += 1
            continue
        assert row['exit'] == 2 and row['stdout'] == '', row
        message = fixture['message']
        match = re.fullmatch(re.escape(str(path)) + r':(\d+):(\d+): error: ' + re.escape(message) +
                             r' \(bytes (\d+)\.\.(\d+)\)\n', row['stderr'])
        assert match, row
        line, column, start, end = map(int, match.groups())
        assert 0 <= start < end <= len(source) and location(source, start) == (line, column), row
        if outcome == 'error':
            diagnostics = [d['Primary'] for d in (fixture['diagnostics'] or []) if d['Code'] in codes.get(message, set())]
            spans = diagnostics or [d['Span'] for d in (fixture['unsupported'] or [])]
            assert any((start, end) == (s['StartByte'], s['EndByte']) for s in spans), (row, spans)
            counts['errors'] += 1
        elif outcome == 'limit':
            assert (start, end) == (0, len(source)), row
            counts['limits'] += 1
        else:
            assert outcome == 'unsupported', fixture
            counts['unsupported'] += 1
    return counts
