"""Verify the Fairy workspace C command and native programs on Windows."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n', encoding='utf-8')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True, help='Go oracle only')
    parser.add_argument('--candidate', type=Path, required=True, help='verified Fairy seed')
    parser.add_argument('--builder', type=Path, required=True)
    parser.add_argument('--fixtures', type=Path, required=True, help='candidate-entry-reference output')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--reuse-build', type=Path, help='reuse a successful compiler with identical Fairy sources/tool inputs')
    args = parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output = args.output.resolve()
    if not output.is_relative_to(ROOT / '.cache'):
        parser.error('output must stay inside repository .cache')
    output.mkdir(parents=True, exist_ok=True)
    fixture = args.fixtures.resolve()
    oracle = json.loads((fixture / 'oracle.json').read_text())
    assert oracle['schema'] == 'fairyc.candidate-entry-reference/v1'
    assert {r['name'] for r in oracle['cases']} == {'Root', 'Missing', 'WrongModule', 'WrongResult', 'Parameter', 'Generic'}
    fixture_inputs = {str(p): digest(p) for p in fixture.rglob('*') if p.is_file()}
    invocations = []

    def run(argv, cwd=ROOT, timeout=60):
        start = time.monotonic()
        r = subprocess.run(list(map(str, argv)), cwd=cwd, capture_output=True, timeout=timeout)
        invocations.append(dict(argv=list(map(str, argv)), cwd=str(cwd), seconds=time.monotonic()-start, exit=r.returncode, stdoutBytes=len(r.stdout), stdoutSHA256=hashlib.sha256(r.stdout).hexdigest(), stderr=r.stderr.decode('utf-8', errors='replace')))
        save(output / 'invocations.json', invocations)
        return r

    # Build the real dispatcher, with a tiny main selecting its file-command path.
    main_source = output / 'main.fy'
    main_source.write_text('module main;\nimport file_cli as cli;\n@effects(filesystem,console) fn i32 main(){return cli::run();}\n', encoding='utf-8')
    modules = {}
    for base in (ROOT / 'examples/lexer_candidate/src', ROOT / 'internal/sdkstdlib/sources'):
        for path in base.rglob('*.fy'):
            name = re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))[1]
            assert name not in modules
            modules[name] = path
    modules['main'] = main_source
    seen, pending = set(), ['main']
    while pending:
        name = pending.pop()
        if name not in seen:
            seen.add(name)
            pending.extend(re.findall(r'^import\s+([\w:]+)', modules[name].read_text(encoding='utf-8'), re.M))
    paths = [main_source] + [modules[name] for name in sorted(seen - {'main'})]
    inputs = [dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in paths]
    inputs.append(dict(path=Path(__file__).relative_to(ROOT).as_posix(), sha256=digest(Path(__file__))))
    tools = {name: dict(path=str(path.resolve()), sha256=digest(path)) for name, path in [('compiler',args.compiler),('candidate',args.candidate),('builder',args.builder)]}
    manifest = output / 'sources.list'
    manifest.write_text(''.join(p.relative_to(ROOT).as_posix()+'\n' for p in paths), encoding='utf-8')
    image = output / 'compiler.exe'
    reused = None
    if args.reuse_build:
        prior = json.loads(args.reuse_build.read_text())
        assert prior['schema'] == 'fairyc.candidate-workspace-emission/v1' and prior['passed']
        assert prior['sourceFiles'] == len(paths) and prior['compilerSHA256'] == digest(image)
        old_sources = {p['path']:p['sha256'] for p in prior['sourceInputs'] if p['path'].endswith('.fy')}
        assert old_sources == {p['path']:p['sha256'] for p in inputs if p['path'].endswith('.fy')}, 'compiler sources changed'
        assert all(prior['tools'][name] == tools[name] for name in ['candidate','builder']), 'compiler build tools changed'
        reused = dict(receiptSHA256=digest(args.reuse_build), compilerSHA256=digest(image))
    else:
        built = run([args.builder.resolve(),'--candidate',args.candidate.resolve(),'--manifest',manifest,'--frontend-timeout','2m','--output',image,'--release'], timeout=900)
        assert built.returncode == 0 and not built.stderr, invocations[-1]
    print('current Fairy workspace dispatcher built', flush=True)

    cases = []
    for row in oracle['cases']:
        name = row['name']
        r = run([image,'--c-workspace',fixture/name,'--locked'])
        assert r.returncode == row['exit'], invocations[-1]
        if name == 'Root':
            assert r.stdout.startswith(b'FYCPARTS1\n') and not r.stderr
            packet = output / 'workspace.bin'
            packet.write_bytes(r.stdout)
        else:
            assert not r.stdout and b'root entry main::main' in r.stderr, invocations[-1]
        cases.append(dict(name=name, exit=r.returncode, stdoutBytes=len(r.stdout), stderr=r.stderr.decode('utf-8')))
    programs = []
    for profile, flags in [('development',[]),('release',['--release'])]:
        program = output / (profile+'.exe')
        r = run([args.builder.resolve(),'--packet',packet,'--packet-sha256',digest(packet),'--output',program,*flags], timeout=600)
        assert r.returncode == 0 and not r.stderr, invocations[-1]
        r = run([program])
        assert r.returncode == 42 and not r.stdout and not r.stderr, invocations[-1]
        programs.append(dict(profile=profile, exit=r.returncode, sha256=digest(program)))
    for name, mutation in [('MissingLock','missing'),('StaleLock','stale'),('SyntaxError','syntax'),('TypeError','type')]:
        directory = output/name
        shutil.copytree(fixture/'Root', directory, dirs_exist_ok=True)
        if mutation == 'missing':
            (directory/'fairy.lock').unlink()
        elif mutation == 'stale':
            (directory/'src/main.fy').write_text('module main;fn i32 main(){return 9;}',encoding='utf-8')
        elif mutation == 'syntax':
            (directory/'src/main.fy').write_text('module main;fn i32 main(){return ;',encoding='utf-8')
        else:
            (directory/'src/main.fy').write_text('module main;fn i32 main(){return true;}',encoding='utf-8')
            # Admit the current source lock before measuring the semantic error.
            refreshed = run([args.compiler.resolve(),'check'],cwd=directory)
            assert refreshed.returncode == 3 and b'E1009' in refreshed.stderr, invocations[-1]
        before = {str(p):digest(p) for p in directory.rglob('*') if p.is_file()}
        reference = run([args.compiler.resolve(),'experimental','emit-workspace-c','--target','x86_64-windows-msvc','--locked'],cwd=directory)
        r = run([image,'--c-workspace',directory,'--locked'])
        assert r.returncode == reference.returncode != 0 and not r.stdout and r.stderr, invocations[-1]
        assert r.returncode == (7 if mutation in ['missing','stale'] else 3), invocations[-1]
        if mutation == 'type':
            assert b'E1009' in r.stderr, 'type fixture rejected before semantic checking'
        assert before == {str(p):digest(p) for p in directory.rglob('*') if p.is_file()}, 'locked failure mutated workspace'
        cases.append(dict(name=name, exit=r.returncode, oracleExit=reference.returncode, stdoutBytes=0, stderr=r.stderr.decode('utf-8'), workspaceUnchanged=True))
    for arguments in [[],['--c-workspace'],['--c-workspace',str(fixture/'Root'),'--bad'],['--c-workspace','']]:
        r = run([image,*arguments])
        assert r.returncode == 2 and not r.stdout and r.stderr, invocations[-1]
    assert fixture_inputs == {str(p):digest(p) for p in fixture.rglob('*') if p.is_file()}, 'fixture workspace changed'
    assert all(digest(ROOT/p['path']) == p['sha256'] for p in inputs), 'sources changed'
    assert all(digest(Path(p['path'])) == p['sha256'] for p in tools.values()), 'tools changed'
    receipt = dict(schema='fairyc.candidate-workspace-emission/v1',passed=True,sourceInputs=inputs,tools=tools,sourceFiles=len(paths),compilerSHA256=digest(image),packetSHA256=digest(packet),cases=cases,programs=programs,invalidArgumentCases=4,fixtureUnchanged=True,GoFrontendFallback=False,invocations=invocations,publicApiChanged=False,sdkAbiChanged=False,dbChanged=False,ciChanged=False,pushed=False)
    if reused:
        receipt['reusedBuild'] = reused
    save(output/'receipt.json',receipt)
    print('workspace entry, lock/source rejection and both native profiles passed',flush=True)


if __name__ == '__main__':
    main()
