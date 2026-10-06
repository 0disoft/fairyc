#!/usr/bin/env python3
"""Test-only enum fixtures and CLI/native observations; no product implementation."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parent
POSITIVES = [
    ('enum-value', 'enum E{A;B;}fn i32 main(){let E e=.B;return switch(e){case .A:{3}case .B:{7}};}', 7),
    ('enum-return', 'enum E{A;B;}fn E id(E e){return e;}fn E make(){return .A;}fn i32 main(){return switch(id(make())){case .B:{2}case .A:{4}};}', 4),
    ('enum-assignment', 'enum E{A;B;}fn i32 main(){var E e=.A;let original=e;e=.B;return switch(original){case .A:{switch(e){case .A:{1}case .B:{8}}}case .B:{2}};}', 8),
    ('enum-join', 'enum E{A;B;}fn E choose(bool yes){return if(yes){.A}else{.B};}fn i32 main(){let E e=switch(choose(false)){case .A:{.B}case .B:{.A}};return switch(e){case .B:{9}case .A:{5}};}', 5),
    ('enum-effects', 'enum E{A;B;}fn i32 main(){var i32 n=1;var E e=.A;let i32 value=switch({n+=1;e}){case .B:{n+=100;8}case .A:{e=.B;n+=10;3}};return value+n;}', 15),
    ('enum-early-return', 'enum E{A;B;}fn i32 f(E e){switch(e){case .A:{return 11;}case .B:{return 12;}};}fn i32 main(){return f(.B);}', 12),
    ('enum-loop', 'enum E{A;B;}fn i32 main(){var E e=.A;var i32 n=0;while(n<3){e=switch(e){case .A:{.B}case .B:{.A}};n+=1;}return switch(e){case .A:{1}case .B:{6}};}', 6),
    ('enum-separate', 'enum E{A;B;}enum F{A;B;}struct P{i32 x;}fn i32 main(){let E e=.A;let F f=.B;let p=P{.x=2};return p.x+switch(e){case .A:{3}case .B:{4}}+switch(f){case .A:{5}case .B:{6}};}', 11),
    ('enum-call-freeze', 'enum E{A;B;}fn i32 f(E a,E b){return switch(a){case .A:{10}case .B:{20}}+switch(b){case .A:{1}case .B:{2}};}fn i32 main(){var E e=.A;return f(e,{e=.B;e});}', 12),
    ('enum-unsigned-join', 'enum E{A;B;}fn i32 main(){let E e=.B;let u8 n=switch(e){case .A:{1}case .B:{255}};assert(n==255);return 0;}', 0),
    ('enum-defer', 'enum E{A;B;}fn void check(E e){assert(switch(e){case .A:{false}case .B:{true}});}fn i32 main(){let E e=.B;switch(e){case .A:{}case .B:{defer check(e);}};return 0;}', 0),
]
NEGATIVES = [
    ('labeled-argument', 'enum E{A;}fn i32 f(E e){return 0;}fn i32 main(){return f(.e=.A);}', 'unsupported name-check syntax'),
    ('empty', 'enum E{}fn i32 main(){return 0;}', 'expected nonempty payloadless enum'),
    ('duplicate-member', 'enum E{A;A;}fn i32 main(){return 0;}', 'duplicate enum member'),
    ('payload-closed', 'enum E{A{i32 n;};}fn i32 main(){return 0;}', 'expected nonempty payloadless enum'),
    ('empty-payload-closed', 'enum E{A{};}fn i32 main(){return 0;}', 'expected nonempty payloadless enum'),
    ('unknown-member', 'enum E{A;}fn i32 main(){let E e=.B;return 0;}', 'unknown enum member'),
    ('untyped', 'enum E{A;}fn i32 main(){let e=.A;return 0;}', 'enum context required'),
    ('missing-case', 'enum E{A;B;}fn i32 main(){let E e=.A;return switch(e){case .A:{0}};}', 'non-exhaustive enum switch'),
    ('duplicate-case', 'enum E{A;B;}fn i32 main(){let E e=.A;return switch(e){case .A:{0}case .A:{1}case .B:{2}};}', 'duplicate switch case'),
    ('unknown-case', 'enum E{A;}fn i32 main(){let E e=.A;return switch(e){case .B:{0}};}', 'unknown enum member'),
    ('different-types', 'enum E{A;}enum F{A;}fn i32 main(){let F f=.A;let E e=f;return 0;}', 'type mismatch'),
    ('integer-coercion', 'enum E{A;}fn i32 main(){let E e=0;return 0;}', 'type mismatch'),
    ('enum-equality', 'enum E{A;}fn i32 main(){let E e=.A;assert(e==e);return 0;}', 'invalid scalar operator'),
    ('enum-field', 'enum E{A;}struct S{E e;}fn i32 main(){return 0;}', 'invalid nominal type'),
    ('enum-array', 'enum E{A;}fn i32 main(){let E[1] a=[.A];return 0;}', None),
    ('join-mismatch', 'enum E{A;B;}fn i32 main(){let E e=.A;return switch(e){case .A:{0}case .B:{true}};}', 'type mismatch'),
    ('immutable-enum', 'enum E{A;B;}fn i32 main(){let E e=.A;e=.B;return 0;}', 'assignment target is immutable'),
    ('view-join', 'enum E{A;}fn i32 main(){let E e=.A;discard switch(e){case .A:{b"A"}};return 0;}', 'unsupported enum pattern'),
]

def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def save(path, value): Path(path).write_text(json.dumps(value, ensure_ascii=False, indent=2)+'\n', encoding='utf-8', newline='\n')
def verify(images, directory, phase, rows):
    directory.mkdir(parents=True, exist_ok=True)
    def run(argv, expected):
        argv=list(map(str, argv));p=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=120)
        row=dict(argv=argv,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'));rows.append(row)
        assert row['exit']==expected,row
        return row
    modes=['--check-types-file','--hir-file'] if phase=='hir' else ['--check-types-file','--hir-file','--mir-file','--c-file']
    units=[]
    for name, source, expected in POSITIVES:
        path=directory/(name+'.fy');path.write_bytes(('module main;'+source).encode());before=sha(path);outputs=[]
        for image in images:
            for mode in modes:
                row=run([image,mode,path],0);assert row['stdout'] and not row['stderr'],row
                if mode=='--hir-file':assert row['stdout'].startswith('fairyc.enum-hir/v1\n'),row
                if mode=='--c-file':outputs.append(row['stdout'])
            assert sha(path)==before
        if outputs:
            assert len(set(outputs))==1
            unit=directory/(name+'.c');unit.write_text(outputs[0],encoding='utf-8',newline='\n');units.append(dict(name=name,source=str(unit),expected=expected))
    for name, source, message in NEGATIVES:
        path=directory/(name+'.fy');path.write_bytes(('module main;'+source).encode());before=sha(path)
        for image in images:
            for mode in modes:
                row=run([image,mode,path],2);assert not row['stdout'] and row['stderr'],row
                if message:assert message in row['stderr'],row
            assert sha(path)==before
    return units

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--images',nargs='+',required=True);parser.add_argument('--phase',choices=['hir','full'],required=True);parser.add_argument('--receipt',required=True);parser.add_argument('--native-runner');args=parser.parse_args()
    images=[Path(p).resolve() for p in args.images];receipt=Path(args.receipt).resolve();rows=[]
    try:
        units=verify(images,receipt.parent,args.phase,rows);native=[]
        if args.phase=='full':
            assert args.native_runner
            request=receipt.parent/'native-request.json';native_report=receipt.parent/'native.json'
            save(request,dict(schema='fairyc.candidate-c-native-request/v1',output=str(receipt.parent/'native'),jobs=[dict(name=u['name'],source=u['source']) for u in units]))
            argv=[str(Path(args.native_runner).resolve()),str(request),str(native_report)];p=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=300);row=dict(argv=argv,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'));rows.append(row);assert p.returncode==0,row
            native=json.loads(native_report.read_text(encoding='utf-8'));assert len(native)==2*len(units)
            expected={u['name']:u['expected'] for u in units}
            for row in native:assert row['trap']==0 and row['value']==expected[row['name']] and sha(row['image'])==row['imageSHA256'],row
        save(receipt,dict(schema='fairyc.enum-candidate-verification/v1',phase=args.phase,images=[dict(path=str(p),sha256=sha(p)) for p in images],cli=rows,cliExecutions=len(images)*(len(POSITIVES)+len(NEGATIVES))*len(['types','hir'] if args.phase=='hir' else ['types','hir','mir','c']),native=native,passed=True))
        print(f'PASS: {args.phase}; {len(rows)} invocations; {len(native)} native executions')
    except Exception:
        save(receipt.with_suffix('.failure.json'),dict(cli=rows,passed=False));raise

if __name__=='__main__': main()
