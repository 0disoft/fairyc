"""Build SDK-using Fairy C packets with the authored MSVC driver, without a Go child builder."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import struct
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parent.parent


def identity(path):
    return dict(path=str(path.resolve()), bytes=path.stat().st_size,
                sha256=hashlib.sha256(path.read_bytes()).hexdigest())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('compiler', 'candidate', 'builder', 'output'):
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    project = output / 'sdk-program'
    (project / 'src').mkdir(parents=True, exist_ok=True)
    (project / 'fairy.toml').write_text('schema="fairyc.manifest/v1"\n[package]\nuid="candidate.msvc.build-proof"\nname="msvc-proof"\nversion="0.1.0"\nlanguage="fairyc-language/20"\nsource="src"\n', encoding='utf-8')
    (project / 'src/main.fy').write_text('module main;\nimport std::io as io;\nimport std::bytes as bytes;\n@effects(console) fn i32 main(){let data=switch(bytes::copy(b"built by Fairy\\n")){case .ok(p):{p}case .err(_):{return 1;}};let written=io::write_bytes(bytes::view(ref data));return if(written.is_ok){42}else{1};}\n', encoding='utf-8')
    invocations = []

    def run(argv, cwd=ROOT, env=None, timeout=300):
        result = subprocess.run(list(map(str, argv)), cwd=cwd, env=env, capture_output=True, timeout=timeout)
        invocations.append(dict(argv=list(map(str, argv)), exit=result.returncode,
                                stdoutBytes=len(result.stdout), stderr=result.stderr.decode('utf-8', errors='replace')))
        (output / 'invocations.json').write_text(json.dumps(invocations, indent=2)+'\n', encoding='utf-8')
        assert result.returncode == 0, result.stdout + result.stderr
        return result

    run([args.compiler.resolve(), 'check'], cwd=project)
    generated = run([args.candidate.resolve(), '--c-workspace', project, '--locked'])
    assert generated.stdout.startswith(b'FYCPARTS1\n') and not generated.stderr
    fixture = output / 'program.bin'
    fixture.write_bytes(generated.stdout)
    images = Path(str(fixture) + '.images')
    images.mkdir(exist_ok=True)
    env = os.environ.copy()
    env['FAIRYC_MSVC_BUILD_FIXTURE'] = str(fixture)
    env['FAIRYC_MSVC_BUILD_RUNTIME_ASSETS'] = '1'
    run(['go', 'test', './internal/candidatebuild', '-run', '^TestPacketMSVCRuntimeFixture$', '-count=1'], env=env)
    asset_pack = Path(str(fixture)+'.runtime.pack')
    run([sys.executable, ROOT / 'tools/package-candidate-runtime-assets.py', '--output', asset_pack])
    assert not Path(str(fixture)+'.runtime.c').exists(), 'use a fresh output directory without a Go runtime fixture'
    inputs = [fixture, asset_pack, Path(str(fixture)+'.compile-error'), Path(str(fixture)+'.link-error')]
    before = [identity(path) for path in inputs]
    temporary = set(Path(tempfile.gettempdir()).glob('fairyc-native-*'))
    argv = [sys.executable, '-X', 'utf8', ROOT / 'tools/verify-candidate-private-tests.py',
            '--compiler', args.compiler.resolve(), '--candidate', args.candidate.resolve(),
            '--builder', args.builder.resolve(), '--module', 'native_msvc_build',
            '--candidate-only', '--fixture', fixture, '--output', output / 'execution']
    for test in ('compiles_links_and_executes_both_profiles_without_go_companion',
                 'separates_compile_link_and_option_failures', 'rejects_invalid_options_and_artifact_headers'):
        argv.extend(['--test', test])
    run(argv)
    assert before == [identity(path) for path in inputs], 'build input changed during execution'
    assert set(Path(tempfile.gettempdir()).glob('fairyc-native-*')) == temporary, 'owned build scratch directory leaked'
    expected = [images / (profile+'-'+mode+'.exe') for profile in ('development', 'release') for mode in ('dev', 'opt')]
    for path in expected:
        data = path.read_bytes()
        pe = struct.unpack_from('<I', data, 60)[0]
        assert struct.unpack_from('<Q', data, pe+24+72)[0] == 65536, 'private stack reservation changed'
    for mode in ('dev', 'opt'):
        assert (images / ('development-'+mode+'.exe')).read_bytes() == (images / ('release-'+mode+'.exe')).read_bytes(), 'authored driver profiles produced different compiler outputs'
    env = os.environ.copy()
    env['FAIRYC_MSVC_BUILT_IMAGES'] = str(images)
    run(['go', 'test', './internal/cdriver', '-run', '^TestCandidateMSVCBuiltImages$', '-count=1'], env=env)
    execution = output / 'execution/receipt.json'
    assert json.loads(execution.read_text(encoding='utf-8'))['passed']
    receipt = dict(schema='fairyc.candidate-msvc-build/v1', passed=True,
                   inputs=before, execution=identity(execution), artifacts=[identity(p) for p in expected],
                   authoredTests=3, buildProfiles=['development', 'release'],
                   candidateProfiles=['development', 'release'], imagesExecuted=4,
                   compileAndLinkFailuresRejected=True, scratchCleaned=True,
                   goPEInspectorPassed=True, goChildBuilderInvoked=False,
                   runtimeAssets='Python packs pinned raw sources; Fairy authenticates, selects and assembles runtime C',
                   goRuntimeSelectionInvoked=False,
                   artifactValidation='Bounded authored COFF/PE checks; full Go COMDAT/unwind proof not ported',
                   cliIntegrated=False, pushed=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8')
    print('authored MSVC build passed: four SDK program images, compile/link failures, exact cross-driver-profile images')


if __name__ == '__main__':
    main()
