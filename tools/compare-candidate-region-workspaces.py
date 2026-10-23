"""Exercise Region fixtures through real SDK workspaces, including known parity gaps.

The direct semantic fixtures supply a small std::alloc stub. Stable workspaces
reserve that namespace for the SDK, so copying the stub tests only its rejection.
This adapter keeps main.fy unchanged and loads the real SDK in a separate module.
It compares CLI results; it does not reproduce the fixtures' internal origin facts.
"""
import argparse
import importlib.util
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('behavior', ROOT / 'tools/compare-candidate-go-behavior.py')
behavior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(behavior)


def matches_expected(observed, expected):
    if observed['timedOut']:
        return False
    if expected['status'] == 'pass':
        return observed['exit'] == 0
    wanted = expected['diagnostics'][0]
    primary = wanted['primary']
    return observed['exit'] == 3 and observed['diagnostics'][:1] == [dict(
        source=primary['source'], code=wanted['code'],
        start=primary['startByte'], end=primary['endByte'])]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--profile', choices=['development', 'release'])
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    built = json.loads(args.build_receipt.read_bytes())
    assert built['passed'] and not built['goFrontendInvoked']
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    inputs = {row['path']: row for row in built['sources']}
    for row in inputs.values():
        assert behavior.sha(ROOT / row['path']) == row['sha256'], row['path']
    manifest = ROOT / 'spec/fixtures/region/manifest-v1.json'
    for path in (manifest, Path(__file__), ROOT / 'tools/compare-candidate-go-behavior.py'):
        relative = path.relative_to(ROOT).as_posix()
        inputs[relative] = dict(path=relative, sha256=behavior.sha(path))
    reference = args.reference.resolve()
    reference_sha = behavior.sha(reference)
    images = [image for image in built['images'] if not args.profile or image['profile'] == args.profile]
    assert images
    rows = []
    adaptations = []
    for index, case in enumerate(json.loads(manifest.read_bytes())['cases']):
        # This adapter intentionally supports the present six-fixture manifest.
        # New dependency graphs require an explicit adapter, not a guessed mapping.
        graph = case.get('packageGraph')
        assert not graph or graph == dict(manifest='../package/manifest-v1.json',
            caseId='pass.package.stdlib-arena-region', rootNode='app')
        assert case['sources'] in (['main.fy'], ['main.fy', 'std/alloc.fy'])
        home = output / case['id']
        (home / 'src').mkdir(parents=True)
        (home / 'fairy.toml').write_text(
            'schema="fairyc.manifest/v1"\n[package]\n'
            f'uid="compare.region.sdk.case{index}"\nname="region-sdk"\n'
            'version="0.1.0"\nlanguage="fairyc-language/20"\nsource="src"\n',
            encoding='utf-8', newline='\n')
        for filename in case['sources']:
            path = (manifest.parent / case['sourceRoot'] / filename).resolve()
            assert path.is_relative_to(ROOT / 'spec/fixtures')
            relative = path.relative_to(ROOT).as_posix()
            inputs[relative] = dict(path=relative, sha256=behavior.sha(path))
            if filename == 'main.fy':
                (home / 'src/main.fy').write_bytes(path.read_bytes())
        adapter = 'module fixture_sdk;\nimport std::alloc as alloc;\n'
        (home / 'src/fixture_sdk.fy').write_text(adapter, encoding='utf-8', newline='\n')
        adaptations.append(dict(id=case['id'], mainSourceUnchanged=True,
            omittedStub='std/alloc.fy' if len(case['sources']) == 2 else None,
            replacedFixtureGraph=graph, addedSource=adapter))
        # Resolve once. Even a failed check must leave the canonical lock before
        # the read-only comparisons; an earlier loader error cannot count as proof.
        setup = behavior.execute(reference, ['check'], home, 180)
        assert not setup['timedOut'] and (home / 'fairy.lock').is_file(), setup
        lock_sha = behavior.sha(home / 'fairy.lock')
        go = behavior.execute(reference, ['check', '--locked'], home, 180)
        for image in images:
            candidate = Path(image['path'])
            assert behavior.sha(candidate) == image['sha256']
            observed = behavior.execute(candidate, ['check', '--locked'], home, 180)
            row = dict(id=case['id'], profile=image['profile'], expectation=case['expect'],
                go=go, candidate=observed, comparison=behavior.comparison(go, observed),
                goMatchesExpected=matches_expected(go, case['expect']),
                candidateMatchesExpected=matches_expected(observed, case['expect']))
            rows.append(row)
            behavior.save(output / 'observations.json', rows)
            assert behavior.sha(candidate) == image['sha256']
            print(image['profile'], case['id'], go['exit'], observed['exit'],
                  'exact=' + str(row['comparison']['exactMatch']), flush=True)
        assert behavior.sha(home / 'fairy.lock') == lock_sha
    for row in inputs.values():
        assert behavior.sha(ROOT / row['path']) == row['sha256'], row['path']
    assert behavior.sha(reference) == reference_sha
    passed = all(row['comparison']['exactMatch'] and row['goMatchesExpected'] and
                 row['candidateMatchesExpected'] for row in rows)
    behavior.save(output / 'receipt.json', dict(
        schema='fairyc.candidate-region-workspace-comparison/v1', passed=passed,
        comparisonCompleted=True, cases=len(rows), exactMatches=sum(row['comparison']['exactMatch'] for row in rows),
        buildReceiptSHA256=behavior.sha(args.build_receipt), referenceSHA256=reference_sha,
        candidateImages=images, sourceInputs=list(inputs.values()), adaptations=adaptations, rows=rows,
        scope='Stable check with real SDK; original main.fy bytes and primary spans retained',
        originFactsCompared=False, nativeExecution=False, skipped=[]))
    raise SystemExit(0 if passed else 1)


if __name__ == '__main__':
    main()
