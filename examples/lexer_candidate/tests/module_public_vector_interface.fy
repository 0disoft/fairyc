module tests::module_public_vector_interface;
import module_public_interface_extract as extract;
import module_public_interface as ordinary;
import module_public_interface_model as records;
import module_state_layout as state;
import module_public_source_types as source_names;
import module_public_type_ids as ids;
import module_unit_interfaces as typed;
import module_unit_generics as generics;
import module_unit_bind as binding;
import module_signatures as signatures;
import generic_source_plan as plan;
import generic_source_plan_input as prepared;
import generic_concrete_types as concrete;
import generic_package_context as context;
import generic_source_requests as requests;
import generic_source_dependencies as dependencies;
import generic_source_closure as closures;
import generic_source_dependency_model as model;
import generic_source_dependency_json as json;
import generic_source_catalog as templates;
import generic_package_bindings as bindings;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import syntax as syntax;
import scanner as scanner;
import std::fs as fs;
import std::text as text;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(value):{value}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(value):{value}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(value):{assert(syntax::problem_count(ref value)==0);value}case .err(_):{assert(false);parsed(raw)}};}
fn usize number(ref u8[] wire,ref mut usize at){assert(*at<=wire.len && wire.len-*at>=8);var usize out=0;var usize i=0;while(i<8){out=out*256+(wire[*at+i] as usize);i+=1;}*at+=8;return out;}
fn ref u8[] field(ref u8[] wire,ref mut usize at){let size=number(wire,at);assert(*at<=wire.len && size<=wire.len-*at);let start=*at;*at+=size;return wire[start..*at];}
fn coordinates::Coordinates coords(ref v::Vec<usize> digits,usize shift){var bytes::Buffer names=bytes::empty();var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();var usize i=0;while(i<v::len<usize>(digits)){let start=bytes::len(ref names);assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize n=0;while(n<64){let digit=*v::at<usize>(digits,i);assert(bytes::append(ref mut names,if(digit==1){b"1"}else if(digit==2){b"2"}else if(digit==3){b"3"}else{b"4"}).is_ok);n+=1;}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=i+shift,.start,.end=bytes::len(ref names)}).is_ok);i+=1;}return coordinates::assemble(names,rows);}
fn bindings::Bindings bound(ref v::Vec<syntax::Parsed> sources,ref v::Vec<catalog::Binding> imports,ref v::Vec<usize> digits,usize shift){var v::Vec<usize> owners=v::empty<usize>();var usize i=0;while(i<v::len<syntax::Parsed>(sources)){assert(v::push<usize>(ref mut owners,i+shift).is_ok);i+=1;}let table=coords(digits,shift);var usize work=0;return switch(bindings::build(sources,ref owners,imports,ref table,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return bound(sources,imports,digits,shift);}};}
@effects(console)
fn templates::Catalog collected(ref v::Vec<syntax::Parsed> sources,ref v::Vec<catalog::Binding> imports,ref v::Vec<usize> digits,usize shift){let resolved=bound(sources,imports,digits,shift);var usize work=0;return switch(templates::build(resolved,sources,ref mut work)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return collected(sources,imports,digits,shift);}};}
@effects(filesystem,console)
fn bytes::Buffer fixture(){var u8[4096] path=[0;4096];let size=switch(process::arg(1,ref mut path)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] view=ref path;let name=switch(text::validate_utf8(view[..size])){case .ok(value):{value}case .err(_):{assert(false);return bytes::empty();}};var fs::File file=switch(fs::open_read_utf8(name)){case .ok(value):{value}case .err(_):{assert(false);return bytes::empty();}};var bytes::Buffer out=bytes::empty();assert(bytes::resize_zeroed(ref mut out,1048577).is_ok);var usize used=0;while(true){let n=switch(fs::read(ref mut file,bytes::view_mut(ref mut out)[used..])){case .ok(value):{value}case .err(_):{assert(false);return out;}};used+=n;assert(used<=1048576);if(n==0){break;}}assert(fs::close(file).is_ok && bytes::truncate(ref mut out,used).is_ok);return out;}

