module tests::module_project_link;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lower as lower;
import module_unit_c_emit as emit;
import module_unit_cleanup_mir_model as clean;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_moves as moves;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn void failed(ref str phase,u32 code,usize start,usize end){discard io::write(phase);discard io::write(" ");discard io::write_u32(code);discard io::write(" ");discard io::write_usize(start);discard io::write("..");discard io::write_usize(end);discard io::write("\n");assert(false);}
@effects(console) fn ir::Program compiled(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first){
    let bound=switch(binding::bind(sources,metadata,first,first+1)){case .ok(p):{p}case .err(e):{failed("binding",e.code,e.start,e.end);return compiled(sources,metadata,first);}};
    let ready=switch(closure::build_unit(sources,metadata,ref bound)){case .ok(p):{p}case .err(e):{failed("closure",e.code,e.start,e.end);return compiled(sources,metadata,first);}};
    let typed=switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(e):{failed("values",e.code,e.start,e.end);return compiled(sources,metadata,first);}};
    let moved=switch(moves::build(sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{failed("moves",e.code,e.start,e.end);return compiled(sources,metadata,first);}};
    return switch(lower::lower(sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{failed("lower",e.code,e.start,e.end);return compiled(sources,metadata,first);}};
}
@effects(console) fn void output(ref v::Vec<syntax::Parsed> sources,usize expected_frames,usize expected_cleanups){
    let metadata=switch(signatures::build(sources)){case .ok(p):{p}case .err(e):{failed("signatures",e.code,e.start,e.end);return;}};var v::Vec<ir::Program> units=v::empty<ir::Program>();var usize i=0;while(i<v::len<syntax::Parsed>(sources)){assert(v::push<ir::Program>(ref mut units,compiled(sources,ref metadata,i)).is_ok);i+=1;}
    let p=switch(link::build(ref metadata,ref units)){case .ok(p):{p}case .err(e):{failed("link",e.code,e.start,e.end);return;}};assert(ir::count(ref p)==expected_frames);var usize entry=0;var usize cleanups=0;i=1;while(i<=ir::count(ref p)){let f=clean::frame_at(ir::cleanup_ir(ref p),i);if(f.kind==2){cleanups+=1;}if(f.module_index==0 && f.kind==3){entry=i;}i+=1;}assert(entry!=0 && cleanups==expected_cleanups);
    var bytes::Buffer text=bytes::empty();let e=emit::render(ref p,entry,ref mut text);if(e.code!=0){failed("emit",e.code,e.start,e.end);return;}assert(io::write_bytes(bytes::view(ref text)).is_ok);
}
@test @effects(console) fn void project_c_links_three_ordinary_units_with_different_type_graphs(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){assert(p::run(40)==42);return 0;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;import helper as h;pub fn i32 run(i32 n){let u8[3] bytes=[1,2,3];discard bytes;return h::add(n,2);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module helper;pub fn i32 add(i32 a,i32 b){return a+b;}")).is_ok);output(ref sources,3,0);
}
@test @effects(console) fn void project_c_namespaces_generic_instances_from_distinct_units(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import left as a;import right as b;fn i32 main(){assert(a::run(40)==40 && b::run(2)==2);return 0;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module left;import shared as s;pub fn i32 run(i32 n){return s::id<i32>(n);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module right;import shared as s;pub fn i32 run(u8 n){let u8 got=s::id<u8>(n);return got as i32;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module shared;pub fn T id<T>(T n){return n;}")).is_ok);output(ref sources,5,0);
}
@test @effects(console) fn void project_c_coalesces_shared_cleanup_bodies_and_preserves_drops(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;struct Dummy{i32 x;}fn i32 main(){let Dummy dummy=Dummy{.x=1};discard dummy;let value=p::make(7);discard value;return 0;}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Item{i32 n;cleanup{unsafe{observe(this.n);};}}@link_name(\"observe_cleanup\")unsafe extern C fn void observe(i32 n);pub fn Item make(i32 n){return Item{.n};}")).is_ok);output(ref sources,3,1);
}
@test @effects(console) fn void project_link_rejects_missing_and_duplicate_ordinary_bodies(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::run(40);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn i32 run(i32 n){return n+2;}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{failed("signatures",e.code,e.start,e.end);return;}};var v::Vec<ir::Program> units=v::empty<ir::Program>();assert(v::push<ir::Program>(ref mut units,compiled(ref sources,ref metadata,0)).is_ok);switch(link::build(ref metadata,ref units)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001 && e.module_index==0 && e.start<e.end);}};
    assert(v::push<ir::Program>(ref mut units,compiled(ref sources,ref metadata,0)).is_ok);switch(link::build(ref metadata,ref units)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001);}};
}
