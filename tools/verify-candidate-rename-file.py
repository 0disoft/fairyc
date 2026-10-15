"""Execute current Fairy C-emitter sources against real file-operation runtimes."""
import argparse
import importlib.util
import json
from pathlib import Path
import re
import shutil
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
PROJECT = ROOT / 'examples/lexer_candidate'
OPERATIONS = {
    'rename-file': ('sdk_c_executes_file_rename_preserves_paths_and_live_owner', 541, 'rename-probe'),
    'hard-link': ('sdk_c_executes_file_hard_link_shared_writes_and_alias_survival', 569, 'hard-link-probe'),
}
spec = importlib.util.spec_from_file_location('behavior', ROOT / 'tools/compare-candidate-go-behavior.py')
behavior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(behavior)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--operation', choices=OPERATIONS, default='rename-file')
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--baseline', type=Path, required=True)
    parser.add_argument('--vcvars', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    case, baseline_line, program = OPERATIONS[args.operation]
    identity = 'tests::module_unit_c_fs::' + case
    assert __debug__ and sys.platform == 'win32'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    reference, baseline = args.reference.resolve(), args.baseline.resolve()
    tools = {name: dict(path=str(path), sha256=behavior.sha(path))
             for name, path in [('reference', reference), ('baseline', baseline)]}
    modules = {}
    for base in (PROJECT / 'src', PROJECT / 'tests', ROOT / 'internal/sdkstdlib/sources'):
        for path in base.rglob('*.fy'):
            name = re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))[1]
            assert name not in modules
            modules[name] = path
    seen, pending = set(), ['tests::module_unit_c_fs']
    while pending:
        name = pending.pop()
        if name not in seen:
            seen.add(name)
            pending.extend(re.findall(r'^import\s+([\w:]+)', modules[name].read_text(encoding='utf-8'), re.M))
    paths = [modules[name] for name in sorted(seen)]
    fixture = ROOT / ('tools/fixtures/candidate-' + args.operation + '.fy')
    extra = [fixture, PROJECT / 'fairy.toml', Path(__file__), ROOT / 'tools/verify-candidate-sdk-c.py',
             ROOT / 'tools/verify-candidate-module-c.py', ROOT / 'tools/compare-candidate-go-behavior.py']
    inputs = [dict(path=p.relative_to(ROOT).as_posix(), sha256=behavior.sha(p)) for p in paths + extra]
    observations = []

    def execute(image, path, argv, label, capture=False):
        row = behavior.execute(image, argv, path, 180)
        observations.append(dict(id=label, **{k: v for k, v in row.items() if not capture or k != 'stdout'}))
        behavior.save(output / 'observations.json', observations)
        assert not row['timedOut'], label
        return row

    def succeeds(row):
        assert row['exit'] == 0 and row['stderr'] == '', {k: v for k, v in row.items() if k != 'stdout'}

    native = output / 'emitter-project'
    for original in paths:
        if original.is_relative_to(PROJECT):
            dest = native / original.relative_to(PROJECT)
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(original, dest)
    shutil.copyfile(PROJECT / 'fairy.toml', native / 'fairy.toml')
    (native / 'src/main.fy').write_text('module main;fn i32 main(){return 0;}\n', encoding='utf-8')
    succeeds(execute(reference, native, ['check'], 'emitter/check'))
    argv = ['test', '--native', '--locked', '--json', '--capture-output', '--exact', identity, '--',
            ROOT / 'internal/sdkstdlib/sources/std/fs.fy', ROOT / 'internal/sdkstdlib/sources/std/bytes.fy', fixture]
    row = execute(reference, native, argv, 'emitter/native', capture=True)
    (output / 'native.json').write_text(row['stdout'], encoding='utf-8', newline='\n')
    succeeds(row)
    report = json.loads(row['stdout'])
    assert report['engine'] == 'native' and report['passed'] == 1 and report['failed'] == 0
    assert [r['identity'] for r in report['tests']] == [identity]
    row = execute(Path(sys.executable), ROOT, [ROOT / 'tools/verify-candidate-sdk-c.py',
        '--report', output / 'native.json', '--case', case, '--vcvars', args.vcvars.resolve(),
        '--output', output / 'c-execution'], 'candidate/C-execution')
    succeeds(row)

    oracle = output / 'Go'
    (oracle / 'src').mkdir(parents=True)
    (oracle / 'fairy.toml').write_text('schema = "fairyc.manifest/v1"\n[package]\n'
        f'uid = "test.candidate-{args.operation}"\nname = "{program}"\nversion = "0.1.0"\n'
        'language = "fairyc-language/20"\nsource = "src"\n', encoding='utf-8', newline='\n')
    shutil.copyfile(fixture, oracle / 'src/main.fy')
    succeeds(execute(reference, oracle, ['check'], 'Go/check'))
    row = execute(baseline, oracle, ['build', '--locked'], 'baseline/build')
    assert row['exit'] == 2 and row['stdout'] == '' and row['stderr'] == (
        f'@sdk/stdlib/std/fs.fy:{baseline_line}:9: fairyc: candidate workspace C emission failed (code 1004)\n'), row
    for profile, flags in [('development', []), ('release', ['--release'])]:
        succeeds(execute(reference, oracle, ['build', '--locked', *flags], 'Go/build/' + profile))
        with tempfile.TemporaryDirectory(prefix=args.operation + '-', dir=output) as scratch:
            row = execute(oracle / ('build/' + program + '.exe'), Path(scratch), [], 'Go/artifact/' + profile)
            succeeds(row)
            assert row['stdout'] == '' and not list(Path(scratch).iterdir()), row
        row = execute(reference, oracle, ['run', '--locked', *flags], 'Go/run/' + profile)
        succeeds(row)
        assert row['stdout'] == ''
    for item in inputs:
        assert behavior.sha(ROOT / item['path']) == item['sha256'], item['path']
    for item in tools.values():
        assert behavior.sha(Path(item['path'])) == item['sha256']
    c = json.loads((output / 'c-execution/receipt.json').read_bytes())
    assert c['passed'] and len(c['executions']) == 2
    behavior.save(output / 'receipt.json', dict(schema='fairyc.candidate-' + args.operation + '/v1', passed=True,
        sourceInputs=inputs, tools=tools, observations=observations, CExecution=c,
        emitterExecution='current authored Fairy sources compiled by Go native test driver',
        candidateCLIRebuilt=False, sdkABIChanged=False, publicAPIChanged=False,
        productTimeoutChanged=False, ciChanged=False, pushed=False,
        skipped=['full candidate CLI rebuild and full-workspace native parity; focused C-emitter/runtime regression only']))
    print(args.operation + ': current Fairy emitter, both generated C profiles and Go oracle pass')


if __name__ == '__main__':
    main()
