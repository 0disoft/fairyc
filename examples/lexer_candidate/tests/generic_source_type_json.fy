module tests::generic_source_type_json;
import generic_source_type_json as types;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import file_source as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;fn T shape<T>(T value){return value;}\n";}
    if(mode==1){return b"module sample;fn Result<Option<T>,ptr mut u8> shape<T>(ref mut T[] values,T[4] array,(pkg::Thing) grouped,unsafe extern C fn i32(ptr void,i64) callback){return .err(null);}\n";}
    if(mode==2){return b"module sample;\r\n// Korean: \xed\x95\x9c\xea\xb8\x80\r\nfn ref (T[Size]) shape<T>(ref T[2][3] data,ptr ptr mut void raw){return ref data;}\r\n";}
    if(mode==3){return b"module sample;fn unsafe extern C fn void() shape<T>(pkg :: Thing < T , ref u8[] > value){return value;}\n";}
    if(mode==4){return b"module sample;\nfn T shape<T>(ref /* comment */ mut T value){return value;}\n";}
    return b"module sample;fn T shape<T>(){return .none;}\n";
}
fn syntax::Parsed parsed(ref u8[] source){
    let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};
    let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};
    let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};
    assert(syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)==2);return p;
}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] value=ref path;return switch(file::manifest(value[..n])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
@test
@effects(filesystem,console)
fn void matches_real_go_ast_types_from_original_source(){
    let fixture=fixture();let data=bytes::view(ref fixture);assert(bytes::equal(data[..8],b"FYASTYP1"));var usize at=8;let count=number(data,ref mut at);assert(count==6);var usize mode=0;
    while(mode<count){let source=field(data,ref mut at);assert(bytes::equal(source,input(mode)));let p=parsed(source);let d=syntax::declaration_at(ref p,1);var usize parameters=0;var usize entry=0;while(entry<d.entry_count){if(syntax::entry_at(ref p,d.entry_start+entry).kind==1){parameters+=1;}entry+=1;}let types_count=number(data,ref mut at);assert(types_count==parameters+1);var usize i=0;
        while(i<types_count){let expected=field(data,ref mut at);let root=if(i==0){d.result_type}else{syntax::entry_at(ref p,d.entry_start+i-1).type_root};var usize work=0;
            let actual=switch(types::encode(ref p,root,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};
            if(!bytes::equal(bytes::view(ref actual),expected)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"type JSON mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b", type ");discard bytes::append_usize(ref mut report,i);discard bytes::append(ref mut report,b"\nactual: ");discard bytes::append(ref mut report,bytes::view(ref actual));discard bytes::append(ref mut report,b"\nexpected: ");discard bytes::append(ref mut report,expected);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}i+=1;}
        mode+=1;
    }assert(at==data.len);
}
@test
fn void rejects_absent_internal_and_out_of_range_type_nodes(){let p=parsed(input(1));var usize work=0;assert(switch(types::encode(ref p,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(types::encode(ref p,syntax::type_count(ref p)+1,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});var usize i=1;var usize rejected=0;while(i<=syntax::type_count(ref p)){let n=syntax::type_at(ref p,i);if(n.kind==7 || n.kind==9 || n.kind==13 || n.kind==14 || n.kind==16){work=0;assert(switch(types::encode(ref p,i,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});rejected+=1;}i+=1;}assert(rejected>5);}
@test
fn void exact_shared_work_limit_and_depth_limit(){let p=parsed(input(1));let d=syntax::declaration_at(ref p,1);var usize cost=0;let original=switch(types::encode(ref p,d.result_type,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let fit=switch(types::encode(ref p,d.result_type,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref original),bytes::view(ref fit)));work=1048576-cost+1;assert(switch(types::encode(ref p,d.result_type,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);var json::Ledger ledger=json::empty();var bytes::Buffer out=bytes::empty();assert(!types::value(ref p,d.result_type,64,ref mut out,ref mut ledger) && ledger.code==10 && bytes::len(ref out)==0);}
@test
fn void json_writer_rejects_invalid_utf8_and_preserves_first_failure(){var json::Ledger ledger=json::empty();var bytes::Buffer out=bytes::empty();var u8[1] invalid=[255];assert(!json::quoted(ref mut out,ref invalid,ref mut ledger) && ledger.code==9 && bytes::len(ref out)==0);assert(!json::raw(ref mut out,b"ignored",ref mut ledger) && ledger.code==9 && bytes::len(ref out)==0);ledger=json::Ledger{.used=1048576,.code=0};assert(!json::number(ref mut out,0,ref mut ledger) && ledger.code==10 && bytes::len(ref out)==0);}
