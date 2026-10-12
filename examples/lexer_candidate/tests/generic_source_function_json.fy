module tests::generic_source_function_json;
import generic_source_function_json as functions;
import generic_source_expression_json as expressions;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import file_source as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;fn void shape<T>(){}\n";}
    if(mode==1){return b"module sample;fn void shape<T>(){let T a=one;var b=2;a=next();b+=1;assert(b>0);defer finish(b);discard take();burn a;return;}\n";}
    if(mode==2){return b"module sample;fn T shape<T>(){return {let x=one;x+2};}\n";}
    if(mode==3){return b"module sample;fn T shape<T>(){let x=if(flag){one}else if(other){two}else{three};return x;}\n";}
    if(mode==4){return b"module sample;fn void shape<T>(){if(flag){discard foo();}else{discard bar();}unsafe {discard foreign();}{discard last();}}\n";}
    if(mode==5){return b"module sample;fn T shape<T>(){while(flag){if(other){break;}continue;}return one;}\n";}
    if(mode==6){return b"module sample;fn void shape<T>(){for(;;){break;}}\n";}
    if(mode==7){return b"module sample;fn T shape<T>(){for(var usize i=0 /* gap */;i<4;i+=1){discard call(i);}return one;}\n";}
    if(mode==8){return b"module sample;fn T shape<T>(){for(index=0;index<4;index=next(index)){continue;}return one;}\n";}
    if(mode==9){return b"module sample;fn T shape<T>(){region arena in pool {let x=place(ref arena);discard x;}return one;}\n";}
    if(mode==10){return b"module sample;fn T shape<T>(){return switch(value){case .some(x):{x}case .none:{one}};}\n";}
    if(mode==11){return b"module sample;fn T shape<T>(){return switch(value){case .Pair{.first=x,.second=_}:{x}case .Empty{}:{one}};}\n";}
    if(mode==12){return b"module sample;fn T shape<T>(){return if(flag){let x=foo<T>(.value=one);x}else{bar()};}\n";}
    if(mode==13){return b"module sample;fn T shape<T>(){return unsafe {call()};}\n";}
    if(mode==14){return b"module sample;fn T shape<T>(){return if(flag){one};}\n";}
    if(mode==15){return b"\xef\xbb\xbfmodule sample;\r\n/// doc\r\n@meta(\"\xed\x95\x9c\xea\xb8\x80\")\r\npub unsafe fn ref T Holder::shape<T>(ref T this){return ref *this;}\r\n";}
    return b"";
}
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};assert(syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)==2);return p;}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] value=ref path;return switch(file::manifest(value[..n])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
@test
@effects(filesystem,console)
fn void matches_real_go_functions_with_blocks_and_controls(){let fixture=fixture();let data=bytes::view(ref fixture);assert(data.len>=16 && bytes::equal(data[..8],b"FYASTFN1"));var usize at=8;let count=number(data,ref mut at);assert(count==16);var usize mode=0;
    while(mode<count){let source=field(data,ref mut at);assert(bytes::equal(source,input(mode)));let expected=field(data,ref mut at);let p=parsed(source);var usize work=0;let actual=switch(functions::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(code):{var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"function encoder failure, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b" code ");discard bytes::append_u32(ref mut report,code);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);return;}};
        if(!bytes::equal(bytes::view(ref actual),expected)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"function JSON mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b"\nactual: ");discard bytes::append(ref mut report,bytes::view(ref actual));discard bytes::append(ref mut report,b"\nexpected: ");discard bytes::append(ref mut report,expected);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}mode+=1;
    }assert(at==data.len);
}
@test
fn void rejects_invalid_nodes_and_unfinished_byte_loop(){let p=parsed(input(7));var usize work=0;assert(switch(functions::encode(ref p,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(functions::encode(ref p,2,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});var json::Ledger ledger=json::empty();var bytes::Buffer out=bytes::empty();assert(!expressions::block(ref p,0,0,ref mut out,ref mut ledger) && ledger.code==9 && bytes::len(ref out)==0);var usize id=1;var usize checked=0;while(id<=syntax::statement_count(ref p)){let n=syntax::statement_at(ref p,id);if(n.kind==7 || n.kind==15){ledger=json::empty();assert(!expressions::statement(ref p,id,0,false,ref mut out,ref mut ledger) && ledger.code==9);checked+=1;}id+=1;}assert(checked>0);let loop=parsed(b"module sample;fn void shape<T>(){for(let u8 byte in data){discard byte;}}");work=0;assert(switch(functions::encode(ref loop,1,ref mut work)){case .ok(_):{false}case .err(code):{code==4503}});}
@test
fn void full_function_retains_shared_work_depth_and_failure(){let p=parsed(input(11));var usize cost=0;let original=switch(functions::encode(ref p,1,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let fit=switch(functions::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref original),bytes::view(ref fit)));work=1048576-cost+1;assert(switch(functions::encode(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);var json::Ledger ledger=json::empty();var bytes::Buffer out=bytes::empty();let body=syntax::declaration_at(ref p,1).body;assert(!expressions::block(ref p,body,64,ref mut out,ref mut ledger) && ledger.code==10 && bytes::len(ref out)==0);ledger=json::Ledger{.used=0,.code=11};assert(!expressions::block(ref p,body,0,ref mut out,ref mut ledger) && ledger.code==11 && ledger.used==0 && bytes::len(ref out)==0);}
