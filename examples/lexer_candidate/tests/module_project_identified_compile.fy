module tests::module_project_identified_compile;
import module_project_identified_compile as project;
import module_project_compile as compile;
import module_identified_ir as identified;
import module_project_identified_ir as linked;
import module_project_link as linking;
import module_project_inputs as inputs;
import module_unit_body_ir_verify as proof;
import module_unit_c_emit as c;
import candidate_limits as limits;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_signatures as signatures;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import generic_package_bindings as bindings;
import generic_source_catalog as templates;
import generic_source_requests as requests;
import generic_source_plan_input as prepared;
import generic_source_plan as plan;
import generic_package_context as context;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(bytes::Buffer raw){let scanned=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};return switch(syntax::parse(scanned,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(bytes::empty())}};}
fn syntax::Parsed text(ref u8[] raw){return parsed(switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}});}
fn syntax::Parsed provider(ref u8[] name,bool invalid){var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module ").is_ok && bytes::append(ref mut raw,name).is_ok && bytes::append(ref mut raw,b";import helper as shared;").is_ok);var u8 i=0;
    while(i<8){let u8[1] letter=[97+i];assert(bytes::append(ref mut raw,b"fn T ").is_ok && bytes::append(ref mut raw,ref letter[..]).is_ok && bytes::append(ref mut raw,b"<T>(T n){return n;}").is_ok);i+=1;}
    assert(bytes::append(ref mut raw,b"pub fn i32 run(){").is_ok);i=0;
    while(i<8){let u8[1] letter=[97+i];var usize type=0;while(type<5){assert(bytes::append(ref mut raw,b"discard ").is_ok && bytes::append(ref mut raw,ref letter[..]).is_ok);let call=if(type==0){b"<bool>(false);"}else if(type==1){b"<u8>(1);"}else if(type==2){b"<u32>(1);"}else if(type==3){b"<i32>(1);"}else{b"<usize>(1);"};assert(bytes::append(ref mut raw,call).is_ok);type+=1;}i+=1;}
    assert(bytes::append(ref mut raw,if(invalid){b"discard shared::id<i32>(1);return true;}"}else{b"return shared::id<i32>(21);}"}).is_ok);return parsed(raw);
}
fn v::Vec<syntax::Parsed> source_set(bool pressure,bool invalid,bool changed){var v::Vec<syntax::Parsed> out=v::empty<syntax::Parsed>();
    if(pressure){assert(v::push<syntax::Parsed>(ref mut out,text(b"module root;import left as a;import right as b;fn i32 main(){return a::run()+b::run();}")).is_ok && v::push<syntax::Parsed>(ref mut out,provider(b"left",false)).is_ok && v::push<syntax::Parsed>(ref mut out,provider(b"right",invalid)).is_ok);}
    else{assert(v::push<syntax::Parsed>(ref mut out,text(b"module root;import helper as h;fn i32 main(){return h::id<i32>(42);}")).is_ok);}
    assert(v::push<syntax::Parsed>(ref mut out,text(if(changed){b"module helper;pub fn T id<T>(T n){return n+0;}"}else{b"module helper;pub fn T id<T>(T n){return n;}"})).is_ok);return out;
}
@effects(console)
fn plan::Plan planned(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"fypc1-sha256-").is_ok);var usize i=0;while(i<64){assert(bytes::append_byte(ref mut text,49).is_ok);i+=1;}var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=bytes::len(ref text)}).is_ok);let coordinates=coordinates::assemble(text,rows);
    var v::Vec<usize> owners=v::empty<usize>();var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();i=0;while(i<v::len<syntax::Parsed>(sources)){assert(v::push<usize>(ref mut owners,0).is_ok);let source=v::at<syntax::Parsed>(sources,i);var usize j=1;while(j<syntax::declaration_count(source)){let d=syntax::declaration_at(source,j);if(d.kind==2){var usize measured=0;let target=signatures::import_target_measured(metadata,i,syntax::source_view(source)[d.name_start..d.name_end],ref mut measured);assert(target.code==0 && target.item>0);assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=i,.declaration=j+1,.target=target.item-1}).is_ok);}j+=1;}i+=1;}
    var context::Budget work=context::empty();let bound=switch(bindings::build(sources,ref owners,ref imports,ref coordinates,ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources,metadata);}};
    let catalog=switch(templates::build(bound,sources,ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources,metadata);}};
    let roots=switch(requests::build(catalog,sources,ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources,metadata);}};
    let input=switch(prepared::build(roots,sources,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.source.code);assert(false);return planned(sources,metadata);}};
    return switch(plan::build(input,plan::defaults(),ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);planned(sources,metadata)}};
}
@effects(console)
fn project::Plan detached(){let sources=source_set(true,false,false);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return detached();}};let original=planned(ref sources,ref metadata);assert(plan::count(ref original)==81);var usize work=0;return switch(project::build(ref sources,ref metadata,ref original,ref mut work)){case .ok(p):{assert(work>0);p}case .err(e):{discard io::write_stderr_u32(e.stage);discard io::write_stderr_u32(e.code);assert(false);detached()}};}
@test @effects(console)
fn void split_units_own_ir_ids_and_shared_specializations_after_sources_drop(){let compiled=detached();assert(project::file_count(ref compiled)==4 && project::count(ref compiled)==2 && project::retries(ref compiled)==1);var usize unit=0;var usize next=0;var usize functions=0;var usize shared=0;var usize calls=0;var bytes::Buffer shared_id=bytes::empty();
    while(unit<project::count(ref compiled)){let range=project::range_at(ref compiled,unit);assert(range.first==next && range.first<range.stop);next=range.stop;let owned=project::at(ref compiled,unit);let program=identified::ir_program(owned);var usize i=1;while(i<=ir::count(program)){let row=identified::frame_at(owned,i);let frame=clean::frame_at(ir::cleanup_ir(program),i);let id=identified::identity(owned,row);if(row.node!=0){assert(frame.kind==1 && id.len==67 && bytes::equal(id[..3],b"g0_"));functions+=1;if(frame.module_index==3){assert(frame.declaration==2);if(shared==0){assert(bytes::append(ref mut shared_id,id).is_ok);}else{assert(bytes::equal(bytes::view(ref shared_id),id));}shared+=1;}}else{assert(id.len==0);}
        let body=ir::at(program,i);var usize j=0;while(j<ir::call_count(body)){let call=ir::call_at(body,j);if(call.kind==2){var usize work=0;let target=switch(identified::call_frame(owned,i,j,ref mut work)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(target>0 && identified::frame_at(owned,target).node!=0);calls+=1;}j+=1;}i+=1;}unit+=1;}
    assert(next==4 && functions==82 && shared==2 && calls==82);
}
@test @effects(console)
fn void later_unit_failure_does_not_publish_checked_prefix(){let sources=source_set(true,true,false);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let original=planned(ref sources,ref metadata);let expected=switch(compile::range(ref sources,ref metadata,2,4)){case .ok(_):{assert(false);return;}case .err(e):{e}};assert(expected.stage==4 && expected.module_index==2 && expected.start<expected.end && !compile::capacity_failure(expected));var usize work=0;switch(project::build(ref sources,ref metadata,ref original,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==expected.stage && e.code==expected.code && e.module_index==expected.module_index && e.first==2 && e.stop==4 && e.start==expected.start && e.end==expected.end && e.expected==expected.expected && e.actual==expected.actual && work>0 && !compile::capacity_failure(e));}};}
@test @effects(console)
fn void attachment_work_and_stale_sources_do_not_retry_or_publish(){let sources=source_set(false,false,false);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let original=planned(ref sources,ref metadata);var usize measured=0;assert(switch(project::build(ref sources,ref metadata,ref original,ref mut measured)){case .ok(p):{project::count(ref p)==1 && project::retries(ref p)==0}case .err(_):{false}} && measured>0);var usize work=16777216-measured;assert(switch(project::build(ref sources,ref metadata,ref original,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==16777216);work=16777216-measured+1;switch(project::build(ref sources,ref metadata,ref original,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==8 && e.code==10 && !compile::capacity_failure(e));}};work=16777217;switch(project::build(ref sources,ref metadata,ref original,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==8 && e.code==10 && work==16777217);}};
    let changed=source_set(false,false,true);let changed_metadata=switch(signatures::build(ref changed)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;switch(project::build(ref changed,ref changed_metadata,ref original,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.stage==8 && e.code==12 && !compile::capacity_failure(e));}};
}

