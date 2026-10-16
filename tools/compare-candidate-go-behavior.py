"""Compare current Go and candidate workspace behavior without treating gaps as passes."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
FAMILIES = {
    'semantic': 'manifest-v0.json', 'resource': 'manifest-v0.json',
    'region': 'manifest-v1.json', 'result': 'manifest-v0.json',
    'mir': 'manifest-v0.json', 'c-emission': 'manifest-v0.json',
}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, data):
    path.write_text(json.dumps(data, indent=2) + '\n', encoding='utf-8', newline='\n')


def execute(image, argv, cwd, timeout):
    started = time.monotonic()
    with subprocess.Popen([str(image), *map(str, argv)], cwd=cwd, stdin=subprocess.DEVNULL,
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                          creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0)) as process:
        timed_out = False
        try:
            out, err = process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out = True
            if os.name == 'nt':
                subprocess.run(['taskkill', '/PID', str(process.pid), '/T', '/F'],
                               capture_output=True, timeout=15)
            if process.poll() is None:
                process.kill()
            out, err = process.communicate()
    errors = err.decode('utf-8', errors='replace')
    diagnostics = []
    for line in errors.splitlines():
        match = re.search(r'(?:^|\s)(E\d{4})\b.*?\[bytes (\d+)\.\.(\d+)', line)
        if match:
            file = line.split(':', 1)[0].replace('\\', '/')
            diagnostics.append(dict(source=file, code=match[1], start=int(match[2]), end=int(match[3])))
    return dict(argv=list(map(str, argv)), exit=process.returncode, timedOut=timed_out,
                seconds=time.monotonic()-started, stdout=out.decode('utf-8', errors='replace'),
                stderr=errors, stdoutSHA256=hashlib.sha256(out).hexdigest(),
                stderrSHA256=hashlib.sha256(err).hexdigest(), diagnostics=diagnostics)


def comparison(go, candidate):
    complete = not go['timedOut'] and not candidate['timedOut']
    return dict(completed=complete, acceptanceMatch=complete and (go['exit'] == 0) == (candidate['exit'] == 0),
                exitMatch=complete and go['exit'] == candidate['exit'],
                firstDiagnosticMatch=complete and go['diagnostics'][:1] == candidate['diagnostics'][:1],
                allDiagnosticsMatch=complete and go['diagnostics'] == candidate['diagnostics'],
                stdoutMatch=complete and go['stdoutSHA256'] == candidate['stdoutSHA256'],
                stderrMatch=complete and go['stderrSHA256'] == candidate['stderrSHA256'],
                exactMatch=complete and all(go[k] == candidate[k] for k in
                                           ['exit', 'stdoutSHA256', 'stderrSHA256']))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--candidate-build-receipt', type=Path, required=True)
    parser.add_argument('--profile', choices=['development', 'release'], default='release')
    parser.add_argument('--suite', choices=['check', 'run', 'cli'], required=True)
    parser.add_argument('--family', action='append', choices=list(FAMILIES))
    parser.add_argument('--case', action='append', help='compare only these exact fixture or CLI ids')
    parser.add_argument('--timeout', type=int, default=60)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__ and 1 <= args.timeout <= 900
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    build = json.loads(args.candidate_build_receipt.read_bytes())
    assert build['passed'] and not build['goFrontendInvoked']
    selected = next(r for r in build['images'] if r['profile'] == args.profile)
    candidate = Path(selected['path'])
    reference = args.reference.resolve()
    assert sha(candidate) == selected['sha256']
    inputs = {r['path']: r for r in build['sources']}
    for row in inputs.values():
        assert sha(ROOT / row['path']) == row['sha256'], row['path']
    for path in ['tools/compare-candidate-go-behavior.py', 'internal/cli/run.go',
                 'examples/lexer_candidate/src/file_cli.fy']:
        inputs[path] = dict(path=path, sha256=sha(ROOT / path))
    # Bind the fresh oracle to the current tracked implementation and SDK inputs.
    tracked = subprocess.check_output(['git', 'ls-files', '-z', '--', 'go.mod', 'go.sum',
                                      '*.go', 'internal/sdkstdlib/sources', 'internal/cstdlib'], cwd=ROOT)
    for raw in tracked.split(b'\0'):
        if raw:
            path = raw.decode('utf-8')
            inputs[path] = dict(path=path, sha256=sha(ROOT / path))
    reference_sha = sha(reference)
    rows, skipped = [], []

    def record(row):
        rows.append(row)
        save(output / 'observations.json', rows)
        if not row['comparison']['exactMatch']:
            print('difference:', row['id'], ','.join(k for k, v in row['comparison'].items() if not v), flush=True)
        if len(rows) % 20 == 0:
            print('compared:', len(rows), flush=True)

    def workspace(family, index, case):
        path = output / family / case['id']
        (path / 'src').mkdir(parents=True)
        (path / 'fairy.toml').write_text(
            'schema="fairyc.manifest/v1"\n[package]\n'
            f'uid="compare.{family.replace("-", "")}.case{index}"\n'
            'name="behavior-fixture"\nversion="0.1.0"\nlanguage="fairyc-language/20"\nsource="src"\n',
            encoding='utf-8', newline='\n')
        manifest = ROOT / 'spec/fixtures' / family / FAMILIES[family]
        sources, names = [], set()
        for filename in case['sources']:
            original = (manifest.parent / case['sourceRoot'] / filename).resolve()
            assert original.is_relative_to(ROOT / 'spec/fixtures')
            raw = original.read_bytes()
            declared = re.search(rb'^\s*module\s+([A-Za-z0-9_:]+)\s*;', raw, re.M)
            relative = declared[1].decode().replace('::', '/')+'.fy' if declared else filename
            target = path / 'src' / relative
            assert target.resolve().is_relative_to(path / 'src') and relative not in names
            names.add(relative)
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(raw)
            source = dict(path=original.relative_to(ROOT).as_posix(), sha256=sha(original))
            inputs[source['path']] = source
            sources.append({**source, 'workspacePath': relative})
        return path, sources

    if args.suite in ['check', 'run']:
        families = args.family or (list(FAMILIES) if args.suite == 'check' else ['c-emission'])
        for family in families:
            manifest = ROOT / 'spec/fixtures' / family / FAMILIES[family]
            data = json.loads(manifest.read_bytes())
            inputs[manifest.relative_to(ROOT).as_posix()] = dict(path=manifest.relative_to(ROOT).as_posix(), sha256=sha(manifest))
            for index, case in enumerate(data['cases']):
                if args.case and case['id'] not in args.case:
                    continue
                if case.get('packageGraph'):
                    skipped.append(dict(family=family, id=case['id'], reason='direct fixture packageGraph requires a separate workspace adapter'))
                    continue
                if args.suite == 'run' and (case.get('entry') != 'main::main' or case['expect']['status'] != 'pass'):
                    skipped.append(dict(family=family, id=case['id'], reason='no passing main::main native entry'))
                    continue
                path, sources = workspace(family, index, case)
                go = execute(reference, ['check'], path, args.timeout)
                if args.suite == 'check':
                    # Both observations read the same source and canonical lock without writing.
                    if go['exit'] == 0:
                        go = execute(reference, ['check', '--locked'], path, args.timeout)
                    candidate_result = execute(candidate, ['check', '--locked'], path, args.timeout)
                else:
                    if go['exit'] != 0:
                        skipped.append(dict(family=family, id=case['id'], reason='Go workspace admission fails', observation=go))
                        continue
                    go = execute(reference, ['run', '--locked'], path, args.timeout)
                    candidate_result = execute(candidate, ['run', '--locked'], path, args.timeout)
                applicable = args.suite == 'check' and family in ['semantic', 'resource', 'region', 'result']
                expected_codes = [d['code'] for d in case['expect'].get('diagnostics', [])]
                fixture_match = ((go['exit'] == 0) == (case['expect']['status'] == 'pass') and
                                 (not expected_codes or [r['code'] for r in go['diagnostics'][:1]] == expected_codes[:1])) if applicable else None
                record(dict(family=family, id=case['id'], suite=args.suite, sources=sources,
                            fixtureExpectation=case['expect'], fixtureExpectationApplicable=applicable,
                            goMatchesFixture=fixture_match, go=go, candidate=candidate_result,
                            comparison=comparison(go, candidate_result)))
    else:
        case = dict(id='cli', sourceRoot='', sources=[])
        path, _ = workspace('semantic', 0, case)
        (path / 'src/main.fy').write_text('module main;fn i32 main(){return 0;}\n', encoding='utf-8', newline='\n')
        execute(reference, ['check'], path, args.timeout)
        invocations = [[], ['check', '--locked'], ['check', '--bad'], ['check', '--locked', '--locked'],
                       ['build', '--bad'], ['run', '--bad'], ['check', '--cache-status']]
        for index, argv in enumerate(invocations):
            if args.case and f'cli-{index}' not in args.case:
                continue
            go = execute(reference, argv, path, args.timeout)
            other = execute(candidate, argv, path, args.timeout)
            record(dict(family='cli', id=f'cli-{index}', go=go, candidate=other, comparison=comparison(go, other)))
    assert rows, 'no matching executable cases'
    assert sha(reference) == reference_sha and sha(candidate) == selected['sha256']
    for row in inputs.values():
        assert sha(ROOT / row['path']) == row['sha256'], row['path']
    measures = list(comparison(rows[0]['go'], rows[0]['candidate'])) if rows else []
    summary = {key: sum(row['comparison'][key] for row in rows) for key in measures}
    report = dict(schema='fairyc.candidate-go-behavior-comparison/v1', comparisonCompleted=True,
                  parity=bool(rows) and all(r['comparison']['exactMatch'] for r in rows) and not skipped,
                  suite=args.suite, profile=args.profile, cases=len(rows), summary=summary, skipped=skipped,
                  scope=build['entryScope'], candidate=dict(path=str(candidate), sha256=sha(candidate)),
                  reference=dict(path=str(reference), sha256=reference_sha), sourceInputs=list(inputs.values()),
                  rows=rows, structureEquivalenceChecked=False, publicApiChanged=False, sdkAbiChanged=False,
                  ciChanged=False, pushed=False)
    save(output / 'receipt.json', report)
    print(json.dumps(dict(suite=args.suite, cases=len(rows), summary=summary, skipped=len(skipped)), indent=2), flush=True)


if __name__ == '__main__':
    main()
