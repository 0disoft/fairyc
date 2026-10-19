"""Build the authored native companion and its pinned runtime distribution asset."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--candidate', required=True, type=Path)
    parser.add_argument('--builder', required=True, type=Path, help='explicit bootstrap builder')
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    assert __debug__, 'run without -O'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    entry = output / 'main.fy'
    entry.write_text('module main;\nimport native_build_cli as cli;\n'
                     '@effects(filesystem,console,environment,clock) fn i32 main(){return cli::run();}\n',
                     encoding='utf-8', newline='\n')
    modules = {}
    for base in (ROOT / 'examples/lexer_candidate/src', ROOT / 'internal/sdkstdlib/sources'):
        for path in base.rglob('*.fy'):
            name = re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))[1]
            assert name not in modules
            modules[name] = path
    modules['main'] = entry
    seen, pending = set(), ['main']
    while pending:
        name = pending.pop()
        if name not in seen:
            seen.add(name)
            pending.extend(re.findall(r'^import\s+([\w:]+)', modules[name].read_text(encoding='utf-8'), re.M))
    paths = [entry] + [modules[name] for name in sorted(seen - {'main'})]
    assert len(paths) <= 512 and all(p.stat().st_size <= 65536 for p in paths)
    manifest = output / 'sources.list'
    manifest.write_text(''.join(p.relative_to(ROOT).as_posix()+'\n' for p in paths), encoding='utf-8')
    sources = [dict(path=p.relative_to(ROOT).as_posix(), sha256=sha(p)) for p in paths]
    identities = {name: dict(path=str(p.resolve()), sha256=sha(p))
                  for name, p in [('candidate', args.candidate), ('bootstrapBuilder', args.builder)]}
    calls = []

    def run(argv):
        started = time.monotonic()
        result = subprocess.run(list(map(str, argv)), cwd=ROOT, capture_output=True, timeout=900)
        calls.append(dict(argv=list(map(str, argv)), seconds=time.monotonic()-started,
                          exit=result.returncode, stderr=result.stderr.decode(errors='replace'),
                          stdoutBytes=len(result.stdout), stdoutSHA256=hashlib.sha256(result.stdout).hexdigest()))
        (output / 'invocations.json').write_text(json.dumps(calls, indent=2)+'\n', encoding='utf-8')
        assert result.returncode == 0 and not result.stderr, calls[-1]
        return result.stdout

    packet = output / 'driver.bin'
    packet.write_bytes(run([args.candidate.resolve(), '--c-project-file', manifest]))
    spec = importlib.util.spec_from_file_location('assets', ROOT / 'tools/package-candidate-runtime-assets.py')
    assets = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(assets)
    data, pin = assets.pack_and_pin()
    assert assets.PIN.read_text(encoding='utf-8') == pin
    raw_sources = [dict(path=p, sha256=sha(ROOT / p)) for p in assets.SOURCES]
    images = []
    for profile, flags in [('development', []), ('release', ['--release'])]:
        directory = output / profile
        directory.mkdir()
        image = directory / 'fairyc-candidate-build.exe'
        run([args.builder.resolve(), '--packet-input', packet, '--output', image, *flags])
        pack = directory / 'fairyc-runtime-assets.pack'
        pack.write_bytes(data)
        images.append(dict(profile=profile, path=str(image), sha256=sha(image),
                           assetPath=str(pack), assetSHA256=sha(pack)))
        print(profile, 'authored native driver built', flush=True)
    assert all(sha(ROOT / row['path']) == row['sha256'] for row in sources + raw_sources)
    assert all(sha(Path(row['path'])) == row['sha256'] for row in identities.values())
    receipt = dict(schema='fairyc.candidate-authored-driver-build/v1', passed=True,
                   sourceFiles=len(paths), sources=sources, runtimeSources=raw_sources,
                   tools=identities, invocations=calls, images=images,
                   packetSHA256=sha(packet), goFrontendInvoked=False,
                   runtimeGoBuilderRequired=False, bootstrapBuilderRequired=True)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8')


if __name__ == '__main__':
    main()
