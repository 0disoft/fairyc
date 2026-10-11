"""Build or verify consecutive Fairy C-part generations without a Go frontend fallback."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def artifact(path):
    return dict(path=str(path.resolve()), bytes=path.stat().st_size, sha256=digest(path))


def checked_artifact(info):
    path = Path(info['path'])
    assert artifact(path) == info, 'artifact changed: '+str(path)
    return path


def sources():
    modules = {}
    for base in (ROOT / 'examples/lexer_candidate/src', ROOT / 'internal/sdkstdlib/sources'):
        for path in base.rglob('*.fy'):
            match = re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))
            assert match and match[1] not in modules, path
            modules[match[1]] = path
    seen, pending = set(), ['main']
    while pending:
        name = pending.pop()
        if name in seen:
            continue
        seen.add(name)
        pending.extend(re.findall(r'^import\s+([\w:]+)', modules[name].read_text(encoding='utf-8'), re.M))
    paths = [modules['main']] + [modules[name] for name in sorted(seen - {'main'})]
    assert 0 < len(paths) <= 512
    assert all(path.stat().st_size <= 65536 for path in paths)
    assert sum(path.stat().st_size for path in paths)+len(paths)-1 <= 4194304
    return [dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in paths]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--seed', type=Path, required=True)
    parser.add_argument('--builder', type=Path)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--reuse-receipt', type=Path)
    parser.add_argument('--frontend-timeout', type=int, default=60, help='verification-only frontend allowance, 1..900 seconds')
    args = parser.parse_args()
    if not 1 <= args.frontend_timeout <= 900:
        parser.error('frontend timeout must be in 1..900 seconds')
    # Assertions are deliberate fail-closed checks, not optional debugging.
    if not __debug__:
        parser.error('verification requires Python without -O')
    seed_identity = artifact(args.seed)
    args.output.mkdir(parents=True, exist_ok=True)
    original = sources()
    inventory = ''.join(item['path']+' '+item['sha256']+'\n' for item in original)
    inventory_sha = hashlib.sha256(inventory.encode()).hexdigest()
    manifest = args.output / 'self-source.manifest'
    manifest.write_text(''.join(item['path']+'\n' for item in original), encoding='utf-8')
    policy = dict(identity='fairyc.private-project-image-repro/v1', compilerFlags=['/Brepro'], linkerFlags=['/INCREMENTAL:NO', '/Brepro'], comparison='Exact full PE image bytes', excludedBytes=[])
    if args.reuse_receipt:
        receipt = json.loads(args.reuse_receipt.read_bytes())
        assert receipt['schema'] in ('fairyc.candidate-two-stage-bootstrap/v1', 'fairyc.candidate-two-stage-bootstrap/v2') and receipt['passed']
        assert receipt['sourceInventorySHA256'] == inventory_sha and receipt['artifactPolicy'] == policy
        assert receipt['sourceFiles'] == len(original)
        assert receipt['frontendTimeoutSeconds'] == args.frontend_timeout
        assert len(receipt['builds']) == 2 and all(item['exit'] == 0 and not item['stderr'] for item in receipt['builds'])
        images = [checked_artifact(receipt[name]) for name in ('stage1', 'stage2')]
        assert images[0].read_bytes() == images[1].read_bytes()
        if receipt['schema'].endswith('/v2'):
            assert receipt['seed'] == seed_identity
            assert [item['stage'] for item in receipt['frontends']] == [1, 2]
            assert len(receipt['packets']) == 2
            for observation in receipt['frontends']:
                assert observation['exit'] == 0 and not observation['stderr'] and observation['seconds'] < args.frontend_timeout
            seed_packet, next_packet = [checked_artifact(item) for item in receipt['packets']]
        else:
            assert receipt['stage1Replay']['exit'] == 0 and not receipt['stage1Replay']['stderr'] and receipt['stage1Replay']['seconds'] < args.frontend_timeout
            seed_packet = Path(receipt['builds'][0]['argv'][1])
            next_packet = Path(receipt['builds'][1]['argv'][1])
            runtime = Path(receipt['builds'][0]['argv'][2])
            assert digest(runtime) == receipt['runtimeSHA256']
        assert digest(seed_packet) == receipt['seedPacketSHA256'] and digest(next_packet) == receipt['packetSHA256']
        assert seed_packet.read_bytes() == next_packet.read_bytes()
        receipt['reusedReceiptSHA256'] = digest(args.reuse_receipt)
    else:
        assert args.builder and args.builder.is_file(), 'builder is required for fresh native builds'
        builder_identity = artifact(args.builder)
        packets, images, observations, frontends = [], [], [], []
        compiler = args.seed
        for stage in (1, 2):
            start = time.monotonic()
            emit_argv = [str(compiler.resolve()), '--c-project-file', str(manifest.resolve())]
            emit = subprocess.run(emit_argv, cwd=ROOT, capture_output=True, timeout=args.frontend_timeout)
            frontends.append(dict(stage=stage, argv=emit_argv, seconds=time.monotonic()-start, exit=emit.returncode, stdoutBytes=len(emit.stdout), stderr=emit.stderr.decode('utf-8', errors='replace')))
            assert emit.returncode == 0 and not emit.stderr and 0 < len(emit.stdout) <= 96 << 20
            packet = args.output / f'stage{stage}-input.bin'
            packet.write_bytes(emit.stdout)
            if packets:
                assert packet.read_bytes() == packets[0].read_bytes()
            image = args.output / f'stage{stage}.exe'
            argv = [str(args.builder.resolve()), '--packet', str(packet.resolve()), '--packet-sha256', digest(packet), '--output', str(image.resolve()), '--release']
            start = time.monotonic()
            built = subprocess.run(argv, cwd=ROOT, capture_output=True, timeout=600)
            assert built.returncode == 0 and not built.stderr, built.stderr
            observations.append(dict(stage=stage, argv=argv, seconds=time.monotonic()-start, exit=built.returncode, stdout=built.stdout.decode('utf-8'), stderr=built.stderr.decode('utf-8')))
            packets.append(packet)
            images.append(image)
            compiler = image
        assert images[0].read_bytes() == images[1].read_bytes()
        assert artifact(args.builder) == builder_identity, 'builder changed during verification'
        receipt = dict(schema='fairyc.candidate-two-stage-bootstrap/v2', passed=True, scope='Consecutive C-part generations after a Go-produced Fairy seed; not byte equality with that monolithic seed image', artifactPolicy=policy, sourceInventorySHA256=inventory_sha, sourceFiles=len(original), frontendTimeoutSeconds=args.frontend_timeout, seed=seed_identity, builder=builder_identity, frontends=frontends, builds=observations, packets=[artifact(p) for p in packets], seedPacketSHA256=digest(packets[0]), packetSHA256=digest(packets[1]), stage1=artifact(images[0]), stage2=artifact(images[1]))
    proof_dir = args.output / 'lexer'
    checked = subprocess.run([sys.executable, str(ROOT/'tools/verify-candidate-bootstrap-lexer.py'), '--reference', str(args.reference.resolve()), '--fixtures', str(ROOT/'spec/fixtures/lexer/candidate-bootstrap-v1.json'), '--stage1', str(images[0].resolve()), '--stage2', str(images[1].resolve()), '--seed', str(args.seed.resolve()), '--output', str(proof_dir.resolve())], cwd=ROOT, capture_output=True, timeout=60)
    assert checked.returncode == 0 and not checked.stderr, checked.stderr
    assert sources() == original, 'compiler sources changed during verification'
    assert artifact(args.seed) == seed_identity, 'seed changed during verification'
    receipt.update(seedImageSHA256=digest(args.seed), manifestSHA256=digest(manifest), lexerDifferential=json.loads((proof_dir/'receipt.json').read_bytes()), productTimeoutChanged=False, promotionGateReady=False, deferred=['Full frontend differential', 'Linux', 'Current-commit hosted CI'], pushed=False)
    (args.output/'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8', newline='\n')
    print('Consecutive compiler images and', receipt['lexerDifferential']['cases'], 'Go lexer cases verified')


if __name__ == '__main__':
    main()
