module tests::closure_value_types;
import syntax as syntax;
import syntax_parse_driver as raw_parser;
import types as nodes;
import generic_source_type_json as encoded;
import generic_source_json as json;
import module_types as values;
import module_catalog as catalog;
import scanner as scanner;
import file_source as file;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
import std::vec as v;

fn scanner::Stream scanned(ref u8[] raw){let owned=switch(bytes::copy(raw)){case .ok(b):{b}case .err(_):{assert(false);return scanned(raw);}};return switch(scanner::collect(owned,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);scanned(raw)}};}
// Raw retention deliberately precedes the separate direct-closure lowering gate.
fn syntax::Parsed retained(ref u8[] raw){return switch(raw_parser::parse_diagnostics(scanned(raw),512)){case .ok(p):{p}case .err(_):{assert(false);retained(raw)}};}
@effects(filesystem) fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(file::manifest(view[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] wire,ref mut usize at){assert(*at<=wire.len && wire.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(wire[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] wire,ref mut usize at){let n=number(wire,at);assert(*at<=wire.len && n<=wire.len-*at);let first=*at;*at+=n;return wire[first..*at];}
fn usize selected(ref syntax::Parsed p,usize kind,usize start,usize end){var usize i=1;var usize found=0;while(i<=syntax::type_count(p)){let n=syntax::type_at(p,i);if((n.kind as usize)==kind && n.start==start && n.end==end){assert(found==0);found=i;}i+=1;}assert(found!=0);return found;}
fn void signature(ref syntax::Parsed p,usize id,ref u8[] wire,ref mut usize at){let n=syntax::type_at(p,id);assert(n.left!=0 && n.left<id);let result=syntax::type_at(p,n.left);assert(result.start==number(wire,at) && result.end==number(wire,at));let expected=number(wire,at);var usize count=0;var usize link=n.right;while(link!=0){let parameter=syntax::type_at(p,link);assert(parameter.kind==16 && parameter.left!=0 && parameter.left<link && parameter.right<link);count+=1;link=parameter.right;}assert(count==expected);var usize i=0;while(i<count){link=n.right;var usize remaining=count-1;while(remaining>i){link=syntax::type_at(p,link).right;remaining-=1;}let parameter=syntax::type_at(p,syntax::type_at(p,link).left);assert(parameter.start==number(wire,at) && parameter.end==number(wire,at));i+=1;}}
@test @effects(filesystem,console) fn void closure_types_match_original_go_structure_and_json(){
    let fixture=fixture();let wire=bytes::view(ref fixture);assert(wire.len>=16 && bytes::equal(wire[..8],b"FYCLTYP1"));var usize at=8;let cases=number(wire,ref mut at);assert(cases>=8);var usize c=0;var usize closures=0;var usize foreign=0;
    while(c<cases){let raw=field(wire,ref mut at);let p=retained(raw);assert(syntax::problem_count(ref p)==0);let count=number(wire,ref mut at);var usize i=0;while(i<count){let kind=number(wire,ref mut at);let start=number(wire,ref mut at);let end=number(wire,ref mut at);let expected=field(wire,ref mut at);let id=selected(ref p,kind,start,end);var usize work=0;let actual=switch(encoded::encode(ref p,id,ref mut work)){case .ok(b):{b}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};
        if(!bytes::equal(bytes::view(ref actual),expected)){discard io::write_stderr("type JSON mismatch\nactual: ");discard io::write_stderr_bytes(bytes::view(ref actual));discard io::write_stderr("\nexpected: ");discard io::write_stderr_bytes(expected);assert(false);}
        if(kind==17 || kind==15){signature(ref p,id,wire,ref mut at);if(kind==17){closures+=1;}else{foreign+=1;}}i+=1;}c+=1;
    }
    assert(closures>10 && foreign>=3);let invalid=number(wire,ref mut at);c=0;while(c<invalid){let raw=field(wire,ref mut at);let expected_start=number(wire,ref mut at);let stream=scanned(raw);var v::Vec<nodes::Node> types=v::empty<nodes::Node>();let failed=nodes::parse(ref stream,0,0,ref mut types);assert(failed.root==0 && failed.count<=4096 && failed.status==0);if(expected_start!=0){assert(failed.expected==5 && scanner::token_info(ref stream,failed.next).start==expected_start);}c+=1;}assert(at==wire.len);
}

fn bytes::Buffer nested(usize depth){var bytes::Buffer raw=bytes::empty();var usize i=0;while(i<depth){assert(bytes::append(ref mut raw,b"fn ").is_ok);i+=1;}assert(bytes::append(ref mut raw,b"i32").is_ok);i=0;while(i<depth){assert(bytes::append(ref mut raw,b"(i32)").is_ok);i+=1;}return raw;}
@test fn void closure_type_bindings_retain_written_types_without_execution(){
    let raw=b"module sample;fn fn i32(i32) identity(fn i32(i32) action){let fn /* binding */ i32(i32) copied=action;return copied;}";
    let p=switch(syntax::parse(scanned(raw),512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref p)==0 && !syntax::closure_prepared(ref p));
    let function=syntax::declaration_at(ref p,1);assert(syntax::type_at(ref p,function.result_type).kind==17);var usize i=0;var usize bindings=0;
    while(i<syntax::entry_count(ref p)){let e=syntax::entry_at(ref p,i);if(e.kind==2){assert(e.type_root!=0 && syntax::type_at(ref p,e.type_root).kind==17);assert(bytes::equal(syntax::type_text(ref p,e.type_root),b"fn /* binding */ i32(i32)"));assert(e.type_start==syntax::type_at(ref p,e.type_root).start && e.type_end==syntax::type_at(ref p,e.type_root).end);bindings+=1;}i+=1;}assert(bindings==1);
    let factory=b"module sample;fn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset+x;};return action;}";
    let written=retained(factory);assert(syntax::problem_count(ref written)==0 && !syntax::closure_prepared(ref written));assert(switch(syntax::parse(scanned(factory),512)){case .ok(_):{false}case .err(code):{code==4}});
}
@test fn void closure_type_parser_keeps_depth_and_node_bounds(){
    let fitting=nested(31);let stream=scanned(bytes::view(ref fitting));var v::Vec<nodes::Node> types=v::empty<nodes::Node>();let fit=nodes::parse(ref stream,0,0,ref mut types);assert(fit.root!=0 && fit.status==0 && v::at<nodes::Node>(ref types,fit.root-1).kind==17);
    let excessive=nested(32);let deep=scanned(bytes::view(ref excessive));var v::Vec<nodes::Node> too_deep=v::empty<nodes::Node>();let rejected=nodes::parse(ref deep,0,0,ref mut too_deep);assert(rejected.root==0 && rejected.status==4 && rejected.count<=4096);
    let small=scanned(b"fn i32(i32)");var v::Vec<nodes::Node> full=v::empty<nodes::Node>();var usize i=0;while(i<4092){assert(v::push<nodes::Node>(ref mut full,nodes::Node{.kind=1,.start=0,.end=0,.left=0,.right=0}).is_ok);i+=1;}let exact=nodes::parse(ref small,0,4092,ref mut full);assert(exact.root==4096 && exact.status==0 && exact.count==4096 && v::len<nodes::Node>(ref full)==4096);
    discard v::pop<nodes::Node>(ref mut full);discard v::pop<nodes::Node>(ref mut full);discard v::pop<nodes::Node>(ref mut full);let exhausted=nodes::parse(ref small,0,4093,ref mut full);assert(exhausted.root==0 && exhausted.status==4 && exhausted.count==4096 && v::len<nodes::Node>(ref full)==4096);
}

@test fn void closure_type_json_keeps_work_and_internal_link_boundaries(){
    let p=retained(b"module sample;fn fn i32(i32,u32) retained(fn void() value){}");assert(syntax::problem_count(ref p)==0);let root=syntax::declaration_at(ref p,1).result_type;var usize cost=0;let baseline=switch(encoded::encode(ref p,root,ref mut cost)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(cost>0 && cost<1048576);var usize work=1048576-cost;let exact=switch(encoded::encode(ref p,root,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return;}};assert(work==1048576 && bytes::equal(bytes::view(ref baseline),bytes::view(ref exact)));work=1048576-cost+1;assert(switch(encoded::encode(ref p,root,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);
    var json::Ledger ledger=json::empty();var bytes::Buffer out=bytes::empty();assert(!encoded::value(ref p,root,64,ref mut out,ref mut ledger) && ledger.code==10 && bytes::len(ref out)==0);assert(!encoded::value(ref p,root,0,ref mut out,ref mut ledger) && ledger.code==10 && bytes::len(ref out)==0);var usize i=1;var usize links=0;while(i<=syntax::type_count(ref p)){if(syntax::type_at(ref p,i).kind==16){work=0;assert(switch(encoded::encode(ref p,i,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});links+=1;}i+=1;}assert(links==2);
}

@test fn void retained_closure_type_does_not_publish_an_executable_graph(){
    let raw=b"module sample;fn fn i32(i32) retained(fn i32(i32) value){return value;}";let p=retained(raw);var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,retained(raw)).is_ok);let c=switch(catalog::build(ref sources)){case .ok(c):{c}case .err(_):{assert(false);return;}};var v::Vec<values::Type> graph=v::empty<values::Type>();assert(values::canonical(ref mut graph,values::Type{.kind=1,.scalar=1,.module_index=0,.declaration=0,.first=0,.second=0})==1);let rejected=values::resolve(ref c,ref p,0,syntax::declaration_at(ref p,1).result_type,ref mut graph);assert(rejected.type==0 && rejected.problem.code!=0 && values::count(ref graph)==1);let original=values::at(ref graph,1);assert(original.kind==1 && original.scalar==1 && original.module_index==0 && original.declaration==0 && original.first==0 && original.second==0);
}
