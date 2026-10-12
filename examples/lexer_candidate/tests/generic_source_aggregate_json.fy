module tests::generic_source_aggregate_json;
import generic_source_aggregate_json as aggregates;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import project_source_file as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;struct Empty<T>{}\n";}
    if(mode==1){return b"module sample;pub struct Pair<T,U>{T first;U second;Option<T> maybe;T[4] values;}\n";}
    if(mode==2){return b"\xef\xbb\xbfmodule sample;\r\n/// \xed\x95\x9c\xea\xb8\x80\r\n@repr(C)\r\n@meta(\"x <&>\")\r\npub struct Pair<T>{pkg :: Thing < T > value;ref mut T[] view;}\r\n";}
    if(mode==3){return b"module sample;resource Bare<T>{T value;}\n";}
    if(mode==4){return b"module sample;resource Empty<T>{cleanup{}}\n";}
    if(mode==5){return b"module sample;resource Handle<T>{T value;@effects(console) cleanup{discard log(this.value);}}\n";}
    if(mode==6){return b"module sample;resource Looper<T>{T value;cleanup{for(let u8 item in data){discard item;}}}\n";}
    if(mode==7){return b"module sample;/// trailing\nenum Flag{On;Off;}\n";}
    if(mode==8){return b"module sample;\n/// choice\n@meta(one :: two)\npub enum Choice{None;Some{T value;};Pair{i32 left;ptr mut void raw;};Empty{};}\n";}
    if(mode==9){return b"module sample;resource Handle<T>{T value;cleanup{defer finish(3);var x=1;if(flag){x+=2;}return;}}\n";}
    if(mode==10){return b"module sample;struct Callback<T>{unsafe extern C fn i32(ptr void,ref mut T[]) callback;ptr mut T raw;}\n";}
    if(mode==11){return b"\xef\xbb\xbfmodule sample;\r\n/// resource\r\n@abilities(drop)\r\npub resource Pair<T,U>{T first;U second;@meta(b\"x\",c\"hi\",\"text\") cleanup{unsafe{discard finish();}}}\r\n";}
    return b"";
}
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};assert(syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)>=2);return p;}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;var usize at=0;var usize separator=0;while(at<length){if(path[at]==47 || path[at]==92){separator=at;}at+=1;}assert(separator>0 && separator+1<length);var usize meter=0;return switch(file::read(path[..separator],path[separator+1..length],1048576,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
@test
@effects(filesystem,console)
fn void matches_go_struct_resource_enum_and_real_sdk_vec(){let fixture=fixture();let data=bytes::view(ref fixture);assert(data.len>=16 && bytes::equal(data[..8],b"FYASTAG1"));var usize at=8;let count=number(data,ref mut at);assert(count==13);var usize mode=0;var usize definitions=0;
    while(mode<count){let source=field(data,ref mut at);if(mode<12){assert(bytes::equal(source,input(mode)));}let expected=field(data,ref mut at);let expected_template=field(data,ref mut at);let expected_digest=field(data,ref mut at);let p=parsed(source);var usize work=0;
        let actual=switch(aggregates::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(code):{var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"aggregate encode failure, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b" code ");discard bytes::append_u32(ref mut report,code);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);return;}};
        if(!bytes::equal(bytes::view(ref actual),expected)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"aggregate JSON mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b"\nactual: ");discard bytes::append(ref mut report,bytes::view(ref actual));discard bytes::append(ref mut report,b"\nexpected: ");discard bytes::append(ref mut report,expected);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}
        if(expected_template.len!=0){work=0;let wrapped=switch(aggregates::template(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref wrapped),expected_template));work=0;let digest=switch(aggregates::definition(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(bytes::len(ref digest)==64 && bytes::equal(bytes::view(ref digest),expected_digest));definitions+=1;}
        if(mode==12){let d=syntax::declaration_at(ref p,1);assert(d.kind==6 && d.public && d.field_count==6 && d.generic_parameters!=0 && d.cleanup_body!=0 && bytes::equal(syntax::source_view(ref p)[d.name_start..d.name_end],b"Vec"));}mode+=1;
    }assert(at==data.len && definitions==11);
}
@test
fn void field_cleanup_and_type_parameter_changes_invalidate(){let p=parsed(b"module sample;resource Hold<T>{T value;cleanup{discard first();}}");var usize work=0;let baseline=switch(aggregates::definition(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};var usize mode=0;while(mode<3){let source=if(mode==0){b"module sample;resource Hold<T>{T second;cleanup{discard first();}}"}else if(mode==1){b"module sample;resource Hold<T>{T value;cleanup{discard other();}}"}else{b"module sample;resource Hold<U>{U value;cleanup{discard first();}}"};let changed=parsed(source);work=0;let actual=switch(aggregates::definition(ref changed,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(!bytes::equal(bytes::view(ref actual),bytes::view(ref baseline)));mode+=1;}}
@test
fn void aggregate_invalid_selection_and_exact_combined_work(){let p=parsed(input(6));var usize cost=0;let original=switch(aggregates::definition(ref p,1,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let fit=switch(aggregates::definition(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref fit),bytes::view(ref original)));work=1048576-cost+1;assert(switch(aggregates::definition(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);work=0;assert(switch(aggregates::encode(ref p,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(aggregates::encode(ref p,2,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});let enumeration=parsed(input(8));work=0;assert(switch(aggregates::definition(ref enumeration,1,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});var json::Ledger ledger=json::Ledger{.used=0,.code=11};var bytes::Buffer out=bytes::empty();assert(!aggregates::value(ref p,1,ref mut out,ref mut ledger) && ledger.code==11 && ledger.used==0 && bytes::len(ref out)==0);}
