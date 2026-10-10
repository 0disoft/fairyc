module tests::module_project_types;
import module_project_types as project;
import module_types as types;
import module_signatures as signatures;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] source){let b=switch(bytes::copy(source)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn void failed(ref str phase,u32 code){discard io::write(phase);discard io::write(" ");discard io::write_u32(code);discard io::write("\n");assert(false);}
@effects(console) fn ir::Program compiled(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first){
    let bound=switch(binding::bind(sources,metadata,first,first+1)){case .ok(p):{p}case .err(e):{failed("binding",e.code);return compiled(sources,metadata,first);}};
    let ready=switch(closure::build_unit(sources,metadata,ref bound)){case .ok(p):{p}case .err(e):{failed("closure",e.code);return compiled(sources,metadata,first);}};
    let typed=switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(e):{failed("values",e.code);return compiled(sources,metadata,first);}};
    let moved=switch(moves::build(sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{failed("moves",e.code);return compiled(sources,metadata,first);}};
    return switch(lower::lower(sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{failed("lower",e.code);return compiled(sources,metadata,first);}};
}
fn v::Vec<types::Type> base(ref signatures::Program metadata){var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;while(i<=signatures::type_count(metadata)){assert(v::push<types::Type>(ref mut graph,signatures::type_at(metadata,i)).is_ok);i+=1;}return graph;}
fn types::Type node(u32 kind,u32 scalar,usize module_index,usize declaration,usize first,usize second){return types::Type{.kind,.scalar,.module_index,.declaration,.first,.second};}
fn types::Graph owned(ref signatures::Program metadata,v::Vec<types::Type> graph){return switch(signatures::own_types(metadata,graph)){case .ok(p):{p}case .err(_):{assert(false);return owned(metadata,base(metadata));}};}
@test @effects(console) fn void independently_lowered_units_share_project_types(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::run(40);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn i32 run(i32 n){let u8[3] bytes=[1,2,3];discard bytes;return n+2;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{failed("signatures",e.code);return;}};
    let caller=compiled(ref sources,ref metadata,0);let provider=compiled(ref sources,ref metadata,1);assert(ir::count(ref caller)==1 && ir::count(ref provider)==1);
    var v::Vec<types::Graph> inputs=v::empty<types::Graph>();let first=switch(project::from_ir(ref metadata,ref caller)){case .ok(p):{p}case .err(e):{failed("caller graph",e.code);return;}};let second=switch(project::from_ir(ref metadata,ref provider)){case .ok(p):{p}case .err(e):{failed("provider graph",e.code);return;}};assert(v::push<types::Graph>(ref mut inputs,first).is_ok);assert(v::push<types::Graph>(ref mut inputs,second).is_ok);
    let p=switch(project::build(ref metadata,ref inputs)){case .ok(p):{p}case .err(e):{failed("project",e.code);return;}};assert(project::unit_count(ref p)==2 && project::work(ref p)<=1048576);
    let caller_result=clean::frame_at(ir::cleanup_ir(ref caller),1).result;let provider_result=clean::frame_at(ir::cleanup_ir(ref provider),1).result;assert(project::mapped(ref p,0,caller_result)==project::mapped(ref p,1,provider_result));
    var usize i=1;var usize arrays=0;while(i<=clean::type_count(ir::cleanup_ir(ref provider))){let t=clean::type_at(ir::cleanup_ir(ref provider),i);let merged=project::type_at(ref p,project::mapped(ref p,1,i));assert(merged.kind==t.kind && merged.scalar==t.scalar && merged.module_index==t.module_index && merged.declaration==t.declaration);if(t.kind==16){arrays+=1;assert(merged.second==3 && merged.first==project::mapped(ref p,1,t.first));}i+=1;}assert(arrays==1);
    assert(project::mapped(ref p,0,0)==0 && project::mapped(ref p,2,1)==0 && project::mapped(ref p,0,4097)==0);
    discard io::write("independent IR units 2 canonical types ");discard io::write_usize(project::type_count(ref p));discard io::write("\n");
}
@test fn void reordered_wrappers_rebind_edges_and_preserve_array_lengths(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;struct Node{i32 value;}fn i32 main(){return 0;}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(signatures::type_count(ref metadata)==1);
    var a=base(ref metadata);let aa=types::canonical(ref mut a,node(16,0,0,0,1,3));let ao=types::canonical(ref mut a,node(5,0,0,0,aa,0));let an=types::canonical(ref mut a,node(2,0,0,2,0,0));let ap=types::canonical(ref mut a,node(12,1,0,0,an,0));let au=types::canonical(ref mut a,node(1,2,0,0,0,0));let ar=types::canonical(ref mut a,node(6,0,0,0,ao,au));
    var b=base(ref metadata);let bu=types::canonical(ref mut b,node(1,2,0,0,0,0));let bn=types::canonical(ref mut b,node(2,0,0,2,0,0));let bp=types::canonical(ref mut b,node(12,1,0,0,bn,0));let ba=types::canonical(ref mut b,node(16,0,0,0,1,3));let bo=types::canonical(ref mut b,node(5,0,0,0,ba,0));let br=types::canonical(ref mut b,node(6,0,0,0,bo,bu));assert(aa!=ba && ao!=bo);
    var v::Vec<types::Graph> inputs=v::empty<types::Graph>();assert(v::push<types::Graph>(ref mut inputs,owned(ref metadata,a)).is_ok);assert(v::push<types::Graph>(ref mut inputs,owned(ref metadata,b)).is_ok);let p=switch(project::build(ref metadata,ref inputs)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(project::type_count(ref p)==7);
    assert(project::mapped(ref p,0,aa)==project::mapped(ref p,1,ba) && project::mapped(ref p,0,ao)==project::mapped(ref p,1,bo) && project::mapped(ref p,0,an)==project::mapped(ref p,1,bn) && project::mapped(ref p,0,ap)==project::mapped(ref p,1,bp) && project::mapped(ref p,0,ar)==project::mapped(ref p,1,br));
    let array=project::type_at(ref p,project::mapped(ref p,0,aa));assert(array.first==1 && array.second==3);let pointer=project::type_at(ref p,project::mapped(ref p,1,bp));assert(pointer.scalar==1 && pointer.first==project::mapped(ref p,1,bn));let result=project::type_at(ref p,project::mapped(ref p,1,br));assert(result.first==project::mapped(ref p,1,bo) && result.second==project::mapped(ref p,1,bu));
    let copy=switch(project::copy_types(ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};let roundtrip=owned(ref metadata,copy);assert(types::graph_count(ref roundtrip)==7);
}
@test fn void project_type_and_unit_limits_and_signature_prefix_reject(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn i32 main(){return 0;}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var v::Vec<types::Graph> inputs=v::empty<types::Graph>();var usize i=0;while(i<256){assert(v::push<types::Graph>(ref mut inputs,owned(ref metadata,base(ref metadata))).is_ok);i+=1;}let p=switch(project::build(ref metadata,ref inputs)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(project::unit_count(ref p)==256 && project::type_count(ref p)==1 && project::mapped(ref p,255,1)==1);assert(v::push<types::Graph>(ref mut inputs,owned(ref metadata,base(ref metadata))).is_ok);switch(project::build(ref metadata,ref inputs)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1002);}};
    var v::Vec<types::Graph> large=v::empty<types::Graph>();var graph=base(ref metadata);i=1;while(i<4096){assert(types::canonical(ref mut graph,node(16,0,0,0,1,i))!=0);i+=1;}assert(v::push<types::Graph>(ref mut large,owned(ref metadata,graph)).is_ok);let full=switch(project::build(ref metadata,ref large)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(project::type_count(ref full)==4096);var extra=base(ref metadata);assert(types::canonical(ref mut extra,node(16,0,0,0,1,4096))!=0);assert(v::push<types::Graph>(ref mut large,owned(ref metadata,extra)).is_ok);switch(project::build(ref metadata,ref large)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1002);}};
    var v::Vec<types::Graph> wrong=v::empty<types::Graph>();var v::Vec<types::Type> bools=v::empty<types::Type>();assert(v::push<types::Type>(ref mut bools,node(1,6,0,0,0,0)).is_ok);assert(v::push<types::Graph>(ref mut wrong,owned(ref metadata,bools)).is_ok);switch(project::build(ref metadata,ref wrong)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001);}};
}
