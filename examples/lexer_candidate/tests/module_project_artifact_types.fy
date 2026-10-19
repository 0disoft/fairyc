module tests::module_project_artifact_types;
import module_project_artifact_types as mapping;
import module_project_compile as compile;
import module_unit_values as values;
import module_unit_body_ir_model as ir;
import module_signatures as signatures;
import module_catalog as catalog;
import module_types as types;
import module_program_artifact_wire as budget;
import tests::module_project_artifact_current_fixture as fixture;
import module_project_artifact_current as current;
import module_project_artifact_rebase as rebase;
import candidate_artifact_wire as wire;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] text){let raw=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let tokens=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn v::Vec<syntax::Parsed> source(ref u8[] text){var v::Vec<syntax::Parsed> p=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut p,parsed(text)).is_ok);return p;}
fn signatures::Program metadata(ref v::Vec<syntax::Parsed> sources){return switch(signatures::build(sources)){case .ok(p):{p}case .err(_):{assert(false);metadata(sources)}};}
@effects(console) fn compile::CheckedBodies checked(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop){return switch(compile::check_range(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.stage);discard io::write_stderr(" ");discard io::write_stderr_u32(e.code);assert(false);checked(sources,metadata,first,stop)}};}
@effects(console) fn ir::Program compiled(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){return switch(compile::range(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.stage);discard io::write_stderr(" ");discard io::write_stderr_u32(e.code);assert(false);compiled(sources,metadata)}};}
@effects(console) fn mapping::Index index(ref ir::Program p,ref mut usize work){return switch(mapping::build(p,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);index(p,work)}};}
@effects(console) fn mapping::Mapping matched(ref mapping::Index p,ref compile::CheckedBodies checked,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){return switch(mapping::match(p,compile::checked_state(checked),sources,metadata,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);matched(p,checked,sources,metadata,work)}};}
fn v::Vec<syntax::Parsed> shapes(u32 mode){
    var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,if(mode==4){b"module main;struct Other{i32 n;}struct Node{i32 n;}"}else{b"module main;struct Node{i32 n;}"}).is_ok);
    let option=b"fn Option<i32> option(Option<i32> n){return n;}";
    let result=if(mode==3){b"fn Result<u32,Node> result(Result<u32,Node> n){return n;}"}else{b"fn Result<Node,u32> result(Result<Node,u32> n){return n;}"};
    let pointer=if(mode==2){b"unsafe fn ptr Node pointer(ptr Node n){return n;}"}else{b"unsafe fn ptr mut Node pointer(ptr mut Node n){return n;}"};
    assert(bytes::append(ref mut text,if(mode==0){option}else{pointer}).is_ok && bytes::append(ref mut text,result).is_ok && bytes::append(ref mut text,if(mode==0){pointer}else{option}).is_ok);
    assert(bytes::append(ref mut text,if(mode==5){b"fn void arrays(){let u8[5] a=[1,2,3,4,5];discard a;}"}else{b"fn void arrays(){let u8[3] a=[1,2,3];let u8[4] b=[4,5,6,7];discard a;discard b;}"}).is_ok);
    assert(bytes::append(ref mut text,b"fn i32 main(){return 42;}").is_ok);return source(bytes::view(ref text));
}
@effects(console) fn mapping::Index detached(){let sources=shapes(0);let metadata=metadata(ref sources);let p=compiled(ref sources,ref metadata);var usize work=0;return index(ref p,ref mut work);}
@test @effects(console) fn void independent_numbering_matches_wrappers_array_lengths_and_pointer_flags(){
    let p=detached();let sources=shapes(1);let metadata=metadata(ref sources);let checked=checked(ref sources,ref metadata,0,1);var usize work=0;let ids=matched(ref p,ref checked,ref sources,ref metadata,ref mut work);let typed=compile::checked_values(ref checked);
    assert(mapping::count(ref ids)==values::type_count(typed) && mapping::target_count(ref ids)==mapping::type_count(ref p));var usize changed=0;var usize arrays=0;var usize results=0;var usize pointers=0;var usize i=1;
    while(i<=values::type_count(typed)){let a=values::type_at(typed,i);let id=mapping::mapped(ref ids,i);let b=mapping::type_at(ref p,id);if(i!=id){changed+=1;}assert(mapping::used(ref ids,id) && a.kind==b.kind && a.scalar==b.scalar && a.module_index==b.module_index && a.declaration==b.declaration && b.first==mapping::mapped(ref ids,a.first));
        if(a.kind==6){results+=1;assert(b.second==mapping::mapped(ref ids,a.second));}else{assert(a.second==b.second);}if(a.kind==16){arrays+=1;assert(b.second==3 || b.second==4);}if(a.kind==12 && a.first!=0){pointers+=1;assert(b.scalar==1);}i+=1;
    }assert(changed>0 && arrays==2 && results==1 && pointers==1 && work>0);assert(mapping::mapped(ref ids,0)==0 && mapping::mapped(ref ids,mapping::count(ref ids)+1)==0 && !mapping::used(ref ids,0) && !mapping::used(ref ids,mapping::target_count(ref ids)+1));
}
@test @effects(console) fn void mismatched_nominals_result_order_pointer_flags_and_array_lengths_reject(){let p=detached();var u32 mode=2;
    while(mode<=5){let sources=shapes(mode);let metadata=metadata(ref sources);let checked=checked(ref sources,ref metadata,0,1);var usize work=0;assert(switch(mapping::match(ref p,compile::checked_state(ref checked),ref sources,ref metadata,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}} && work>0);mode+=1;}
}
@test @effects(console) fn void shared_work_is_exact_and_exhaustion_publishes_no_map(){let sources=shapes(0);let metadata=metadata(ref sources);let program=compiled(ref sources,ref metadata);let checked=checked(ref sources,ref metadata,0,1);var usize required=0;let p=index(ref program,ref mut required);assert(required>0 && required<budget::work_limit());var usize work=budget::work_limit()-required;let exact=index(ref program,ref mut work);assert(work==budget::work_limit() && mapping::type_count(ref exact)==mapping::type_count(ref p));work=budget::work_limit()-required+1;assert(switch(mapping::build(ref program,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}});work=budget::work_limit()+1;assert(switch(mapping::build(ref program,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==budget::work_limit()+1);
    required=0;let ids=matched(ref p,ref checked,ref sources,ref metadata,ref mut required);work=budget::work_limit()-required;let fit=matched(ref p,ref checked,ref sources,ref metadata,ref mut work);assert(work==budget::work_limit() && mapping::count(ref ids)==mapping::count(ref fit));work=budget::work_limit()-required+1;assert(switch(mapping::match(ref p,compile::checked_state(ref checked),ref sources,ref metadata,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}});work=budget::work_limit()+1;assert(switch(mapping::match(ref p,compile::checked_state(ref checked),ref sources,ref metadata,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==budget::work_limit()+1);
}
@test @effects(console) fn void changed_current_sources_cannot_reuse_checked_type_facts(){let sources=source(b"module main;fn i32 main(){return 42;}");let metadata=metadata(ref sources);let program=compiled(ref sources,ref metadata);let checked=checked(ref sources,ref metadata,0,1);var usize work=0;let p=index(ref program,ref mut work);let changed=source(b"module main;fn i32 main(){return 43;}");work=0;assert(switch(mapping::match(ref p,compile::checked_state(ref checked),ref changed,ref metadata,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});let missing=v::empty<syntax::Parsed>();work=0;assert(switch(mapping::match(ref p,compile::checked_state(ref checked),ref missing,ref metadata,ref mut work)){case .ok(_):{false}case .err(code):{code==1001}});}
@test @effects(filesystem,console) fn void real_rebased_generic_vec_state_project_matches_each_checked_unit(){let packet=fixture::packet();let sources=fixture::current(1);var usize work=0;let rebound=switch(current::build_sources(fixture::restored(wire::content(ref packet)),fixture::sources(ref sources),fixture::owners(ref sources),fixture::imports(ref sources),fixture::coordinate_table(ref sources),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};work=0;let program=switch(rebase::build(ref rebound,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};
    let catalog=switch(catalog::build_scoped(fixture::sources(ref sources),fixture::owners(ref sources),fixture::imports(ref sources))){case .ok(p):{p}case .err(_):{assert(false);return;}};let metadata=switch(signatures::build_catalog(fixture::sources(ref sources),catalog)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let p=index(ref program,ref mut work);var usize unit=0;var usize vectors=0;
    while(unit<v::len<syntax::Parsed>(fixture::sources(ref sources))){let checked=checked(fixture::sources(ref sources),ref metadata,unit,unit+1);work=0;let ids=matched(ref p,ref checked,fixture::sources(ref sources),ref metadata,ref mut work);let typed=compile::checked_values(ref checked);var usize i=1;while(i<=values::type_count(typed)){let row=values::type_at(typed,i);if(row.kind==13){let target=mapping::type_at(ref p,mapping::mapped(ref ids,i));assert(target.kind==13 && target.module_index==row.module_index && target.declaration==row.declaration && target.first==mapping::mapped(ref ids,row.first));vectors+=1;}i+=1;}unit+=1;
    }assert(vectors>0 && unit>2);
}