@effects(console)
fn linked::Program linked_program(){let sources=source_set(true,false,false);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return linked_program();}};let original=planned(ref sources,ref metadata);var usize work=0;let units=switch(project::build(ref sources,ref metadata,ref original,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return linked_program();}};work=0;return switch(linked::build(ref metadata,ref units,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(b"link=");discard io::write_stderr_u32(e.code);assert(false);linked_program()}};}
@test @effects(console)
fn void identified_project_link_deduplicates_calls_and_detaches_ids(){let owned=linked_program();let program=linked::ir_program(ref owned);assert(ir::count(program)==84 && proof::check(program).code==0);var usize i=1;var usize functions=0;var usize shared=0;var usize shared_calls=0;var usize shared_frame=0;
    while(i<=ir::count(program)){let frame=clean::frame_at(ir::cleanup_ir(program),i);let row=linked::frame_at(ref owned,i);let id=linked::identity(ref owned,row);if(row.node!=0){assert(frame.kind==1 && frame.context==i && id.len==67 && bytes::equal(id[..3],b"g0_"));functions+=1;if(frame.module_index==3){assert(frame.declaration==2);shared+=1;shared_frame=i;}}else{assert(id.len==0);}i+=1;}
    assert(functions==81 && shared==1 && shared_frame>0);i=1;while(i<=ir::count(program)){let body=ir::at(program,i);var usize j=0;while(j<ir::call_count(body)){let call=ir::call_at(body,j);var usize work=0;let target=switch(linked::call_frame(ref owned,i,j,ref mut work)){case .ok(n):{n}case .err(_):{assert(false);0}};assert(target>0);if(call.kind==2){assert(call.instance==target && linked::frame_at(ref owned,target).node!=0);if(call.target_module==3){assert(target==shared_frame);shared_calls+=1;}}j+=1;}i+=1;}assert(shared_calls==2);
    i=1;while(i<=clean::type_count(ir::cleanup_ir(program))){assert(linked::identity(ref owned,linked::type_at(ref owned,i)).len==0);i+=1;}
}
@test @effects(console)
fn void project_link_work_limits_and_disjoint_input_domains(){let sources=source_set(false,false,false);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let original=planned(ref sources,ref metadata);var usize work=0;let named=switch(project::build(ref sources,ref metadata,ref original,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize measured=0;assert(switch(linked::build(ref metadata,ref named,ref mut measured)){case .ok(_):{true}case .err(_):{false}} && measured>0);work=limits::project_work()-measured;assert(switch(linked::build(ref metadata,ref named,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==limits::project_work());work=limits::project_work()-measured+1;switch(linked::build(ref metadata,ref named,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1002);}};work=limits::project_work()+1;switch(linked::build(ref metadata,ref named,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1002 && work==limits::project_work()+1);}};
    var v::Vec<ir::Program> ordinary=v::empty<ir::Program>();let unit=switch(compile::range(ref sources,ref metadata,0,2)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<ir::Program>(ref mut ordinary,unit).is_ok);switch(linking::prepare_inputs(ref metadata,ref ordinary,ref named)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001);}};assert(switch(linking::build(ref metadata,ref ordinary)){case .ok(p):{proof::check(ref p).code==0 && ir::count(ref p)==2}case .err(_):{false}});
}
