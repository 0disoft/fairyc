module tests::generic_source_item_json;
import generic_source_item_json as items;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import project_source_file as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn ref u8[] input(usize mode){
    if(mode==0){return b"module sample;import pkg as p;\n";}
    if(mode==1){return b"\xef\xbb\xbfmodule sample;\r\nimport pkg /* path */ :: deep\r\n :: leaf /* alias */ as selected;\r\n";}
    if(mode==2){return b"module sample;const u8 amount=255;\n";}
    if(mode==3){return b"module sample;\n/// unattached constant\n@meta(\"<&>\")\npub const i64 LIMIT=-0x80;\n";}
    if(mode==4){return b"module sample;const ref str MESSAGE=\"\xed\x95\x9c\xea\xb8\x80 <&>\";\n";}
    if(mode==5){return b"module sample;static_assert(true);\n";}
    if(mode==6){return b"module sample;@meta(one :: two) static_assert(((LIMIT as i64)+3)>2);\n";}
    if(mode==7){return b"module sample;@contract_uid(\"sample.empty\") state{}\n";}
    if(mode==8){return b"module sample;@contract_uid(\"sample.storage\") state{cell u64 counter;map bytes20 u64 balances;}\n";}
    if(mode==9){return b"\xef\xbb\xbfmodule sample;\r\n@meta(b\"x\",c\"hi\",\"text\")\r\npub state{cell ref mut pkg :: Type [] view;map u32 Option<i32> entries;}\r\n";}
    if(mode==10){return b"module sample;unsafe extern C fn void close();\n";}
    if(mode==11){return b"module sample;\n/// external\n@effects(console)\n@link_name(\"external_call\")\npub unsafe extern C fn i64 call(i32 value,ptr mut void output,ref u8[] data);\n";}
    if(mode==12){return b"\xef\xbb\xbfmodule sample;\r\n/// foreign \xed\x95\x9c\xea\xb8\x80\r\n@meta(\"a\\n<&>\")\r\nunsafe extern C fn ptr void load(unsafe extern C fn i32(ptr void) callback,ref mut u8[] output);\r\n";}
    return b"";
}
fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(source);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(source);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(source);}};assert(syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)>=2);return p;}
@effects(filesystem)
fn bytes::Buffer fixture(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;var usize at=0;var usize separator=0;while(at<length){if(path[at]==47 || path[at]==92){separator=at;}at+=1;}assert(separator>0 && separator+1<length);var usize meter=0;return switch(file::read(path[..separator],path[separator+1..length],1048576,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] data,ref mut usize at){assert(*at<=data.len && data.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(data[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] data,ref mut usize at){let n=number(data,at);assert(*at<=data.len && n<=data.len-*at);let first=*at;*at+=n;return data[first..*at];}
@test
@effects(filesystem,console)
fn void matches_go_import_const_assert_state_and_foreign(){let fixture=fixture();let data=bytes::view(ref fixture);assert(data.len>=16 && bytes::equal(data[..8],b"FYASTIT1"));var usize at=8;let count=number(data,ref mut at);assert(count==13);var usize mode=0;
    while(mode<count){let source=field(data,ref mut at);assert(bytes::equal(source,input(mode)));let expected=field(data,ref mut at);let p=parsed(source);var usize work=0;
        let actual=switch(items::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(code):{var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"item encode failure, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b" code ");discard bytes::append_u32(ref mut report,code);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);return;}};
        if(!bytes::equal(bytes::view(ref actual),expected)){var bytes::Buffer report=bytes::empty();discard bytes::append(ref mut report,b"item JSON mismatch, source ");discard bytes::append_usize(ref mut report,mode);discard bytes::append(ref mut report,b"\nactual: ");discard bytes::append(ref mut report,bytes::view(ref actual));discard bytes::append(ref mut report,b"\nexpected: ");discard bytes::append(ref mut report,expected);discard bytes::append(ref mut report,b"\n");discard io::write_stderr_bytes(bytes::view(ref report));assert(false);}mode+=1;
    }assert(at==data.len);
}
@test
fn void field_alias_expression_and_foreign_signature_changes_are_retained(){var usize mode=0;while(mode<4){let a=if(mode==0){b"module sample;import pkg as first;"}else if(mode==1){b"module sample;const i32 LIMIT=3;"}else if(mode==2){b"module sample;state{cell u64 first;}"}else{b"module sample;unsafe extern C fn i32 load(i32 value);"};let b=if(mode==0){b"module sample;import pkg as other;"}else if(mode==1){b"module sample;const i32 LIMIT=4;"}else if(mode==2){b"module sample;state{cell u64 other;}"}else{b"module sample;unsafe extern C fn i64 load(i32 value);"};let p=parsed(a);let q=parsed(b);var usize work=0;let left=switch(items::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};work=0;let right=switch(items::encode(ref q,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(!bytes::equal(bytes::view(ref left),bytes::view(ref right)));mode+=1;}}
@test
fn void item_invalid_selection_and_exact_work(){let p=parsed(input(8));var usize cost=0;let original=switch(items::encode(ref p,1,ref mut cost)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let fit=switch(items::encode(ref p,1,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref fit),bytes::view(ref original)));work=1048576-cost+1;assert(switch(items::encode(ref p,1,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);work=0;assert(switch(items::encode(ref p,0,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});work=0;assert(switch(items::encode(ref p,2,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});let function=parsed(b"module sample;fn void sample(){}");work=0;assert(switch(items::encode(ref function,1,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});var json::Ledger ledger=json::Ledger{.used=0,.code=11};var bytes::Buffer out=bytes::empty();assert(!items::value(ref p,1,ref mut out,ref mut ledger) && ledger.code==11 && ledger.used==0 && bytes::len(ref out)==0);}
