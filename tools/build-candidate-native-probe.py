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
    parser.add_argument('--stack-reserve-bytes', type=int, default=0)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    if args.stack_reserve_bytes and not (65536 <= args.stack_reserve_bytes <= 16777216 and args.stack_reserve_bytes % 65536 == 0):
        parser.error('stack reserve must be zero or 64KiB aligned in [64KiB,16MiB]')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    entry = output / 'main.fy'
    entry.write_text('module main;\nimport file_cli as cli;\n@effects(filesystem,console,environment,clock) fn i32 main(){return cli::run();}\n', encoding='utf-8', newline='\n')
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
    manifest.write_text(''.join(p.relative_to(ROOT).as_posix() + '\n' for p in paths), encoding='utf-8', newline='\n')
    calls = []

    def run(argv, timeout):
        started = time.monotonic()
        result = subprocess.run(list(map(str, argv)), cwd=ROOT, capture_output=True, timeout=timeout)
        calls.append(dict(argv=list(map(str, argv)), seconds=time.monotonic()-started, exit=result.returncode, stdoutBytes=len(result.stdout), stdoutSHA256=hashlib.sha256(result.stdout).hexdigest(), stderr=result.stderr.decode('utf-8', errors='replace')))
        (output / 'invocations.json').write_text(json.dumps(calls, indent=2) + '\n', encoding='utf-8', newline='\n')
        assert result.returncode == 0 and not result.stderr, calls[-1]
        return result.stdout

    packet = output / 'candidate-packet.bin'
    packet.write_bytes(run([args.candidate.resolve(), '--c-project-file', manifest], 900))
    assert packet.read_bytes().startswith(b'FYCPARTS1\n')
    print('authored dispatcher emitted:', len(paths), 'sources,', packet.stat().st_size, 'packet bytes', flush=True)
    images = []
    for profile, flags in [('development', []), ('release', ['--release'])]:
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
        print(profile, 'native dispatcher built', flush=True)
    assert all(sha(ROOT / s['path']) == s['sha256'] for s in sources)
    assert all(sha(Path(t['path'])) == t['sha256'] for t in tools.values())
    assert sha(PROJECT / 'fairy.lock') == lock_sha
    receipt = dict(schema='fairyc.candidate-native-probe-build/v1', passed=True, entryScope='authored file_cli::run with real OS argv', sources=sources, tools=tools, lockSHA256=lock_sha, images=images, candidatePacketSHA256=sha(packet), invocations=calls, stackReserveBytes=args.stack_reserve_bytes, goFrontendInvoked=False, productStackChanged=False, productTimeoutChanged=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')


if __name__ == '__main__':
    main()
