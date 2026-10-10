module tests::module_unit_calls;
import module_unit_call_closure as closure;
import module_unit_calls as calls;
import module_unit_interfaces as interfaces;
import module_unit_bodies as bodies;
import module_unit_names as names;
import module_unit_ordinary_names as ordinary;
import module_unit_bind as binding;
import module_unit_drops as drops;
import module_unit_plan as units;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@test @effects(console) fn void private_call_results_close_nested_fields_and_cleanup_call_results(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::go<i32>(1);}")).is_ok);
    assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;resource Extra{u8 n;cleanup{}}resource Leaf{i32 n;cleanup{let inferred=cleanup_value();discard inferred;}}resource Hidden{Leaf item;cleanup{}}fn Extra cleanup_value(){return Extra{.n=1};}fn Hidden helper(){return Hidden{.item=Leaf{.n=2}};}pub fn T go<T>(T n){let Option<i32> maybe=.some(1);discard maybe;let inferred=helper();discard inferred;return n;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};
    let initial=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(interfaces::shape_count(ref initial)==0);
    let p=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" at ");discard io::write_usize(e.module_index);discard io::write(" ");discard io::write_usize(e.start);discard io::write("..");discard io::write_usize(e.end);discard io::write("\n");assert(false);return;}};let imported=closure::imported(ref p);let plan=closure::call_plan(ref p);assert(closure::passes(ref p)>=3 && closure::work(ref p)<=1048576 && calls::target_count(plan)==0 && calls::count(plan)==2 && calls::variants(plan)==1 && interfaces::shape_count(imported)==3 && bodies::cleanup_count(closure::owned(ref p))==3);var usize i=0;var usize functions=0;var usize cleaning=0;
    while(i<calls::count(plan)){let c=calls::at(plan,i);assert(c.module_index==1 && c.target_module==1 && c.result==signatures::at(ref metadata,c.signature).result && c.parameter_count==0 && !c.foreign);let e=syntax::expression_at(v::at<syntax::Parsed>(ref sources,1),c.source_index);assert(e.kind==6 && e.start==c.start && e.end==c.end);if(c.context_kind==1){functions+=1;assert(c.declaration==7 && c.target_declaration==6);}else{cleaning+=1;assert(c.context_kind==2 && c.declaration==3 && c.target_declaration==5);}assert(drops::at(closure::drop_plan(ref p),c.result).complete);i+=1;}assert(functions==1 && cleaning==1);
}
@test fn void local_callable_shadow_is_rejected_at_the_callee_span(){
    let text=b"module main;fn i32 helper(i32 n){return n;}fn i32 main(){let i32 helper=0;return helper(1);}";var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};switch(units::build(ref sources,ref metadata)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==9 && p.module_index==0 && bytes::equal(text[p.start..p.end],b"helper"));}};
}
@test fn void call_result_in_array_cast_keeps_its_original_callee_binding(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;resource Context{u32 n;cleanup{}}fn u32 helper(ref Context p,usize id){return p.n;}fn usize emit(ref Context p){let usize[20] fields=[0,helper(p,0) as usize,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0];return fields[0];}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let p=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let plan=closure::call_plan(ref p);assert(calls::count(plan)==1 && calls::target_count(plan)==0);let c=calls::at(plan,0);assert(c.context_kind==3 && c.declaration==4 && c.target_declaration==3 && c.parameter_count==2 && interfaces::type_at(closure::imported(ref p),c.result).scalar==4);
}
@test fn void foreign_call_contracts_and_lexical_unsafe_context_are_retained(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;@link_name(\"fairyc_probe\")unsafe extern C fn i32 leaf(i32 n);unsafe fn i32 main(){return leaf(1);}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let p=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let plan=closure::call_plan(ref p);assert(calls::count(plan)==1 && calls::target_count(plan)==0);let c=calls::at(plan,0);assert(c.context_kind==3 && c.declaration==3 && c.target_declaration==2 && c.parameter_count==1 && c.foreign && c.unsafe_call && c.unsafe_context && c.effects==0 && !c.must_use);let imported=closure::imported(ref p);assert(interfaces::type_at(imported,c.result).scalar==1 && interfaces::type_at(imported,signatures::parameter_at(ref metadata,c.parameter_start)).scalar==1);
}
@test fn void stale_call_input_is_rejected_before_signature_publication(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn i32 helper(){return 1;}fn i32 main(){return helper();}")).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(b):{b}case .err(_):{assert(false);return;}};let p=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};discard v::pop<syntax::Parsed>(ref mut sources);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;fn i32 helper(){return 2;}fn i32 main(){return helper();}")).is_ok);let fresh=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};switch(calls::build(ref sources,ref fresh,closure::imported(ref p),closure::owned(ref p),closure::resolved_names(ref p),closure::original_names(ref p))){case .ok(_):{assert(false);}case .err(p):{assert(p.code==12 && p.module_index==0);}};
}
