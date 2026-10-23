"""Check authored frontend and consecutive native-companion generations exactly."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]
HELPER = ROOT / 'tools/verify-candidate-native-fixed-point.py'
spec = importlib.util.spec_from_file_location('fixed_point', HELPER)
proof = importlib.util.module_from_spec(spec)
spec.loader.exec_module(proof)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    assert __debug__, 'run without -O'
    verifier_identity = proof.artifact(Path(__file__))
    helper_identity = proof.artifact(HELPER)
    build_path = args.build_receipt.resolve()
    build_identity = proof.artifact(build_path)
    build = json.loads(build_path.read_text(encoding='utf-8'))
    assert build['schema'] == 'fairyc.candidate-native-probe-build/v1' and build['passed']
    assert not any(build.get(k) for k in ('goFrontendInvoked', 'commandsOnly', 'compilerOnly'))
    release = next(row for row in build['images'] if row['profile'] == 'release')
    compiler = Path(release['path'])
    assert proof.sha(compiler) == release['sha256']
    for source in build['sources']:
        assert proof.sha(ROOT / source['path']) == source['sha256'], source['path']
    companion = proof.checked_companion(build, release)
    assert companion['implementation'] == 'authored Fairy'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    scratch = output / 'scratch'
    scratch.mkdir()
    environment = os.environ.copy()
    environment.update(TMP=str(scratch), TEMP=str(scratch))
    rows = []

    def run(argv, destination=None):
        start = time.monotonic()
        child = subprocess.Popen(list(map(str, argv)), cwd=ROOT, stdin=subprocess.DEVNULL,
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=environment,
                                 creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
        timed_out = False
        try:
            out, err = child.communicate(timeout=300)
        except subprocess.TimeoutExpired:
            timed_out = True
            subprocess.run(['taskkill', '/PID', str(child.pid), '/T', '/F'],
                           capture_output=True, timeout=15)
            if child.poll() is None:
                child.kill()
            out, err = child.communicate()
        rows.append(dict(argv=list(map(str, argv)), exit=child.returncode, timedOut=timed_out,
                         seconds=time.monotonic()-start, stdoutBytes=len(out),
                         stdoutSHA256=hashlib.sha256(out).hexdigest(),
                         stderr=err.decode(errors='replace')))
        (output/'progress.json').write_text(json.dumps(rows, indent=2)+'\n', encoding='utf-8')
        assert not timed_out and child.returncode == 0 and not err, rows[-1]
        if destination:
            destination.write_bytes(out)
        print('passed:', round(rows[-1]['seconds'], 3), Path(argv[0]).name, flush=True)

    packet = output / 'driver.bin'
    run([compiler, '--c-project-file', companion['manifest']['path']], packet)
    assert packet.read_bytes() == Path(companion['packet']['path']).read_bytes(), 'driver C packets differ'
    current = Path(companion['driver']['path'])
    generations = [proof.artifact(current)]
    for name in ('generation2', 'generation3'):
        home = output / name
        home.mkdir()
        image = home / 'fairyc-candidate-build.exe'
        run([current, '--packet-input', packet, '--output-root', home,
             '--output', image.name, '--release', '--compile-timeout', '1m'])
        assert image.read_bytes() == current.read_bytes(), 'consecutive companion images differ'
        assets = home / 'fairyc-runtime-assets.pack'
        shutil.copyfile(companion['assets']['path'], assets)
        assert proof.sha(assets) == companion['assets']['sha256']
        generations.append(proof.artifact(image))
        current = image
    assert not list(scratch.iterdir()), 'owned scratch directory leaked'
    scratch.rmdir()
    assert proof.artifact(build_path) == build_identity
    assert proof.sha(compiler) == release['sha256']
    assert proof.checked_companion(build, release) == companion
    assert proof.artifact(Path(__file__)) == verifier_identity
    assert proof.artifact(HELPER) == helper_identity
    for source in build['sources']:
        assert proof.sha(ROOT / source['path']) == source['sha256'], source['path']
    report = dict(schema='fairyc.candidate-companion-fixed-point/v1', passed=True,
                  buildReceipt=build_identity, compiler=proof.artifact(compiler), companion=companion,
                  originalPacket=companion['packet'], renewedPacket=proof.artifact(packet),
                  packetBytesEqual=True, generations=generations, optimizedImageBytesEqual=True,
                  excludedImageBytes=[], invocations=rows, verifier=verifier_identity,
                  helper=helper_identity, scratchDirectoryRemoved=True,
                  bootstrapBuilderInvoked=False, goFrontendInvoked=False, linuxExecuted=False,
                  productTimeoutChanged=False, targetMaturityChanged=False, pushed=False,
                  scope='Authored frontend reproduces the companion packet; authored companions compile two exact successive release images.',
                  limitations=['The driver consumes a C packet; this does not make it a Fairy frontend.',
                               'MSVC remains the native backend; Linux and clean-machine execution are separate.'])
    (output/'receipt.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    print('authored companion consecutive-generation equality verified', flush=True)


if __name__ == '__main__':
    main()
