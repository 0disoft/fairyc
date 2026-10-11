"""Compare Fairy manifest value rules with Go in native and candidate-emitted C."""
import argparse
import base64
import hashlib
import json
from pathlib import Path
import re
import subprocess
import time

ROOT=Path(__file__).resolve().parents[1]
PROJECT=ROOT/'examples/lexer_candidate'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path,data):
    path.write_text(json.dumps(data,indent=2)+'\n',encoding='utf-8',newline='\n')


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler',type=Path,required=True)
    parser.add_argument('--reference',type=Path,required=True)
    parser.add_argument('--candidate',type=Path,required=True)
    parser.add_argument('--builder',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--reuse-native',action='store_true',help='reuse a native report only when compiler, lock, source and input hashes match')
    args=parser.parse_args()
    if not __debug__:
        parser.error('verification requires Python without -O')
    output=args.output.resolve();output.mkdir(parents=True,exist_ok=True)
    source_paths=[PROJECT/'src/project_manifest_values.fy',PROJECT/'tests/project_manifest_values.fy',ROOT/'spec/fixtures/project-manifest/candidate-values-v1.json',ROOT/'cmd/fairyc-candidate-manifest-reference/main.go',Path(__file__)]
    identities=[dict(path=p.relative_to(ROOT).as_posix(),sha256=digest(p)) for p in source_paths]
    images={name:dict(path=str(path.resolve()),sha256=digest(path)) for name,path in [('compiler',args.compiler),('reference',args.reference),('candidate',args.candidate),('builder',args.builder)]}
    invocations=[]

    def run(argv,cwd=ROOT,timeout=60):
        start=time.monotonic();result=subprocess.run(list(map(str,argv)),cwd=cwd,capture_output=True,timeout=timeout)
        row=dict(argv=list(map(str,argv)),cwd=str(cwd),seconds=time.monotonic()-start,exit=result.returncode,stdoutBytes=len(result.stdout),stdoutSHA256=hashlib.sha256(result.stdout).hexdigest(),stderr=result.stderr.decode('utf-8',errors='replace'))
        invocations.append(row);save(output/'invocations.json',invocations)
        assert result.returncode==0 and not result.stderr,row
        return result

    corpus=json.loads(source_paths[2].read_bytes());assert corpus['schema']=='fairyc.candidate-manifest-values-corpus/v1'
    cases=list(corpus['cases'])
    for mode in corpus['exhaustiveSingleByteModes']:
        for byte in range(256):
            cases.append(dict(name=f'byte-{mode}-{byte}',mode=mode,value=base64.b64encode(bytes([byte])).decode()))
    request=output/'reference-request.json';binary=output/'reference.bin';reference_report=output/'reference.json'
    save(request,dict(schema='fairyc.candidate-manifest-values-request/v1',cases=cases))
    run([args.reference.resolve(),request,binary,reference_report])
    observed=json.loads(reference_report.read_bytes());assert len(observed['cases'])==len(cases)
    context=dict(compilerSHA256=digest(args.compiler),binarySHA256=digest(binary),lockSHA256=digest(PROJECT/'fairy.lock'),sources=[item for item in identities if item['path'].endswith('.fy')])
    if args.reuse_native:
        # Prove the actual source graph still matches the recorded lock too;
        # an unchanged lock file alone would not detect edits to other imports.
        run([args.compiler.resolve(),'check','--locked'],cwd=PROJECT)
        previous=json.loads((output/'native-context.json').read_bytes())
        assert previous['inputs']==context and previous['reportSHA256']==digest(output/'native.json'),'native inputs changed'
        native_invocation=previous['invocation'];assert native_invocation['exit']==0 and not native_invocation['stderr']
        native_report=json.loads((output/'native.json').read_bytes())
    else:
        native=run([args.compiler.resolve(),'test','--native','--locked','--json','--capture-output','--exact','tests::project_manifest_values::keeps_exact_manifest_names_paths_and_versions','--exact','tests::project_manifest_values::matches_go_manifest_value_reference','--',binary],cwd=PROJECT,timeout=600)
        native_invocation=invocations[-1]
        native_report=json.loads(native.stdout);save(output/'native.json',native_report)
        save(output/'native-context.json',dict(inputs=context,reportSHA256=digest(output/'native.json'),invocation=native_invocation))
    assert native_report['passed']==2 and native_report['failed']==0
    assert native_report['engine']=='native' and native_report['lock']=='verified'
    assert {item['identity'] for item in native_report['tests']}=={'tests::project_manifest_values::keeps_exact_manifest_names_paths_and_versions','tests::project_manifest_values::matches_go_manifest_value_reference'}

    # Retain the authored Fairy test bodies. Only their test annotations and
    # module name change for a private executable entry; Python evaluates none
    # of the manifest predicates or semantic expected results.
    authored=source_paths[1].read_text(encoding='utf-8')
    authored=authored.replace('module tests::project_manifest_values;','module main;',1).replace('@test\n','')
    authored+='\n@effects(filesystem,console) fn i32 main(){keeps_exact_manifest_names_paths_and_versions();matches_go_manifest_value_reference();return 0;}\n'
    main_source=output/'main.fy';main_source.write_text(authored,encoding='utf-8',newline='\n')
    modules={}
    for base in (PROJECT/'src',ROOT/'internal/sdkstdlib/sources'):
        for path in base.rglob('*.fy'):
            name=re.search(r'^module\s+([\w:]+);',path.read_text(encoding='utf-8'))[1]
            assert name not in modules
            modules[name]=path
    modules['main']=main_source
    seen=set();pending=['main']
    while pending:
        name=pending.pop()
        if name in seen:continue
        seen.add(name);pending.extend(re.findall(r'^import\s+([\w:]+)',modules[name].read_text(encoding='utf-8'),re.M))
    paths=[main_source]+[modules[n] for n in sorted(seen-{'main'})]
    manifest=output/'sources.list';manifest.write_text(''.join(p.relative_to(ROOT).as_posix()+'\n' for p in paths),encoding='utf-8',newline='\n')
    packet=run([args.candidate.resolve(),'--c-project-file',manifest])
    packet_path=output/'candidate-packet.bin';packet_path.write_bytes(packet.stdout)
    assert packet.stdout.startswith(b'FYCPARTS1\n')
    executions=[]
    for profile,flags in [('development',[]),('release',['--release'])]:
        image=output/(profile+'.exe')
        run([args.builder.resolve(),'--packet',packet_path,'--packet-sha256',digest(packet_path),'--output',image,*flags],timeout=600)
        executed=run([image,binary]);assert not executed.stdout
        executions.append(dict(profile=profile,path=str(image),sha256=digest(image),exit=executed.returncode))
    assert [dict(path=p.relative_to(ROOT).as_posix(),sha256=digest(p)) for p in source_paths]==identities,'sources changed during verification'
    assert all(digest(Path(item['path']))==item['sha256'] for item in images.values()),'tool changed during verification'
    assert context['lockSHA256']==digest(PROJECT/'fairy.lock'),'lock changed during verification'
    receipt=dict(schema='fairyc.candidate-manifest-values/v1',passed=True,scope='Decoded manifest values only; TOML parsing, source discovery and project CLI are subsequent implementation units',sourceInputs=identities,tools=images,cases=len(cases),accepted=sum(item['accepted'] for item in observed['cases']),referenceSHA256=digest(reference_report),binaryFixtureSHA256=digest(binary),binaryFixtureBytes=binary.stat().st_size,nativeTests=2,native=native_report,nativeInputs=context,nativeInvocation=native_invocation,nativeReused=args.reuse_native,candidateInputFiles=len(paths),candidatePacketSHA256=digest(packet_path),candidateExecutions=executions,invocations=invocations,manifestParsingComplete=False,projectDiscoveryComplete=False,publicApiChanged=False,dbChanged=False,ciChanged=False,pushed=False)
    save(output/'receipt.json',receipt)
    print('Manifest value differential passed:',len(cases),'cases; native API and both C profiles')


if __name__=='__main__':
    main()
