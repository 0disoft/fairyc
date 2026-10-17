"""Reuse an authenticated CLI build as Stage1 and verify its next C generation.

The source manifest and frozen version module remain exactly those of Stage1.
This verifies the full file_cli dispatcher, not the separate stdin lexer protocol.
"""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def artifact(path):
    return dict(path=str(path.resolve()), bytes=path.stat().st_size, sha256=sha(path))


def stack_reserve(path):
    raw = path.read_bytes()
    assert raw[:2] == b'MZ'
    pe = struct.unpack_from('<I', raw, 60)[0]
    assert raw[pe:pe+4] == b'PE\0\0' and struct.unpack_from('<H', raw, pe+24)[0] == 0x20b
    return struct.unpack_from('<Q', raw, pe+24+72)[0]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--frontend-timeout', type=int, default=900,
                        help='verification-only stage allowance, 1..1800 seconds')
    parser.add_argument('--compile-timeout', type=int, default=900,
                        help='verification-only stage allowance, 1..1800 seconds')
    args = parser.parse_args()
    assert __debug__ and 1 <= args.frontend_timeout <= 1800 and 1 <= args.compile_timeout <= 1800
    receipt_path = args.build_receipt.resolve()
    receipt_identity = artifact(receipt_path)
    built = json.loads(receipt_path.read_text(encoding='utf-8'))
    assert built['schema'] == 'fairyc.candidate-native-probe-build/v1' and built['passed']
    assert not any(built.get(k) for k in ('goFrontendInvoked', 'commandsOnly', 'compilerOnly', 'frontendReuse'))
    inputs = built['sources']
    assert 0 < len(inputs) <= 512
    for row in inputs:
        assert sha(ROOT / row['path']) == row['sha256'], 'source changed: ' + row['path']
    version = built['buildVersion']
    assert sha(ROOT / version['template']['path']) == version['template']['sha256']
    assert sha(Path(version['oracle']['path'])) == version['oracle']['sha256']
    source = receipt_path.parent
    manifest = source / 'sources.list'
    assert manifest.read_text(encoding='utf-8') == ''.join(row['path']+'\n' for row in inputs)
    packet1 = source / 'candidate-packet.bin'
    assert sha(packet1) == built['candidatePacketSHA256'] and packet1.read_bytes().startswith(b'FYCPARTS1\n')
    for item in built['tools'].values():
        assert sha(Path(item['path'])) == item['sha256']
    first = built['invocations'][0]
    assert first['argv'] == [built['tools']['candidate']['path'], '--c-project-file', str(manifest)]
    assert first['exit'] == 0 and not first['stderr']
    assert first['stdoutSHA256'] == sha(packet1) and first['stdoutBytes'] == packet1.stat().st_size
    release = next(row for row in built['images'] if row['profile'] == 'release')
    stage1, builder = Path(release['path']), Path(built['tools']['builder']['path'])
    stage1_identity, builder_identity = artifact(stage1), artifact(builder)
    assert stage1_identity['sha256'] == release['sha256']
    assert sha(stage1.with_name('fairyc-candidate-build.exe')) == release['driverSHA256'] == sha(builder)
    build1 = next(row for row in built['invocations'] if '--output' in row['argv']
                  and row['argv'][row['argv'].index('--output')+1] == str(stage1))
    assert build1['exit'] == 0 and not build1['stderr'] and '--release' in build1['argv']
    assert build1['argv'][0] == str(builder)
    assert build1['argv'][build1['argv'].index('--packet')+1] == str(packet1)
    assert build1['argv'][build1['argv'].index('--packet-sha256')+1] == sha(packet1)
    reserve = built['stackReserveBytes']
    flags = ['--stack-reserve-bytes', str(reserve)] if reserve else []
    assert build1['argv'] == list(map(str, [builder, '--packet', packet1, '--packet-sha256',
                                           sha(packet1), '--output', stage1, '--release', *flags]))
    assert not reserve or stack_reserve(stage1) == reserve
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    calls = []

    def save(name, value):
        (output / name).write_text(json.dumps(value, indent=2)+'\n', encoding='utf-8', newline='\n')

    def run(argv, timeout, destination=None):
        start = time.monotonic()
        process = subprocess.Popen(list(map(str, argv)), cwd=ROOT, stdin=subprocess.DEVNULL,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
        timed_out = False
        try:
            out, err = process.communicate(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out = True
            subprocess.run(['taskkill', '/PID', str(process.pid), '/T', '/F'],
                           capture_output=True, timeout=15)
            if process.poll() is None:
                process.kill()
            out, err = process.communicate()
        row = dict(argv=list(map(str, argv)), exit=process.returncode, timedOut=timed_out,
                   seconds=time.monotonic()-start, stdoutBytes=len(out),
                   stdoutSHA256=hashlib.sha256(out).hexdigest(), stderr=err.decode('utf-8', errors='replace'))
        calls.append(row)
        if destination is not None:
            destination.write_bytes(out)
        save('progress.json', dict(passed=False, adoptedBuild=receipt_identity, invocations=calls))
        assert not timed_out and process.returncode == 0 and not err, row
        print('passed:', round(row['seconds'], 3), argv[1], flush=True)
        return out

    packet2 = output / 'stage2-input.bin'
    run([stage1, '--c-project-file', manifest], args.frontend_timeout, packet2)
    assert packet2.read_bytes() == packet1.read_bytes(), 'consecutive C packets differ'
    print('consecutive C packets exactly equal', flush=True)
    stage2 = output / 'stage2.exe'
    run([builder, '--packet', packet2, '--packet-sha256', sha(packet2), '--output', stage2,
         '--release', *flags], args.compile_timeout)
    assert stage2.read_bytes() == stage1.read_bytes(), 'consecutive optimized PE images differ'
    assert stack_reserve(stage1) == stack_reserve(stage2)
    assert not reserve or stack_reserve(stage2) == reserve
    assert run([stage2, '--version'], 30) == built['buildVersion']['text'].encode('utf-8')
    for row in inputs:
        assert sha(ROOT / row['path']) == row['sha256'], row['path']
    assert sha(ROOT / version['template']['path']) == version['template']['sha256']
    assert sha(Path(version['oracle']['path'])) == version['oracle']['sha256']
    assert artifact(receipt_path) == receipt_identity and artifact(stage1) == stage1_identity
    assert artifact(builder) == builder_identity and sha(packet1) == built['candidatePacketSHA256']
    save('receipt.json', dict(schema='fairyc.candidate-native-cli-fixed-point/v1', passed=True,
         scope='Consecutive full file_cli C-part generations; exact complete packet and optimized PE bytes',
         adoptedStage1Build=receipt_identity, sourceInputs=inputs, sourceFiles=len(inputs),
         frozenBuildVersion=built['buildVersion'], originalFrontend=first, originalReleaseBuild=build1,
         invocations=calls, packets=[artifact(packet1), artifact(packet2)],
         stage1=stage1_identity, stage2=artifact(stage2), builder=builder_identity,
         packetBytesEqual=True, optimizedImageBytesEqual=True, excludedImageBytes=[],
         stackReserveBytes=stack_reserve(stage2), goFrontendInvoked=False,
         productTimeoutChanged=False, pushed=False,
         limitations=['Windows full file_cli dispatcher only; stdin lexer protocol and non-Windows targets are not included.']))
    print('full CLI consecutive-generation fixed point verified', flush=True)


if __name__ == '__main__':
    main()
