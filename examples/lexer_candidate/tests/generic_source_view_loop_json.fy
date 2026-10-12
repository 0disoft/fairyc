module tests::generic_source_view_loop_json;
import generic_source_function_json as functions;
import generic_source_view_loop_json as loops;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import project_source_file as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;fn void shape<T>(){for(let u8 byte in data){discard byte;}}\n";}
    if(mode==1){return b"module sample;fn void shape<T>(){for(let ref mut Item item in holder.items){discard item;}}\n";}
    if(mode==2){return b"module sample;fn void shape<T>(){for(let ref u8 item in data){discard item;}}\n";}
    if(mode==3){return b"module sample;fn void shape<T>(){for(let u8 item in (data)){discard item;}}\n";}
    if(mode==4){return b"module sample;fn void shape<T>(){for(let u8 item in data[0]){discard item;}}\n";}
    if(mode==5){return b"module sample;fn void shape<T>(){for(let u8 item in fetch()){discard item;}}\n";}
    if(mode==6){return b"module sample;fn void shape<T>(){for(let u8 item in (holder).data){discard item;}}\n";}
    if(mode==7){return b"module sample;fn void shape<T>(){for(let u8 item in holder.data[1..]){discard item;}}\n";}
    if(mode==8){return b"module sample;fn void shape<T>(){for(let u8 outer in data){for(let u8 inner in data[..]){discard inner;}discard outer;}}\n";}
    if(mode==9){return b"module sample;fn void shape<T>(){for(let ptr u8 item in data){discard item;}}\n";}
    if(mode==10){return b"module sample;fn void shape<T>(){for(let Item<T> item in data){discard item;}}\n";}
    if(mode==11){return b"module sample;fn void shape<T>(){for(let u8[4] item in data){discard item;}}\n";}
    if(mode==12){return b"module sample;fn void shape<T>(){for(let ref mut ptr u8 item in data){discard item;}}\n";}
    if(mode==13){return b"module sample;fn void shape<T>(){for(let u8 item in (holder.data)){discard item;}}\n";}
    if(mode==14){return b"module sample;fn void shape<T>(){for(let u8 item in data[..]){}}\n";}
    if(mode==15){return b"\xef\xbb\xbfmodule sample;\r\n/// doc \xed\x95\x9c\xea\xb8\x80\r\nfn void shape<T>(){for(let u8 item in bundle.items[..2]){if(flag){continue;}break;}}\r\n";}
    return b"";
}
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};assert(syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)==2);return p;}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;var usize at=0;var usize separator=0;while(at<length){if(path[at]==47 || path[at]==92){separator=at;}at+=1;}assert(separator>0 && separator+1<length);var usize meter=0;return switch(file::read(path[..separator],path[separator+1..length],1048576,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
fn syntax::Statement loop(ref syntax::Parsed p){let body=syntax::declaration_at(p,1).body;assert(syntax::block_statement_count(p,body)==1);let s=syntax::statement_at(p,syntax::block_statement(p,body,0));assert(s.kind==18);return s;}
@test
@effects(filesystem,console)
fn void matches_go_written_and_synthetic_view_loops(){let fixture=fixture();let data=bytes::view(ref fixture);assert(data.len>=16 && bytes::equal(data[..8],b"FYASTVF1"));var usize at=8;let count=number(data,ref mut at);assert(count==16);var usize mode=0;var usize prepared=0;
    while(mode<count){let source=field(data,ref mut at);assert(bytes::equal(source,input(mode)));let expected=field(data,ref mut at);let synthetic=number(data,ref mut at);let p=parsed(source);var json::Ledger ledger=json::empty();let admitted=loops::admitted(ref p,loop(ref p),ref mut ledger);if(ledger.code!=0 || admitted!=(synthetic==1)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"view loop preparation mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b" code ");discard bytes::append_u32(ref mut report,ledger.code);discard bytes::append(ref mut report,b" actual ");discard bytes::append_bool(ref mut report,admitted);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}if(admitted){prepared+=1;}var usize work=0;let actual=switch(functions::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(code):{var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"view loop encoder failure, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b" code ");discard bytes::append_u32(ref mut report,code);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);return;}};
        if(!bytes::equal(bytes::view(ref actual),expected)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"view loop JSON mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b"\nactual: ");discard bytes::append(ref mut report,bytes::view(ref actual));discard bytes::append(ref mut report,b"\nexpected: ");discard bytes::append(ref mut report,expected);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}mode+=1;
    }assert(at==data.len && prepared==8);
}
@test
fn void malformed_loop_facts_and_failure_reject(){let p=parsed(input(0));let original=loop(ref p);var s=original;var json::Ledger work=json::empty();s.target=0;assert(!loops::valid(ref p,s,ref mut work) && work.code==9);s=original;s.expression=syntax::expression_count(ref p)+1;work=json::empty();assert(!loops::admitted(ref p,s,ref mut work) && work.code==9);s=original;s.first=0;work=json::empty();assert(!loops::valid(ref p,s,ref mut work) && work.code==9);s=original;s.second=syntax::type_count(ref p)+1;work=json::empty();var bytes::Buffer out=bytes::empty();assert(!loops::synthetic(ref p,s,b"null",b"null",ref mut out,ref mut work) && work.code==9 && bytes::len(ref out)==0);work=json::Ledger{.used=0,.code=11};assert(!loops::admitted(ref p,original,ref mut work) && work.code==11 && work.used==0);}
@test
fn void nested_view_loop_obeys_exact_shared_work(){let p=parsed(input(8));var usize cost=0;let original=switch(functions::encode(ref p,1,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let fit=switch(functions::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref original),bytes::view(ref fit)));work=1048576-cost+1;assert(switch(functions::encode(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);}
