module project_pack_source;
import project_pack_options as options;
import project_pack as pack;
import project_pack_query as query_policy;
import project_atlas_source_version as source_version;
import project_atlas_facts as facts;
import project_atlas_instances as instances;
import project_atlas_produce as produce;
import project_atlas_schema as schema;
import project_compiler_source_mode as mode;
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
import std::vec as v;
import std::bytes as bytes;

// 1 workspace/lock, 2 unsupported/invalid source, 3 internal preparation,
// 4 selector/output budget. Successful output is private until final encoding.
pub struct Problem{u32 stage;u32 code;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code};}
fn Problem source_failure(u32 code){return fail(if(code>=101 && code<=104){2}else{3},code);}

// Derive the requested map family from current Complete facts, never from an
// existing map/cache or a child compiler. Unlocked requests use ordinary lock
// refresh policy; locked requests must not change the workspace.
@effects(filesystem,environment)
pub fn Result<bytes::Buffer,Problem> query(ref u8[] root,ref options::Options request){
    var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
    let loaded=switch(loader::load(root,options::locked(request),ref mut budget,ref mut meter)){case .ok(p):{p}case .err(_):{return .err(fail(1,0));}};
    let workspace=loader::workspace(ref loaded);let graph=visibility::workspace_graph(workspace);
    let version=options::version(request);
    let lock=switch(locks::project(workspace,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(1,e.code));}};
    let table=switch(coordinates::collect(graph,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(fail(3,e.code));}};
    let source=switch(input::project(workspace,ref mut meter)){case .ok(p):{p}case .err(e):{return .err(source_failure(e.code));}};
    switch(source_version::select(ref source,version,ref mut meter)){case .ok(_):{}case .err(code):{return .err(source_failure(code));}};
    let aggregates=switch(mode::aggregates(ref source,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(source_failure(code));}};
    var usize source_work=0;var usize link_work=0;
    let complete=switch(if(aggregates){compiler::build_complete_aggregate_sources(input::sources(ref source),input::owners(ref source),input::bindings(ref source),ref table,ref mut source_work,ref mut link_work)}else{compiler::build_complete(ref source,ref table,ref mut source_work,ref mut link_work)}){case .ok(p):{p}case .err(e):{return .err(fail(2,e.code));}};
    let program=identified::ir_program(compiler::checked_program(ref complete));
    let metadata=switch(if(aggregates){schema::prepare_aggregates(ref source,ref table,ref complete,ref mut meter)}else{schema::prepare(ref source,ref table,ref complete,ref mut meter)}){case .ok(p):{p}case .err(code):{return .err(source_failure(code));}};
    var rows=switch(schema::type_rows(ref metadata,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(source_failure(code));}};
    let observed=switch(if(aggregates){instances::collect_aggregates(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)}else{instances::collect(ref source,ref table,compiler::checked_program(ref complete),ref metadata,ref mut meter)}){case .ok(p):{p}case .err(code):{return .err(source_failure(code));}};
    var usize image_work=0;
    let build=switch(environment::compiler_current(ref mut image_work)){case .ok(p):{p}case .err(_):{return .err(fail(3,0));}};
    var packages=v::empty<produce::Package>();var usize selected=graph::node_count(graph);
    var usize matches=0;var usize owner=0;var usize total=0;var usize external=0;
    while(owner<graph::node_count(graph)){
        let f=switch(facts::collect_version(graph,ref lock,ref table,compiler::public_interfaces(ref complete),program,image::value(ref build),owner,version,instances::template_count(ref observed)>0,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(source_failure(code));}};
        let part=switch(produce::package_version(f,ref source,ref table,ref complete,ref metadata,ref mut rows,ref observed,owner,version,ref mut meter)){case .ok(p):{p}case .err(code):{return .err(source_failure(code));}};
        if(bytes::equal(produce::uid(ref part),options::uid(request))){selected=owner;matches+=1;}
        total+=produce::call_count(ref part);external+=produce::external_count(ref part);
        if(!v::push<produce::Package>(ref mut packages,part).is_ok){return .err(fail(3,2));}owner+=1;
    }
    if(matches!=1){return .err(fail(4,9));}
    // Match the existing SDK source/query ledger separation. All package facts
    // above come from the same Complete; admission checks the selected package.
    var usize query_work=if(graph::has_sdk(graph)){0}else{meter};
    let ready=switch(produce::ready(v::at<produce::Package>(ref packages,selected),total-external,external,ref mut query_work)){case .ok(p):{p}case .err(code):{return .err(fail(3,code));}};
    let action=options::action(request);
    if(bytes::equal(action,b"overview")){
        return switch(pack::encode_overview(ref ready,version,options::records(request),options::selection(request),options::output(request),ref mut query_work)){case .ok(p):{.ok(p)}case .err(code):{.err(fail(4,code))}};
    }
    var usize mask=0;var usize kind=0;while(kind<10){if(options::has_relation(request,kind)){mask|=1<<kind;}kind+=1;}
    let direction=options::direction(request);
    let policy=query_policy::Policy{.direction=if(bytes::equal(direction,b"outgoing")){0}else if(bytes::equal(direction,b"incoming")){1}else{2},.mask,.records=options::records(request),.relations=options::relations(request),.depth=options::depth(request),.selection=options::selection(request)};
    return switch(if(bytes::equal(action,b"show")){pack::encode_show(ref ready,version,options::symbol(request),policy,options::output(request),ref mut query_work)}else{pack::encode_around(ref ready,version,options::symbol(request),policy,options::output(request),ref mut query_work)}){case .ok(p):{.ok(p)}case .err(code):{.err(fail(4,code))}};
}
