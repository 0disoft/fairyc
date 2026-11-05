module project_atlas_source;
import project_atlas_facts as facts;
import project_atlas_instances as instances;
import project_atlas_produce as produce;
import project_atlas_schema as schema;
import project_atlas_current as current;
import project_atlas_join as join;
import project_atlas_impact as impact;
import project_atlas_impact_json as output;
import project_atlas_options as options;
import project_workspace_lock as loader;
import project_lock_document as locks;
import project_coordinate_graph as coordinates;
import project_module_visibility as visibility;
import project_workspace_graph as graph;
import project_source_set as sources;
import project_compiler_input as input;
import project_compiler_core as compiler;
import project_cache_environment as environment;
import project_compiler_image_hash as image;
import module_project_identified_ir as identified;
import module_unit_body_ir_model as ir;
import std::vec as v;
import std::bytes as bytes;

pub struct Problem{u32 stage;u32 code;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code};}
// Always freshly resolve and verify the existing lock. No cache/map file or Go
// child supplies current facts. All successful data comes from this Complete.
@effects(filesystem,environment) pub fn Result<bytes::Buffer,Problem> query(ref u8[] root,ref options::Options o){var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
 let loaded=switch(loader::load(root,true,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(fail(1,0));}};let workspace=loader::workspace(ref loaded);let graph=visibility::workspace_graph(workspace);if(graph::node_count(graph)>options::packages(o)){return .err(fail(4,3));}if(graph::has_sdk(graph)){return .err(fail(2,100));}
 let lock=switch(locks::project(workspace,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(1,e.code));}};let table=switch(coordinates::collect(graph,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};let projected=switch(input::project(workspace,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};var usize source_work=0;var usize link_work=0;let complete=switch(compiler::build_complete(ref projected,ref table,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};let program=identified::ir_program(compiler::checked_program(ref complete));var usize image_work=0;let build=switch(environment::compiler_current(ref mut image_work)){case .ok(p):{p}case .err(_):{return .err(fail(3,0));}};
 let metadata=switch(schema::prepare(ref projected,ref table,ref complete,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};var type_rows=switch(schema::type_rows(ref metadata,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
 let observed=switch(instances::collect(ref projected,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};
 var v::Vec<produce::Package> packages=v::empty<produce::Package>();var usize owner=0;var usize total_calls=0;var usize total_external=0;var usize selected=graph::node_count(graph);var usize matches=0;
 while(owner<graph::node_count(graph)){let f=switch(facts::collect(graph,ref lock,ref table,compiler::public_interfaces(ref complete),program,image::value(ref build),owner,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};let p=switch(produce::package_generics(f,ref projected,program,ref table,ref metadata,ref mut type_rows,ref observed,owner,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(2,code));}};if(bytes::equal(produce::uid(ref p),options::uid(o))){matches+=1;selected=owner;}total_calls+=produce::call_count(ref p);total_external+=produce::external_count(ref p);if(total_calls-total_external>options::calls(o)){return .err(fail(4,3));}if(!v::push<produce::Package>(ref mut packages,p).is_ok){return .err(fail(3,2));}owner+=1;}
 if(matches!=1){return .err(fail(4,9));}let selected_instance=produce::instance(v::at<produce::Package>(ref packages,selected));var v::Vec<current::Ready> admitted=v::empty<current::Ready>();owner=0;while(owner<v::len<produce::Package>(ref packages)){let ready=switch(produce::ready(v::at<produce::Package>(ref packages,owner),total_calls-total_external,total_external,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(3,code));}};if(!v::push<current::Ready>(ref mut admitted,ready).is_ok){return .err(fail(3,2));}owner+=1;}
 let joined=switch(join::join(admitted,options::packages(o),options::calls(o),ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(4,code));}};let subject=switch(join::resolve(ref joined,selected_instance,options::symbol(o),ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(4,code));}};let result=switch(impact::explore(ref joined,subject,options::depth(o),options::symbols(o),ref mut meter)){case .ok(p):{p}case .err(code):{return .err(fail(4,code));}};return switch(output::render(ref joined,ref result,options::output(o),ref mut meter)){case .ok(p):{.ok(p)}case .err(code):{.err(fail(4,code))}};
}
