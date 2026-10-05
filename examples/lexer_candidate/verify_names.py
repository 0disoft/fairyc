"""Name-check CLI tests; Go is an independent test reference, never product logic."""
import re


def verify_names(image, reference, directory, rows, run, sha, location):
    counts = dict(files=0, accepted=0, unresolved=0, duplicate=0, unsupported=0, limits=0, unchangedFiles=0)
    for index, fixture in enumerate(reference['fixtures']):
        source = bytes.fromhex(fixture['inputHex'])
        path = directory / ('한글 이름.fy' if index == 1 else f'names-{index}.fy')
        path.write_bytes(source)
        before = sha(path)
        row = run([image, '--check-names-file', path], rows, 'names-' + fixture['name'])
        row['input'] = dict(path=str(path), beforeSHA256=before, afterSHA256=sha(path), bytes=len(source))
        assert before == sha(path), row
        counts['files'] += 1
        counts['unchangedFiles'] += 1
        outcome = fixture['outcome']
        if outcome == 'pass':
            assert (row['exit'], row['stdout'], row['stderr']) == (0, 'names ok\n', ''), row
            counts['accepted'] += 1
        else:
            assert row['exit'] == 2 and row['stdout'] == '', row
            expected = fixture['expected']
            if expected:
                start, end = expected['start'], expected['end']
                line, column = location(source, start)
                message = f"{path}:{line}:{column}: error: {expected['message']} (bytes {start}..{end})\n"
                assert row['stderr'] == message, dict(row, expected=message)
                counts[outcome] += 1
            else:
                message = 'unsupported name-check syntax' if outcome == 'unsupported' else 'name-check limit'
                match = re.fullmatch(re.escape(str(path)) + r':(\d+):(\d+): error: ' + message + r' \(bytes (\d+)\.\.(\d+)\)\n', row['stderr'])
                assert match, row
                line, column, start, end = map(int, match.groups())
                assert 0 <= start < end <= len(source) and location(source, start) == (line, column), row
                counts['unsupported' if outcome == 'unsupported' else 'limits'] += 1
    return counts
