module tests::generic_source_module_json;
import generic_source_module_json as modules;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import project_source_file as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;\n";}
    if(mode==1){return b"module sample;fn void f(){let x=3;var i32 y=4;discard x+y;}\n";}
    if(mode==2){return b"module sample;fn i32 f(){let outer={let inner=2;inner+3};return outer;}\n";}
    if(mode==3){return b"module sample;fn void f(){for(var usize i=0;i<3;i+=1){let u8[2] values=[4,5];discard values;}}\n";}
    if(mode==4){return b"module sample;fn void f(){for(let u8 item in data[1..3]){let value=item;discard value;}}\n";}
    if(mode==5){return b"module sample;fn void f(){region r in arena{let x=9;discard x;}unsafe{var x=null;discard x;}}\n";}
    if(mode==6){return b"module sample;struct S{u8[2][3] values;}enum E{A{i32[4] items;};B;}state{cell u64 count;map bytes20 u64 entries;}\n";}
    if(mode==7){return b"module sample;@meta(123,\"ignored\",b\"ignored\") const i32 LIMIT=7;static_assert(LIMIT==7);\n";}
    if(mode==8){return b"module sample;fn void f(){discard true;discard false;discard null;discard 0xBEEF;discard 0b101;discard 1_000;discard 1.25e+2;discard \"\xed\x95\x9c\xea\xb8\x80 <&>\";discard b\"x\\x00\";discard c\"hi\";}\n";}
    if(mode==9){return b"module sample;resource Handle{u8[3] field;cleanup{let x=1;for(var i=0;i<2;i+=1){let y=3;discard y;}}}\n";}
    if(mode==10){return b"module sample;fn i32 f(i32 x){let result=switch(x){case .Some(value):{let y=2;y}case .None:{3}};return result;}\n";}
    if(mode==11){return b"\xef\xbb\xbfmodule sample;\r\nfn void f(){let u8[4] a=[0;4];if(true){let x=1;}else{let x=2;}while(false){let z=3;}}\r\n";}
    if(mode==12){return b"module sample;fn u8[3] f(u8[2] a,unsafe extern C fn i32[5](u8[4]) cb){let i32[6] local=[0;6];return [1;3];}\n";}
    if(mode==13){return b"module sample;unsafe extern C fn i32[8] load(unsafe extern C fn i32[7](u8[6],unsafe extern C fn u8[5](u8[4])) cb,u8[3] data);\n";}
    if(mode==14){return b"module sample;import pkg :: deep as p;@meta(\"<&>\") const i32 LIMIT=7;static_assert(LIMIT>0);@contract_uid(\"sample.storage\") state{cell u64 counter;map bytes20 u64 balances;}pub struct Packet{u8[2] data;}pub enum E{A;B{i32 value;};}resource Owned{i32 value;@effects(console) cleanup{discard log(this.value);}}struct Generic<T>{T value;}@link_name(\"external\") unsafe extern C fn i32 raw(ptr void data);@meta(3) pub fn T helper<T>(T value){return value;}fn i32 main(){let Packet packet=Packet{.data=[1,2]};return helper<i32>(LIMIT);}\n";}
    if(mode==15){return b"\xef\xbb\xbf// heading\r\nmodule sample;\r\n/// record \xed\x95\x9c\xea\xb8\x80\r\n@repr(C) pub struct Pair{i32 first;i64 second;}\r\nfn i32 f(){let x=1;return x;}\r\n// trailing\r\n";}
    if(mode==16){return b"module sample;\n\n";}
    return b"";
}
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};assert(syntax::declaration_count(ref p)>=1);return p;}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;var usize at=0;var usize separator=0;while(at<length){if(path[at]==47 || path[at]==92){separator=at;}at+=1;}assert(separator>0 && separator+1<length);var usize meter=0;return switch(file::read(path[..separator],path[separator+1..length],1048576,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
@test
@effects(filesystem,console)
fn void matches_go_complete_modules_and_nominal_definitions(){let fixture=fixture();let data=bytes::view(ref fixture);assert(data.len>=16 && bytes::equal(data[..8],b"FYASTMD1"));var usize at=8;let count=number(data,ref mut at);assert(count==18);var usize mode=0;var usize definitions=0;
    while(mode<count){let source=field(data,ref mut at);if(mode<17){assert(bytes::equal(source,input(mode)));}let expected=field(data,ref mut at);let p=parsed(source);assert(syntax::problem_count(ref p)==0);var usize work=0;
        let actual=switch(modules::encode(ref p,ref mut work)){case .ok(value):{value}case .err(code):{var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"module encode failure, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b" code ");discard bytes::append_u32(ref mut report,code);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);return;}};
        if(!bytes::equal(bytes::view(ref actual),expected)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"module JSON mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b"\nactual: ");discard bytes::append(ref mut report,bytes::view(ref actual));discard bytes::append(ref mut report,b"\nexpected: ");discard bytes::append(ref mut report,expected);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}
        let nominal_count=number(data,ref mut at);var usize n=0;while(n<nominal_count){let name=field(data,ref mut at);let digest=field(data,ref mut at);var usize index=1;while(index<syntax::declaration_count(ref p)){let d=syntax::declaration_at(ref p,index);if(bytes::equal(syntax::source_view(ref p)[d.name_start..d.name_end],name)){break;}index+=1;}assert(index<syntax::declaration_count(ref p));work=0;let actual_digest=switch(modules::nominal_definition(ref p,index,ref mut work)){case .ok(value):{value}case .err(code):{var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"nominal definition failure, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b" code ");discard bytes::append_u32(ref mut report,code);discard io::write_stderr_bytes(bytes::view(ref report));assert(false);return;}};assert(bytes::equal(bytes::view(ref actual_digest),digest));definitions+=1;n+=1;}
        mode+=1;
    }assert(at==data.len && definitions==5);
}
@test
fn void nominal_module_hash_retains_helpers_fields_and_geometry(){let p=parsed(b"module sample;struct S{i32 value;}fn i32 helper(){return 1;}\n");var usize work=0;let baseline=switch(modules::nominal_definition(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};var usize mode=0;while(mode<3){let source=if(mode==0){b"module sample;struct S{i64 value;}fn i32 helper(){return 1;}\n"}else if(mode==1){b"module sample;struct S{i32 value;}fn i32 helper(){return 2;}\n"}else{b"module sample;struct S{i32 value;}fn i32 helper(){return 1;}\r\n"};let changed=parsed(source);work=0;let actual=switch(modules::nominal_definition(ref changed,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(!bytes::equal(bytes::view(ref actual),bytes::view(ref baseline)));mode+=1;}}
@test
fn void module_exact_work_invalid_source_and_selection(){let p=parsed(input(9));var usize cost=0;let original=switch(modules::nominal_definition(ref p,1,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let fit=switch(modules::nominal_definition(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref fit),bytes::view(ref original)));work=1048576-cost+1;assert(switch(modules::nominal_definition(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);work=0;assert(switch(modules::nominal_definition(ref p,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(modules::nominal_definition(ref p,2,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});let generic=parsed(b"module sample;struct S<T>{T value;}");work=0;assert(switch(modules::nominal_definition(ref generic,1,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});let invalid=parsed(b"module sample;const i32 bad=;");assert(syntax::problem_count(ref invalid)>0);work=0;assert(switch(modules::encode(ref invalid,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});var json::Ledger ledger=json::Ledger{.used=0,.code=11};var bytes::Buffer out=bytes::empty();assert(!modules::value(ref p,ref mut out,ref mut ledger) && ledger.code==11 && ledger.used==0 && bytes::len(ref out)==0);}
