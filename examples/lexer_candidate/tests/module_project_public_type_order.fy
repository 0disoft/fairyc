module tests::module_project_public_type_order;
import module_project_public_type_order as order;
import module_unit_cleanup_mir_model as clean;
import module_unit_cleanup_mir_types as identity;
import module_unit_body_ir_model as ir;
import module_signatures as signatures;
import module_types as types;
import module_state_layout as budget;
import module_project_artifact_current as current;
import module_project_artifact_rebase as rebase;
import module_catalog as catalog;
import tests::module_project_artifact_current_fixture as fixture;
import candidate_artifact_wire as wire;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let tokens=switch(scanner::collect(fixture::copied(raw),false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};return switch(syntax::parse(tokens,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(b"module empty;")}};}
fn v::Vec<syntax::Parsed> make_sources(bool reversed,bool changed){var v::Vec<syntax::Parsed> out=v::empty<syntax::Parsed>();var usize i=0;while(i<2){let old=if(reversed){1-i}else{i};let raw=if(old==0){b"module alpha;pub struct A{i32 n;}pub fn Option<ref A> shared(ref A x){return .some(x);}fn void array(u8[3] x){}"}else if(changed){b"module beta;pub struct C{bool n;}pub struct B{bool n;}pub fn Result<B,i32> value(Result<B,i32> x){return x;}"}else{b"module beta;pub struct B{bool n;}pub fn Result<B,i32> value(Result<B,i32> x){return x;}"};assert(v::push<syntax::Parsed>(ref mut out,parsed(raw)).is_ok);i+=1;}return out;}
@effects(console) fn signatures::Program metadata(ref v::Vec<syntax::Parsed> p){return switch(signatures::build(p)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);metadata(p)}};}
fn clean::Program retained(ref signatures::Program p,bool reversed,bool extra){var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<clean::Glue> glue=v::empty<clean::Glue>();var usize scalar=0;var usize i=1;
    while(i<=signatures::type_count(p)){var row=signatures::type_at(p,i);if(row.kind==1 && row.scalar==1){scalar=i;}if(reversed && (row.kind==2 || row.kind==3 || row.kind==4 || row.kind==13)){row.module_index=1-row.module_index;}assert(v::push<types::Type>(ref mut graph,row).is_ok);i+=1;}
    if(extra){assert(scalar>0 && v::push<types::Type>(ref mut graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=scalar,.second=9}).is_ok);}i=0;
    while(i<v::len<types::Type>(ref graph)){let t=*v::at<types::Type>(ref graph,i);assert(v::push<clean::Glue>(ref mut glue,clean::Glue{.kind=if(t.kind==5){2}else if(t.kind==6){3}else{0},.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);i+=1;}
    return switch(clean::assemble(graph,v::empty<clean::Frame>(),glue,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),0)){case .ok(p):{p}case .err(_):{assert(false);retained(p,reversed,extra)}};
}
@effects(console) fn order::Order ordered(ref clean::Program p,ref signatures::Program metadata,ref mut usize work){return switch(order::build(p,metadata,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);ordered(p,metadata,work)}};}
fn usize verify(ref order::Order p,ref clean::Program retained,ref signatures::Program metadata){assert(order::count(p)==clean::type_count(retained));var usize moved=0;var usize i=1;
    while(i<=order::count(p)){let old=order::original(p,i);assert(old>0 && old<=order::count(p) && order::mapped(p,old)==i);var row=clean::type_at(retained,old);if(row.first!=0){row.first=order::mapped(p,row.first);}if(row.kind==6){row.second=order::mapped(p,row.second);}assert(identity::same(row,order::at(p,i)));if(i<=signatures::type_count(metadata)){assert(identity::same(row,signatures::type_at(metadata,i)));}if(old!=i){moved+=1;}i+=1;}return moved;
}
@effects(console) fn order::Order detached(){let old_sources=make_sources(false,false);let original=metadata(ref old_sources);let p=retained(ref original,true,true);let now=make_sources(true,false);let metadata=metadata(ref now);var usize work=0;let result=ordered(ref p,ref metadata,ref mut work);assert(verify(ref result,ref p,ref metadata)>0 && work>0);return result;}
@test @effects(console) fn void current_signature_prefix_and_all_composite_references_reorder_and_detach(){let p=detached();assert(order::count(ref p)>signatures_count());var usize i=1;var bool extra=false;while(i<=order::count(ref p)){let t=order::at(ref p,i);assert(t.first==0 || t.first<i);if(t.kind==6){assert(t.second>0 && t.second<i);}if(t.kind==16 && t.second==9){extra=true;}i+=1;}assert(extra && order::mapped(ref p,0)==0 && order::mapped(ref p,order::count(ref p)+1)==0);}
@effects(console) fn usize signatures_count(){let sources=make_sources(false,false);let p=metadata(ref sources);return signatures::type_count(ref p);}
@test @effects(console) fn void unchanged_cold_numbering_and_exact_exhausted_invalid_work_are_preserved(){let sources=make_sources(false,false);let metadata=metadata(ref sources);let p=retained(ref metadata,false,true);var usize work=0;let normal=ordered(ref p,ref metadata,ref mut work);assert(verify(ref normal,ref p,ref metadata)==0);let needed=work;work=1048576-needed;let exact=ordered(ref p,ref metadata,ref mut work);assert(work==1048576 && order::count(ref exact)==order::count(ref normal));work=1048576-needed+1;assert(switch(order::build(ref p,ref metadata,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});work=1048577;assert(switch(order::build(ref p,ref metadata,ref mut work)){case .ok(_):{false}case .err(code):{code==10}} && work==1048577);}
@test @effects(console) fn void absent_current_nominal_types_reject_without_a_partial_mapping(){let sources=make_sources(false,false);let original=metadata(ref sources);let p=retained(ref original,true,true);let now=make_sources(true,true);let changed=metadata(ref now);var usize work=0;assert(switch(order::build(ref p,ref changed,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});}
@test @effects(filesystem,console) fn void actual_rebased_generic_vec_state_type_order_matches_current_signatures(){let data=fixture::packet();let now=fixture::current(1);let sources=fixture::sources(ref now);var usize work=0;let rebound=switch(current::build_sources(fixture::restored(wire::content(ref data)),sources,fixture::owners(ref now),fixture::imports(ref now),fixture::coordinate_table(ref now),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};work=0;let p=switch(rebase::build(ref rebound,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};
    let scope=switch(catalog::build_scoped(sources,fixture::owners(ref now),fixture::imports(ref now))){case .ok(p):{p}case .err(_):{assert(false);return;}};let metadata=switch(signatures::build_catalog(sources,scope)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let ordered=ordered(ir::cleanup_ir(ref p),ref metadata,ref mut work);assert(verify(ref ordered,ir::cleanup_ir(ref p),ref metadata)>0);var usize i=1;var usize vectors=0;while(i<=order::count(ref ordered)){if(order::at(ref ordered,i).kind==13){vectors+=1;}i+=1;}assert(vectors>0 && ir::count(ref p)>100 && work>0);
}
