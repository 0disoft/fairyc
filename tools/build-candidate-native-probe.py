"""Build the authored Fairy file_cli dispatcher through candidate C, without Go fallback."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import struct
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
PROJECT = ROOT / 'examples/lexer_candidate'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--builder', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--reuse-frontend', type=Path,
                        help='reuse a successful source-authenticated C emission while rebuilding native images')
    parser.add_argument('--stack-reserve-bytes', type=int,
                        help='private compiler-host reservation; omitted inherits the seed PE, zero uses the linker default')
    parser.add_argument('--profile', action='append', choices=['development', 'release'],
                        help='build only the selected compiler profile; repeat for both (default: both)')
    parser.add_argument('--commands-only', action='store_true',
                        help='verify the authored check/build/run command entry points without unrelated private frontend commands')
    parser.add_argument('--compiler-only', action='store_true',
                        help='build only the private --c-project-file transport as a bounded bootstrap seed')
    args = parser.parse_args()
    if args.commands_only and args.compiler_only:
        parser.error('commands-only and compiler-only are mutually exclusive')
    if not __debug__:
        parser.error('verification requires Python without -O')
    inherited_stack = args.stack_reserve_bytes is None
    if inherited_stack:
        seed = args.candidate.read_bytes()
        assert seed[:2] == b'MZ', 'candidate must be a Windows PE image'
        offset = struct.unpack_from('<I', seed, 60)[0]
        assert seed[offset:offset+4] == b'PE\0\0' and struct.unpack_from('<H', seed, offset+24)[0] == 0x20b
        args.stack_reserve_bytes = struct.unpack_from('<Q', seed, offset+24+72)[0]
    if args.stack_reserve_bytes and not (65536 <= args.stack_reserve_bytes <= 16777216 and args.stack_reserve_bytes % 65536 == 0):
        parser.error('stack reserve must be zero or 64KiB aligned in [64KiB,16MiB]')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    entry = output / 'main.fy'
    command_entry = '''module main;
import project_check_cli as check;
import native_cli as native;
import std::bytes as bytes;
import std::process as process;
@effects(filesystem,console,environment,clock) fn i32 main(){
    if(process::arg_count()<2){return 2;}var u8[16] name=[0;16];
    let n=switch(process::arg(1,ref mut name)){case .ok(p):{p}case .err(_):{return 2;}};
    let ref u8[] view=ref name;
    if(bytes::equal(view[..n],b"check")){return check::arguments();}
    if(bytes::equal(view[..n],b"build")){return native::run(false);}
    if(bytes::equal(view[..n],b"run")){return native::run(true);}return 2;
}
'''
    compiler_entry = '''module main;
import module_project_cli as project;
import std::process as process;
import std::bytes as bytes;
@effects(filesystem,console) fn i32 main(){
    if(process::arg_count()!=3){return 2;}
    var u8[32] command=[0;32];let count=switch(process::arg(1,ref mut command)){case .ok(n):{n}case .err(_):{return 2;}};
    let ref u8[] flag=ref command;if(!bytes::equal(flag[..count],b"--c-project-file")){return 2;}
    var u8[4096] path=[0;4096];let size=switch(process::arg(2,ref mut path)){case .ok(n):{n}case .err(_):{return 2;}};
    let ref u8[] name=ref path;return project::run(name[..size]);
}
'''
    entry.write_text(compiler_entry if args.compiler_only else command_entry if args.commands_only else 'module main;\nimport file_cli as cli;\n@effects(filesystem,console,environment,clock) fn i32 main(){return cli::run();}\n', encoding='utf-8', newline='\n')
    modules = {}
    for base in (PROJECT / 'src', ROOT / 'internal/sdkstdlib/sources'):
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
    sources = [dict(path=p.relative_to(ROOT).as_posix(), sha256=sha(p), bytes=p.stat().st_size) for p in paths]
    tools = {name: dict(path=str(p.resolve()), sha256=sha(p)) for name, p in [('candidate', args.candidate), ('builder', args.builder)]}
    lock_sha = sha(PROJECT / 'fairy.lock')
    manifest = output / 'sources.list'
    calls = []
    frontend_reuse = None
    reused_packet = None
    if args.reuse_frontend:
        prior = args.reuse_frontend.resolve()
        snapshot_path = prior / 'source-snapshot.json'
        snapshot = json.loads(snapshot_path.read_text(encoding='utf-8'))
        original = snapshot['sources']
        # Only the generated entry's path may differ. Bind all bytes, every
        # other source identity, the seed executable, and the production lock.
        assert original[0]['sha256'] == sources[0]['sha256']
        assert original[1:] == sources[1:]
        assert snapshot['tools']['candidate'] == tools['candidate']
        assert snapshot['lockSHA256'] == lock_sha
        assert all(sha(ROOT / row['path']) == row['sha256'] for row in original)
        prior_manifest = prior / 'sources.list'
        assert prior_manifest.read_text(encoding='utf-8') == ''.join(row['path'] + '\n' for row in original)
        prior_calls_path = prior / 'invocations.json'
        first = json.loads(prior_calls_path.read_text(encoding='utf-8'))[0]
        assert first['argv'] == [str(args.candidate.resolve()), '--c-project-file', str(prior_manifest)]
        reused_packet = prior / 'candidate-packet.bin'
        assert first['exit'] == 0 and not first['stderr']
        assert first['stdoutBytes'] == reused_packet.stat().st_size and first['stdoutSHA256'] == sha(reused_packet)
        assert reused_packet.read_bytes().startswith(b'FYCPARTS1\n')
        sources = original
        calls.append(dict(first, reused=True))
        frontend_reuse = dict(path=str(prior), sourceSnapshotSHA256=sha(snapshot_path),
                              invocationsSHA256=sha(prior_calls_path), packetSHA256=sha(reused_packet))
    manifest.write_text(''.join(row['path'] + '\n' for row in sources), encoding='utf-8', newline='\n')
    (output / 'source-snapshot.json').write_text(
        json.dumps(dict(sources=sources, tools=tools, lockSHA256=lock_sha), indent=2) + '\n',
        encoding='utf-8', newline='\n')

    def receipt(path, images):
        assert all(sha(ROOT / s['path']) == s['sha256'] for s in sources)
        assert all(sha(Path(t['path'])) == t['sha256'] for t in tools.values())
        assert all(sha(Path(row['path'])) == row['sha256'] for row in images)
        assert sha(PROJECT / 'fairy.lock') == lock_sha
        data = dict(schema='fairyc.candidate-native-probe-build/v1', passed=True,
                    entryScope='private --c-project-file transport only' if args.compiler_only else 'authored check/build/run entry points with real OS argv' if args.commands_only else 'authored file_cli::run with real OS argv',
                    commandsOnly=args.commands_only, compilerOnly=args.compiler_only, sources=sources, tools=tools,
                    frontendReuse=frontend_reuse,
                    lockSHA256=lock_sha, images=images, candidatePacketSHA256=sha(packet),
                    invocations=calls, stackReserveBytes=args.stack_reserve_bytes,
                    stackReserveOrigin='seed PE' if inherited_stack else 'explicit',
                    goFrontendInvoked=False, productStackChanged=False, productTimeoutChanged=False)
        path.write_text(json.dumps(data, indent=2) + '\n', encoding='utf-8', newline='\n')

    def run(argv, timeout):
        started = time.monotonic()
        result = subprocess.run(list(map(str, argv)), cwd=ROOT, capture_output=True, timeout=timeout)
        calls.append(dict(argv=list(map(str, argv)), seconds=time.monotonic()-started, exit=result.returncode, stdoutBytes=len(result.stdout), stdoutSHA256=hashlib.sha256(result.stdout).hexdigest(), stderr=result.stderr.decode('utf-8', errors='replace')))
        (output / 'invocations.json').write_text(json.dumps(calls, indent=2) + '\n', encoding='utf-8', newline='\n')
        assert result.returncode == 0 and not result.stderr, calls[-1]
        return result.stdout

    packet = output / 'candidate-packet.bin'
    if reused_packet:
        shutil.copyfile(reused_packet, packet)
        assert sha(packet) == frontend_reuse['packetSHA256']
        (output / 'invocations.json').write_text(json.dumps(calls, indent=2) + '\n', encoding='utf-8')
    else:
        packet.write_bytes(run([args.candidate.resolve(), '--c-project-file', manifest], 900))
    assert packet.read_bytes().startswith(b'FYCPARTS1\n')
    print('authored dispatcher emitted:', len(paths), 'sources,', packet.stat().st_size, 'packet bytes', flush=True)
    images = []
    for profile, flags in [('development', []), ('release', ['--release'])]:
        if args.profile and profile not in args.profile:
            continue
        directory = output / profile
        directory.mkdir()
        image = directory / 'fairyc.exe'
        reserve = ['--stack-reserve-bytes', str(args.stack_reserve_bytes)] if args.stack_reserve_bytes else []
        run([args.builder.resolve(), '--packet', packet, '--packet-sha256', sha(packet), '--output', image, *flags, *reserve], 900)
        if args.stack_reserve_bytes:
            data = image.read_bytes()
            offset = struct.unpack_from('<I', data, 60)[0]
            assert data[:2] == b'MZ' and data[offset:offset+4] == b'PE\0\0'
            assert struct.unpack_from('<H', data, offset+24)[0] == 0x20b
            assert struct.unpack_from('<Q', data, offset+24+72)[0] == args.stack_reserve_bytes
        companion = directory / 'fairyc-candidate-build.exe'
        shutil.copyfile(args.builder.resolve(), companion)
        assert sha(companion) == tools['builder']['sha256']
        images.append(dict(profile=profile, path=str(image), sha256=sha(image), driverSHA256=sha(companion)))
        receipt(output / f'receipt-{profile}.json', images[-1:])
        print(profile, 'native dispatcher built', flush=True)
    receipt(output / 'receipt.json', images)


if __name__ == '__main__':
    main()
