"""Build a private process probe of the authored check CLI with candidate C."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
PROJECT = ROOT / 'examples/lexer_candidate'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--candidate', type=Path, required=True)
    p.add_argument('--builder', type=Path, required=True)
    p.add_argument('--bootstrap', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--stack-reserve-bytes', type=int, default=0,
                   help='private compiler probe reserve; zero/default or 64KiB aligned in [64KiB,16MiB]')
    p.add_argument('--native-stack-editor', type=Path,
                   help='explicit editbin for applying the same reserve to the private Go-built dispatcher')
    args = p.parse_args()
    if args.stack_reserve_bytes and (not 65536 <= args.stack_reserve_bytes <= 16777216 or args.stack_reserve_bytes % 65536):
        p.error('stack reserve must be zero or 64KiB aligned in [64KiB,16MiB]')
    if args.stack_reserve_bytes and args.native_stack_editor is None:
        p.error('explicit native stack editor required for a non-default probe reserve')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    entry = output / 'main.fy'
    # This entry only forwards real OS arguments. All checking, rendering, file
    # discovery, lock policy and side effects stay in the authored Fairy code.
    entry.write_text('module main;\nimport project_check_cli as cli;\n@effects(filesystem,console) fn i32 main(){return cli::arguments();}\n', encoding='utf-8', newline='\n')
    modules = {}
    for base in [PROJECT / 'src', ROOT / 'internal/sdkstdlib/sources']:
        for path in base.rglob('*.fy'):
            name = re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))[1]
            assert name not in modules
            modules[name] = path
    modules['main'] = entry
    seen, pending = set(), ['main']
    while pending:
        name = pending.pop()
        if name in seen:
            continue
        seen.add(name)
        pending.extend(re.findall(r'^import\s+([\w:]+)', modules[name].read_text(encoding='utf-8'), re.M))
    paths = [entry] + [modules[name] for name in sorted(seen - {'main'})]
    sources = [dict(path=path.relative_to(ROOT).as_posix(), sha256=sha(path), bytes=path.stat().st_size) for path in paths]
    manifest = output / 'sources.list'
    manifest.write_text(''.join(path.relative_to(ROOT).as_posix() + '\n' for path in paths), encoding='utf-8', newline='\n')
    calls = []
    tools = {name: dict(path=str(path.resolve()), sha256=sha(path)) for name, path in [('candidate', args.candidate), ('builder', args.builder), ('bootstrap', args.bootstrap)]}
    if args.native_stack_editor:
        tools['nativeStackEditor'] = dict(path=str(args.native_stack_editor.resolve()), sha256=sha(args.native_stack_editor))
    dispatcher_sources = [dict(path=path.relative_to(ROOT).as_posix(), sha256=sha(path), bytes=path.stat().st_size)
                          for path in sorted((PROJECT / 'src').glob('*.fy'))]

    def run(argv, timeout, cwd=ROOT):
        start = time.monotonic()
        r = subprocess.run(list(map(str, argv)), cwd=cwd, capture_output=True, timeout=timeout)
        calls.append(dict(argv=list(map(str, argv)), cwd=str(cwd), seconds=time.monotonic()-start, exit=r.returncode,
                          stdoutBytes=len(r.stdout), stdoutSHA256=hashlib.sha256(r.stdout).hexdigest(), stderr=r.stderr.decode('utf-8', errors='replace')))
        (output / 'invocations.json').write_text(json.dumps(calls, indent=2) + '\n', encoding='utf-8', newline='\n')
        assert r.returncode == 0 and not r.stderr, calls[-1]
        return r.stdout

    print(run([args.bootstrap.resolve(), 'check'], 120, PROJECT).decode('utf-8').strip(), flush=True)
    lock = PROJECT / 'fairy.lock'
    lock_digest = sha(lock)
    # Build the exact same authored check closure with the Go frontend. The
    # full dispatcher also imports private Windows adapters outside this probe.
    native_project = output / 'native-project'
    (native_project / 'src').mkdir(parents=True)
    shutil.copyfile(PROJECT / 'fairy.toml', native_project / 'fairy.toml')
    copied = []
    for path in paths:
        if path == entry or path.is_relative_to(PROJECT / 'src'):
            target = native_project / 'src' / path.name
            assert not target.exists(), target
            shutil.copyfile(path, target)
            assert sha(target) == sha(path)
            copied.append(dict(source=path.relative_to(ROOT).as_posix(), path=str(target), sha256=sha(target)))
    run([args.bootstrap.resolve(), 'check'], 120, native_project)
    run([args.bootstrap.resolve(), 'build', '--locked'], 900, native_project)
    dispatcher = output / 'go-dispatch.exe'
    shutil.copyfile(native_project / 'build/lexer-candidate.exe', dispatcher)
    if args.stack_reserve_bytes:
        run([args.native_stack_editor.resolve(), '/nologo', '/stack:' + str(args.stack_reserve_bytes), dispatcher], 30)
    print('isolated Go-frontend check probe built', flush=True)
    packet = output / 'candidate-packet.bin'
    packet.write_bytes(run([args.candidate.resolve(), '--c-project-file', manifest], 900))
    assert packet.read_bytes().startswith(b'FYCPARTS1\n')
    images = []
    for name, flags in [('development', []), ('release', ['--release'])]:
        image = output / (name + '.exe')
        reserve = [] if not args.stack_reserve_bytes else ['--stack-reserve-bytes', str(args.stack_reserve_bytes)]
        run([args.builder.resolve(), '--packet', packet, '--packet-sha256', sha(packet), '--output', image, *flags, *reserve], 600)
        images.append(dict(profile=name, path=str(image), sha256=sha(image)))
        print(name, 'probe built', flush=True)
    assert all(sha(ROOT / s['path']) == s['sha256'] for s in sources)
    assert all(sha(ROOT / s['path']) == s['sha256'] for s in dispatcher_sources)
    assert all(sha(Path(s['path'])) == s['sha256'] for s in copied)
    assert sha(lock) == lock_digest
    assert all(sha(Path(t['path'])) == t['sha256'] for t in tools.values())
    receipt = dict(schema='fairyc.candidate-check-build/v1', entryScope='project_check_cli::arguments with real OS argv; excludes file_cli dispatcher',
                   sources=sources, dispatcherSources=dispatcher_sources,
                   dispatcherImage=dict(path=str(dispatcher), sha256=sha(dispatcher)),
                   lockSHA256=lock_digest, tools=tools, images=images, candidatePacketSHA256=sha(packet), invocations=calls,
                   stackReserveBytes=args.stack_reserve_bytes, productStackChanged=False,
                   nativeIsolation=dict(path=str(native_project), copiedInputs=copied))
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')


if __name__ == '__main__':
    main()
