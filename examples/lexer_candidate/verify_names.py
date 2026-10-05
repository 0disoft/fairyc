"""Name-check CLI tests; Go is an independent test reference, never product logic."""
import re


def verify_names(image, reference, directory, rows, run, sha, location):
    counts = dict(bindingFiles=0, bindingReferences=0, files=0, accepted=0, unresolved=0, duplicate=0, unsupported=0, limits=0, unchangedFiles=0)
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
        bindings = run([image, '--bindings-file', path], rows, 'bindings-' + fixture['name'])
        assert before == sha(path), bindings
        bindings['input'] = dict(path=str(path),beforeSHA256=before,afterSHA256=sha(path),bytes=len(source))
        counts['bindingFiles'] += 1
        outcome = fixture['outcome']
        if outcome == 'pass':
            assert (row['exit'], row['stdout'], row['stderr']) == (0, 'names ok\n', ''), row
            assert bindings['exit'] == 0 and bindings['stderr'] == '', bindings
            counts['bindingReferences'] += compare_bindings(bindings['stdout'], fixture, source)
            counts['accepted'] += 1
        else:
            assert row['exit'] == 2 and row['stdout'] == '', row
            assert (bindings['exit'],bindings['stdout'],bindings['stderr']) == (row['exit'],row['stdout'],row['stderr']), bindings
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


def compare_bindings(output, fixture, source):
    lines = output.splitlines()
    assert lines.pop(0) == 'record\tid\tkind\tstart\tend\tscope\towner\ttarget'
    symbols, references = {}, []
    for line in lines:
        record, id_, kind, start, end, scope, owner, target = line.split('\t')
        id_, start, end, scope, owner, target = map(int, (id_, start, end, scope, owner, target))
        assert 0 <= start < end <= len(source)
        if record == 'symbol':
            assert id_ == len(symbols) + 1 and target == 0
            symbols[id_] = dict(kind=kind,start=start,end=end,scope=scope,owner=owner)
        else:
            assert record == 'reference' and kind == 'name' and scope == owner == 0
            references.append((id_,start,end,target))
    actual_symbols = sorted((v['kind'],v['start'],v['end']) for v in symbols.values())
    expected_symbols = sorted((v['kind'],v['start'],v['end']) for v in (fixture['symbols'] or []))
    assert actual_symbols == expected_symbols, (fixture['name'],actual_symbols,expected_symbols)
    for symbol in symbols.values():
        if symbol['kind'] == 'function':
            assert symbol['scope'] == symbol['owner'] == 0
        else:
            assert symbol['scope'] > 0 and symbols[symbol['owner']]['kind'] == 'function'
    actual, seen = [], set()
    for expression, start, end, target in references:
        assert expression not in seen
        seen.add(expression)
        symbol = symbols[target]
        assert source[start:end] == source[symbol['start']:symbol['end']]
        kind = 'local' if symbol['kind'].startswith('local-') else symbol['kind']
        actual.append((start,end,symbol['start'],symbol['end'],kind))
    expected = [(v['start'],v['end'],v['targetStart'],v['targetEnd'],v['targetKind']) for v in (fixture['resolutions'] or [])]
    assert sorted(actual) == sorted(expected), (fixture['name'],actual,expected)
    return len(references)
