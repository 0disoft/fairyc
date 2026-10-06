#!/usr/bin/env python3
"""Compare Fairy-emitted C bytes and native outcomes with the Go MIR oracle.

Python only prepares fixtures, observes processes and compares results. The C
source's semantic operations come exclusively from the Fairy executable.
"""
import argparse, base64, hashlib, json, re, subprocess
from pathlib import Path

ROOT=Path(__file__).resolve().parent
def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def save(path,obj): path.write_text(json.dumps(obj,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
def run(argv,rows,case):
    p=subprocess.run(list(map(str,argv)),cwd=ROOT,capture_output=True,timeout=30)
    row=dict(case=case,argv=list(map(str,argv)),cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'))
    rows.append(row);return row
def main():
    parser=argparse.ArgumentParser();parser.add_argument('--images',nargs=2,required=True);parser.add_argument('--reference',required=True);parser.add_argument('--native-runner',required=True);parser.add_argument('--receipt',required=True);parser.add_argument('--nominal',action='store_true')
    args=parser.parse_args(); images=list(map(lambda x:Path(x).resolve(),args.images)); reference=Path(args.reference).resolve();runner=Path(args.native_runner).resolve();receipt=Path(args.receipt).resolve();directory=receipt.parent;directory.mkdir(parents=True,exist_ok=True)
    data=json.loads(reference.read_text(encoding='utf-8'));fixtures=data['fixtures'];assert data['schema']==('fairyc.nominal-mir-oracle/v1' if args.nominal else 'fairyc.mir-file-oracle/v1');assert len(fixtures)<=120 and all(f['admitted'] for f in fixtures)
    if args.nominal:
        # The Go bootstrap currently rejects empty-struct field stores. Use an
        # explicit fixed expectation for this extra native control, not a Go claim.
        source=b'module main;struct E{}struct B{E e;i32 n;}fn i32 main(){var b=B{.e=E{},.n=1};b.e={b.n=9;E{}};return b.n;}'
        fixtures.append(dict(name='empty-field-rhs-effects',inputHex=source.hex(),admitted=True,outcome=dict(Kind='return',Value=9),outcomeOrigin='fixed-native-control'))
    rows=[];jobs=[];expected={};units=[]
    for fixture in fixtures:
        source=bytes.fromhex(fixture['inputHex']);path=directory/(fixture['name']+'.fy');path.write_bytes(source);before=sha(path)
        outputs=[]
        for image in images:
            row=run([image,'--c-file',path],rows,fixture['name']);assert (row['exit'],row['stderr'])==(0,''),row;assert sha(path)==before
            outputs.append(row['stdout']);row['input']=dict(path=str(path),sha256=before,unchanged=True)
        assert outputs[0]==outputs[1],fixture['name']
        emitted=directory/(fixture['name']+'.c');emitted.write_text(outputs[0],encoding='utf-8',newline='\n')
        unit=outputs[0];assert unit.startswith('/* fairyc.candidate-'+('nominal' if args.nominal else 'scalar')+'-c/v1;')
        adapted='#define FY_CANDIDATE_NO_MAIN\n#include <inttypes.h>\n#include <stdio.h>\n'+unit
        trace=fixture.get('traceCalls',False)
        if trace:
            # Observe the explicit fixture trace function at entry. Its original
            # emitted body still runs; no expression/result is replaced.
            # Go serializes functions by canonical name, while the candidate's
            # private target IDs retain source declaration order.
            ordered=sorted(fixture['program']['Functions'],key=lambda f:f['Span']['StartByte'])
            target=next(i+1 for i,f in enumerate(ordered) if f['Name']=='main::defer_trace')
            pattern=r'(int32_t fy_fn_'+str(target)+r'\(int32_t (fy_l_\d+)\)\{\n)'
            matches=list(re.finditer(pattern,adapted));assert len(matches)==1,(fixture['name'],target)
            # The private trap runtime exits without flushing C stdio. Match
            # the oracle's immediate trace byte before any subsequent trap.
            adapted=re.sub(pattern,lambda m:m[1]+'    (void)fputc((unsigned char)'+m[2]+',stdout);(void)fflush(stdout);\n',adapted,count=1)
        # Keep the observer reachable to the C compiler even for an entry that
        # always traps. The volatile call still executes the original body.
        adapted+='int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t result=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",result);return 0;}\n'
        instrumented=directory/(fixture['name']+'.observed.c');instrumented.write_text(adapted,encoding='utf-8',newline='\n')
        jobs.append(dict(name=fixture['name'],source=str(instrumented),observe=True));expected[fixture['name']]=fixture['outcome']
        units.append(dict(case=fixture['name'],source=str(path),sourceSHA256=before,emitted=str(emitted),emittedSHA256=sha(emitted),observed=str(instrumented),observedSHA256=sha(instrumented),traceHook=trace,outcomeOrigin=fixture['outcomeOrigin']))
    # The ordinary generated main is also compiled and run without the observer.
    standalone=[('standalone-main','construct-read'),('standalone-trap','field-overflow'),('standalone-empty','empty-return')] if args.nominal else [('standalone-main','constant'),('standalone-trap','overflow-i32')]
    for name,fixture_name in standalone:
        plain=next(f for f in fixtures if f['name']==fixture_name);jobs.append(dict(name=name,source=str(directory/(fixture_name+'.c')),observe=False));expected[name]=plain['outcome']
    controls=[('no-main',b'module main;fn i32 f(){return 0;}', 'error: C entry requires fn i32 main()\n'),('void-main',b'module main;fn void main(){}','error: C entry requires fn i32 main()\n'),('parameter-main',b'module main;fn i32 main(i32 x){return x;}','error: C entry requires fn i32 main()\n'),('empty-program',b'module main;','error: C entry requires fn i32 main()\n'),('bad-utf8',b'module main;\xff',None),('bad-field',b'module main;fn i32 main(){return absent;}',None),('bad-number',b'module main;fn i32 main(){return 9223372036854775808;}',None),('missing',None,'error: input file read failed\n')]
    if args.nominal:
        controls += [('unknown-member',b'module main;struct A{i32 x;}fn i32 main(){return A{.x=1}.bad;}',None),('missing-member',b'module main;struct A{i32 x;}fn i32 main(){let a=A{};return 0;}',None),('wrong-nominal',b'module main;struct A{i32 x;}struct B{i32 x;}fn i32 f(A a){return a.x;}fn i32 main(){return f(B{.x=1});}',None),('cyclic-layout',b'module main;struct A{B b;}struct B{A a;}fn i32 main(){return 0;}',None)]
    for name,source,message in controls:
        path=directory/(name+'.fy')
        if source is not None:path.write_bytes(source)
        for image in images:
            row=run([image,'--c-file',path],rows,name);assert row['exit']==2 and row['stdout']=='' and row['stderr'],row
            if message is not None:assert row['stderr']==message,row
            if source is not None:assert path.read_bytes()==source
    request=directory/'native-request.json';native_report=directory/'native.json';output=directory/'native';save(request,dict(schema='fairyc.candidate-c-native-request/v1',output=str(output),jobs=jobs))
    command=[str(runner),str(request),str(native_report)];p=subprocess.run(command,cwd=ROOT,capture_output=True,timeout=600)
    invocation=dict(argv=command,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'));save(directory/'native-invocation.json',invocation);assert p.returncode==0,invocation
    native=json.loads(native_report.read_text(encoding='utf-8'));categories={1:'integer-overflow',2:'division-by-zero',3:'signed-division-overflow',4:'invalid-shift',5:'assertion-failed',6:'bounds',7:'unreachable'}
    for row in native:
        outcome=expected[row['name']];assert sha(row['image'])==row['imageSHA256'] and sha(row['source'])==row['sourceSHA256']
        if outcome['Kind']=='trap':assert row['trap'] and categories[row['trap']]==outcome['TrapCategory'],(row,outcome)
        else:assert row['trap']==0 and row['value']==outcome['Value'],(row,outcome)
        wanted=base64.b64decode(outcome.get('Stdout') or '')
        assert base64.b64decode(row['stdoutBytes'] or '')==wanted,(row,outcome)
    groups={(x['compiler'],x['profile']) for x in native};assert len(native)==len(jobs)*len(groups) and all(sum(x['name']==j['name'] for x in native)==len(groups) for j in jobs)
    save(receipt,dict(schema='fairyc.candidate-c-file-verification/v1',images=[dict(path=str(p),sha256=sha(p)) for p in images],reference=dict(path=str(reference),sha256=sha(reference)),sources=[dict(path=str(p),sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))],units=units,cli=rows,controls=len(controls),cliExecutions=len(rows),nativeRunner=dict(path=str(runner),sha256=sha(runner)),nativeInvocation=invocation,nativeReport=dict(path=str(native_report),sha256=sha(native_report)),nativeExecutions=len(native),compilerProfiles=sorted(groups),passed=True))
    print(f'PASS: {len(rows)} C CLI executions; {len(fixtures)} identical C units; {len(native)} native executions')
if __name__=='__main__':main()
