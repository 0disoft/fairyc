module tests::generic_source_declaration_json;
import generic_source_declaration_json as declarations;
import generic_source_documentation as docs;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import file_source as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;fn T shape<T>(T value){return value;}\n";}
    if(mode==1){return b"module sample;\n/// first\n  /// second\n@effects(console)\n@must_use\npub unsafe fn T shape<T,U>(T value,U other){return value;}\n";}
    if(mode==2){return b"\xef\xbb\xbfmodule sample;\r\n \t/// \xed\x95\x9c\xea\xb8\x80\r\n \t/// second\r\n \t@demo(\"x <&>\xe2\x80\xa8\",23,one :: two)\r\n pub fn ref mut (T[]) Holder::shape<T>(ref mut T[] this,usize count){let usize local=count;return this;}\r\n";}
    if(mode==3){return b"module sample;\n/// first\n\nfn i32 plain(unsafe extern C fn void() callback){return 0;}\n";}
    if(mode==4){return b"module sample;/// trailing\nfn i32 plain(){return 0;}\n";}
    if(mode==5){return b"module sample;\n/// not attached\n// ordinary\nfn T shape<T>(){return .none;}\n";}
    if(mode==6){return b"module sample;\n/// not attached\n/* gap */\nfn T shape<T>(){return .none;}\n";}
    if(mode==7){return b"module sample;\n//// start\n/// next\nfn T shape<T>(){return .none;}\n";}
    if(mode==8){return b"module sample;\n/** ordinary */\nfn T shape<T>(){return .none;}\n";}
    if(mode==9){return b"module sample;\n/// attached\n@meta(b\"hi\",c\"x\",\"\",9)\nfn T shape<T>(T value){return value;}\n";}
    if(mode==10){return b"module sample;\n/// attached\n@one :: two(foo :: bar)\nfn T shape<T>(){return .none;}\n";}
    return b"";
}
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};assert(syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)==2);return p;}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] value=ref path;return switch(file::manifest(value[..n])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
fn usize header(ref u8[] data){assert(data.len>=16 && bytes::equal(data[..8],b"FYASTHD1"));var usize at=8;let count=number(data,ref mut at);assert(count==11);return count;}
@test
@effects(filesystem,console)
fn void matches_real_go_function_headers_and_original_spans(){let fixture=fixture();let data=bytes::view(ref fixture);let count=header(data);var usize at=16;var usize mode=0;
    while(mode<count){let source=field(data,ref mut at);assert(bytes::equal(source,input(mode)));let expected=field(data,ref mut at);discard number(data,ref mut at);discard number(data,ref mut at);let p=parsed(source);var usize work=0;
        let actual=switch(declarations::header(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};
        if(!bytes::equal(bytes::view(ref actual),expected)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"header JSON mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b"\nactual: ");discard bytes::append(ref mut report,bytes::view(ref actual));discard bytes::append(ref mut report,b"\nexpected: ");discard bytes::append(ref mut report,expected);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}mode+=1;
    }assert(at==data.len);
}
@test
@effects(filesystem)
fn void documentation_matches_exact_go_attachment_and_breaks(){let fixture=fixture();let data=bytes::view(ref fixture);let count=header(data);var usize at=16;var usize mode=0;var usize attached=0;
    while(mode<count){let source=field(data,ref mut at);discard field(data,ref mut at);let start=number(data,ref mut at);let end=number(data,ref mut at);let p=parsed(source);var json::Ledger work=json::empty();let span=docs::declaration(ref p,1,ref mut work);assert(work.code==0 && span.start==start && span.end==end && span.present==(end>start));if(span.present){attached+=1;assert(bytes::equal(syntax::source_view(ref p)[span.start..span.start+3],b"///"));}let ignored=docs::declaration(ref p,0,ref mut work);assert(!ignored.present && ignored.start==0 && ignored.end==0 && work.code==0);mode+=1;}assert(at==data.len && attached==5);
}
@test
fn void rejects_nonfunction_invalid_index_and_preserves_failure(){let p=parsed(input(0));var usize work=0;assert(switch(declarations::header(ref p,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(declarations::header(ref p,2,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});var json::Ledger ledger=json::empty();let missing=docs::declaration(ref p,2,ref mut ledger);assert(!missing.present && ledger.code==9);var bytes::Buffer out=bytes::empty();assert(!declarations::function_fields(ref p,1,ref mut out,ref mut ledger) && ledger.code==9 && bytes::len(ref out)==0);}
@test
fn void header_honors_exact_shared_work_allowance(){let p=parsed(input(2));var usize cost=0;let original=switch(declarations::header(ref p,1,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let fit=switch(declarations::header(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref original),bytes::view(ref fit)));work=1048576-cost+1;assert(switch(declarations::header(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);}
