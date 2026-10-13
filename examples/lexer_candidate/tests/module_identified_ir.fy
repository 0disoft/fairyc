module tests::module_identified_ir;
import module_source_specializations as selected;
import module_identified_ir as identified;
import module_project_compile as compile;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_emit as c;
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

fn ref u8[] root(){return b"module root;import helper as a;import helper as b;import std::vec as vec;resource Owned{i32 n;cleanup{discard a::id<i32>(1);}}fn Owned make(){return Owned{.n=1};}fn i32 main(){discard a::flip<i32,bool>(1,true);discard b::flip<i32,bool>(2,false);discard a::flip<bool,i32>(true,1);let vec::Vec<a::Token> items=vec::empty<a::Token>();discard items;let Owned x=make();discard x;return a::ordinary()+1;}";}
fn ref u8[] provider(){return b"module helper;pub struct Token{i32 n;}pub fn B flip<A,B>(A a,B b){return id<B>(b);}pub fn T id<T>(T value){return value;}pub fn i32 ordinary(){return 41;}";}
@effects(filesystem,console)
fn v::Vec<syntax::Parsed> sources(){let sdk=fixture();var v::Vec<syntax::Parsed> out=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut out,parsed(root())).is_ok && v::push<syntax::Parsed>(ref mut out,parsed(provider())).is_ok && v::push<syntax::Parsed>(ref mut out,parsed(bytes::view(ref sdk))).is_ok);return out;}
fn v::Vec<catalog::Binding> imports(){var v::Vec<catalog::Binding> out=v::empty<catalog::Binding>();assert(v::push<catalog::Binding>(ref mut out,catalog::Binding{.source=0,.declaration=2,.target=1}).is_ok && v::push<catalog::Binding>(ref mut out,catalog::Binding{.source=0,.declaration=3,.target=1}).is_ok && v::push<catalog::Binding>(ref mut out,catalog::Binding{.source=0,.declaration=4,.target=2}).is_ok);return out;}
fn v::Vec<usize> digits(){var v::Vec<usize> out=v::empty<usize>();assert(v::push<usize>(ref mut out,1).is_ok && v::push<usize>(ref mut out,2).is_ok && v::push<usize>(ref mut out,3).is_ok);return out;}
@effects(console)
fn signatures::Program signatures_for(ref v::Vec<syntax::Parsed> sources,usize shift){let bindings=imports();var v::Vec<usize> owners=v::empty<usize>();var usize i=0;while(i<v::len<syntax::Parsed>(sources)){assert(v::push<usize>(ref mut owners,i+shift).is_ok);i+=1;}let declarations=switch(catalog::build_scoped(sources,ref owners,ref bindings)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return signatures_for(sources,shift);}};return switch(signatures::build_catalog(sources,declarations)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);signatures_for(sources,shift)}};}
@effects(console)
fn typed::Interfaces retained(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return retained(sources,metadata);}};return switch(typed::build(sources,metadata,ref bound)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);retained(sources,metadata)}};}
@effects(console)
fn plan::Plan planned(ref v::Vec<syntax::Parsed> sources,usize shift){let bindings=imports();let digits=digits();let original=collected(sources,ref bindings,ref digits,shift);var context::Budget work=context::empty();let roots=switch(requests::build(original,sources,ref mut work.work)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources,shift);}};let input=switch(prepared::build(roots,sources,ref mut work)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.source.code);assert(false);return planned(sources,shift);}};return switch(plan::build(input,plan::defaults(),ref mut work.work)){case .ok(value):{value}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);planned(sources,shift)}};}

