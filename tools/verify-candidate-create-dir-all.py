"""Compare authenticated recursive-directory C emission and execution with Go."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import shutil
import tempfile

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('behavior', ROOT / 'tools/compare-candidate-go-behavior.py')
behavior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(behavior)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--baseline', type=Path, required=True)
    parser.add_argument('--profile', choices=['development', 'release'], default='release',
                        help='compiler profile; both generated program profiles are exercised')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__ and os.name == 'nt'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    build = json.loads(args.build_receipt.read_bytes())
    assert build['passed'] and not build['goFrontendInvoked']
    assert {r['profile'] for r in build['images']} == {'development', 'release'}
    inputs = {r['path']: r for r in build['sources']}
    for path in ['tools/fixtures/candidate-create-dir-all.fy', 'tools/verify-candidate-create-dir-all.py',
                 'tools/compare-candidate-go-behavior.py', 'internal/sdkstdlib/sources/std/fs.fy',
                 'internal/cstdlib/stage0_windows_file.c']:
        inputs[path] = dict(path=path, sha256=behavior.sha(ROOT / path))
    reference, baseline = args.reference.resolve(), args.baseline.resolve()
    for row in inputs.values():
        assert behavior.sha(ROOT / row['path']) == row['sha256'], row['path']
    images = [dict(profile='Go', path=str(reference), sha256=behavior.sha(reference)),
              *[row for row in build['images'] if row['profile'] == args.profile]]
    before_sha = behavior.sha(baseline)
    observations = []
    golden_builds = {}

    def execute(image, path, argv, label):
        row = behavior.execute(image, argv, path, 180)
        observations.append(dict(id=label, **row))
        behavior.save(output / 'observations.json', observations)
        assert not row['timedOut'], row
        return row

    def succeeds(row):
        assert row['exit'] == 0 and row['stderr'] == '', row

    for image in images:
        executable = Path(image['path'])
        assert behavior.sha(executable) == image['sha256']
        path = output / image['profile']
        (path / 'src').mkdir(parents=True)
        (path / 'fairy.toml').write_text('schema = "fairyc.manifest/v1"\n[package]\n'
            'uid = "test.candidate-create-dir-all"\nname = "directory-probe"\nversion = "0.1.0"\n'
            'language = "fairyc-language/20"\nsource = "src"\n', encoding='utf-8', newline='\n')
        shutil.copyfile(ROOT / 'tools/fixtures/candidate-create-dir-all.fy', path / 'src/main.fy')
        if image['profile'] == 'Go':
            succeeds(execute(reference, path, ['check'], 'Go/check'))
            lock = (path / 'fairy.lock').read_bytes()
            before = execute(baseline, path, ['build', '--locked'], 'baseline/build')
            assert before['exit'] == 2 and before['stdout'] == '' and 'code 1004' in before['stderr'], before
            assert '@sdk/stdlib/std/fs.fy:488:9:' in before['stderr'], before
        else:
            (path / 'fairy.lock').write_bytes(lock)
        for mode, flags in [('development', []), ('release', ['--release'])]:
            label = image['profile'] + '/' + mode
            built = execute(executable, path, ['build', '--locked', *flags], label + '/build')
            succeeds(built)
            if image['profile'] == 'Go':
                golden_builds[mode] = built
            else:
                assert behavior.comparison(golden_builds[mode], built)['exactMatch'], built
            with tempfile.TemporaryDirectory(prefix='directory-', dir=output) as scratch:
                row = execute(path / 'build/directory-probe.exe', Path(scratch), [], label + '/artifact')
                assert row['exit'] == 42 and row['stdout'] == row['stderr'] == '', row
                assert not list(Path(scratch).iterdir()), 'fixture left filesystem effects behind'
            row = execute(executable, path, ['run', '--locked', *flags], label + '/run')
            assert row['exit'] == 42 and row['stdout'] == row['stderr'] == '', row
        assert behavior.sha(executable) == image['sha256']
        print(image['profile'], 'build/artifact/run pass at both output profiles', flush=True)
    assert behavior.sha(baseline) == before_sha
    for row in inputs.values():
        assert behavior.sha(ROOT / row['path']) == row['sha256'], row['path']
    behavior.save(output / 'receipt.json', dict(schema='fairyc.candidate-create-dir-all/v1', passed=True,
        sourceInputs=list(inputs.values()), images=images, observations=observations,
        baseline=dict(path=str(baseline), sha256=before_sha), expectedExit=42,
        scope='recursive-directory admission, original SDK result bridge and real Windows runtime',
        goFrontendFallback=False, sdkAbiChanged=False, productTimeoutChanged=False,
        ciChanged=False, pushed=False))


if __name__ == '__main__':
    main()
