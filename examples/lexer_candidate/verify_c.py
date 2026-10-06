#!/usr/bin/env python3
"""Compare Fairy-emitted C bytes and native outcomes with Go MIR or explicit fixed test expectations.

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
    parser=argparse.ArgumentParser();parser.add_argument('--images',nargs=2,required=True);parser.add_argument('--reference',required=True);parser.add_argument('--native-runner',required=True);parser.add_argument('--receipt',required=True);parser.add_argument('--nominal',action='store_true');parser.add_argument('--arrays',action='store_true');parser.add_argument('--literals',action='store_true');parser.add_argument('--views',action='store_true');parser.add_argument('--mutable',action='store_true');parser.add_argument('--aggregate-arrays',action='store_true')
    args=parser.parse_args(); args.arrays=args.arrays or args.aggregate_arrays; images=list(map(lambda x:Path(x).resolve(),args.images)); reference=Path(args.reference).resolve();runner=Path(args.native_runner).resolve();receipt=Path(args.receipt).resolve();directory=receipt.parent;directory.mkdir(parents=True,exist_ok=True)
    data=json.loads(reference.read_text(encoding='utf-8'));fixtures=data['fixtures'];assert data['schema']==('fairyc.aggregate-array-fixed-oracle/v1' if args.aggregate_arrays else 'fairyc.mutable-mir-oracle/v1' if args.mutable else 'fairyc.view-mir-oracle/v1' if args.views else 'fairyc.literal-mir-oracle/v1' if args.literals else 'fairyc.array-mir-oracle/v1' if args.arrays else 'fairyc.nominal-mir-oracle/v1' if args.nominal else 'fairyc.mir-file-oracle/v1');assert len(fixtures)<=120 and all(f['admitted'] for f in fixtures)
    rows=[];jobs=[];expected={};units=[]
    for fixture in fixtures:
        source=bytes.fromhex(fixture['inputHex']);path=directory/(fixture['name']+'.fy');path.write_bytes(source);before=sha(path)
        outputs=[]
        for image in images:
            row=run([image,'--c-file',path],rows,fixture['name']);assert (row['exit'],row['stderr'])==(0,''),row;assert sha(path)==before
            outputs.append(row['stdout']);row['input']=dict(path=str(path),sha256=before,unchanged=True)
        assert outputs[0]==outputs[1],fixture['name']
        emitted=directory/(fixture['name']+'.c');emitted.write_text(outputs[0],encoding='utf-8',newline='\n')
        unit=outputs[0];assert unit.startswith('/* fairyc.candidate-'+('view' if args.mutable or args.views else 'literal' if args.literals else 'array' if args.arrays else 'nominal' if args.nominal else 'scalar')+'-c/v1;')
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
    standalone=[('standalone-main','field-copy'),('standalone-trap','outer-bounds'),('standalone-return','rank2-return')] if args.aggregate_arrays else [('standalone-main','mutable-write'),('standalone-trap','mutable-reverse'),('standalone-return','mutable-bound-return')] if args.mutable else [('standalone-main','view-borrow'),('standalone-trap','view-reverse'),('standalone-return','view-return')] if args.views else [('standalone-main','literal-bytes'),('standalone-trap','literal-bounds'),('standalone-return','literal-return')] if args.literals else [('standalone-main','array-copy'),('standalone-trap','array-bounds-read'),('standalone-return','array-return')] if args.arrays else [('standalone-main','construct-read'),('standalone-trap','field-overflow'),('standalone-empty','empty-return')] if args.nominal else [('standalone-main','constant'),('standalone-trap','overflow-i32')]
    for name,fixture_name in standalone:
        plain=next(f for f in fixtures if f['name']==fixture_name);jobs.append(dict(name=name,source=str(directory/(fixture_name+'.c')),observe=False));expected[name]=plain['outcome']
    controls=[('no-main',b'module main;fn i32 f(){return 0;}', 'error: C entry requires fn i32 main()\n'),('void-main',b'module main;fn void main(){}','error: C entry requires fn i32 main()\n'),('parameter-main',b'module main;fn i32 main(i32 x){return x;}','error: C entry requires fn i32 main()\n'),('empty-program',b'module main;','error: C entry requires fn i32 main()\n'),('bad-utf8',b'module main;\xff',None),('bad-field',b'module main;fn i32 main(){return absent;}',None),('bad-number',b'module main;fn i32 main(){return 9223372036854775808;}',None),('missing',None,'error: input file read failed\n')]
    if args.nominal:
        controls += [('unknown-member',b'module main;struct A{i32 x;}fn i32 main(){return A{.x=1}.bad;}',None),('missing-member',b'module main;struct A{i32 x;}fn i32 main(){let a=A{};return 0;}',None),('wrong-nominal',b'module main;struct A{i32 x;}struct B{i32 x;}fn i32 f(A a){return a.x;}fn i32 main(){return f(B{.x=1});}',None),('cyclic-layout',b'module main;struct A{B b;}struct B{A a;}fn i32 main(){return 0;}',None)]
    if args.arrays:
        controls += [('empty-array',b'module main;fn i32 main(){let a=[];return 0;}',None),('large-array',b'module main;fn i32 main(){let i32[65] a=[1;65];return 0;}',None),('wrong-length',b'module main;fn i32 main(){let i32[2] a=[1];return 0;}',None),('wrong-element',b'module main;fn i32 main(){let i32[2] a=[1,true];return 0;}',None),('immutable-array',b'module main;fn i32 main(){let a=[1];a[0]=2;return 0;}',None),('negative-index',b'module main;fn i32 main(){let a=[1];return a[-1];}',None),('named-count',b'module main;fn i32 main(){let A=1;let a=[1;A];return 0;}',None)]
    if args.aggregate_arrays:
        controls += [
            ('wrong-row-length',b'module main;fn i32 main(){let i32[2][1] a=[[1]];return 0;}',None),
            ('wrong-outer-length',b'module main;fn i32 main(){let i32[1][2] a=[[1]];return 0;}',None),
            ('mixed-rows',b'module main;fn i32 main(){let a=[[1],[true]];return 0;}',None),
            ('rank3-inferred',b'module main;fn i32 main(){let a=[[[1]]];return 0;}',None),
            ('rank3-explicit',b'module main;fn i32 main(){let i32[1][1][1] a=[[1]];return 0;}',None),
            ('large-dimension',b'module main;fn i32 main(){let i32[65][1] a=[[1]];return 0;}',None),
            ('zero-dimension',b'module main;struct A{i32[0] a;}fn i32 main(){return 0;}',None),
            ('nominal-element',b'module main;struct A{i32 x;}fn i32 main(){let a=[A{.x=1}];return 0;}',None),
            ('view-field',b'module main;struct A{ref mut u8[] a;}fn i32 main(){return 0;}',None),
            ('immutable-field',b'module main;struct A{i32[1] a;}fn i32 main(){let a=A{.a=[1]};a.a[0]=2;return 0;}',None),
            ('parameter-field',b'module main;struct A{i32[1] a;}fn void bad(A a){a.a[0]=2;}fn i32 main(){return 0;}',None),
            ('row-parameter',b'module main;fn void bad(i32[1][1] a){a[0][0]=2;}fn i32 main(){return 0;}',None),
            ('temporary-field',b'module main;struct A{i32[1] a;}fn i32 main(){A{.a=[1]}.a[0]=2;return 0;}',None),
            ('compound-row',b'module main;fn i32 main(){var a=[[1]];a[0]+=[1];return 0;}',None),
            ('row-view',b'module main;fn i32 main(){var u8[1][1] a=[[1]];let ref u8[] b=ref a[0];return 0;}',None),
            ('field-view',b'module main;struct A{u8[1] a;}fn i32 main(){var a=A{.a=[1]};let ref mut u8[] b=ref mut a.a;return 0;}',None),
            ('cyclic-struct',b'module main;struct A{A a;i32[1] x;}fn i32 main(){return 0;}',None),
        ]
    if args.literals:
        controls += [('text-index',rb'module main;fn i32 main(){discard "a"[0];return 0;}',None),('byte-write',rb'module main;fn i32 main(){var b=b"a";b[0]=1;return 0;}',None),('group-byte-write',rb'module main;fn i32 main(){var b=b"a";(b[0])=1;return 0;}',None),('metadata-write',rb'module main;fn i32 main(){var b=b"a";b.len=1;return 0;}',None),('wrong-literal-type',rb'module main;fn i32 main(){let ref str s=b"a";return 0;}',None),('bad-escape',rb'module main;fn i32 main(){let s="\q";return 0;}',None),('bad-text-utf8',rb'module main;fn i32 main(){let s="\xff";return 0;}',None),('surrogate',rb'module main;fn i32 main(){let s="\u{d800}";return 0;}',None),('byte-unicode-escape',rb'module main;fn i32 main(){let s=b"\u{41}";return 0;}',None),('c-string-closed',rb'module main;fn i32 main(){let s=c"a";return 0;}',None),('literal-field',rb'module main;struct A{ref str s;}fn i32 main(){return 0;}',None),('oversize',rb'module main;fn i32 main(){let s="'+b'A'*32769+b'";return 0;}',None)]
    if args.views or args.mutable:
        controls += [
            ('index-held-owner-write',rb'module main;fn i32 main(){var u8[1] a=[1];let ref u8[] b=ref a;discard b[{a[0]=2;0}];return 0;}',None),
            ('defer-view-capture',rb'module main;fn void later(ref u8[] b){discard b[0];}fn i32 main(){var u8[1] a=[1];let ref u8[] b=ref a;defer later(b);a[0]=2;return 0;}',None),
            ('live-owner-write',rb'module main;fn i32 main(){var u8[1] a=[1];let ref u8[] b=ref a;a[0]=2;discard b[0];return 0;}',None),
            ('live-alias-write',rb'module main;fn i32 main(){var u8[1] a=[1];let ref u8[] b=ref a;let s=b[..];a=[2];discard s[0];return 0;}',None),
            ('loop-held-write',rb'module main;fn i32 main(){var u8[1] a=[1];let ref u8[] b=ref a;var usize i=0;while(i<2){discard b[0];a[0]=2;i+=1;}return 0;}',None),
            ('escaping-local',rb'module main;fn ref u8[] bad(){let u8[1] a=[1];return ref a;}fn i32 main(){return 0;}',None),
            ('escaping-scope',rb'module main;fn i32 main(){let ref u8[] b={let u8[1] a=[1];ref a};discard b[0];return 0;}',None),
            ('held-view-argument',rb'module main;fn i32 f(ref u8[] b,i32 n){discard n;return 0;}fn i32 main(){var u8[1] a=[1];return f(ref a,{a[0]=2;0});}',None),
            ('wrong-array-element',rb'module main;fn i32 main(){let i32[1] a=[1];let ref u8[] b=ref a;return 0;}',None),
            ('temporary-array-view',rb'module main;fn i32 main(){let ref u8[] b=ref [1];return 0;}',None),
            ('borrowed-descriptor-var',rb'module main;fn i32 main(){let u8[1] a=[1];var ref u8[] b=ref a;return 0;}',None),
            ('byte-view-write',rb'module main;fn i32 main(){let u8[1] a=[1];let ref u8[] b=ref a;b[0]=2;return 0;}',None),
            ('negative-slice-bound',rb'module main;fn i32 main(){discard b"A"[-1..];return 0;}',None),
            ('text-slice',rb'module main;fn i32 main(){discard "A"[..];return 0;}',None),
            ('mixed-view-origins',rb'module main;fn ref u8[] bad(ref u8[] a,ref u8[] b){return if(true){a}else{b};}fn i32 main(){return 0;}',None),
        ]
    if args.mutable:
        controls += [
            ('immutable-mut-owner',rb'module main;fn i32 main(){let u8[1] a=[1];let ref mut u8[] b=ref mut a;b[0]=2;return 0;}',None),
            ('mut-owner-read',rb'module main;fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;discard a[0];b[0]=2;return 0;}',None),
            ('mut-parent-access',rb'module main;fn i32 main(){var u8[2] a=[1,2];let ref mut u8[] b=ref mut a;let s=b[..1];b[1]=7;s[0]=8;return 0;}',None),
            ('mut-sibling-conflict',rb'module main;fn i32 main(){var u8[2] a=[1,2];let ref mut u8[] b=ref mut a;let x=b[..1];let y=b[1..];x[0]=7;y[0]=8;return 0;}',None),
            ('mut-rhs-owner',rb'module main;fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;b[0]={a[0]=7;8};return 0;}',None),
            ('mut-call-overlap',rb'module main;fn void fill(ref mut u8[] b,ref mut u8[] c){b[0]=8;c[0]=9;}fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;fill(b,b);return 0;}',None),
            ('mut-local-return',rb'module main;fn ref mut u8[] bad(){var u8[1] a=[1];return ref mut a;}fn i32 main(){return 0;}',None),
            ('mut-to-shared-closed',rb'module main;fn void read(ref u8[] b){discard b[0];}fn i32 main(){var u8[1] a=[1];let ref mut u8[] b=ref mut a;read(b);return 0;}',None),
        ]
    for name,source,message in controls:
        path=directory/(name+'.fy')
        if source is not None:path.write_bytes(source)
        for image in images:
            row=run([image,'--c-file',path],rows,name);assert row['exit']==2 and row['stdout']=='' and row['stderr'],row
            if message is not None:assert row['stderr']==message,row
            if name in ('text-index','text-slice'):
                access=b'"a"[0]' if name=='text-index' else b'"A"[..]';start=source.index(access);end=start+len(access)
                assert row['stderr']==f'{path}:1:{start+1}: error: text indexing and slicing are not supported (bytes {start}..{end})\n',row
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
