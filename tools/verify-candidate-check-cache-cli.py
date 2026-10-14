"""Verify actual private check CLI cache selection across fresh Windows processes."""
import argparse
import hashlib
import json
import subprocess
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    builds = parser.add_mutually_exclusive_group(required=True)
    builds.add_argument('--build-receipt', type=Path,
                        help='candidate C build receipt with both profiles')
    builds.add_argument('--go-reference-build-receipt', type=Path,
                        help='Go-frontend private command receipt; does not verify candidate C')
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__, 'verification requires Python without -O'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    go_reference = args.go_reference_build_receipt is not None
    build_path = (args.go_reference_build_receipt or args.build_receipt).resolve()
    build = json.loads(build_path.read_text(encoding='utf-8'))
    assert build['passed'] and build['goFrontendInvoked'] == go_reference
    if go_reference:
        build = {**build, 'sources': build['inputs'],
                 'images': [{'profile': 'go-reference', **build['image']}]}
    sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
    for row in build['sources']:
        assert sha(ROOT / row['path']) == row['sha256'], row['path']
    lock_sha = sha(ROOT / 'examples/lexer_candidate/fairy.lock')
    if not go_reference:
        assert lock_sha == build['lockSHA256']
        assert {p['profile'] for p in build['images']} == {'development', 'release'}
    reference = args.reference.resolve()
    reference_sha = sha(reference)
    rows = []

    def run(image, cwd, flags, expected=0, status=b'', label=''):
        start = time.monotonic()
        result = subprocess.run([str(image), 'check', *map(str, flags)], cwd=cwd, capture_output=True, timeout=900)
        rows.append(dict(label=label, image=str(image), argv=['check', *map(str,flags)], cwd=str(cwd), seconds=time.monotonic()-start, exit=result.returncode, stdout=result.stdout.decode('utf-8'), stderr=result.stderr.decode('utf-8')))
        (output/'executions.json').write_text(json.dumps(rows,indent=2)+'\n',encoding='utf-8')
        assert result.returncode == expected, rows[-1]
        assert result.stderr.startswith(status), rows[-1]
        print('passed:',label,flush=True)
        return result.stdout, result.stderr[len(status):]

    manifest = "schema='fairyc.manifest/v1'\n[package]\nuid='example.check-cache.{name}'\nname='cache-app'\nversion='0.1.0'\nlanguage='fairyc-language/20'\nsource='src'\n"
    program = 'module main;fn i32 main(){return 42;}\n'
    for selected in build['images']:
        profile = selected['profile']
        image = Path(selected['path'])
        assert sha(image) == selected['sha256']
        base = output/profile/'프로젝트'
        (base/'src').mkdir(parents=True)
        (base/'fairy.toml').write_text(manifest.format(name=profile),encoding='utf-8',newline='\n')
        source = base/'src/main.fy'
        source.write_text(program,encoding='utf-8',newline='\n')
        cache = output/profile/'캐시 한글'
        cache.mkdir()
        flags = ['--cache-root',cache,'--cache-status']
        out, err = run(image,base,flags,status=b'cache: cold stored\n',label=profile+' cold population')
        assert not err and b'lock updated' in out
        leaves = list(cache.glob('fyproject0-*.fyc'))
        assert len(leaves)==1 and leaves[0].read_bytes().startswith(b'FYCACHE0')
        before = {p.name:(sha(p),p.stat().st_mtime_ns) for p in leaves}
        locked = ['--locked',*flags]
        actual = run(image,base,locked,status=b'cache: hit\n',label=profile+' fresh-process verified hit')
        oracle = run(reference,base,['--locked'],label=profile+' Go locked oracle')
        assert actual == oracle
        assert before == {p.name:(sha(p),p.stat().st_mtime_ns) for p in cache.glob('fyproject0-*.fyc')}
        assert run(image,base,['--locked','--cache-root',cache],label=profile+' silent hit') == oracle
        leaves[0].write_bytes(b'broken')
        assert run(image,base,locked,status=b'cache: cold stored\n',label=profile+' corrupt generation fallback') == oracle
        assert run(image,base,locked,status=b'cache: hit\n',label=profile+' repaired generation hit') == oracle
        assert run(image,base,['--locked','--cache-root',cache/'missing','--cache-status'],status=b'cache: cold uncached\n',label=profile+' unavailable root fallback') == oracle

        source.write_text(program.replace('42','43'),encoding='utf-8',newline='\n')
        out, err = run(image,base,flags,status=b'cache: cold stored\n',label=profile+' changed body cold population')
        assert not err and b'lock updated' in out
        actual = run(image,base,locked,status=b'cache: hit\n',label=profile+' changed body verified hit')
        assert actual == run(reference,base,['--locked'],label=profile+' changed body Go oracle')
        retained = {p.name:sha(p) for p in cache.glob('fyproject0-*.fyc')}
        source.write_text(program.replace('42','true'),encoding='utf-8',newline='\n')
        actual = run(image,base,flags,3,b'cache: cold uncached\n',profile+' invalid body fallback')
        assert actual == run(reference,base,['--locked'],3,label=profile+' exact invalid-body Go diagnostic')
        assert retained == {p.name:sha(p) for p in cache.glob('fyproject0-*.fyc')}

        # Private configuration errors are admitted before filesystem work.
        result = run(image,base,['--cache-status'],2,label=profile+' status requires root')
        assert result == (b'',b'fairyc: --cache-status requires --cache-root\n')
        result = run(image,base,['--cache-root',cache,'--cache-root',cache],2,label=profile+' duplicate root')
        assert result == (b'',b'fairyc: command "check" received --cache-root more than once\n')
        result = run(image,base,['--bad'],2,label=profile+' legacy invalid option')
        assert result == (b'',b'fairyc: command "check" accepts only optional --locked\n')
        assert not list(cache.glob('.stage*'))

        for name, body in (
            ('generic', 'module main;import helper as h;fn i32 main(){return h::id<i32>(42);}\n'),
            ('vec', 'module main;import helper as h;import std::vec as v;fn i32 main(){var v::Vec<i32> items=v::empty<i32>();discard v::push<i32>(ref mut items,h::id<i32>(1));return h::id<i32>(42);}\n'),
        ):
            project = output/profile/name
            (project/'src').mkdir(parents=True)
            (project/'helper/src').mkdir(parents=True)
            (project/'fairy.toml').write_text(manifest.format(name=profile+'.'+name)+"[workspace]\nmembers=['helper']\n[dependencies.h]\npath='helper'\nversion='0.1.0'\nfeatures=[]\n",encoding='utf-8',newline='\n')
            (project/'src/main.fy').write_text(body,encoding='utf-8',newline='\n')
            (project/'helper/fairy.toml').write_text(manifest.format(name='helper'),encoding='utf-8',newline='\n')
            (project/'helper/src/helper.fy').write_text('module helper;pub fn T id<T>(T value){return value;}\n',encoding='utf-8',newline='\n')
            cold, error = run(image,project,flags,status=b'cache: cold stored\n',label=profile+' '+name+' cold population')
            assert not error and b'lock updated' in cold
            hit = run(image,project,locked,status=b'cache: hit\n',label=profile+' '+name+' cross-process hit')
            assert hit == run(reference,project,['--locked'],label=profile+' '+name+' Go oracle')
        assert not list(cache.glob('.stage*'))

    assert sha(reference)==reference_sha
    for row in build['sources']:
        assert sha(ROOT/row['path'])==row['sha256']
    assert sha(ROOT/'examples/lexer_candidate/fairy.lock')==lock_sha
    report=dict(schema='fairyc.candidate-check-cache-cli/v0',passed=True,goFrontendInvoked=go_reference,candidateCVerified=not go_reference,buildReceipt=dict(path=str(build_path.relative_to(ROOT)),sha256=sha(build_path)),sourceInputs=build['sources'],rootLockSHA256=lock_sha,reference=dict(path=str(reference),sha256=reference_sha),images=build['images'],executions=rows,explicitRoot=True,defaultCacheEnabled=False,publicGoCliChanged=False,ciChanged=False,pushed=False)
    (output/'receipt.json').write_text(json.dumps(report,indent=2)+'\n',encoding='utf-8')
    print('private check cache CLI passed:',len(rows),'fresh process observations',flush=True)

if __name__=='__main__':
    main()