fn ref u8[] root(){return b"module root;import helper as a;import helper as b;import std::vec as vec;resource Owned{i32 n;cleanup{discard a::id<i32>(1);}}fn bool main(){discard a::flip<i32,bool>(1,true);discard b::flip<i32,bool>(2,false);discard a::flip<bool,i32>(true,1);let vec::Vec<a::Token> items=vec::empty<a::Token>();discard items;let Owned x=Owned{.n=1};discard x;return true;}pub fn void expose(ref vec::Vec<i32> p0,ref mut vec::Vec<a::Token> p1,Option<vec::Vec<a::Token>> p2,Result<vec::Vec<i32>,i32> p3){discard p2;discard p3;}";}
fn ref u8[] provider(){return b"module helper;pub struct Token{i32 n;}pub fn B flip<A,B>(A a,B b){return id<B>(b);}pub fn T id<T>(T value){return value;}";}
@effects(filesystem,console)
fn v::Vec<syntax::Parsed> source_set(){let sdk=fixture();var v::Vec<syntax::Parsed> out=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut out,parsed(root())).is_ok && v::push<syntax::Parsed>(ref mut out,parsed(provider())).is_ok && v::push<syntax::Parsed>(ref mut out,parsed(sdk_text(bytes::view(ref sdk)))).is_ok);return out;}
fn v::Vec<catalog::Binding> imports(){var v::Vec<catalog::Binding> out=v::empty<catalog::Binding>();assert(v::push<catalog::Binding>(ref mut out,catalog::Binding{.source=0,.declaration=2,.target=1}).is_ok && v::push<catalog::Binding>(ref mut out,catalog::Binding{.source=0,.declaration=3,.target=1}).is_ok && v::push<catalog::Binding>(ref mut out,catalog::Binding{.source=0,.declaration=4,.target=2}).is_ok);return out;}
fn v::Vec<usize> digits(){var v::Vec<usize> out=v::empty<usize>();assert(v::push<usize>(ref mut out,1).is_ok && v::push<usize>(ref mut out,2).is_ok && v::push<usize>(ref mut out,3).is_ok);return out;}
@effects(console)
fn signatures::Program signatures_for(ref v::Vec<syntax::Parsed> sources,usize shift){let bindings=imports();var v::Vec<usize> owners=v::empty<usize>();var usize i=0;while(i<v::len<syntax::Parsed>(sources)){assert(v::push<usize>(ref mut owners,i+shift).is_ok);i+=1;}let declarations=switch(catalog::build_scoped(sources,ref owners,ref bindings)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return signatures_for(sources,shift);}};return switch(signatures::build_catalog(sources,declarations)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);signatures_for(sources,shift)}};}
@effects(console)
fn typed::Interfaces retained(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return retained(sources,metadata);}};return switch(typed::build(sources,metadata,ref bound)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);retained(sources,metadata)}};}
@effects(console)
fn plan::Plan planned(ref v::Vec<syntax::Parsed> sources,usize shift){let bindings=imports();let digits=digits();let original=collected(sources,ref bindings,ref digits,shift);var context::Budget work=context::empty();let roots=switch(requests::build(original,sources,ref mut work.work)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources,shift);}};let input=switch(prepared::build(roots,sources,ref mut work)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.source.code);assert(false);return planned(sources,shift);}};return switch(plan::build(input,plan::defaults(),ref mut work.work)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);planned(sources,shift)}};}

fn ref u8[] sdk_text(ref u8[] wire){assert(wire.len>=8 && bytes::equal(wire[..8],b"FYGVTYPE"));var usize at=8;return field(wire,ref mut at);}
@effects(console)
fn ids::Table named(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph){let original=planned(sources,0);var usize work=0;return switch(source_names::build(ref original,sources,metadata,graph,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return ids::empty();}};}
@effects(filesystem,console)
fn ids::Table detached(){let sources=source_set();let metadata=signatures_for(ref sources,0);let graph=retained(ref sources,ref metadata);return named(ref sources,ref metadata,ref graph);}

