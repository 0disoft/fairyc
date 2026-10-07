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
int32_t fixture_read(const int32_t *p){return *p;}
void fixture_store(int32_t *p,int32_t n){*p=n;}
int32_t fixture_read_opaque(const void *p){return *(const int32_t*)p;}
int32_t fixture_bytes(const uint8_t *p,uintptr_t n){return n==2&&p[0]==65&&p[1]==66?7:0;}
void fixture_emit(int32_t n){(void)fputc((unsigned char)n,stdout);(void)fflush(stdout);}
'''
PROVIDER += '\n#include <stddef.h>\n_Bool fixture_make_bool(void){return 1;}\nint32_t fixture_consume_bool(_Bool n){return n==1?7:0;}\nchar fixture_make_char(void){return -3;}\nint32_t fixture_consume_char(char n){return n==-3?7:0;}\ndouble fixture_make_double(void){return 1.25;}\nint32_t fixture_consume_double(double n){return n==1.25?7:0;}\nfloat fixture_make_float(void){return 1.25f;}\nint32_t fixture_consume_float(float n){return n==1.25f?7:0;}\nint fixture_make_int(void){return -7;}\nint32_t fixture_consume_int(int n){return n==-7?7:0;}\nlong fixture_make_long(void){return -9L;}\nint32_t fixture_consume_long(long n){return n==-9L?7:0;}\nlong long fixture_make_longlong(void){return -11LL;}\nint32_t fixture_consume_longlong(long long n){return n==-11LL?7:0;}\nsigned char fixture_make_schar(void){return -13;}\nint32_t fixture_consume_schar(signed char n){return n==-13?7:0;}\nshort fixture_make_short(void){return -15;}\nint32_t fixture_consume_short(short n){return n==-15?7:0;}\nsize_t fixture_make_size_t(void){return 17;}\nint32_t fixture_consume_size_t(size_t n){return n==17?7:0;}\nunsigned char fixture_make_uchar(void){return 19;}\nint32_t fixture_consume_uchar(unsigned char n){return n==19?7:0;}\nunsigned int fixture_make_uint(void){return 23U;}\nint32_t fixture_consume_uint(unsigned int n){return n==23U?7:0;}\nunsigned long fixture_make_ulong(void){return 29UL;}\nint32_t fixture_consume_ulong(unsigned long n){return n==29UL?7:0;}\nunsigned long long fixture_make_ulonglong(void){return 31ULL;}\nint32_t fixture_consume_ulonglong(unsigned long long n){return n==31ULL?7:0;}\nunsigned short fixture_make_ushort(void){return 37;}\nint32_t fixture_consume_ushort(unsigned short n){return n==37?7:0;}\n'
PROVIDER += 'int32_t fixture_slot(void **out){static int32_t n=7;*out=&n;return 7;}\nvoid fixture_clear_slot(void **out){*out=NULL;}\nint32_t fixture_mut_opaque(void *p){return *(int32_t*)p;}\n'
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
CAST_CASES=[
 ('scalar-shared','unsafe extern C fn i32 fixture_read(ptr i32 p);fn i32 main(){let i32 n=7;return unsafe{fixture_read(ref n as ptr i32)};}','',7),
 ('scalar-mutable','unsafe extern C fn void fixture_store(ptr mut i32 p,i32 n);fn i32 main(){var i32 n=1;unsafe{fixture_store(ref mut n as ptr mut i32,7);};return n;}','',7),
 ('mutable-to-shared','unsafe extern C fn i32 fixture_read(ptr i32 p);fn i32 main(){var i32 n=7;let ref mut i32 r=ref mut n;return unsafe{fixture_read(r as ptr i32)};}','',7),
 ('opaque-owner','unsafe extern C fn i32 fixture_read_opaque(ptr void p);fn i32 main(){let i32 n=7;return unsafe{fixture_read_opaque(ref n as ptr void)};}','',7),
 ('byte-view-data','unsafe extern C fn i32 fixture_bytes(ptr u8 p,usize n);fn i32 main(){let u8[2] a=[65,66];let ref u8[] r=ref a;return unsafe{fixture_bytes(r as ptr u8,2)};}','',7),
 ('text-data','unsafe extern C fn i32 fixture_bytes(ptr u8 p,usize n);fn i32 main(){let ref str s="AB";return unsafe{fixture_bytes(s as ptr u8,2)};}','',7),
 ('unsafe-wrapper','unsafe fn ptr i32 address(ref i32 n){return n as ptr i32;}unsafe extern C fn i32 fixture_read(ptr i32 p);fn i32 main(){let i32 n=7;return unsafe{fixture_read(address(ref n))};}','',7),
]
CAST_REJECTIONS=[
 ('safe-cast','fn i32 main(){let i32 n=7;let ptr i32 p=ref n as ptr i32;return 0;}','E5101'),
 ('strengthened-cast','fn i32 main(){let i32 n=7;let p=unsafe{ref n as ptr mut i32};return 0;}','unsupported explicit cast'),
 ('wrong-pointee-cast','fn i32 main(){let i32 n=7;let p=unsafe{ref n as ptr u8};return 0;}','unsupported explicit cast'),
 ('integer-pointer-cast','fn i32 main(){let p=unsafe{7 as ptr i32};return 0;}','unsupported explicit cast'),
 ('pointer-integer-cast','fn i32 main(){let ptr i32 p=null;let n=unsafe{p as usize};return 0;}','unsupported explicit cast'),
 ('text-mutable-cast','fn i32 main(){let ref str s="AB";let p=unsafe{s as ptr mut u8};return 0;}','unsupported explicit cast'),
 ('opaque-projection-cast','struct S{i32 n;}fn i32 main(){let S s=S{.n=7};let p=unsafe{ref s.n as ptr void};return 0;}','unsupported explicit cast'),
]
ABI_CASES=[
 ('all-C-scalar-families','unsafe extern C fn c::bool fixture_make_bool();unsafe extern C fn i32 fixture_consume_bool(c::bool n);unsafe extern C fn c::char fixture_make_char();unsafe extern C fn i32 fixture_consume_char(c::char n);unsafe extern C fn c::double fixture_make_double();unsafe extern C fn i32 fixture_consume_double(c::double n);unsafe extern C fn c::float fixture_make_float();unsafe extern C fn i32 fixture_consume_float(c::float n);unsafe extern C fn c::int fixture_make_int();unsafe extern C fn i32 fixture_consume_int(c::int n);unsafe extern C fn c::long fixture_make_long();unsafe extern C fn i32 fixture_consume_long(c::long n);unsafe extern C fn c::longlong fixture_make_longlong();unsafe extern C fn i32 fixture_consume_longlong(c::longlong n);unsafe extern C fn c::schar fixture_make_schar();unsafe extern C fn i32 fixture_consume_schar(c::schar n);unsafe extern C fn c::short fixture_make_short();unsafe extern C fn i32 fixture_consume_short(c::short n);unsafe extern C fn c::size_t fixture_make_size_t();unsafe extern C fn i32 fixture_consume_size_t(c::size_t n);unsafe extern C fn c::uchar fixture_make_uchar();unsafe extern C fn i32 fixture_consume_uchar(c::uchar n);unsafe extern C fn c::uint fixture_make_uint();unsafe extern C fn i32 fixture_consume_uint(c::uint n);unsafe extern C fn c::ulong fixture_make_ulong();unsafe extern C fn i32 fixture_consume_ulong(c::ulong n);unsafe extern C fn c::ulonglong fixture_make_ulonglong();unsafe extern C fn i32 fixture_consume_ulonglong(c::ulonglong n);unsafe extern C fn c::ushort fixture_make_ushort();unsafe extern C fn i32 fixture_consume_ushort(c::ushort n);fn i32 main(){unsafe{let c::bool n_bool=fixture_make_bool();assert(fixture_consume_bool(n_bool)==7);let c::char n_char=fixture_make_char();assert(fixture_consume_char(n_char)==7);let c::double n_double=fixture_make_double();assert(fixture_consume_double(n_double)==7);let c::float n_float=fixture_make_float();assert(fixture_consume_float(n_float)==7);let c::int n_int=fixture_make_int();assert(fixture_consume_int(n_int)==7);let c::long n_long=fixture_make_long();assert(fixture_consume_long(n_long)==7);let c::longlong n_longlong=fixture_make_longlong();assert(fixture_consume_longlong(n_longlong)==7);let c::schar n_schar=fixture_make_schar();assert(fixture_consume_schar(n_schar)==7);let c::short n_short=fixture_make_short();assert(fixture_consume_short(n_short)==7);let c::size_t n_size_t=fixture_make_size_t();assert(fixture_consume_size_t(n_size_t)==7);let c::uchar n_uchar=fixture_make_uchar();assert(fixture_consume_uchar(n_uchar)==7);let c::uint n_uint=fixture_make_uint();assert(fixture_consume_uint(n_uint)==7);let c::ulong n_ulong=fixture_make_ulong();assert(fixture_consume_ulong(n_ulong)==7);let c::ulonglong n_ulonglong=fixture_make_ulonglong();assert(fixture_consume_ulonglong(n_ulonglong)==7);let c::ushort n_ushort=fixture_make_ushort();assert(fixture_consume_ushort(n_ushort)==7);};return 7;}','',7),
 ('c-int-casts','fn i32 main(){let c::int n=7 as c::int;return n as i32;}','',7),
 ('c-int-extremes','fn i32 main(){let c::int low=-2147483648 as c::int;let c::int high=2147483647 as c::int;assert((low as i32)==-2147483648);assert((high as i32)==2147483647);return 7;}','',7),
 ('c-field-copy','struct S{c::int n;}fn i32 main(){let S s=S{.n=7 as c::int};let S copy=s;return copy.n as i32;}','',7),
]
ABI_REJECTIONS=[
 ('c-int-implicit','fn i32 main(){let c::int n=7;return 0;}','type mismatch'),
 ('c-int-comparison','fn i32 main(){let c::int n=7 as c::int;assert(n==n);return 0;}','invalid scalar operator'),
 ('c-int-arithmetic','fn i32 main(){let c::int n=7 as c::int;let m=n+n;return 0;}','invalid scalar operator'),
 ('c-uint-cast','fn i32 main(){let u32 n=7;let m=n as c::uint;return 0;}','unsupported explicit cast'),
 ('c-bool-implicit','fn i32 main(){let c::bool n=true;return 0;}','type mismatch'),
 ('c-float-literal','fn i32 main(){let c::float n=7;return 0;}','type mismatch'),
 ('c-scalar-pointer','unsafe extern C fn i32 host(ptr c::int p);fn i32 main(){return 0;}','E5102'),
 ('c-ssize-target-pending','unsafe extern C fn c::ssize_t host();fn i32 main(){return 0;}','E5102'),
 ('c-unknown','unsafe extern C fn c::longdouble host();fn i32 main(){return 0;}','E5102'),
]
SLOT_CASES=[
 ('direct-output-slot','unsafe extern C fn i32 fixture_slot(ptr mut ptr mut void out);unsafe extern C fn i32 fixture_mut_opaque(ptr mut void p);unsafe extern C fn void fixture_clear_slot(ptr mut ptr mut void out);fn i32 main(){var ptr mut void p=null;unsafe{assert(fixture_slot(ref mut p as ptr mut ptr mut void)==7);assert(fixture_mut_opaque(p)==7);fixture_clear_slot(ref mut p as ptr mut ptr mut void);};assert(p==null);return 7;}','',7),
 ('borrowed-output-slot','unsafe extern C fn i32 fixture_slot(ptr mut ptr mut void out);unsafe extern C fn i32 fixture_mut_opaque(ptr mut void p);unsafe fn i32 fill(ref mut ptr mut void out){return fixture_slot(out as ptr mut ptr mut void);}fn i32 main(){var ptr mut void p=null;unsafe{assert(fill(ref mut p)==7);assert(fixture_mut_opaque(p)==7);};return 7;}','',7),
 ('slot-null','unsafe extern C fn ptr mut ptr mut void fixture_slot_identity(ptr mut ptr mut void p);fn i32 main(){let ptr mut ptr mut void p=null;assert(p==null);return 7;}','',7),
]
SLOT_REJECTIONS=[
 ('slot-shared-outer','unsafe extern C fn i32 host(ptr ptr mut void p);fn i32 main(){return 0;}','E5102'),
 ('slot-shared-inner','unsafe extern C fn i32 host(ptr mut ptr void p);fn i32 main(){return 0;}','E5102'),
 ('slot-other-pointee','unsafe extern C fn i32 host(ptr mut ptr mut u8 p);fn i32 main(){return 0;}','E5102'),
 ('slot-third-level','unsafe extern C fn i32 host(ptr mut ptr mut ptr mut void p);fn i32 main(){return 0;}','E5102'),
 ('slot-shared-borrow','fn i32 main(){var ptr mut void p=null;let q=unsafe{ref p as ptr mut ptr mut void};return 0;}','unsupported explicit cast'),
 ('slot-unsafe-required','fn i32 main(){var ptr mut void p=null;let q=ref mut p as ptr mut ptr mut void;return 0;}','E5101'),
 ('slot-tagged-ref','fn i32 main(){var ptr mut void p=null;let Option<ref mut ptr mut void> q=.some(ref mut p);return 0;}','error:'),
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
 ap=argparse.ArgumentParser();ap.add_argument('--images',nargs='+',required=True);ap.add_argument('--native-runner',required=True);ap.add_argument('--receipt',required=True);ap.add_argument('--cli-only',action='store_true');ap.add_argument('--lane',choices=['scalar','pointer','cast','abi','slot'],default='scalar');args=ap.parse_args()
 cases=SLOT_CASES if args.lane=='slot' else ABI_CASES if args.lane=='abi' else CAST_CASES if args.lane=='cast' else POINTER_CASES if args.lane=='pointer' else CASES;rejections=SLOT_REJECTIONS if args.lane=='slot' else ABI_REJECTIONS if args.lane=='abi' else CAST_REJECTIONS if args.lane=='cast' else POINTER_REJECTIONS if args.lane=='pointer' else REJECTIONS
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
