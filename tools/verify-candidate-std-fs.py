"""Compare the shipped filesystem example through real Go and candidate commands."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('behavior', ROOT / 'tools/compare-candidate-go-behavior.py')
behavior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(behavior)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--baseline', type=Path, help='optional pre-fix command image reproducing the internal error')
    parser.add_argument('--profile', choices=['development', 'release'], default='release',
                        help='compiler image profile; both generated program profiles are always exercised')
    parser.add_argument('--check-only', action='store_true', help='verify the frontend crash regression without requiring native backend parity')
    parser.add_argument('--timeout', type=int, default=180, help='verification-only per-command allowance in seconds (1..900)')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if not 1 <= args.timeout <= 900:
        parser.error('timeout must be in 1..900 seconds')
    assert __debug__ and os.name == 'nt', 'requires Windows and Python without -O'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    build = json.loads(args.build_receipt.read_bytes())
    assert build['passed'] and not build['goFrontendInvoked']
    assert {r['profile'] for r in build['images']} == {'development', 'release'}
    reference = args.reference.resolve()
    reference_sha = behavior.sha(reference)
    inputs = {r['path']: r for r in build['sources']}
    tracked = subprocess.run(['git', 'ls-files', 'examples/std_fs_quickstart'], cwd=ROOT,
                             check=True, capture_output=True, text=True).stdout.splitlines()
    files = [Path(p) for p in tracked if p.endswith('.fy') or p.endswith('/fairy.toml')]
    for path in files + [Path('tools/verify-candidate-std-fs.py'), Path('tools/compare-candidate-go-behavior.py'),
                         Path('cmd/fairyc-windows-host-smoke/stdlib_quickstart_proofs.go')]:
        inputs[path.as_posix()] = dict(path=path.as_posix(), sha256=behavior.sha(ROOT / path))
    observations = []

    def workspace(name):
        path = output / name
        for original in files:
            dest = path / original.relative_to('examples/std_fs_quickstart')
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(ROOT / original, dest)
        return path

    def execute(image, path, argv, label):
        row = behavior.execute(image, argv, path, args.timeout)
        observations.append(dict(id=label, **row))
        behavior.save(output / 'observations.json', observations)
        assert not row['timedOut'], label
        return row

    def fixtures(path):
        # Direct artifact execution uses the Windows host smoke fixtures.
        folder = (path / 'build').resolve()
        assert folder.is_relative_to(output) and folder.is_dir()
        for target in folder.glob('fairyc-std-fs-*'):
            assert target.resolve().is_relative_to(folder)
            if target.is_dir():
                shutil.rmtree(target)
            else:
                target.chmod(stat.S_IWRITE)
                target.unlink()
        readonly = folder / 'fairyc-std-fs-read-only-metadata.tmp'
        readonly.write_bytes(b'metadata')
        readonly.chmod(stat.S_IREAD)
        linked = folder / 'fairyc-std-fs-hard-link-metadata.tmp'
        linked.write_bytes(b'metadata')
        os.link(linked, folder / 'fairyc-std-fs-hard-link-metadata-alias.tmp')

    def returned(row, stderr='', status=42):
        assert row['exit'] == status and row['stdout'] == '' and row['stderr'] == stderr, row

    base = workspace('Go')
    assert execute(reference, base, ['check'], 'Go/admit')['exit'] == 0
    golden = execute(reference, base, ['check', '--locked'], 'Go/check')
    assert golden['exit'] == 0 and golden['stderr'] == ''
    baseline = None
    if args.baseline:
        previous = args.baseline.resolve()
        baseline = dict(path=str(previous), sha256=behavior.sha(previous))
        before = execute(previous, base, ['check', '--locked'], 'baseline/check')
        assert before['exit'] == 4 and before['stdout'] == '' and before['stderr'] == 'fairyc: internal compiler invariant failed\n'
    oracles = {}
    for mode, flags in ([] if args.check_only else [('development', []), ('release', ['--release'])]):
        row = execute(reference, base, ['build', '--locked', *flags], 'Go/build/' + mode)
        assert row['exit'] == 0 and row['stderr'] == ''
        oracles[mode] = row
        fixtures(base)
        returned(execute(base / 'build/std-fs-quickstart.exe', base / 'build', [], 'Go/artifact/' + mode))
        returned(execute(reference, base, ['run', '--locked', *flags], 'Go/run/' + mode), status=356)
    checks = []
    cache_reuse = False
    selected_images = [r for r in build['images'] if r['profile'] == args.profile]
    for selected in selected_images:
        image = Path(selected['path'])
        assert behavior.sha(image) == selected['sha256']
        profile = selected['profile']
        path = workspace(profile)
        shutil.copyfile(base / 'fairy.lock', path / 'fairy.lock')
        row = execute(image, path, ['check', '--locked'], profile + '/check')
        assert row['exit'] == 0 and row['stderr'] == '', row
        checks.append(dict(profile=profile, stdoutMatch=row['stdout'] == golden['stdout']))
        for mode, flags in ([] if args.check_only else [('development', []), ('release', ['--release'])]):
            row = execute(image, path, ['build', '--locked', *flags], profile + '/build/' + mode)
            assert all(row[k] == oracles[mode][k] for k in ['exit', 'stdout', 'stderr']), row
            fixtures(path)
            returned(execute(path / 'build/std-fs-quickstart.exe', path / 'build', [], profile + '/artifact/' + mode))
            returned(execute(image, path, ['run', '--locked', *flags], profile + '/run/' + mode), status=356)
        cache = output / (profile + '-cache')
        cache.mkdir()
        flags = ['--locked', '--cache-root', str(cache), '--cache-status']
        cold = execute(image, path, ['check', *flags], profile + '/cache/cold')
        assert cold['exit'] == 0 and cold['stderr'] in ['cache: cold stored\n', 'cache: cold uncached\n'], cold
        hit = execute(image, path, ['check', *flags], profile + '/cache/hit')
        statuses = ['cache: hit\n', 'cache: units hit\n', 'cache: mixed\n',
                    'cache: cold stored\n', 'cache: cold uncached\n']
        assert hit['exit'] == 0 and hit['stdout'] == cold['stdout'] and hit['stderr'] in statuses, hit
        cache_rows = [hit]
        if not args.check_only:
            built = execute(image, path, ['build', *flags], profile + '/cache/build')
            assert built['exit'] == 0 and built['stderr'] in statuses, built
            run = execute(image, path, ['run', *flags], profile + '/cache/run')
            assert run['stderr'] in statuses, run
            returned(run, run['stderr'], status=356)
            cache_rows.extend([built, run])
        cache_reuse |= any(r['stderr'] in statuses[:3] for r in cache_rows)
        assert behavior.sha(image) == selected['sha256']
        print(profile, 'checks pass' if args.check_only else 'check/build/run and both output profiles pass', flush=True)
    assert behavior.sha(reference) == reference_sha
    if baseline:
        assert behavior.sha(Path(baseline['path'])) == baseline['sha256']
    for row in inputs.values():
        assert behavior.sha(ROOT / row['path']) == row['sha256'], row['path']
    behavior.save(output / 'receipt.json', dict(schema='fairyc.candidate-std-fs/v1', passed=True,
                  mode='check-only' if args.check_only else 'check-build-run',
                  executionTimeoutSeconds=args.timeout, productTimeoutChanged=False,
                  reference=dict(path=str(reference), sha256=reference_sha), profiles=selected_images,
                  sourceInputs=list(inputs.values()), observations=observations, checkStatistics=checks,
                  artifactExpectedExit=None if args.check_only else 42, runExpectedExit=None if args.check_only else 356,
                  cacheReuseObserved=cache_reuse,
                  baseline=baseline,
                  runNote='Both CLIs execute in fresh scratch directories without the external metadata fixtures.',
                  scope=build['entryScope'], goFrontendFallback=False, publicApiChanged=False,
                  sdkAbiChanged=False, ciChanged=False, pushed=False))


if __name__ == '__main__':
    main()
