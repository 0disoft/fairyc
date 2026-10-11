module project_module_visibility;
import project_workspace_graph as graph;
import project_instance_graph as local_graph;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_source_modules as modules;
import project_source_set as sources;
import project_import_path as paths;
import project_path as work;
import project_sdk_bundle as bundled;
import project_sdk_embedded as embedded;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Lookup{usize owner;usize source_module;usize declaration;usize provider;usize provider_module;usize path_start;usize path_end;usize start;usize end;}
pub struct Range{usize first;usize count;}
pub struct Candidate{usize node;usize module_index;}
pub struct Problem{u32 code;u32 reason;usize owner;usize module_index;usize declaration;usize start;usize end;}
// 1 missing visible module; 2 ambiguous visible module; 3 allocation;
// 4 work bound; 5 invalid import declaration. Candidate indices are graph nodes.
pub resource Resolved{graph::Graph graph;bytes::Buffer names;v::Vec<Lookup> lookups;v::Vec<Range> ranges;cleanup{}}
pub resource Error{graph::Graph graph;Problem problem;bytes::Buffer path;v::Vec<Lookup> lookups;bytes::Buffer names;v::Vec<Candidate> providers;v::Vec<Candidate> hidden;cleanup{}}
pub resource LoadError{u32 phase;v::Vec<local_graph::LoadError> input;v::Vec<Error> visibility;v::Vec<graph::Error> sdk;cleanup{}}
fn Problem fail(u32 code,u32 reason,usize owner,usize module_index,usize declaration,usize start,usize end){return Problem{.code,.reason,.owner,.module_index,.declaration,.start,.end};}
pub fn ref str reason(u32 code){if(code==1){return "missing-visible-module";}if(code==2){return "ambiguous-visible-module";}if(code==3){return "allocation-failed";}if(code==4){return "implementation-bound";}return "invalid-import-declaration";}
pub fn Problem error(ref Error error){return error.problem;}
pub fn ref local_graph::Graph error_graph(ref Error error){return graph::local_graph(ref error.graph);}
pub fn ref graph::Graph error_workspace_graph(ref Error error){return ref error.graph;}
pub fn ref u8[] error_path(ref Error error){return bytes::view(ref error.path);}
pub fn ref u8[] error_file(ref Error error){return graph::module_file(ref error.graph,error.problem.owner,error.problem.module_index);}
pub fn ref u8[] error_source(ref Error error){return graph::module_source(ref error.graph,error.problem.owner,error.problem.module_index);}
pub fn usize provider_count(ref Error error){return v::len<Candidate>(ref error.providers);}
pub fn Candidate provider(ref Error error,usize at){assert(at<provider_count(error));return *v::at<Candidate>(ref error.providers,at);}
pub fn usize hidden_count(ref Error error){return v::len<Candidate>(ref error.hidden);}
pub fn Candidate hidden(ref Error error,usize at){assert(at<hidden_count(error));return *v::at<Candidate>(ref error.hidden,at);}
pub fn usize error_lookup_count(ref Error error){return v::len<Lookup>(ref error.lookups);}
pub fn ref local_graph::Graph instance_graph(ref Resolved resolved){return graph::local_graph(ref resolved.graph);}
pub fn ref graph::Graph workspace_graph(ref Resolved resolved){return ref resolved.graph;}
pub fn usize lookup_count(ref Resolved resolved){return v::len<Lookup>(ref resolved.lookups);}
pub fn Lookup lookup(ref Resolved resolved,usize at){assert(at<lookup_count(resolved));return *v::at<Lookup>(ref resolved.lookups,at);}
pub fn Range owner_range(ref Resolved resolved,usize at){assert(at<graph::node_count(ref resolved.graph));return *v::at<Range>(ref resolved.ranges,at);}
pub fn ref u8[] import_path(ref Resolved resolved,usize at){let Lookup l=lookup(resolved,at);return bytes::view(ref resolved.names)[l.path_start..l.path_end];}
pub fn u32 load_phase(ref LoadError error){return error.phase;}
pub fn ref local_graph::LoadError input_error(ref LoadError error){assert(error.phase==1);return v::at<local_graph::LoadError>(ref error.input,0);}
pub fn ref Error visibility_error(ref LoadError error){assert(error.phase==2);return v::at<Error>(ref error.visibility,0);}
pub fn ref graph::Error sdk_error(ref LoadError error){assert(error.phase==4);return v::at<graph::Error>(ref error.sdk,0);}
fn void number(ref mut v::Vec<usize> values,usize at,usize value){let ref mut usize slot=v::at_mut<usize>(values,at);*slot=value;}
// Heap-sort the direct providers by graph position, preserving own-instance first.
fn u32 sift(ref mut v::Vec<usize> values,usize root,usize size,ref mut usize meter){var usize cursor=root;while(cursor<size/2){if(!work::charge(meter,1)){return 4;}var usize child=cursor*2+1;if(child+1<size && *v::at<usize>(values,child+2)>*v::at<usize>(values,child+1)){child+=1;}let usize parent=*v::at<usize>(values,cursor+1);let usize next=*v::at<usize>(values,child+1);if(parent>=next){return 0;}number(values,cursor+1,next);number(values,child+1,parent);cursor=child;}return 0;}
fn u32 visible(ref graph::Graph graph,usize owner,ref mut v::Vec<usize> marks,ref mut v::Vec<usize> output,ref mut usize meter){
    discard v::clear<usize>(output);let usize stamp=owner+1;if(!v::push<usize>(output,owner).is_ok){return 3;}number(marks,owner,stamp);let n=graph::node(graph,owner);var usize at=0;
    while(at<n.count){if(!work::charge(meter,1)){return 4;}let usize target=graph::edge(graph,n.first+at).target;if(*v::at<usize>(marks,target)!=stamp){number(marks,target,stamp);if(!v::push<usize>(output,target).is_ok){return 3;}}at+=1;}
    let usize size=v::len<usize>(output)-1;var usize root=size/2;while(root>0){root-=1;let u32 code=sift(output,root,size,meter);if(code!=0){return code;}}var usize end=size;while(end>1){let usize value=*v::at<usize>(output,1);let usize tail=*v::at<usize>(output,end);number(output,1,tail);number(output,end,value);end-=1;let u32 code=sift(output,0,end,meter);if(code!=0){return code;}}return 0;
}
fn Result<usize,u32> provides(ref graph::Graph graph,usize node,ref u8[] path,ref mut usize meter){return graph::find_module(graph,node,path,meter);}
fn Problem bind(ref graph::Graph graph,ref mut bytes::Buffer names,ref mut v::Vec<Lookup> lookups,ref mut v::Vec<Range> ranges,ref mut bytes::Buffer failed_path,ref mut v::Vec<Candidate> providers,ref mut v::Vec<Candidate> hidden,ref mut usize meter){
    var v::Vec<usize> marks=v::empty<usize>();var v::Vec<usize> visible_nodes=v::empty<usize>();let usize count=graph::node_count(graph);if(!work::charge(meter,count)){return fail(4,0,0,0,0,0,0);}var usize at=0;while(at<count){if(!v::push<usize>(ref mut marks,0).is_ok){return fail(3,0,0,0,0,0,0);}at+=1;}
    var usize owner=0;while(owner<count){let u32 visible_code=visible(graph,owner,ref mut marks,ref mut visible_nodes,meter);if(visible_code!=0){return fail(visible_code,0,owner,0,0,0,0);}let usize first=v::len<Lookup>(lookups);var usize ordinal=0;
        while(ordinal<graph::module_count(graph,owner)){let usize module_index=graph::module_order(graph,owner,ordinal);let tree=graph::module_parser(graph,owner,module_index);var usize declaration=1;
            while(declaration<syntax::declaration_count(tree)){let d=syntax::declaration_at(tree,declaration);if(d.kind!=2){declaration+=1;continue;}let paths::Path path=switch(paths::read(tree,declaration,meter)){case .ok(value):{value}case .err(code):{return fail(if(code==2){3}else if(code==3){4}else{5},code,owner,module_index,declaration,d.start,d.end);}};
                discard v::clear<Candidate>(providers);var usize index=0;while(index<v::len<usize>(ref visible_nodes)){let usize node=*v::at<usize>(ref visible_nodes,index);let usize found=switch(provides(graph,node,paths::name(ref path),meter)){case .ok(value):{value}case .err(code):{return fail(4,code,owner,module_index,declaration,paths::start(ref path),paths::end(ref path));}};if(found!=0 && !v::push<Candidate>(providers,Candidate{.node,.module_index=found-1}).is_ok){return fail(3,0,owner,module_index,declaration,paths::start(ref path),paths::end(ref path));}index+=1;}
                if(v::len<Candidate>(providers)!=1){if(!bytes::append(failed_path,paths::name(ref path)).is_ok){return fail(3,0,owner,module_index,declaration,paths::start(ref path),paths::end(ref path));}if(v::len<Candidate>(providers)==0){index=0;while(index<count){if(!work::charge(meter,1)){return fail(4,0,owner,module_index,declaration,paths::start(ref path),paths::end(ref path));}if(*v::at<usize>(ref marks,index)!=owner+1){let usize found=switch(provides(graph,index,paths::name(ref path),meter)){case .ok(value):{value}case .err(code):{return fail(4,code,owner,module_index,declaration,paths::start(ref path),paths::end(ref path));}};if(found!=0 && !v::push<Candidate>(hidden,Candidate{.node=index,.module_index=found-1}).is_ok){return fail(3,0,owner,module_index,declaration,paths::start(ref path),paths::end(ref path));}}index+=1;}}return fail(if(v::len<Candidate>(providers)==0){1}else{2},0,owner,module_index,declaration,paths::start(ref path),paths::end(ref path));}
                let Candidate provider=*v::at<Candidate>(providers,0);let usize path_start=bytes::len(names);if(!bytes::append(names,paths::name(ref path)).is_ok || !v::push<Lookup>(lookups,Lookup{.owner,.source_module=module_index,.declaration,.provider=provider.node,.provider_module=provider.module_index,.path_start,.path_end=bytes::len(names),.start=paths::start(ref path),.end=paths::end(ref path)}).is_ok){return fail(3,0,owner,module_index,declaration,paths::start(ref path),paths::end(ref path));}declaration+=1;
            }ordinal+=1;
        }if(!v::push<Range>(ranges,Range{.first,.count=v::len<Lookup>(lookups)-first}).is_ok){return fail(3,0,owner,0,0,0,0);}owner+=1;
    }return fail(0,0,0,0,0,0,0);
}
pub fn Result<Resolved,Error> resolve(graph::Graph graph,ref mut usize meter){var bytes::Buffer names=bytes::empty();var bytes::Buffer path=bytes::empty();var v::Vec<Lookup> lookups=v::empty<Lookup>();var v::Vec<Range> ranges=v::empty<Range>();var v::Vec<Candidate> providers=v::empty<Candidate>();var v::Vec<Candidate> hidden=v::empty<Candidate>();let Problem p=bind(ref graph,ref mut names,ref mut lookups,ref mut ranges,ref mut path,ref mut providers,ref mut hidden,meter);if(p.code!=0){return .err(Error{.graph,.problem=p,.path,.lookups,.names,.providers,.hidden});}return .ok(Resolved{.graph,.names,.lookups,.ranges});}
fn LoadError input_failure(local_graph::LoadError e){var v::Vec<local_graph::LoadError> errors=v::empty<local_graph::LoadError>();let bool ok=v::push<local_graph::LoadError>(ref mut errors,e).is_ok;return LoadError{.phase=if(ok){1}else{3},.input=errors,.visibility=v::empty<Error>(),.sdk=v::empty<graph::Error>()};}
fn LoadError visibility_failure(Error e){var v::Vec<Error> errors=v::empty<Error>();let bool ok=v::push<Error>(ref mut errors,e).is_ok;return LoadError{.phase=if(ok){2}else{3},.input=v::empty<local_graph::LoadError>(),.visibility=errors,.sdk=v::empty<graph::Error>()};}
fn LoadError sdk_failure(graph::Error e){var v::Vec<graph::Error> errors=v::empty<graph::Error>();let bool ok=v::push<graph::Error>(ref mut errors,e).is_ok;return LoadError{.phase=if(ok){4}else{3},.input=v::empty<local_graph::LoadError>(),.visibility=v::empty<Error>(),.sdk=errors};}
@effects(filesystem)
pub fn Result<Resolved,LoadError> load(ref u8[] root,ref mut sources::Budget budget,ref mut usize meter){return load_embedded(root,false,budget,meter);}
@effects(filesystem)
pub fn Result<Resolved,LoadError> load_embedded(ref u8[] root,bool candidate,ref mut sources::Budget budget,ref mut usize meter){var sources::Budget staged=*budget;let local_graph::Graph local=switch(local_graph::load(root,ref mut staged,meter)){case .ok(value):{value}case .err(e):{return .err(input_failure(e));}};
    let local_graph::SDKSelection selected=switch(local_graph::select_sdk(ref local,candidate,meter)){case .ok(value):{value}case .err(p):{return .err(sdk_failure(graph::selection_failure(local,p)));}};
    let graph::Graph loaded=if(local_graph::sdk_profile(ref selected)==0){graph::without_sdk(local)}else{let bundled::Bundle library=switch(embedded::load(local_graph::sdk_profile(ref selected),meter)){case .ok(value):{value}case .err(p):{return .err(sdk_failure(graph::bundle_failure(local,p)));}};switch(graph::inject(local,library,meter)){case .ok(value):{value}case .err(e):{return .err(sdk_failure(e));}}};
    let Resolved result=switch(resolve(loaded,meter)){case .ok(value):{value}case .err(e):{return .err(visibility_failure(e));}};*budget=staged;return .ok(result);}
@effects(filesystem)
pub fn Result<Resolved,LoadError> load_sdk(ref u8[] root,bundled::Bundle library,ref mut sources::Budget budget,ref mut usize meter){var sources::Budget staged=*budget;let local_graph::Graph local=switch(local_graph::load(root,ref mut staged,meter)){case .ok(value):{value}case .err(e):{return .err(input_failure(e));}};let graph::Graph injected=switch(graph::inject(local,library,meter)){case .ok(value):{value}case .err(e):{return .err(sdk_failure(e));}};let Resolved result=switch(resolve(injected,meter)){case .ok(value):{value}case .err(e):{return .err(visibility_failure(e));}};*budget=staged;return .ok(result);}
