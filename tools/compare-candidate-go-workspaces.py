"""Compare shipped workspaces and shared command boundaries in isolated copies."""
import argparse
import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('behavior', ROOT / 'tools/compare-candidate-go-behavior.py')
behavior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(behavior)
NATIVE = [
    'workspace_quickstart', 'std_io_quickstart', 'std_io_scalar_quickstart',
    'process_args_quickstart', 'std_alloc_quickstart', 'regions',
    'std_bytes_quickstart', 'std_text_quickstart', 'std_text_from_buffer_quickstart',
    'std_vec_quickstart', 'std_fs_quickstart', 'i64_quickstart',
    'contract_capabilities_quickstart', 'asset_burn_quickstart',
    'copied_mutable_struct_quickstart', 'repeated_array', 'struct_vec_view',
]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--candidate-build-receipt', type=Path, required=True)
    parser.add_argument('--profile', choices=['development', 'release'], default='release')
    parser.add_argument('--suite', choices=['examples', 'cli'], required=True)
    parser.add_argument('--case', action='append', help='compare only these example directory names or CLI ids')
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
    candidate, reference = Path(selected['path']), args.reference.resolve()
    assert behavior.sha(candidate) == selected['sha256']
    reference_sha = behavior.sha(reference)
    inputs = {r['path']: r for r in build['sources']}
    for path in ['tools/compare-candidate-go-behavior.py', 'tools/compare-candidate-go-workspaces.py']:
        inputs[path] = dict(path=path, sha256=behavior.sha(ROOT / path))
    files = subprocess.check_output(['git', 'ls-files', '-z', '--', 'examples', '*.go',
                                    'go.mod', 'go.sum', 'internal/sdkstdlib/sources',
                                    'internal/cstdlib'], cwd=ROOT).split(b'\0')
    tracked = [Path(raw.decode('utf-8')) for raw in files if raw]
    rows, skipped = [], []

    def record(name, go, other, expected_exit=None):
        row = dict(id=name, go=go, candidate=other, comparison=behavior.comparison(go, other))
        if expected_exit is not None:
            row['expectedExit'] = expected_exit
            row['programContractMatch'] = not go['timedOut'] and not other['timedOut'] and go['exit'] == other['exit'] == expected_exit
        rows.append(row)
        behavior.save(output / 'observations.json', rows)
        print(name, go['exit'], other['exit'], flush=True)
        return row

    def run(name, argv, path):
        return record(name, behavior.execute(reference, argv, path, args.timeout),
                      behavior.execute(candidate, argv, path, args.timeout))

    def filesystem_native(path):
        # The stable run command executes in its own scratch directory. This
        # quickstart requires host-prepared metadata files, so build and execute
        # each image in a fresh retained directory, as the native smoke does.
        with tempfile.TemporaryDirectory(prefix='fy-fs-', dir=ROOT / '.cache') as temporary:
            runtime_root = Path(temporary).resolve()
            assert runtime_root.is_relative_to(ROOT / '.cache')
            filesystem_profiles(path, runtime_root)

    def filesystem_profiles(path, runtime_root):
        for profile, flags in [('development', []), ('release', ['--release'])]:
            projects = []
            built = []
            for engine, image in [('go', reference), ('candidate', candidate)]:
                project = runtime_root / profile / engine
                shutil.copytree(path, project)
                projects.append(project)
                built.append(behavior.execute(image, ['build', *flags, '--locked'], project, args.timeout))
            row = record('std_fs_quickstart/build-' + profile, *built, expected_exit=0)
            if not row['programContractMatch']:
                continue
            executed = []
            for project in projects:
                runtime = project / 'build'
                readonly = runtime / 'fairyc-std-fs-read-only-metadata.tmp'
                readonly.write_bytes(b'metadata')
                readonly.chmod(0o444)
                linked = runtime / 'fairyc-std-fs-hard-link-metadata.tmp'
                linked.write_bytes(b'metadata')
                (runtime / 'fairyc-std-fs-hard-link-metadata-alias.tmp').hardlink_to(linked)
                try:
                    executed.append(behavior.execute(runtime / 'std-fs-quickstart.exe', [], runtime, args.timeout))
                finally:
                    readonly.chmod(0o666)
            record('std_fs_quickstart/execute-built-' + profile, *executed, expected_exit=42)

    def copy_example(name):
        source = Path('examples') / name
        target = output / name
        for relative in tracked:
            if relative.is_relative_to(source):
                original = ROOT / relative
                dest = target / relative.relative_to(source)
                dest.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(original, dest)
                key = relative.as_posix()
                inputs[key] = dict(path=key, sha256=behavior.sha(original))
        return target

    for relative in tracked:
        if relative.suffix == '.go' or relative.as_posix().startswith('internal/') or relative.name in ['go.mod', 'go.sum']:
            key = relative.as_posix()
            inputs[key] = dict(path=key, sha256=behavior.sha(ROOT / relative))
    if args.suite == 'examples':
        for manifest in sorted((ROOT / 'examples').glob('*/fairy.toml')):
            name = manifest.parent.name
            if args.case and name not in args.case:
                continue
            if name == 'lexer_candidate':
                skipped.append(dict(id=name, reason='compiler self-source is a separate bootstrap/fixed-point gate'))
                continue
            path = copy_example(name)
            admission = behavior.execute(reference, ['check'], path, args.timeout)
            if admission['exit'] != 0:
                skipped.append(dict(id=name, reason='Go cannot admit this shipped workspace', go=admission))
                continue
            row = run(name + '/check', ['check', '--locked'], path)
            if name not in NATIVE or row['candidate']['exit'] != 0:
                if name in NATIVE:
                    skipped.append(dict(id=name + '/native', reason='candidate workspace check fails'))
                continue
            if name == 'std_fs_quickstart':
                filesystem_native(path)
                continue
            run(name + '/run', ['run', '--locked'], path)
            if name in ['workspace_quickstart', 'std_io_scalar_quickstart', 'std_vec_quickstart']:
                run(name + '/run-release', ['run', '--release', '--locked'], path)
                run(name + '/build', ['build', '--locked'], path)
                run(name + '/build-release', ['build', '--release', '--locked'], path)
    else:
        path = copy_example('workspace_quickstart')
        admission = behavior.execute(reference, ['check'], path, args.timeout)
        assert admission['exit'] == 0
        invocations = [
            ['check', '--locked'], ['check', '--bad'], ['check', '--locked', '--locked'],
            ['check', '--help'], ['check', '--cache-status'], ['check', '--release'],
            ['check', '--', 'argument'], ['build', '--bad'], ['run', '--bad'],
            ['build', '--locked', '--locked'], ['run', '--locked', '--locked'],
            ['build', '--release', '--release'], ['run', '--release', '--release'],
            ['build', '--help'], ['run', '--help'], ['build', '--', 'argument'],
            ['run', '--locked', '--', '--program-option'],
            [], ['--help'], ['--version'], ['--unknown'],
            ['new', '--help'], ['test', '--help'],
            ['test', '--list', '--json', '--locked'],
            ['explain', 'E4101'], ['targets', '--json'],
            ['experimental', '--help'], ['emit'],
        ]
        for index, argv in enumerate(invocations):
            if args.case and f'cli-{index}' not in args.case:
                continue
            run(f'cli-{index}', argv, path)
    assert rows, 'no matching executable cases'
    assert behavior.sha(reference) == reference_sha and behavior.sha(candidate) == selected['sha256']
    for row in inputs.values():
        assert behavior.sha(ROOT / row['path']) == row['sha256'], row['path']
    measures = list(rows[0]['comparison']) if rows else []
    summary = {key: sum(row['comparison'][key] for row in rows) for key in measures}
    report = dict(schema='fairyc.candidate-go-workspace-comparison/v1', comparisonCompleted=True,
                  suite=args.suite, profile=args.profile, cases=len(rows), summary=summary, skipped=skipped,
                  scope=build['entryScope'], candidate=dict(path=str(candidate), sha256=behavior.sha(candidate)),
                  reference=dict(path=str(reference), sha256=reference_sha), sourceInputs=list(inputs.values()),
                  rows=rows, nativeContractFailures=[r['id'] for r in rows if not r.get('programContractMatch', True)],
                  parity=bool(rows) and all(r['comparison']['exactMatch'] and r.get('programContractMatch', True) for r in rows) and not skipped,
                  note='Run status is a program status; check/build status is a compiler status. No timing or IR equality claim.',
                  publicApiChanged=False, sdkAbiChanged=False, ciChanged=False, pushed=False)
    behavior.save(output / 'receipt.json', report)
    print(json.dumps(dict(suite=args.suite, cases=len(rows), summary=summary, skipped=len(skipped)), indent=2), flush=True)


if __name__ == '__main__':
    main()
