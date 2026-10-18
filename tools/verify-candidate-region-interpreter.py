"""Run the authored Region interpreter against checked IR and the authentic SDK.

Uses a native host harness to inject abstract Arena arguments, without pretending
that public semantic tests can execute SDK Arena host constructors/destructors.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
TEST = ROOT / 'examples/lexer_candidate/tests/module_unit_interpreter_regions.fy'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--release', action='store_true')
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    home = output / 'workspace'
    (home / 'src').mkdir(parents=True)
    (home / 'tests').mkdir()
    modules = {}
    for path in (ROOT / 'examples/lexer_candidate/src').glob('*.fy'):
        modules[re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))[1]] = path
    pending = re.findall(r'^import\s+([\w:]+)', TEST.read_text(encoding='utf-8'), re.M)
    used = set()
    sources = [TEST]
    while pending:
        name = pending.pop()
        if name in used or name.startswith('std::'):
            continue
        used.add(name)
        path = modules[name]
        sources.append(path)
        shutil.copyfile(path, home / 'src' / path.name)
        pending.extend(re.findall(r'^import\s+([\w:]+)', path.read_text(encoding='utf-8'), re.M))
    shutil.copyfile(TEST, home / 'tests' / TEST.name)
    (home / 'fairy.toml').write_text(
        'schema = "fairyc.manifest/v1"\n[package]\nuid = "probe.region-interpreter"\n'
        'name = "region-interpreter"\nversion = "0.1.0"\n'
        'language = "fairyc-language/20"\nsource = "src"\n', encoding='utf-8')
    snapshots = {p.relative_to(ROOT).as_posix(): digest(p) for p in sources}
    allocator = ROOT / 'internal/sdkstdlib/sources/std/alloc.fy'
    snapshots[allocator.relative_to(ROOT).as_posix()] = digest(allocator)
    calls = []

    def run(argv):
        start = time.monotonic()
        result = subprocess.run([str(args.reference.resolve()), *argv], cwd=home,
                                input=b'', capture_output=True, timeout=900)
        row = dict(argv=argv, exit=result.returncode, stdout=result.stdout.decode(),
                   stderr=result.stderr.decode(), seconds=time.monotonic() - start)
        calls.append(row)
        (output / 'observations.json').write_text(json.dumps(calls, indent=2) + '\n', encoding='utf-8')
        assert result.returncode == 0, row
        return result

    run(['check'])
    flags = ['--release'] if args.release else []
    result = run(['test', '--native', '--locked', '--json', '--capture-output',
                  '--timeout-seconds', '60', *flags, '--', str(allocator)])
    report = json.loads(result.stdout)
    assert report['count'] == report['passed'] == 2 and report['failed'] == 0, report
    assert report['status'] == 'ok' and report['engine'] == 'native', report
    assert all(digest(ROOT / p) == sha for p, sha in snapshots.items())
    receipt = dict(schema='fairyc.candidate-region-interpreter/v1', passed=True,
                   profile='release' if args.release else 'development',
                   referenceSHA256=digest(args.reference), verifierSHA256=digest(Path(__file__)),
                   sources=snapshots, invocations=calls, report=report,
                   authenticSDK=True, abstractArenaArguments=True,
                   publicCLIRegionParity=False, fullCandidateRebuilt=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print('Region interpreter harness passed', flush=True)


if __name__ == '__main__':
    main()
