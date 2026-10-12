module tests::generic_source_function_definition;
import generic_source_function_json as functions;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import project_source_file as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;fn T shape<T>(T value){return value;}\n";}
    if(mode==1){return b"module sample;fn T shape<T>(T value){return value+value;}\n";}
    if(mode==2){return b"module sample;pub fn T shape<T>(T value){return value;}\n";}
    if(mode==3){return b"module sample;\n/// documented\nfn T shape<T>(T value){return value;}\n";}
    if(mode==4){return b"module sample;fn T shape<T>(T other){return other;}\n";}
    if(mode==5){return b"module sample;fn U shape<U>(U value){return value;}\n";}
    return b"";
}
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};assert(syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)==2);return p;}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;var usize at=0;var usize separator=0;while(at<length){if(path[at]==47 || path[at]==92){separator=at;}at+=1;}assert(separator>0 && separator+1<length);var usize meter=0;return switch(file::read(path[..separator],path[separator+1..length],1048576,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
@test
@effects(filesystem,console)
fn void matches_real_go_source_function_definitions(){let fixture=fixture();let data=bytes::view(ref fixture);assert(data.len>=16 && bytes::equal(data[..8],b"FYASTDG1"));var usize at=8;let count=number(data,ref mut at);assert(count==38);var usize mode=0;
    while(mode<count){let source=field(data,ref mut at);let expected_json=field(data,ref mut at);let expected=field(data,ref mut at);if(mode>=32){assert(bytes::equal(source,input(mode-32)));}let p=parsed(source);var usize work=0;let actual_json=switch(functions::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref actual_json),expected_json));work=0;let actual=switch(functions::definition(ref p,1,ref mut work)){case .ok(value):{value}case .err(code):{var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"function definition failure, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b" code ");discard bytes::append_u32(ref mut report,code);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);return;}};assert(bytes::len(ref actual)==64 && bytes::equal(bytes::view(ref actual),expected));mode+=1;
    }assert(at==data.len);
}
@test
fn void body_visibility_documentation_and_names_invalidate(){var usize work=0;let p=parsed(input(0));let baseline=switch(functions::definition(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};var usize mode=1;while(mode<6){let changed=parsed(input(mode));work=0;let actual=switch(functions::definition(ref changed,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(!bytes::equal(bytes::view(ref baseline),bytes::view(ref actual)));mode+=1;}}
@test
fn void definition_shares_json_and_hash_budget_and_rejects_selection(){let p=parsed(input(0));var usize cost=0;let original=switch(functions::definition(ref p,1,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};var usize json_cost=0;let encoded=switch(functions::encode(ref p,1,ref mut json_cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(bytes::len(ref encoded)>64 && cost>json_cost && cost<1048576);var usize work=1048576-cost;let fit=switch(functions::definition(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref original),bytes::view(ref fit)));work=1048576-cost+1;assert(switch(functions::definition(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);work=0;assert(switch(functions::definition(ref p,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(functions::definition(ref p,2,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});}
