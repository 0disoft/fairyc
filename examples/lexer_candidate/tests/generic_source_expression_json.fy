module tests::generic_source_expression_json;
import generic_source_expression_json as expressions;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import file_source as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;fn T shape<T>(){return call(0xff,0b101,1.25,true,false,null,\"x <&>\\u{2028}\",b\"\\xff\",c\"x\");}\n";}
    if(mode==1){return b"module sample;fn T shape<T>(){return pkg :: call(.first=one :: two,.second=(3 + -4) * 5);}\n";}
    if(mode==2){return b"module sample;fn T shape<T>(){return ref mut data[2].value;}\n";}
    if(mode==3){return b"module sample;fn T shape<T>(){return (ref data) as ptr mut T;}\n";}
    if(mode==4){return b"module sample;fn T shape<T>(){return [1,2 + 3,foo()];}\n";}
    if(mode==5){return b"module sample;fn T shape<T>(){return [];}\n";}
    if(mode==6){return b"module sample;fn T shape<T>(){return [foo();4];}\n";}
    if(mode==7){return b"module sample;fn T shape<T>(){return call(data[..],data[1..],data[..2],data[1..2]);}\n";}
    if(mode==8){return b"module sample;fn T shape<T>(){return pkg :: Item{.first,.second=foo(2)};}\n";}
    if(mode==9){return b"module sample;fn T shape<T>(){return .some(value)?;}\n";}
    if(mode==10){return b"module sample;fn T shape<T>(){return pkg::call<ref T[],Option<ptr mut u8>>(.value=ref data,.count=3);}\n";}
    if(mode==11){return b"module sample;fn T shape<T>(){return !((one << 2) | (two & 3)) && left != right;}\n";}
    if(mode==12){return b"\xef\xbb\xbfmodule sample;\r\nfn T shape<T>(){return pkg /* c */ :: call(\"\xed\x95\x9c\xea\xb8\x80\",c\"\xe2\x80\xa9\");}\r\n";}
    if(mode==13){return b"module sample;fn T shape<T>(){return .Variant{.x,.y=2};}\n";}
    if(mode==14){return b"module sample;fn T shape<T>(){return *ref value;}\n";}
    if(mode==15){return b"module sample;fn T shape<T>(){return pkg :: Item < T , ref u8[] > {.value};}\n";}
    return b"";
}
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};assert(syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)==2);return p;}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] value=ref path;return switch(file::manifest(value[..n])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
fn usize root(ref syntax::Parsed p){let id=syntax::declaration_at(p,1).expression;assert(id!=0);return id;}
@test
@effects(filesystem,console)
fn void matches_real_go_source_expressions(){let fixture=fixture();let data=bytes::view(ref fixture);assert(data.len>=16 && bytes::equal(data[..8],b"FYASTEX1"));var usize at=8;let count=number(data,ref mut at);assert(count==16);var usize mode=0;
    while(mode<count){let source=field(data,ref mut at);assert(bytes::equal(source,input(mode)));let expected=field(data,ref mut at);let p=parsed(source);var usize work=0;let actual=switch(expressions::encode(ref p,root(ref p),ref mut work)){case .ok(value):{value}case .err(code):{discard io::write_stderr("expression encoder failed\n");assert(false);return;}};
        if(!bytes::equal(bytes::view(ref actual),expected)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"expression JSON mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b"\nactual: ");discard bytes::append(ref mut report,bytes::view(ref actual));discard bytes::append(ref mut report,b"\nexpected: ");discard bytes::append(ref mut report,expected);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}mode+=1;
    }assert(at==data.len);
}
@test
fn void rejects_invalid_internal_and_unfinished_control_nodes(){let p=parsed(input(7));var usize work=0;assert(switch(expressions::encode(ref p,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(expressions::encode(ref p,syntax::expression_count(ref p)+1,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});var usize id=1;var usize checked=0;while(id<=syntax::expression_count(ref p)){let n=syntax::expression_at(ref p,id);if(n.kind==7 || n.kind==27){work=0;assert(switch(expressions::encode(ref p,id,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});checked+=1;}id+=1;}assert(checked>=8);let control=parsed(b"module sample;fn T shape<T>(){return {1};}");work=0;assert(switch(expressions::encode(ref control,root(ref control),ref mut work)){case .ok(_):{false}case .err(code):{code==4503}});}
@test
fn void expression_obeys_shared_work_depth_and_failure(){let p=parsed(input(10));let id=root(ref p);var usize cost=0;let original=switch(expressions::encode(ref p,id,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let fit=switch(expressions::encode(ref p,id,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref original),bytes::view(ref fit)));work=1048576-cost+1;assert(switch(expressions::encode(ref p,id,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);var json::Ledger ledger=json::empty();var bytes::Buffer out=bytes::empty();assert(!expressions::value(ref p,id,64,ref mut out,ref mut ledger) && ledger.code==10 && bytes::len(ref out)==0);ledger=json::Ledger{.used=0,.code=11};assert(!expressions::value(ref p,id,0,ref mut out,ref mut ledger) && ledger.code==11 && ledger.used==0 && bytes::len(ref out)==0);}
