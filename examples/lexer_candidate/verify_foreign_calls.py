#!/usr/bin/env python3
"""Observe Fairy-emitted scalar C imports using a separate provider C object."""
import argparse,hashlib,json,subprocess,sys
from pathlib import Path
ROOT=Path(__file__).resolve().parent;REPO=ROOT.parents[1]
PROVIDER='''#include <stdint.h>
#include <stdio.h>
int32_t fixture_add(int32_t n){return n+1;}
const int32_t *fixture_pointer(const int32_t *p){static const int32_t x=7;return p?p:&x;}
void *fixture_opaque(void *p){return p;}
int32_t fixture_null(const uint8_t *p){return p==NULL?7:0;}
int64_t fixture_wide(int64_t n){return n+1;}
uint8_t fixture_byte(uint8_t n){return n;}
uint32_t fixture_word(uint32_t n){return n;}
uintptr_t fixture_size(uintptr_t n){return n;}
int32_t fixture_signed(int32_t n){return n;}
int32_t fixture_trace(int32_t n){(void)fputc((unsigned char)n,stdout);(void)fflush(stdout);return n;}
int32_t fixture_pair(int32_t a,int32_t b){return a+b;}
void fixture_emit(int32_t n){(void)fputc((unsigned char)n,stdout);(void)fflush(stdout);}
'''
CASES=[
 ('default-symbol','unsafe extern C fn i32 fixture_add(i32 n);fn i32 main(){return unsafe{fixture_add(6)};}','',7),
 ('linked-alias','@link_name("fixture_add")unsafe extern C fn i32 foreign(i32 n);fn i32 main(){return unsafe{foreign(6)};}','',7),
 ('wide-i64','unsafe extern C fn i64 fixture_wide(i64 n);fn i32 main(){let i64 n=unsafe{fixture_wide(4000000000)};assert(n==4000000001);return 7;}','',7),
 ('scalar-extremes','unsafe extern C fn u8 fixture_byte(u8 n);unsafe extern C fn u32 fixture_word(u32 n);unsafe extern C fn usize fixture_size(usize n);unsafe extern C fn i32 fixture_signed(i32 n);fn i32 main(){unsafe{assert(fixture_byte(255)==255);assert(fixture_word(4294967295)==4294967295);assert(fixture_size(18446744073709551615)==18446744073709551615);assert(fixture_signed(-2147483648)==-2147483648);};return 7;}','',7),
 ('ordered-arguments','@effects(console)unsafe extern C fn i32 fixture_trace(i32 n);unsafe extern C fn i32 fixture_pair(i32 a,i32 b);@effects(console)fn i32 main(){return unsafe{fixture_pair(fixture_trace(65),fixture_trace(66))};}','AB',131),
 ('effect-and-must-use','@must_use @effects(console) @link_name("fixture_trace")unsafe extern C fn i32 token(i32 n);@effects(console)fn i32 main(){return unsafe{token(65)};}','A',65),
 ('void-defer','@effects(console)unsafe extern C fn void fixture_emit(i32 n);@effects(console)fn i32 main(){unsafe{defer fixture_emit(65);fixture_emit(88);};return 7;}','XA',7),
 ('cleanup-import','@effects(console)unsafe extern C fn void fixture_emit(i32 n);@effects(console)resource R{cleanup{unsafe{fixture_emit(65);};}}@effects(console)fn i32 main(){let R r=R{};return 7;}','A',7),
 ('same-link-compatible','@link_name("fixture_add")unsafe extern C fn i32 a(i32 n);@link_name("fixture_add")unsafe extern C fn i32 b(i32 n);fn i32 main(){return unsafe{a(2)+b(3)};}','',7),
]
POINTER_CASES=[
 ('typed-null','unsafe extern C fn i32 fixture_null(ptr u8 p);fn i32 main(){return unsafe{fixture_null(null)};}','',7),
 ('pointer-roundtrip','unsafe extern C fn ptr i32 fixture_pointer(ptr i32 p);fn ptr i32 forward(ptr i32 p){return p;}fn i32 main(){let ptr i32 p=unsafe{fixture_pointer(null)};assert(p!=null);assert(null!=p);assert(forward(p)==p);return 7;}','',7),
 ('mutable-opaque','unsafe extern C fn ptr mut void fixture_opaque(ptr mut void p);fn i32 main(){var ptr mut void p=null;p=unsafe{fixture_opaque(p)};assert(p==null);assert(null==p);return 7;}','',7),
 ('pointer-field','struct S{ptr u8 p;}fn i32 main(){let S s=S{.p=null};assert(s.p==null);return 7;}','',7),
]
POINTER_REJECTIONS=[
 ('null-integer','fn i32 main(){let i32 x=null;return x;}','type mismatch'),
 ('null-inferred','fn i32 main(){let x=null;return 0;}','type mismatch'),
 ('mismatch-mutability','fn i32 main(){let ptr i32 a=null;let ptr mut i32 b=a;return 0;}','type mismatch'),
 ('mismatch-pointee','fn i32 main(){let ptr i32 a=null;let ptr u8 b=a;return 0;}','type mismatch'),
 ('pointer-order','fn i32 main(){let ptr i32 a=null;assert(a<a);return 0;}','invalid scalar operator'),
 ('raw-dereference','fn i32 main(){let ptr i32 a=null;return unsafe{*a};}','invalid nominal type'),
 ('pointer-arithmetic','fn i32 main(){let ptr i32 a=null;let b=unsafe{a+1};return 0;}','type mismatch'),
 ('nested-pointer','unsafe extern C fn i32 foreign(ptr ptr i32 p);fn i32 main(){return 0;}','E5102'),
 ('null-safe-borrow','fn i32 main(){let ref i32 a=null;return 0;}','type mismatch'),
]
REJECTIONS=[
 ('no-unsafe','unsafe extern C fn i32 fixture_add(i32 n);fn i32 main(){return fixture_add(6);}','E5101'),
 ('missing-effect','@effects(console)unsafe extern C fn i32 fixture_trace(i32 n);fn i32 main(){return unsafe{fixture_trace(65)};}','E7102'),
 ('ignored-value','@must_use unsafe extern C fn i32 fixture_add(i32 n);fn i32 main(){unsafe{fixture_add(6);};return 0;}','E4202'),
 ('safe-reference','unsafe extern C fn i32 foreign(ref i32 n);fn i32 main(){return 0;}','E5102'),
 ('bool-abi','unsafe extern C fn bool foreign();fn i32 main(){return 0;}','E5102'),
 ('aggregate-abi','struct S{i32 n;}unsafe extern C fn S foreign();fn i32 main(){return 0;}','E5102'),
 ('pointer-bool','unsafe extern C fn i32 foreign(ptr bool n);fn i32 main(){return 0;}','E5102'),
 ('empty-link','@link_name("")unsafe extern C fn i32 foreign();fn i32 main(){return 0;}','E5106'),
 ('multiple-link','@link_name("fixture_add")@link_name("fixture_add")unsafe extern C fn i32 foreign(i32 n);fn i32 main(){return 0;}','E5106'),
 ('link-number','@link_name(7)unsafe extern C fn i32 foreign();fn i32 main(){return 0;}','E5106'),
 ('link-keyword','@link_name("while")unsafe extern C fn i32 foreign();fn i32 main(){return 0;}','E5106'),
 ('link-injection','@link_name("x);system(0)")unsafe extern C fn i32 foreign();fn i32 main(){return 0;}','E5106'),
 ('runtime-namespace','@link_name("fy_runtime_trap")unsafe extern C fn i32 foreign();fn i32 main(){return 0;}','E5106'),
 ('c-macro','@link_name("INT32_MAX")unsafe extern C fn i32 foreign();fn i32 main(){return 0;}','E5106'),
 ('conflicting-link','@link_name("fixture_add")unsafe extern C fn i32 a(i32 n);@link_name("fixture_add")unsafe extern C fn u32 b(u32 n);fn i32 main(){return 0;}','E5106'),
]
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def save(p,data):p.write_text(json.dumps(data,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
def main():
 ap=argparse.ArgumentParser();ap.add_argument('--images',nargs='+',required=True);ap.add_argument('--native-runner',required=True);ap.add_argument('--receipt',required=True);ap.add_argument('--cli-only',action='store_true');ap.add_argument('--lane',choices=['scalar','pointer'],default='scalar');args=ap.parse_args()
 cases=POINTER_CASES if args.lane=='pointer' else CASES;rejections=POINTER_REJECTIONS if args.lane=='pointer' else REJECTIONS
 images=[Path(s).resolve() for s in args.images];runner=Path(args.native_runner).resolve();receipt=Path(args.receipt).resolve();receipt.parent.mkdir(parents=True,exist_ok=True);rows=[];fixtures=[];jobs=[];native=[];passed=False
 def run(argv,expected=0):
  q=subprocess.run(list(map(str,argv)),cwd=ROOT,capture_output=True,encoding='utf-8',timeout=360);row=dict(argv=list(map(str,argv)),cwd=str(ROOT),exit=q.returncode,stdout=q.stdout,stderr=q.stderr);rows.append(row);save(receipt.parent/'raw.json',rows);assert q.returncode==expected,row;return row
 provider=receipt.parent/'provider.c';provider.write_text(PROVIDER,encoding='utf-8',newline='\n')
 try:
  for name,body,trace,value in cases:
   source=receipt.parent/(name+'.fy');source.write_text('module main;'+body,encoding='utf-8',newline='\n');outputs=[];fixtures.append(dict(name=name,inputHex=source.read_bytes().hex(),sha256=sha(source),expectedTrace=trace,expectedValue=value))
   for image in images:
    for mode in ('--check-types-file','--hir-file','--mir-file','--c-file'):
     row=run([image,mode,source]);assert row['stdout'] and not row['stderr'],row
     if mode=='--c-file':outputs.append(row['stdout'])
   assert all(s==outputs[0] for s in outputs)
   plain=receipt.parent/(name+'.c');plain.write_text(outputs[0],encoding='utf-8',newline='\n')
   observed=receipt.parent/(name+'.observed.c');observed.write_text('#define FY_CANDIDATE_NO_MAIN\n#include <stdio.h>\n#include <inttypes.h>\n'+outputs[0]+'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t n=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",n);return 0;}\n',encoding='utf-8',newline='\n')
   jobs.append(dict(name=name,source=str(observed),foreignSource=str(provider),observe=True))
  for name,body,diagnostic in rejections:
   source=receipt.parent/(name+'.fy');source.write_text('module main;'+body,encoding='utf-8',newline='\n');fixtures.append(dict(name=name,inputHex=source.read_bytes().hex(),sha256=sha(source),expectedDiagnostic=diagnostic))
   for image in images:
    for mode in ('--mir-file','--c-file'):
     row=run([image,mode,source],2);assert not row['stdout'] and diagnostic in row['stderr'],row
  if not args.cli_only:
   request=receipt.parent/'native-request.json';report=receipt.parent/'native.json';save(request,dict(schema='fairyc.candidate-c-native-request/v1',output=str(receipt.parent/'native'),jobs=jobs));run([runner,request,report]);native=json.loads(report.read_text(encoding='utf-8'));assert len(native)==(2 if sys.platform=='win32' else 4)*len(jobs)
  expected={n:(t,v) for n,_,t,v in cases}
  for row in native:
   trace,value=expected[row['name']];assert not row.get('error') and row['trap']==0 and row['exit']==0 and row['value']==value and row['stdout']==trace and row['foreignSHA256']==sha(provider),row
  passed=True
 finally:
  save(receipt,dict(schema='fairyc.candidate-'+args.lane+'-foreign-native/v1',passed=passed,nativeValidated=passed and not args.cli_only,images=[dict(path=str(p),sha256=sha(p)) for p in images],nativeRunner=dict(path=str(runner),sha256=sha(runner)),foreignProvider=dict(path=str(provider),sha256=sha(provider),inputHex=provider.read_bytes().hex(),method='separate runtime/provider translation unit; ordinary C linking, no generated product logic replaced'),fixtures=fixtures,cli=rows,native=native,sources=[dict(path=str(p.relative_to(REPO)),sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))],observer=dict(path=str(Path(__file__).relative_to(REPO)),sha256=sha(Path(__file__))),lock=dict(sha256=sha(ROOT/'fairy.lock'),packages=json.loads((ROOT/'fairy.lock').read_text(encoding='utf-8'))['packages'])))
 print(f'PASS {args.lane} foreign calls: {len(fixtures)} fixtures, {len(native)} native runs')
if __name__=='__main__':main()
