"""Verify Result obligation diagnostics and cache error preservation against Go."""
import argparse
import importlib.util
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('behavior', ROOT / 'tools/compare-candidate-go-behavior.py')
behavior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(behavior)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    build = json.loads(args.build_receipt.read_bytes())
    assert build['passed'] and not build['goFrontendInvoked']
    assert {r['profile'] for r in build['images']} == {'development', 'release'}
    reference = args.reference.resolve()
    reference_sha = behavior.sha(reference)
    inputs = {r['path']: r for r in build['sources']}
    for name in ['tools/verify-candidate-result-flow.py', 'tools/compare-candidate-go-behavior.py',
                 'docs/language/0004-results-traps-and-control-flow.md']:
        inputs[name] = dict(path=name, sha256=behavior.sha(ROOT / name))
    cases, skipped, observations = [], [], []
    for family in ['result', 'mir']:
        manifest = ROOT / 'spec/fixtures' / family / 'manifest-v0.json'
        key = manifest.relative_to(ROOT).as_posix()
        inputs[key] = dict(path=key, sha256=behavior.sha(manifest))
        for case in json.loads(manifest.read_bytes())['cases']:
            if family == 'mir' and case['id'] != 'pass.mir.result-error-cleanup':
                continue
            codes = [d['code'] for d in case['expect'].get('diagnostics', [])]
            if family == 'result' and case['expect']['status'] != 'pass' and codes != ['E4101']:
                skipped.append(dict(id=case['id'], reason='E4102/E4103 diagnostic parity is outside this fix'))
                continue
            path = output / 'fixtures' / case['id']
            files = {}
            for filename in case['sources']:
                original = (manifest.parent / case['sourceRoot'] / filename).resolve()
                assert original.is_relative_to(ROOT / 'spec/fixtures')
                key = original.relative_to(ROOT).as_posix()
                inputs[key] = dict(path=key, sha256=behavior.sha(original))
                raw = original.read_text(encoding='utf-8')
                module = re.search(r'^\s*module\s+([\w:]+)\s*;', raw, re.M)[1]
                files['src/' + module.replace('::', '/') + '.fy'] = raw
            workspace(path, files)
            go = behavior.execute(reference, ['check'], path, 60)
            if go['exit'] == 0:
                go = behavior.execute(reference, ['check', '--locked'], path, 60)
            assert not go['timedOut'] and (go['exit'] == 0 or
                   go['exit'] == 3 and [d['code'] for d in go['diagnostics']] == ['E4101']), go
            cases.append(dict(id=case['id'], path=path, go=go))
    baseline_cases = len(cases)
    baseline_errors = sum(c['go']['exit'] == 3 for c in cases)
    assert baseline_cases == 23 and baseline_errors == 11
    extra_sources = {
        'iteration': 'module main;fn void f(ref u8[] items){for(let u8 item in items){let Result<i32,i32> value=.ok(1);HANDLE}}',
        'region': 'module main;import std::alloc as a;fn void f(a::Arena arena){var a::Arena owner=arena;region scratch in ref mut owner{let Result<i32,i32> value=.ok(1);HANDLE}}',
    }
    for kind, text in extra_sources.items():
        for accepted in [False, True]:
            name = kind + ('-handled' if accepted else '-pending')
            path = output / 'fixtures' / name
            workspace(path, {'src/main.fy': text.replace('HANDLE', 'discard value;' if accepted else '')})
            go = behavior.execute(reference, ['check'], path, 60)
            if accepted:
                assert go['exit'] == 0, go
                go = behavior.execute(reference, ['check', '--locked'], path, 60)
            else:
                assert go['exit'] == 3 and [d['code'] for d in go['diagnostics']] == ['E4101'], go
            cases.append(dict(id=name, path=path, go=go, additional=True))

    def execute(image, path, argv, label):
        row = behavior.execute(image, argv, path, 120)
        observations.append(dict(label=label, image=str(image), **row))
        behavior.save(output / 'observations.json', observations)
        assert not row['timedOut'], observations[-1]
        return row

    for selected in build['images']:
        profile, image = selected['profile'], Path(selected['path'])
        assert behavior.sha(image) == selected['sha256']
        for case in cases:
            other = execute(image, case['path'], ['check', '--locked'], profile + '/' + case['id'])
            matched = behavior.comparison(case['go'], other)
            assert (matched['exitMatch'] and matched['stderrMatch'] if case.get('additional') else matched['exactMatch']), observations[-1]
        print(profile, 'fixture diagnostics and successful checks:', len(cases), flush=True)
        path = output / profile / 'workspace'
        valid = 'module main;import helper as h;fn i32 main(){discard h::make();return 42;}\n'
        workspace(path, {'src/main.fy': valid,
                        'helper/fairy.toml': manifest_text('helper'),
                        'helper/src/helper.fy': 'module helper;pub fn Result<i32,i32> make(){return .ok(42);}\n'},
                  "[workspace]\nmembers=['helper']\n[dependencies.helper]\npath='helper'\nversion='1.0.0'\nfeatures=[]\n")
        go = execute(reference, path, ['check'], profile + '/Go valid')
        assert go['exit'] == 0
        cache = output / profile / 'cache'
        cache.mkdir()
        flags = ['--locked', '--cache-root', str(cache), '--cache-status']
        cold = execute(image, path, ['build', *flags], profile + '/cold cached build')
        assert cold['exit'] == 0 and cold['stderr'] == 'cache: cold stored\n', cold
        binaries = list((path / 'build').glob('*.exe'))
        assert len(binaries) == 1
        binary_sha = behavior.sha(binaries[0])
        hit = execute(image, path, ['check', *flags], profile + '/verified project hit')
        assert hit['exit'] == 0 and hit['stderr'] == 'cache: hit\n'
        run = execute(image, path, ['run', *flags], profile + '/valid cached run')
        assert run['exit'] == 42 and run['stdout'] == '' and run['stderr'] == 'cache: hit\n'
        before = snapshot(cache)
        assert len(before) >= 3
        (path / 'src/main.fy').write_text(valid.replace('discard h::make()', 'h::make()'), encoding='utf-8', newline='\n')
        go = execute(reference, path, ['check'], profile + '/Go pending Result')
        assert go['exit'] == 3 and [d['code'] for d in go['diagnostics']] == ['E4101']
        for command in ['check', 'build', 'run']:
            cold = execute(image, path, [command, '--locked'], profile + '/cold invalid ' + command)
            assert cold['exit'] == go['exit'] and cold['stdout'] == go['stdout'] and cold['stderr'] == go['stderr']
            other = execute(image, path, [command, *flags], profile + '/mixed invalid ' + command)
            status, diagnostic = other['stderr'].split('\n', 1)
            assert status.startswith('cache: ') and status not in ['cache: hit', 'cache: units hit']
            assert other['exit'] == go['exit'] and other['stdout'] == go['stdout'] and diagnostic == go['stderr']
            assert snapshot(cache) == before and behavior.sha(binaries[0]) == binary_sha
        (path / 'src/main.fy').write_text(valid, encoding='utf-8', newline='\n')
        assert execute(reference, path, ['check'], profile + '/Go restored valid')['exit'] == 0
        restored = execute(image, path, ['check', *flags], profile + '/restored valid hit')
        assert restored['exit'] == 0 and restored['stderr'] == 'cache: hit\n'
        assert snapshot(cache) == before and behavior.sha(binaries[0]) == binary_sha
        print(profile, 'cold/hit/mixed errors preserve cache and executable', flush=True)
        assert behavior.sha(image) == selected['sha256']
    assert behavior.sha(reference) == reference_sha
    for row in inputs.values():
        assert behavior.sha(ROOT / row['path']) == row['sha256'], row['path']
    receipt = dict(schema='fairyc.candidate-result-flow/v1', passed=True, fixtureCases=len(cases),
                   baselineFixtureCases=baseline_cases, baselineE4101Cases=baseline_errors, additionalCases=4,
                   e4101Cases=sum(c['go']['exit'] == 3 for c in cases),
                   acceptedCases=sum(c['go']['exit'] == 0 for c in cases), profiles=build['images'],
                   scope=build['entryScope'], reference=dict(path=str(reference), sha256=reference_sha),
                   candidateGoFrontendFallback=False, sourceInputs=list(inputs.values()), observations=observations,
                   fixtureOracles=[dict(id=c['id'], go=c['go']) for c in cases],
                   skipped=skipped, publicApiChanged=False, sdkAbiChanged=False, ciChanged=False, pushed=False)
    behavior.save(output / 'receipt.json', receipt)
    print('passed:', len(observations), 'recorded command observations', flush=True)


def manifest_text(uid):
    return ('schema="fairyc.manifest/v1"\n[package]\n' + f'uid="example.result-flow.{uid}"\n'
            'name="result-flow"\nversion="1.0.0"\nlanguage="fairyc-language/20"\nsource="src"\n')


def workspace(path, files, extra=''):
    files = {'fairy.toml': manifest_text('root') + extra, **files}
    for name, text in files.items():
        target = path / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(text, encoding='utf-8', newline='\n')


def snapshot(path):
    return {p.name: behavior.sha(p) for p in path.glob('*.fyc')}


if __name__ == '__main__':
    main()
