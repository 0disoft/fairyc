module tests::module_unit_loan_records;
import module_unit_origin_diagnostics as diagnostics;
import module_unit_origin_returns as origins;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import module_unit_origin_check as checker;
import std::process as process;
import file_source as files;
import module_unit_origin_inputs as lifetime_inputs;
import module_unit_origin_flow_model as lifetime;
import module_unit_origin_sets as lifetime_sets;
import module_unit_origin_provenance as lifetime_sites;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn values::Plan primary(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(b):{b}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};return switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);primary(sources,metadata)}};}


@effects(console) fn origins::Plan owned(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);
    if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return owned(root,provider);}};
    let typed=primary(ref sources,ref metadata);
    return switch(origins::build(ref sources,ref metadata,ref typed)){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);assert(false);owned(root,provider)}};
}
@effects(console) fn void show(ref origins::Plan p){var usize frame=1;while(frame<=origins::count(p)){
    discard io::write("frame ");discard io::write_usize(frame);discard io::write(" ");discard io::write_usize(origins::loan_record_count(p,frame));discard io::write("\n");
    var usize i=0;while(i<origins::loan_record_count(p,frame)){let r=origins::loan_record_at(p,frame,i);
        discard io::write("loan ");discard io::write_usize(i);discard io::write(" ");discard io::write_u32(if(r.subview){1}else{0});discard io::write(" ");discard io::write_usize(r.owner);discard io::write(" ");discard io::write_usize(r.parent);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");i+=1;}frame+=1;}}
@test @effects(console) fn void root_and_child_records_survive_source_drop(){
    let p=owned(b"module main;fn void empty(){}fn void f(){var u8[2] data=[1,2];let ref mut u8[] parent=ref mut data;let ref mut u8[] child=parent[..1];discard child[0];}",b"");show(ref p);
    assert(origins::count(ref p)==2 && origins::loan_record_count(ref p,1)==0 && origins::loan_record_count(ref p,2)==2);
    let input=origins::owned_inputs(ref p);let frame=lifetime_inputs::frame(input,2);
    let root=origins::loan_record_at(ref p,2,0);let child=origins::loan_record_at(ref p,2,1);
    assert(!root.subview && root.mutable && root.owner==frame.local_start+1 && root.parent==0);
    assert(child.subview && child.mutable && child.owner==0 && child.parent==frame.local_start+2);
    assert(origins::loan_origin_count(ref p,2,0)==1 && origins::loan_origin_count(ref p,2,1)==1);
    assert(origins::loan_origin_at(ref p,2,0,0)==origins::loan_origin_at(ref p,2,1,0));
    var copy=child;copy.parent=0;assert(origins::loan_record_at(ref p,2,1).parent==child.parent);
}
@test @effects(console) fn void converged_parameter_generation_is_not_duplicated(){
    let p=owned(b"module main;fn ref mut u8[] f(ref mut u8[] p){return p[..1];}",b"");show(ref p);
    assert(origins::rounds(ref p)>=2 && origins::loan_record_count(ref p,1)==1);
    let r=origins::loan_record_at(ref p,1,0);let input=origins::owned_inputs(ref p);let parameter=lifetime_inputs::parameter(input,1,0);
    assert(r.subview && r.owner==0 && r.parent==parameter && origins::summary_subview(ref p,1));
    assert(origins::loan_origin_at(ref p,1,0,0)==parameter+2);
}
@test @effects(console) fn void mixed_origins_remain_readable_after_branch_join(){
    let p=owned(b"module main;fn void f(ref mut u8[] a,ref mut u8[] b,bool c){var ref mut u8[] parent=a;if(c){parent=b;};let ref mut u8[] child=parent[..1];discard child[0];}",b"");show(ref p);
    assert(origins::loan_record_count(ref p,1)==1 && origins::loan_origin_count(ref p,1,0)==2);
    let input=origins::owned_inputs(ref p);assert(origins::loan_origin_at(ref p,1,0,0)==lifetime_inputs::parameter(input,1,0)+2);
    assert(origins::loan_origin_at(ref p,1,0,1)==lifetime_inputs::parameter(input,1,1)+2);
}
@test @effects(console) fn void provider_and_generic_records_keep_separate_frames(){
    let p=owned(b"module main;import provider as pr;fn void f(ref mut u8[] p){let ref mut u8[] child=pr::tail<u8>(p);discard child[0];}",b"module provider;pub fn ref mut T[] tail<T>(ref mut T[] p){return p[..1];}");show(ref p);
    assert(origins::count(ref p)==2 && origins::loan_record_count(ref p,1)==1 && origins::loan_record_count(ref p,2)==1);
    let input=origins::owned_inputs(ref p);let usize caller_frame=if(lifetime_inputs::frame(input,1).value.module_index==0){1}else{2};let usize provider_frame=if(caller_frame==1){2}else{1};
    let caller=origins::loan_record_at(ref p,caller_frame,0);let provider=origins::loan_record_at(ref p,provider_frame,0);
    assert(lifetime_inputs::frame(input,caller_frame).value.module_index==0 && lifetime_inputs::frame(input,provider_frame).value.module_index==1);
    assert(caller.subview && caller.parent==0 && provider.subview && provider.parent!=0);
    assert(origins::loan_origin_at(ref p,caller_frame,0,0)==lifetime_inputs::parameter(input,caller_frame,0)+2);
    assert(origins::loan_origin_at(ref p,provider_frame,0,0)==lifetime_inputs::parameter(input,provider_frame,0)+2);
}
