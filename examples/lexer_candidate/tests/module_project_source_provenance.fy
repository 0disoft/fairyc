module tests::module_project_source_provenance;
import module_project_identified_compile as project;
import module_project_identified_ir as linked;
import module_project_link as linking;
import module_source_provenance as provenance;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_signatures as signatures;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import generic_package_bindings as bindings;
import generic_source_catalog as templates;
import generic_source_requests as requests;
import generic_source_plan_input as prepared;
import generic_source_plan as plan;
import generic_package_context as context;
import candidate_limits as limits;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){let owned=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};let scanned=switch(scanner::collect(owned,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"");}};return switch(syntax::parse(scanned,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(b"")}};}
fn v::Vec<syntax::Parsed> source_set(bool changed){var v::Vec<syntax::Parsed> out=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut out,parsed(if(changed){b"module root;import helper as h;fn i32 main(){return h::id<i32>(42);}fn i32 ordinary(){return 2;}"}else{b"module root;import helper as h;fn i32 main(){return h::id<i32>(42);}fn i32 ordinary(){return 1;}"})).is_ok);assert(v::push<syntax::Parsed>(ref mut out,parsed(b"module helper;pub fn T id<T>(T n){return n;}")).is_ok && v::push<syntax::Parsed>(ref mut out,parsed(b"module helper;pub fn T id<T>(T n){return n;}")).is_ok);return out;}
fn v::Vec<usize> owners(bool changed){var v::Vec<usize> out=v::empty<usize>();assert(v::push<usize>(ref mut out,0).is_ok && v::push<usize>(ref mut out,if(changed){2}else{1}).is_ok && v::push<usize>(ref mut out,if(changed){1}else{2}).is_ok);return out;}
fn v::Vec<catalog::Binding> imports(bool changed){var v::Vec<catalog::Binding> out=v::empty<catalog::Binding>();assert(v::push<catalog::Binding>(ref mut out,catalog::Binding{.source=0,.declaration=2,.target=if(changed){2}else{1}}).is_ok);return out;}
fn coordinates::Coordinates coordinate_table(bool changed,bool reverse){var bytes::Buffer names=bytes::empty();var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();var usize i=0;while(i<3){let owner=if(reverse){2-i}else{i};let start=bytes::len(ref names);assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize j=0;while(j<64){assert(bytes::append_byte(ref mut names,if(changed && owner==1){52}else{if(owner==0){49}else if(owner==1){50}else{51}}).is_ok);j+=1;}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start,.end=bytes::len(ref names)}).is_ok);i+=1;}return coordinates::assemble(names,rows);}
@effects(console)
fn signatures::Program make_metadata(ref v::Vec<syntax::Parsed> sources,bool owner_change,bool provider_change){let owners=owners(owner_change);let imports=imports(provider_change);let catalog=switch(catalog::build_scoped(sources,ref owners,ref imports)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return make_metadata(sources,owner_change,provider_change);}};return switch(signatures::build_catalog(sources,catalog)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);make_metadata(sources,owner_change,provider_change)}};}
@effects(console)
fn plan::Plan planned(ref v::Vec<syntax::Parsed> sources){let owners=owners(false);let imports=imports(false);let coordinates=coordinate_table(false,false);var context::Budget work=context::empty();let bound=switch(bindings::build(sources,ref owners,ref imports,ref coordinates,ref mut work.work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);return planned(sources);}};let catalog=switch(templates::build(bound,sources,ref mut work.work)){case .ok(p):{p}case .err(_):{assert(false);return planned(sources);}};let roots=switch(requests::build(catalog,sources,ref mut work.work)){case .ok(p):{p}case .err(_):{assert(false);return planned(sources);}};let input=switch(prepared::build(roots,sources,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return planned(sources);}};return switch(plan::build(input,plan::defaults(),ref mut work.work)){case .ok(p):{p}case .err(_):{assert(false);planned(sources)}};}
@effects(console)
fn project::Plan compile_units(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let original=planned(sources);var usize work=0;return switch(project::build(sources,metadata,ref original,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.stage);discard io::write_stderr_u32(e.code);assert(false);compile_units(sources,metadata)}};}
@effects(console)
fn linked::Program detached(){let sources=source_set(false);let metadata=make_metadata(ref sources,false,false);let units=compile_units(ref sources,ref metadata);var usize work=0;return switch(linked::build(ref metadata,ref units,ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);assert(false);detached()}};}
@test @effects(console)
fn void linked_owner_keeps_current_source_namespaces_and_resolved_imports(){let p=detached();assert(proof::check(linked::ir_program(ref p)).code==0);let stamps=linked::source_provenance(ref p);assert(bindings::count(stamps)==3 && bindings::import_count(stamps)==1);let row=bindings::import_at(stamps,0);assert(row.module_index==0 && row.target==1 && bytes::equal(bindings::text(stamps,row.alias),b"h"));assert(bindings::module_at(stamps,0).owner==0 && bindings::module_at(stamps,1).owner==1 && bindings::module_at(stamps,2).owner==2);
 let sources=source_set(false);let metadata=make_metadata(ref sources,false,false);let coordinates=coordinate_table(false,false);var usize work=0;assert(linked::current(ref p,ref sources,ref metadata,ref coordinates,ref mut work)==0 && work>0);let reverse=coordinate_table(false,true);work=0;assert(linked::current(ref p,ref sources,ref metadata,ref reverse,ref mut work)==0);
 let changed=source_set(true);let changed_metadata=make_metadata(ref changed,false,false);work=0;assert(linked::current(ref p,ref changed,ref changed_metadata,ref coordinates,ref mut work)==12);work=0;assert(linked::current(ref p,ref changed,ref metadata,ref coordinates,ref mut work)==12);
 let moved=make_metadata(ref sources,true,false);work=0;assert(linked::current(ref p,ref sources,ref moved,ref coordinates,ref mut work)==9);let rebound=make_metadata(ref sources,false,true);work=0;assert(linked::current(ref p,ref sources,ref rebound,ref coordinates,ref mut work)==9);
 let wrong=coordinate_table(true,false);work=0;assert(linked::current(ref p,ref sources,ref metadata,ref wrong,ref mut work)==9);let missing=coordinates::assemble(bytes::empty(),v::empty<coordinates::Coordinate>());work=0;assert(linked::current(ref p,ref sources,ref metadata,ref missing,ref mut work)==9);
}
@test @effects(console)
fn void stale_metadata_rejects_preparation_linking_and_prepared_reuse(){let sources=source_set(false);let metadata=make_metadata(ref sources,false,false);let units=compile_units(ref sources,ref metadata);let ordinary=v::empty<ir::Program>();let ready=switch(linking::prepare_inputs(ref metadata,ref ordinary,ref units)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let changed=source_set(true);let changed_metadata=make_metadata(ref changed,false,false);var usize work=0;switch(linked::build(ref changed_metadata,ref units,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001 && work==0);}};switch(linking::prepare_inputs(ref changed_metadata,ref ordinary,ref units)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001);}};work=0;switch(linking::program_inputs(ref changed_metadata,ref ordinary,ref units,ref ready,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001 && work>0);}};
 let rebound=make_metadata(ref sources,false,true);work=0;switch(linked::build(ref rebound,ref units,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001);}};let moved=make_metadata(ref sources,true,false);work=0;switch(linked::build(ref moved,ref units,ref mut work)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001);}};
 let coordinates=coordinate_table(false,false);work=0;let linked=switch(linked::build(ref metadata,ref units,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;assert(linked::current(ref linked,ref sources,ref metadata,ref coordinates,ref mut work)==0);
}
@test @effects(console)
fn void detached_copy_and_current_source_checks_retain_exact_work_limits(){let p=detached();let sources=source_set(false);let metadata=make_metadata(ref sources,false,false);let coordinates=coordinate_table(false,false);var usize required=0;assert(linked::current(ref p,ref sources,ref metadata,ref coordinates,ref mut required)==0 && required>0);var usize work=16777216-required;assert(linked::current(ref p,ref sources,ref metadata,ref coordinates,ref mut work)==0 && work==16777216);work=16777216-required+1;assert(linked::current(ref p,ref sources,ref metadata,ref coordinates,ref mut work)==10);work=16777217;assert(linked::current(ref p,ref sources,ref metadata,ref coordinates,ref mut work)==10 && work==16777217);
 required=0;let copy=switch(bindings::copy(linked::source_provenance(ref p),ref mut required)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bindings::count(ref copy)==3 && required>0);work=16777216-required;assert(switch(bindings::copy(ref copy,ref mut work)){case .ok(_):{true}case .err(_):{false}} && work==16777216);work=16777216-required+1;switch(bindings::copy(ref copy,ref mut work)){case .ok(_):{assert(false);}case .err(code):{assert(code==10);}};
 work=0;assert(provenance::match_metadata(ref copy,ref metadata,ref mut work)==0);let empty=bindings::empty();work=0;assert(provenance::match_metadata(ref empty,ref metadata,ref mut work)==9);
}
