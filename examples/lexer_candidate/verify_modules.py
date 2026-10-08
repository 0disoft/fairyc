"""Multi-file frontend verification; Python prepares inputs and checks observations."""
from pathlib import Path
import re


def verify_modules(image, reference, directory, rows, run, sha, location):
    counts = dict(fixtures=0, accepted=0, semanticErrors=0, unsupported=0, limits=0, unchangedFiles=0)
    for index, fixture in enumerate(reference['fixtures']):
        folder = directory / f'modules-{index}'
        folder.mkdir(exist_ok=True)
        inputs = []
        for file in fixture['files']:
            path = folder / file['path']
            path.parent.mkdir(parents=True, exist_ok=True)
            source = bytes.fromhex(file['inputHex'])
            path.write_bytes(source)
            inputs.append((path, source, sha(path), file['path']))
        manifest = folder / 'files.txt'
        # Exercise all admitted line endings, final-line spelling and relative
        # cwd-based paths independently of source-file line endings.
        paths = [str(p.relative_to(Path(__file__).resolve().parent)) for p, *_ in inputs]
        separator = '\r\n' if index % 2 else '\n'
        manifest.write_bytes((separator.join(paths) + (separator if index % 3 else '')).encode('utf-8'))
        before = sha(manifest)
        row = run([image, '--check-modules-file', manifest], rows, 'modules-' + fixture['name'])
        row['inputs'] = [dict(path=str(p), beforeSHA256=digest, afterSHA256=sha(p), bytes=len(source)) for p, source, digest, _ in inputs]
        row['manifest'] = dict(path=str(manifest), beforeSHA256=before, afterSHA256=sha(manifest))
        assert before == sha(manifest) and all(x['beforeSHA256'] == x['afterSHA256'] for x in row['inputs']), row
        counts['unchangedFiles'] += len(inputs) + 1
        counts['fixtures'] += 1
        outcome = fixture['outcome']
        if outcome == 'pass':
            assert (row['exit'], row['stdout'], row['stderr']) == (0, 'modules ok\n', ''), row
            counts['accepted'] += 1
            continue
        assert row['exit'] == 2 and row['stdout'] == '' and row['stderr'], row
        if outcome == 'error':
            # Candidate messages are private. Compare their rejection category
            # to real Go diagnostics and verify the candidate's byte location.
            match = re.fullmatch(r'(.+):(\d+):(\d+): error: (.+) \(bytes (\d+)\.\.(\d+)\)\n', row['stderr'])
            assert match, row
            reported, line, column, message, start, end = match.groups()
            start, end = int(start), int(end)
            selected = next(((s, logical) for p, s, _, logical in inputs if reported in (str(p), str(p.relative_to(Path(__file__).resolve().parent)))), None)
            assert selected is not None, row
            source, logical = selected
            assert 0 <= start < end <= len(source), row
            assert location(source, start) == (int(line), int(column)), row
            categories = {
                'unresolved name': {'E0101'}, 'unresolved import': {'E0101'},
                'unresolved imported item': {'E0101'}, 'private imported item': {'E0103'},
                'unresolved import module': {'E0101'}, 'unresolved import alias': {'E0101'},
                'duplicate name in scope': {'E0102'}, 'duplicate import alias': {'E0102'},
                'type mismatch': {'E1006', 'E1009'}, 'call argument count mismatch': {'E1010'},
                'integer literal out of range': {'E1013'},
            }
            assert message in categories, row
            matches = []
            for d in fixture['diagnostics']:
                spans = [d['Primary']]
                if d['Code'] == 'E0102':
                    spans += [r['Span'] for r in d['Related'] or []]
                matches.extend(d['Code'] in categories[message] and span['Source'] == logical and
                               start < span['EndByte'] and span['StartByte'] < end for span in spans)
            assert any(matches), dict(row, goldens=fixture)
            counts['semanticErrors'] += 1
        elif outcome == 'unsupported':
            assert 'unsupported' in row['stderr'], row
            counts['unsupported'] += 1
        elif outcome == 'limit':
            assert 'limit' in row['stderr'], row
            counts['limits'] += 1
        else:
            raise AssertionError(fixture)
    # Transport, parser and duplicate-module controls are independent of the Go
    # semantic fixture set. No candidate diagnostics are used as an oracle.
    folder = directory / 'module-transport'
    folder.mkdir(exist_ok=True)
    source = folder / '한글 소스.fy'
    source.write_bytes(b'module main;')
    other = folder / 'other.fy'
    other.write_bytes(b'module main;')
    relative = lambda p: str(p.relative_to(Path(__file__).resolve().parent))
    cases = [
        ('empty-manifest', b'', 'error: invalid module manifest\n'),
        ('blank-line', (relative(source)+'\n\n').encode('utf-8'), 'error: invalid module manifest\n'),
        ('invalid-UTF8', b'\xff\n', 'error: invalid module manifest\n'),
        ('BOM', b'\xef\xbb\xbf'+relative(source).encode('utf-8'), 'error: invalid module manifest\n'),
        ('NUL', b'a\0.fy\n', 'error: invalid module manifest\n'),
        ('TAB', b'a\t.fy\n', 'error: invalid module manifest\n'),
        ('absolute-entry', str(source).encode('utf-8'), 'error: invalid module manifest\n'),
        ('parent-read', (relative(folder/'missing.fy')+'\n').encode('utf-8'), 'error: input file read failed\n'),
        ('path-over', b'a'*4096, 'error: file path limit (4095 UTF-8 bytes)\n'),
        ('manifest-over', b'a'*65537, 'error: manifest limit (65536 bytes)\n'),
    ]
    manifest = folder / 'files.txt'
    for name, data, expected in cases:
        manifest.write_bytes(data)
        before = sha(manifest)
        row = run([image, '--check-modules-file', manifest], rows, 'modules-'+name)
        row['manifest'] = dict(path=str(manifest), beforeSHA256=before, afterSHA256=sha(manifest), bytes=len(data))
        assert (row['exit'],row['stdout'],row['stderr']) == (2,'',expected), row
        assert sha(manifest) == before
        counts['unchangedFiles'] += 1
    for name, content in [('duplicate-path', relative(source)+'\n'+relative(source)),
                          ('duplicate-module', relative(source)+'\n'+relative(other))]:
        manifest.write_text(content,encoding='utf-8',newline='\n')
        before = sha(manifest)
        input_before = [(p,sha(p)) for p in (source,other)]
        row = run([image,'--check-modules-file',manifest],rows,'modules-'+name)
        row['manifest'] = dict(path=str(manifest),beforeSHA256=before,afterSHA256=sha(manifest))
        row['inputs'] = [dict(path=str(p),beforeSHA256=digest,afterSHA256=sha(p)) for p,digest in input_before]
        assert row['exit']==2 and row['stdout']=='' and 'duplicate module' in row['stderr'], row
        assert sha(manifest)==before and all(x['beforeSHA256']==x['afterSHA256'] for x in row['inputs'])
    for name, content, expected in [('lexical',b'module main;\xff','E0001'),
                                     ('syntax',b'module main','expected ;')]:
        source.write_bytes(content)
        before = sha(source)
        manifest.write_text(relative(source),encoding='utf-8',newline='\n')
        row = run([image,'--check-modules-file',manifest],rows,'modules-'+name)
        row['input'] = dict(path=str(source),beforeSHA256=before,afterSHA256=sha(source),bytes=len(content))
        assert row['exit']==2 and row['stdout']=='' and expected in row['stderr'], row
        assert sha(source)==before
    return counts