fn usize interface_start(ref u8[] wire){var usize at=8;discard field(wire,ref mut at);let vectors=number(wire,ref mut at);var usize i=0;while(i<vectors){discard field(wire,ref mut at);i+=1;}let parameters=number(wire,ref mut at);i=0;while(i<parameters){discard field(wire,ref mut at);i+=1;}assert(number(wire,ref mut at)==3);return at;}
@effects(filesystem,console)
fn extract::Projection detached_projection(){let sources=source_set();let metadata=signatures_for(ref sources,0);let graph=retained(ref sources,ref metadata);let names=named(ref sources,ref metadata,ref graph);let digits=digits();let table=coords(ref digits,0);var usize work=0;return switch(extract::extract_identified(ref names,ref sources,ref metadata,ref graph,ref table,2,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e);assert(false);return detached_projection();}};}
@test @effects(filesystem,console)
fn void full_descriptors_and_ordinary_hashes_match_real_go(){let sources=source_set();let metadata=signatures_for(ref sources,0);let graph=retained(ref sources,ref metadata);let names=named(ref sources,ref metadata,ref graph);let digits=digits();let table=coords(ref digits,0);let raw=fixture();let wire=bytes::view(ref raw);var usize at=interface_start(wire);var usize i=0;while(i<3){let owner=number(wire,ref mut at);let expected=field(wire,ref mut at);let hash=field(wire,ref mut at);discard field(wire,ref mut at);var usize work=0;let projected=switch(extract::extract_identified(ref names,ref sources,ref metadata,ref graph,ref table,owner,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_usize(owner);discard io::write_stderr_u32(e);assert(false);return;}};work=0;let encoded=switch(ordinary::encode(signatures::state_layouts(ref metadata),owner,extract::descriptor(ref projected),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref encoded),expected));work=0;let actual=switch(ordinary::hash(signatures::state_layouts(ref metadata),owner,extract::descriptor(ref projected),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref actual),hash));i+=1;}assert(at==wire.len);}
@test @effects(filesystem,console)
fn void sdk_projection_survives_all_source_and_plan_owners(){let projected=detached_projection();assert(extract::has_templates(ref projected));let layout=state::empty();var usize work=0;let actual=switch(ordinary::encode(ref layout,2,extract::descriptor(ref projected),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let raw=fixture();let wire=bytes::view(ref raw);var usize at=interface_start(wire);var usize i=0;while(i<3){let owner=number(wire,ref mut at);let expected=field(wire,ref mut at);discard field(wire,ref mut at);discard field(wire,ref mut at);if(owner==2){assert(bytes::equal(bytes::view(ref actual),expected));}i+=1;}let p=extract::descriptor(ref projected);var usize item=0;while(item<records::item_count(p)){let name=records::text(p,records::item_at(p,item).name);assert(name.len<12 || !bytes::equal(name[..12],b"_fy_generic_"));item+=1;}}
@test @effects(filesystem,console)
fn void identified_descriptor_exact_work_and_bad_owner_fail_closed(){let sources=source_set();let metadata=signatures_for(ref sources,0);let graph=retained(ref sources,ref metadata);let names=named(ref sources,ref metadata,ref graph);let digits=digits();let table=coords(ref digits,0);var usize measured=0;let expected=switch(extract::extract_identified(ref names,ref sources,ref metadata,ref graph,ref table,2,ref mut measured)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(measured>0 && measured<1048576);var usize work=1048576-measured;let exact=switch(extract::extract_identified(ref names,ref sources,ref metadata,ref graph,ref table,2,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(work==1048576 && records::item_count(extract::descriptor(ref exact))==records::item_count(extract::descriptor(ref expected)));work=1048577-measured;assert(switch(extract::extract_identified(ref names,ref sources,ref metadata,ref graph,ref table,2,ref mut work)){case .ok(_):{false}case .err(e):{e==10}});work=0;assert(switch(extract::extract_identified(ref names,ref sources,ref metadata,ref graph,ref table,512,ref mut work)){case .ok(_):{false}case .err(e):{e==9}});work=0;assert(switch(extract::extract(ref sources,ref metadata,ref graph,ref table,2,ref mut work)){case .ok(_):{false}case .err(e):{e==4503}});}