@effects(filesystem,console)
fn v::Vec<syntax::Parsed> changed_sources(u32 mode){let sdk=fixture();var v::Vec<syntax::Parsed> out=v::empty<syntax::Parsed>();let raw=if(mode==1){b"module root;import helper as a;import helper as b;import std::vec as vec;fn i32 main(){discard a::id<i32>(1);return true;}"}else if(mode==2){b"module root;import helper as a;import helper as b;import std::vec as vec;resource Owned{i32 n;cleanup{}}fn Owned make(){return Owned{.n=1};}fn void take(Owned x){}fn i32 main(){discard a::id<i32>(1);let Owned x=make();take(x);take(x);return 0;}"}else{b"module root;import helper as a;import helper as b;import std::vec as vec;fn ref i32 invalid(){var i32 x=1;return ref x;}fn i32 main(){discard a::id<i32>(1);return 0;}"};assert(v::push<syntax::Parsed>(ref mut out,parsed(raw)).is_ok && v::push<syntax::Parsed>(ref mut out,parsed(provider())).is_ok && v::push<syntax::Parsed>(ref mut out,parsed(bytes::view(ref sdk))).is_ok);return out;}
@effects(filesystem,console)
fn identified::Program compiled(){let sources=sources();let metadata=signatures_for(ref sources,0);let original=planned(ref sources,0);var usize work=0;return switch(compile::range_identified(ref sources,ref metadata,ref original,0,3,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(b"stage=");discard io::write_stderr_u32(e.stage);discard io::write_stderr_bytes(b" code=");discard io::write_stderr_u32(e.code);assert(false);compiled()}};}
@test
@effects(filesystem,console)
fn void checked_ir_frames_types_and_calls_keep_source_ids(){let actual=compiled();let program=identified::ir_program(ref actual);assert(identified::frame_count(ref actual)==ir::count(program));var usize i=1;var usize functions=0;var usize cleaned_count=0;var usize ordinary=0;var usize entry=0;var usize specialized_calls=0;var usize ordinary_calls=0;var usize work=0;
while(i<=ir::count(program)){let f=clean::frame_at(ir::cleanup_ir(program),i);let row=identified::frame_at(ref actual,i);let id=identified::identity(ref actual,row);if(row.node!=0){assert(id.len==67 && bytes::equal(id[..3],b"g0_"));if(f.kind==1){functions+=1;}else{assert(f.kind==2);assert(bytes::equal(id,identified::identity(ref actual,identified::type_at(ref actual,f.receiver_type))));cleaned_count+=1;}}else{assert(id.len==0);ordinary+=1;}if(f.module_index==0 && f.kind==3 && clean::type_at(ir::cleanup_ir(program),f.result).kind==1 && clean::type_at(ir::cleanup_ir(program),f.result).scalar==1){entry=i;}let body=ir::at(program,i);var usize j=0;while(j<ir::call_count(body)){let call=ir::call_at(body,j);let target=switch(identified::call_frame(ref actual,i,j,ref mut work)){case .ok(n):{n}case .err(code):{discard io::write_stderr_u32(code);assert(false);0}};if(call.kind==2){assert(target!=0 && identified::frame_at(ref actual,target).node!=0);specialized_calls+=1;}else if(target!=0){assert(identified::frame_at(ref actual,target).node==0);ordinary_calls+=1;}j+=1;}i+=1;}
i=1;var usize vectors=0;while(i<=clean::type_count(ir::cleanup_ir(program))){let t=clean::type_at(ir::cleanup_ir(program),i);let row=identified::type_at(ref actual,i);assert((row.node!=0)==(t.kind==13));if(t.kind==13){vectors+=1;}i+=1;}if(!(functions==5 && cleaned_count==6 && ordinary>=3 && vectors==6 && specialized_calls>=5 && ordinary_calls>=2 && entry!=0)){discard io::write_stderr_bytes(b"counts ");discard io::write_stderr_usize(functions);discard io::write_stderr_bytes(b" ");discard io::write_stderr_usize(cleaned_count);discard io::write_stderr_bytes(b" ");discard io::write_stderr_usize(ordinary);discard io::write_stderr_bytes(b" ");discard io::write_stderr_usize(vectors);discard io::write_stderr_bytes(b" ");discard io::write_stderr_usize(specialized_calls);discard io::write_stderr_bytes(b" ");discard io::write_stderr_usize(ordinary_calls);discard io::write_stderr_bytes(b" ");discard io::write_stderr_usize(entry);}assert(functions==5 && cleaned_count==6 && ordinary>=3 && vectors==6 && specialized_calls>=5 && ordinary_calls>=2 && entry!=0);var bytes::Buffer output=bytes::empty();let rendered=c::render(program,entry,ref mut output);if(rendered.code!=0){discard io::write_stderr_bytes(b"render=");discard io::write_stderr_u32(rendered.code);}assert(rendered.code==0 && bytes::len(ref output)>1000);}
@test
@effects(filesystem,console)
fn void identified_compilation_preserves_type_move_and_borrow_failure_gates(){var u32 mode=1;while(mode<=3){let sources=changed_sources(mode);let metadata=signatures_for(ref sources,0);let original=planned(ref sources,0);let expected=switch(compile::range(ref sources,ref metadata,0,1)){case .ok(_):{discard io::write_stderr_bytes(b"unexpected success mode=");discard io::write_stderr_u32(mode);assert(false);return;}case .err(e):{e}};var usize work=0;switch(compile::range_identified(ref sources,ref metadata,ref original,0,1,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==expected.stage && e.code==expected.code && e.module_index==expected.module_index && e.start==expected.start && e.end==expected.end && e.first==expected.first && e.stop==expected.stop);if(e.stage!=if(mode==1){4}else if(mode==2){5}else{7}){discard io::write_stderr_bytes(b"mode=");discard io::write_stderr_u32(mode);discard io::write_stderr_bytes(b" stage=");discard io::write_stderr_u32(e.stage);discard io::write_stderr_bytes(b" code=");discard io::write_stderr_u32(e.code);}assert(e.stage==if(mode==1){4}else if(mode==2){5}else{7});assert(work==0);}};mode+=1;}}
@test
@effects(filesystem,console)
fn void failed_identity_attachment_returns_no_checked_prefix(){let sources=sources();let metadata=signatures_for(ref sources,0);let original=planned(ref sources,0);var usize work=16777216;switch(compile::range_identified(ref sources,ref metadata,ref original,0,1,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==8 && e.code==10 && e.first==0 && e.stop==1);}};let shifted=planned(ref sources,19);work=0;switch(compile::range_identified(ref sources,ref metadata,ref shifted,0,1,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==8 && e.code==9);}};}
