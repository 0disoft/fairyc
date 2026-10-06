#!/usr/bin/env python3
"""Test-only payload enum inputs and raw observations; product logic stays Fairy."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parent
POSITIVES = [
    ('mixed', 'enum E{None;Data{i64 n;bool ready;};}fn E make(){return .Data{.ready=true,.n=4000000000};}fn i32 main(){let i64 n=switch(make()){case .Data{.ready=_,.n=value}:{value}case .None:{0}};assert(n==4000000000);return 0;}'),
    ('widths', 'enum E{A{i32 a;i64 b;u8 c;u32 d;usize e;bool f;};}fn i32 main(){let E v=.A{.a=-2147483648,.b=-9223372036854775808,.c=255,.d=4294967295,.e=18446744073709551615,.f=true};return switch(v){case .A{.a=a,.b=b,.c=c,.d=d,.e=e,.f=f}:{assert(a==-2147483648);assert(b==-9223372036854775808);assert(c==255);assert(d==4294967295);assert(e==18446744073709551615);assert(f);0}};}'),
    ('shorthand', 'enum E{A{i32 n;};}fn E f(i32 n){return .A{.n};}fn i32 main(){return switch(f(9)){case .A{.n=value}:{value}};}'),
    ('copy-return', 'enum E{A{i32 n;};B;}fn E id(E e){return e;}fn i32 main(){var E e=.A{.n=3};let E saved=id(e);e=.B;return switch(saved){case .A{.n=n}:{n}case .B:{0}};}'),
    ('join', 'enum E{A{bool b;};B;}fn E f(bool yes){return if(yes){.A{.b=true}}else{.B};}fn i32 main(){return switch(f(true)){case .A{.b=b}:{if(b){1}else{0}}case .B:{2}};}'),
    ('nested-shadow', 'enum E{A{i32 n;};}fn i32 main(){let i32 n=4;let E e=.A{.n=n};return switch(e){case .A{.n=n}:{switch(e){case .A{.n=n}:{n}}}}+n;}'),
    ('arm-scope', 'enum E{A{i32 x;};B{i32 y;};}fn i32 main(){let E e=.B{.y=2};return switch(e){case .A{.x=n}:{n}case .B{.y=n}:{n}};}'),
    ('effects', 'enum E{A{i32 a;i32 b;};}fn i32 main(){var i32 n=0;let E e=.A{.b={n+=1;n},.a={n+=1;n}};return switch(e){case .A{.a=a,.b=b}:{a*10+b}};}'),
    ('discard', 'enum E{A{i32 a;bool b;};}fn i32 main(){let E e=.A{.a=1,.b=true};return switch(e){case .A{.b=_,.a=_}:{0}};}'),
    ('sixteen-fields', 'enum E{A{'+''.join(f'i32 f{i};' for i in range(16))+'};}fn i32 main(){let E e=.A{'+','.join(f'.f{i}={i}' for i in range(16))+'};return switch(e){case .A{'+','.join(f'.f{i}=_' for i in range(16))+'}:{0}};}'),
    ('freeze', 'enum E{A{i32 a;i32 b;};}fn i32 main(){var i32 n=1;let E e=.A{.a=n,.b={n=2;n}};return switch(e){case .A{.a=a,.b=b}:{a*10+b}};}'),
    ('subject-freeze', 'enum E{A{i32 a;i32 b;};B;}fn i32 main(){var E e=.A{.a=1,.b=2};return switch(e){case .B:{0}case .A{.a=a,.b=b}:{e=.B;a*10+b}};}'),
    ('loop', 'enum E{A{i32 n;};B;}fn i32 main(){var E e=.A{.n=0};var i32 i=0;while(i<3){e=switch(e){case .A{.n=n}:{.A{.n=n+1}}case .B:{.A{.n=0}}};i+=1;}return switch(e){case .A{.n=n}:{n}case .B:{0}};}'),
    ('early-return', 'enum E{A{i32 a;i32 b;};}fn i32 main(){let E e=.A{.a={return 6;0},.b=99};return switch(e){case .A{.a=a,.b=b}:{a+b}};}'),
    ('multiple-payloads', 'enum E{A{i32 n;};B{bool ready;i64 n;};None;}fn i32 main(){let E e=.B{.n=-4000000000,.ready=true};return switch(e){case .A{.n=n}:{n}case .None:{2}case .B{.ready=b,.n=n}:{assert(b);assert(n==-4000000000);4}};}'),
]
EXPECTED = dict(zip((name for name,_ in POSITIVES), [0,0,9,3,1,8,2,21,0,0,12,12,3,6,4]))
NEGATIVES = [
    ('empty-payload', 'enum E{A{};}fn i32 main(){return 0;}', 'enum payload requires'),
    ('duplicate-declaration-field', 'enum E{A{i32 n;bool n;};}fn i32 main(){return 0;}', 'duplicate field'),
    ('unsupported-text', 'enum E{A{ref str n;};}fn i32 main(){return 0;}', 'enum payload requires'),
    ('unsupported-struct', 'struct S{i32 n;}enum E{A{S n;};}fn i32 main(){return 0;}', 'enum payload requires'),
    ('unsupported-enum', 'enum F{B;}enum E{A{F n;};}fn i32 main(){return 0;}', 'enum payload requires'),
    ('unsupported-array', 'enum E{A{i32[1] n;};}fn i32 main(){return 0;}', 'enum payload requires'),
    ('seventeen-fields', 'enum E{A{'+''.join(f'i32 f{i};' for i in range(17))+'};}fn i32 main(){return 0;}', 'enum payload requires'),
    ('missing-construction', 'enum E{A{i32 n;};}fn i32 main(){let E e=.A;return 0;}', 'enum payload fields'),
    ('missing-field', 'enum E{A{i32 n;bool b;};}fn i32 main(){let E e=.A{.n=1};return 0;}', 'enum payload fields'),
    ('duplicate-initializer', 'enum E{A{i32 n;bool b;};}fn i32 main(){let E e=.A{.n=1,.n=2};return 0;}', 'duplicate field'),
    ('unknown-initializer', 'enum E{A{i32 n;};}fn i32 main(){let E e=.A{.x=1};return 0;}', 'unknown field'),
    ('wrong-type', 'enum E{A{i32 n;};}fn i32 main(){let E e=.A{.n=true};return 0;}', 'type mismatch'),
    ('overflow', 'enum E{A{u8 n;};}fn i32 main(){let E e=.A{.n=256};return 0;}', 'integer literal out of range'),
    ('untyped', 'enum E{A{i32 n;};}fn i32 main(){let e=.A{.n=1};return 0;}', 'enum context required'),
    ('extra-empty-construction', 'enum E{A;}fn i32 main(){let E e=.A{};return 0;}', 'enum payload fields'),
    ('missing-pattern', 'enum E{A{i32 n;};}fn i32 main(){let E e=.A{.n=1};return switch(e){case .A:{0}};}', 'enum payload fields'),
    ('duplicate-pattern-field', 'enum E{A{i32 n;bool b;};}fn i32 main(){let E e=.A{.n=1,.b=true};return switch(e){case .A{.n=a,.n=b}:{0}};}', 'duplicate field'),
    ('unknown-pattern-field', 'enum E{A{i32 n;};}fn i32 main(){let E e=.A{.n=1};return switch(e){case .A{.x=n}:{0}};}', 'unknown field'),
    ('duplicate-pattern-binding', 'enum E{A{i32 n;bool b;};}fn i32 main(){let E e=.A{.n=1,.b=true};return switch(e){case .A{.n=n,.b=n}:{0}};}', 'duplicate name'),
    ('immutable-binding', 'enum E{A{i32 n;};}fn i32 main(){let E e=.A{.n=1};return switch(e){case .A{.n=n}:{n=2;n}};}', 'assignment target is immutable'),
    ('binding-escapes', 'enum E{A{i32 n;};}fn i32 main(){let E e=.A{.n=1};discard switch(e){case .A{.n=n}:{n}};return n;}', 'unresolved name'),
    ('arm-leak', 'enum E{A{i32 n;};B;}fn i32 main(){let E e=.A{.n=1};return switch(e){case .A{.n=n}:{n}case .B:{n}};}', 'unresolved name'),
    ('missing-case', 'enum E{A{i32 n;};B;}fn i32 main(){let E e=.A{.n=1};return switch(e){case .A{.n=n}:{n}};}', 'non-exhaustive enum switch'),
    ('foreign-payload-field', 'enum E{A{i32 n;};B{bool b;};}fn i32 main(){let E e=.A{.b=true};return 0;}', 'unknown field'),
]

def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def main():
    p=argparse.ArgumentParser();p.add_argument('--images',nargs='+',required=True);p.add_argument('--receipt',required=True);p.add_argument('--phase',choices=['hir','full'],default='full');p.add_argument('--native-runner');args=p.parse_args()
    receipt=Path(args.receipt).resolve();receipt.parent.mkdir(parents=True,exist_ok=True);rows=[];passed=False
    images=[Path(i).resolve() for i in args.images];units=[];native=[]
    def save(path,data): path.write_text(json.dumps(data,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
    def run(image,mode,path,expected):
        argv=list(map(str,[image,mode,path]));proc=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=120)
        row=dict(argv=argv,cwd=str(ROOT),exit=proc.returncode,stdout=proc.stdout.decode('utf-8'),stderr=proc.stderr.decode('utf-8'));rows.append(row)
        assert row['exit']==expected,row
        return row
    try:
        for name,source in POSITIVES:
            path=receipt.parent/(name+'.fy');path.write_bytes(('module main;'+source).encode('utf-8'));digest=sha(path);hirs=[]
            codes=[];mirs=[]
            for image in images:
                for mode in ['--check-types-file','--hir-file']:
                    row=run(image,mode,path,0);assert row['stdout'] and not row['stderr'],row
                    if mode=='--hir-file':assert row['stdout'].startswith('fairyc.enum-payload-hir/v1\n'),row;hirs.append(row['stdout'])
                for mode in ['--mir-file','--c-file']:
                    row=run(image,mode,path,0);assert row['stdout'] and not row['stderr'],row
                    if mode=='--mir-file':assert row['stdout'].startswith('fairyc.enum-payload-mir/v1\n');mirs.append(row['stdout'])
                    else:assert row['stdout'].startswith('/* fairyc.candidate-enum-payload-c/v1;');codes.append(row['stdout'])
                assert sha(path)==digest
            assert len(set(hirs))==1
            if args.phase=='full':
                assert len(set(mirs))==1 and len(set(codes))==1
                unit=receipt.parent/(name+'.c');unit.write_text('#define FY_CANDIDATE_NO_MAIN\n#include <inttypes.h>\n#include <stdio.h>\n'+codes[0]+'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t result=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",result);return 0;}\n',encoding='utf-8',newline='\n');units.append(dict(name=name,source=str(unit),observe=True))
                if name=='shorthand':
                    # Test-only corruption of the emitted constructor's active
                    # tag. The unchanged extraction must trap before union read.
                    needle='=UINT32_C(0)';assert codes[0].count(needle)==1
                    corrupt=receipt.parent/'wrong-active-tag.c';corrupt.write_text(codes[0].replace(needle,'=UINT32_C(1)',1),encoding='utf-8',newline='\n')
                    units.append(dict(name='wrong-active-tag',source=str(corrupt),observe=False))
        for name,source,message in NEGATIVES:
            path=receipt.parent/(name+'.fy');path.write_bytes(('module main;'+source).encode('utf-8'));digest=sha(path)
            for image in images:
                for mode in (['--check-types-file','--hir-file','--mir-file','--c-file'] if args.phase=='full' else ['--check-types-file','--hir-file']):
                    row=run(image,mode,path,2);assert not row['stdout'] and message in row['stderr'],row
                assert sha(path)==digest
        if args.phase=='full':
            assert args.native_runner
            request=receipt.parent/'native-request.json';report=receipt.parent/'native.json'
            save(request,dict(schema='fairyc.candidate-c-native-request/v1',output=str(receipt.parent/'native'),jobs=units))
            argv=[str(Path(args.native_runner).resolve()),str(request),str(report)];proc=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=300)
            row=dict(argv=argv,cwd=str(ROOT),exit=proc.returncode,stdout=proc.stdout.decode('utf-8'),stderr=proc.stderr.decode('utf-8'));rows.append(row);assert row['exit']==0,row
            native=json.loads(report.read_text(encoding='utf-8'));assert len(native)==2*len(units)
            for row in native:
                if row['name']=='wrong-active-tag':
                    assert row['trap']==7 and not row['stdout'] and not row['stderr'],row
                    assert sha(row['image'])==row['imageSHA256'] and sha(row['source'])==row['sourceSHA256'],row
                    continue
                assert row['trap']==0 and row['exit']==0 and row['value']==EXPECTED[row['name']] and not row['stdout'],row
                assert row['stderr'].replace('\r\n','\n')==f"fairyc-candidate-result/v1:{EXPECTED[row['name']]}\n",row
                assert sha(row['image'])==row['imageSHA256'] and sha(row['source'])==row['sourceSHA256'],row
        passed=True
    finally:
        save(receipt,dict(schema='fairyc.enum-payload-verification/v1',phase=args.phase,passed=passed,images=[dict(path=str(i),sha256=sha(i)) for i in images],cli=rows,native=native,expected=EXPECTED))
    print(f'PASS: payload {args.phase}; {len(rows)} invocations; {len(native)} native executions')

if __name__=='__main__':main()
