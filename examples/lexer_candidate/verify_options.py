#!/usr/bin/env python3
"""Option candidate fixtures; Python only supplies inputs and observations."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parent
POSITIVES = [
    ('some', 'fn i32 main(){let Option<i32> v=.some(7);return switch(v){case .some(n):{n}case .none:{0}};}', 7),
    ('none', 'fn i32 main(){let Option<i32> v=.none;return switch(v){case .none:{5}case .some(n):{n}};}', 5),
    ('copy', 'fn Option<i32> id(Option<i32> v){return v;}fn i32 main(){var Option<i32> v=.some(3);let saved=id(v);v=.none;return switch(saved){case .some(n):{n}case .none:{0}};}', 3),
    ('discard', 'fn i32 main(){let Option<bool> v=.some(true);return switch(v){case .some(_):{4}case .none:{0}};}', 4),
    ('shadow', 'fn i32 main(){let i32 n=4;let Option<i32> v=.some(3);return switch(v){case .some(n):{n}case .none:{0}}+n;}', 7),
    ('enum-coexist', 'enum E{Data{i32 n;};}fn i32 main(){let E e=.Data{.n=2};let Option<i32> v=.some(3);return switch(e){case .Data{.n=n}:{n}}+switch(v){case .some(n):{n}case .none:{0}};}', 5),
    ('multiple-instances', 'fn i32 main(){let Option<i32> a=.some(3);let Option<bool> b=.none;return switch(a){case .some(n):{n}case .none:{0}}+switch(b){case .some(v):{if(v){1}else{2}}case .none:{4}};}', 7),
    ('if', 'fn Option<i32> pick(bool yes){return if(yes){.some(9)}else{.none};}fn i32 main(){return switch(pick(true)){case .some(n):{n}case .none:{0}}+switch(pick(false)){case .some(n):{n}case .none:{2}};}', 11),
    ('loop', 'fn i32 main(){var Option<i32> v=.some(0);var i32 i=0;while(i<3){v=switch(v){case .some(n):{.some(n+1)}case .none:{.none}};i+=1;}return switch(v){case .some(n):{n}case .none:{99}};}', 3),
    ('loop-control', 'fn i32 main(){var Option<i32> v=.none;var i32 i=0;while(i<5){i+=1;if(i==2){continue;}if(i==4){break;}v=.some(i);}return switch(v){case .some(n):{n}case .none:{99}};}', 3),
    ('subject-once', 'fn i32 main(){var i32 count=0;var Option<i32> v=.some(8);return switch({count+=1;v}){case .some(n):{v=.none;count+=2;n+count}case .none:{0}};}', 11),
    ('early-construction', 'fn i32 main(){let Option<i32> v=.some({return 6;7});return switch(v){case .some(n):{n}case .none:{0}};}', 6),
    ('early-call', 'fn i32 pick(Option<i32> v){return switch(v){case .some(n):{n}case .none:{0}};}fn i32 main(){return pick(.some({return 8;1}));}', 8),
    ('return-arm', 'fn i32 pick(Option<i32> v){switch(v){case .some(n):{return n;}case .none:{return 2;}};return 99;}fn i32 main(){return pick(.some(8));}', 8),
    ('assert-arm', 'fn i32 main(){let Option<i32> v=.some(7);return switch(v){case .none:{assert(false);}case .some(n):{n}};}', 7),
    ('assert-arm-infer', 'fn i32 main(){let Option<i32> v=.some(7);let n=switch(v){case .none:{assert(false);}case .some(n):{n}};return n;}', 7),
]
TRAPS = [
    ('payload-overflow', 'fn i32 main(){let Option<i32> v=.some(2147483647+1);return switch(v){case .some(n):{n}case .none:{0}};}', 1),
    ('payload-division', 'fn i32 main(){let Option<i32> v=.some(1/0);return switch(v){case .some(n):{n}case .none:{0}};}', 2),
    ('payload-assert', 'fn i32 main(){let Option<i32> v=.some({assert(false);1});return switch(v){case .some(n):{n}case .none:{0}};}', 5),
    ('assert-arm-trap', 'fn i32 main(){let Option<i32> v=.none;return switch(v){case .some(n):{n}case .none:{assert(false);}};}', 5),
]
for type_name,number in [('i32','-2147483648'),('i64','-9223372036854775808'),('u8','255'),('u32','4294967295'),('usize','18446744073709551615'),('bool','true')]:
    POSITIVES.append((type_name, f'fn Option<{type_name}> make(){{return .some({number});}}fn i32 main(){{let {type_name} n=switch(make()){{case .some(n):{{n}}case .none:{{{number}}}}};assert(n=={number});return 0;}}', 0))
NEGATIVES = [
    ('type-identity', 'fn i32 main(){let Option<i32> a=.none;let Option<i64> b=a;return 0;}', 'type mismatch'),
    ('wrong-payload', 'fn i32 main(){let Option<i32> a=.some(true);return 0;}', 'type mismatch'),
    ('overflow', 'fn i32 main(){let Option<u8> a=.some(256);return 0;}', 'integer literal out of range'),
    ('without-context', 'fn i32 main(){let a=.some(1);return 0;}', 'Option context required'),
    ('some-without-value', 'fn i32 main(){let Option<i32> a=.some;return 0;}', 'invalid Option construction'),
    ('some-empty', 'fn i32 main(){let Option<i32> a=.some();return 0;}', 'invalid Option construction'),
    ('some-two-values', 'fn i32 main(){let Option<i32> a=.some(1,2);return 0;}', 'invalid Option construction'),
    ('none-call', 'fn i32 main(){let Option<i32> a=.none(1);return 0;}', 'invalid Option construction'),
    ('unknown-variant', 'fn i32 main(){let Option<i32> a=.other(1);return 0;}', 'invalid Option construction'),
    ('missing-none', 'fn i32 main(){let Option<i32> a=.some(1);return switch(a){case .some(n):{n}};}', 'non-exhaustive Option switch'),
    ('missing-some', 'fn i32 main(){let Option<i32> a=.none;return switch(a){case .none:{0}};}', 'non-exhaustive Option switch'),
    ('duplicate', 'fn i32 main(){let Option<i32> a=.none;return switch(a){case .none:{0}case .none:{1}case .some(n):{n}};}', 'duplicate switch case'),
    ('none-binding', 'fn i32 main(){let Option<i32> a=.none;return switch(a){case .some(n):{n}case .none(n):{n}};}', 'invalid Option pattern'),
    ('some-no-binding', 'fn i32 main(){let Option<i32> a=.none;return switch(a){case .some:{1}case .none:{0}};}', 'invalid Option pattern'),
    ('immutable-binding', 'fn i32 main(){let Option<i32> a=.some(1);return switch(a){case .some(n):{n=2;n}case .none:{0}};}', 'assignment target is immutable'),
    ('arm-leak', 'fn i32 main(){let Option<i32> a=.some(1);return switch(a){case .some(n):{n}case .none:{n}};}', 'unresolved name'),
    ('binding-escapes', 'fn i32 main(){let Option<i32> a=.some(1);discard switch(a){case .some(n):{n}case .none:{0}};return n;}', 'unresolved name'),
    ('wrong-arity', 'fn i32 main(){let Option<i32,bool> a=.none;return 0;}', None),
    ('void', 'fn i32 main(){let Option<void> a=.none;return 0;}', None),
    ('user-enum-positional', 'enum E{A{i32 n;};}fn i32 main(){let E e=.A{.n=1};return switch(e){case .A(n):{n}};}', 'unsupported enum pattern'),
    ('both-assert-arms', 'fn i32 main(){let Option<i32> v=.none;return switch(v){case .some(n):{assert(false);}case .none:{assert(false);}};}', 'invalid Option pattern'),
    ('conditional-assert-arm', 'fn i32 main(){let Option<i32> v=.some(1);return switch(v){case .some(n):{n}case .none:{assert(true);}};}', 'type mismatch'),
]

def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def save(path, value): path.write_text(json.dumps(value, ensure_ascii=False, indent=2)+'\n', encoding='utf-8', newline='\n')

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--images',nargs='+',required=True);parser.add_argument('--phase',choices=['mir','native'],required=True);parser.add_argument('--receipt',required=True);parser.add_argument('--native-runner');args=parser.parse_args()
    receipt=Path(args.receipt).resolve();receipt.parent.mkdir(parents=True,exist_ok=True);images=[Path(p).resolve() for p in args.images];rows=[];native=[];passed=False;units=[];fixtures=[]
    def run(argv, expected, timeout=120):
        argv=list(map(str,argv));p=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=timeout);row=dict(argv=argv,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'));rows.append(row);assert row['exit']==expected,row;return row
    modes=['--check-types-file','--hir-file','--mir-file']+(['--c-file'] if args.phase=='native' else [])
    try:
        for name,source,expected in POSITIVES+TRAPS:
            path=receipt.parent/(name+'.fy');data=('module main;'+source).encode('utf-8');path.write_bytes(data);digest=sha(path);fixtures.append(dict(name=name,inputHex=data.hex(),sha256=digest,expected=expected));outputs={mode:[] for mode in modes}
            for image in images:
                for mode in modes:
                    row=run([image,mode,path],0);assert row['stdout'] and not row['stderr'],row;outputs[mode].append(row['stdout'])
                assert sha(path)==digest
            assert all(len(set(values))==1 for values in outputs.values())
            assert '\t1966' in outputs['--mir-file'][0] and 'option:' in outputs['--hir-file'][0]
            if args.phase=='native':
                is_trap=name in {n for n,_,_ in TRAPS};unit=receipt.parent/(name+'.c');unit.write_text(outputs['--c-file'][0] if is_trap else '#define FY_CANDIDATE_NO_MAIN\n#include <inttypes.h>\n#include <stdio.h>\n'+outputs['--c-file'][0]+'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t result=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",result);return 0;}\n',encoding='utf-8',newline='\n');units.append(dict(name=name,source=str(unit),observe=not is_trap))
                if name=='some':
                    # Test-only corrupted tag and branch choice. The extraction
                    # guard is unchanged and must prevent inactive union reads.
                    code=outputs['--c-file'][0];assert code.count('=UINT32_C(0)')==1 and code.count('.fy_tag == UINT32_C(0)')==1
                    code=code.replace('=UINT32_C(0)','=UINT32_C(1)',1).replace('.fy_tag == UINT32_C(0)','.fy_tag == UINT32_C(1)',1)
                    fault=receipt.parent/'none-as-some.c';fault.write_text(code,encoding='utf-8',newline='\n');units.append(dict(name='none-as-some',source=str(fault),observe=False))
        for name,source,message in NEGATIVES:
            path=receipt.parent/(name+'.fy');path.write_bytes(('module main;'+source).encode());digest=sha(path)
            for image in images:
                for mode in modes:
                    row=run([image,mode,path],2);assert not row['stdout'] and row['stderr'],row
                    if message: assert message in row['stderr'],row
                assert sha(path)==digest
        if args.phase=='native':
            assert args.native_runner;request=receipt.parent/'native-request.json';report=receipt.parent/'native.json';save(request,dict(schema='fairyc.candidate-c-native-request/v1',output=str(receipt.parent/'native'),jobs=units))
            run([Path(args.native_runner).resolve(),request,report],0,300);native=json.loads(report.read_text(encoding='utf-8'));assert len(native)==(2 if sys.platform=='win32' else 4)*len(units)
            expected={name:value for name,_,value in POSITIVES}
            traps={name:value for name,_,value in TRAPS}
            for row in native:
                if row['name']=='none-as-some' or row['name'] in traps:
                    assert row['trap']==traps.get(row['name'],7) and row['exit']!=0 and not row['stdout'] and not row['stderr'],row
                    assert sha(row['image'])==row['imageSHA256'] and sha(row['source'])==row['sourceSHA256'],row
                    continue
                assert row['exit']==0 and row['trap']==0 and row['value']==expected[row['name']] and not row['stdout'],row
                assert row['stderr'].replace('\r\n','\n')==f"fairyc-candidate-result/v1:{expected[row['name']]}\n",row
                assert sha(row['image'])==row['imageSHA256'] and sha(row['source'])==row['sourceSHA256'],row
        passed=True
    finally:
        save(receipt,dict(schema='fairyc.option-candidate-verification/v1',phase=args.phase,passed=passed,images=[dict(path=str(p),sha256=sha(p)) for p in images],cli=rows,native=native,fixtures=fixtures,sources=[dict(path=str(p.relative_to(ROOT)),sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))]))
    print(f'PASS: Option {args.phase}; {len(rows)} invocations; {len(native)} native executions')

if __name__=='__main__': main()
