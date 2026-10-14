"""Verify opt-in Fairy build/run cache reuse through actual native commands."""
import argparse
import hashlib
import json
import subprocess
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__, 'verification requires Python without -O'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
    build_path = args.build_receipt.resolve()
    build_sha = sha(build_path)
    build = json.loads(build_path.read_text(encoding='utf-8'))
    assert build['passed'] and not build['goFrontendInvoked']
    assert {row['profile'] for row in build['images']} == {'development', 'release'}
    reference = args.reference.resolve()
    reference_sha = sha(reference)
    source_inputs = build['sources'] + [dict(path=Path(__file__).relative_to(ROOT).as_posix(), sha256=sha(Path(__file__)))]
    for row in source_inputs:
        assert sha(ROOT / row['path']) == row['sha256'], row['path']
    rows = []

    def run(image, cwd, command, flags=(), expected=0, status=b'', label=''):
        argv = [str(image), *([] if command is None else [command]), *map(str, flags)]
        started = time.monotonic()
        result = subprocess.run(argv, cwd=cwd, capture_output=True, timeout=900)
        rows.append(dict(label=label, argv=argv, cwd=str(cwd), seconds=time.monotonic()-started,
                         exit=result.returncode, stdout=result.stdout.decode('utf-8'), stderr=result.stderr.decode('utf-8')))
        (output/'executions.json').write_text(json.dumps(rows, indent=2)+'\n', encoding='utf-8')
        assert result.returncode == expected and result.stderr.startswith(status), rows[-1]
        print('passed:', label, flush=True)
        return result.stdout, result.stderr[len(status):]

    manifest = "schema='fairyc.manifest/v1'\n[package]\nuid='example.native-cache.{name}'\nname='cache-app'\nversion='0.1.0'\nlanguage='fairyc-language/20'\nsource='src'\n"

    def fixture(profile, name, source):
        project = output/profile/('한글 '+name)
        (project/'src').mkdir(parents=True)
        (project/'fairy.toml').write_text(manifest.format(name=profile+'.'+name), encoding='utf-8', newline='\n')
        (project/'src/main.fy').write_text(source, encoding='utf-8', newline='\n')
        return project

    plain = 'module main;fn i32 main(){return 42;}\n'
    vector = 'module main;import helper as h;import std::vec as v;fn i32 main(){var v::Vec<i32> values=v::empty<i32>();assert(v::push<i32>(ref mut values,h::id<i32>(20)).is_ok);assert(v::push<i32>(ref mut values,h::id<i32>(22)).is_ok);return *v::at<i32>(ref values,0)+*v::at<i32>(ref values,1);}\n'
    forwarded = '''module main;import std::process as p;import std::bytes as b;
fn i32 main(){if(p::arg_count()!=3){return 0;}var u8[64] data=[0;64];
let n=switch(p::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{return 0;}};{let ref u8[] view=ref data;if(!b::equal(view[..n],b"--cache-root")){return 0;}}
let m=switch(p::arg(2,ref mut data)){case .ok(n):{n}case .err(_):{return 0;}};let ref u8[] view=ref data;return if(b::equal(view[..m],b"--cache-status")){42}else{0};}
'''
    for selected in build['images']:
        profile = selected['profile']
        image = Path(selected['path'])
        assert sha(image) == selected['sha256']
        assert sha(image.parent/'fairyc-candidate-build.exe') == selected['driverSHA256']
        cache = output/profile/'캐시 root'
        cache.mkdir(parents=True)
        flags = ['--cache-root', cache, '--cache-status']
        locked = ['--locked', *flags]
        project = fixture(profile, 'plain', plain)
        assert run(image, project, 'build', flags, status=b'cache: cold stored\n', label=profile+' cold cached build') == (b'built: build/cache-app.exe (x86_64-windows-msvc, development, lock updated)\n', b'')
        product = project/'build/cache-app.exe'
        assert run(product, project, None, expected=42, label=profile+' built program') == (b'', b'')
        leaves = list(cache.glob('fyproject0-*.fyc'))
        assert len(leaves) == 1
        retained = {p.name: (sha(p), p.stat().st_mtime_ns) for p in leaves}
        actual = run(image, project, 'run', locked, 42, b'cache: hit\n', profile+' checked hit run')
        assert actual == run(reference, project, 'run', ['--locked'], 42, label=profile+' Go run oracle') == (b'', b'')
        assert retained == {p.name: (sha(p), p.stat().st_mtime_ns) for p in cache.glob('fyproject0-*.fyc')}
        assert run(image, project, 'run', ['--locked', '--cache-root', cache], 42, label=profile+' silent hit run') == (b'', b'')
        assert run(image, project, 'run', ['--locked', '--cache-root', cache/'missing', '--cache-status'], 42, b'cache: cold uncached\n', profile+' unavailable-root cold run') == (b'', b'')
        prior = sha(product)
        (project/'src/main.fy').write_text(plain.replace('42', 'true'), encoding='utf-8', newline='\n')
        actual = run(image, project, 'build', flags, 3, b'cache: cold uncached\n', profile+' invalid-body cached build')
        assert actual == run(reference, project, 'build', ['--locked'], 3, label=profile+' Go invalid-body diagnostic')
        assert sha(product) == prior and retained == {p.name: (sha(p), p.stat().st_mtime_ns) for p in cache.glob('fyproject0-*.fyc')}
        assert run(image, project, 'build', ['--cache-status'], 2, label=profile+' status without root') == (b'', b'fairyc: --cache-status requires --cache-root\n')
        assert run(image, project, 'run', ['--cache-root', cache, '--cache-root', cache], 2, label=profile+' duplicate run root') == (b'', b'fairyc: command "run" received --cache-root more than once\n')
        project = fixture(profile, 'generic-vec', vector)
        (project/'src/helper.fy').write_text('module helper;pub fn T id<T>(T value){return value;}\n', encoding='utf-8', newline='\n')
        assert run(image, project, 'build', flags, status=b'cache: cold stored\n', label=profile+' generic Vec cached build') == (b'built: build/cache-app.exe (x86_64-windows-msvc, development, lock updated)\n', b'')
        assert run(project/'build/cache-app.exe', project, None, expected=42, label=profile+' generic Vec built program') == (b'', b'')
        assert run(image, project, 'run', locked, 42, b'cache: hit\n', profile+' generic Vec hit run') == (b'', b'')
        project = fixture(profile, 'argv', forwarded)
        assert run(image, project, 'run', [*flags, '--', '--cache-root', '--cache-status'], 42, b'cache: cold stored\n', profile+' cache flags forwarded after delimiter') == (b'', b'')
        assert not list(cache.glob('.stage*'))

    assert sha(reference) == reference_sha and sha(build_path) == build_sha
    for selected in build['images']:
        assert sha(Path(selected['path'])) == selected['sha256']
        assert sha(Path(selected['path']).parent/'fairyc-candidate-build.exe') == selected['driverSHA256']
    for row in source_inputs:
        assert sha(ROOT/row['path']) == row['sha256'], row['path']
    report = dict(schema='fairyc.candidate-native-cache-cli/v0', passed=True, candidateCVerified=True,
                  buildReceipt=dict(path=str(build_path.relative_to(ROOT)), sha256=build_sha), sourceInputs=source_inputs,
                  reference=dict(path=str(reference), sha256=reference_sha), images=build['images'], executions=rows,
                  explicitRoot=True, defaultCacheEnabled=False, publicGoCliChanged=False, ciChanged=False, pushed=False)
    (output/'receipt.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    print('native cached build/run passed:', len(rows), 'process observations', flush=True)


if __name__ == '__main__':
    main()
