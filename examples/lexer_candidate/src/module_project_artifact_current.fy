module module_project_artifact_current;
import candidate_limits as limits;
import module_project_artifact_model as artifact;
import module_project_artifact_plan_index as indexing;
import module_source_artifact_model as stamps;
import module_source_artifact_rebind as sources;
import module_identity_artifact_model as identities;
import module_source_specializations as selection;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import generic_package_bindings as bindings;
import generic_source_plan as source;
import generic_source_plan_input as prepared;
import generic_source_catalog as templates;
import generic_concrete_types as concrete;
import project_source_plan as planning;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import project_compiler_input as input;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

struct Owners{usize[512] slots;}
resource Mapping{usize count;v::Vec<usize> modules;Owners owners;cleanup{}}
fn usize module_slot(ref Mapping p,usize index){assert(index<p.count);return *v::at<usize>(ref p.modules,index);}
// Current source facts and source-planner handles, still in the artifact's IR
// order. Whole-IR renumbering and completed Flow/interface admission come next.
pub resource Rebound{artifact::Restored artifact;bindings::Bindings provenance;source::Plan plan;identities::Snapshot identities;Mapping mapping;cleanup{}}
pub fn ref artifact::Restored restored(ref Rebound p){return ref p.artifact;}
pub fn ref bindings::Bindings provenance(ref Rebound p){return ref p.provenance;}
pub fn ref source::Plan source_plan(ref Rebound p){return ref p.plan;}
pub fn ref identities::Snapshot source_identities(ref Rebound p){return ref p.identities;}
pub fn usize module_count(ref Rebound p){return p.mapping.count;}
pub fn usize module_at(ref Rebound p,usize old){assert(old<p.mapping.count);return module_slot(ref p.mapping,old)-1;}
pub fn usize owner_at(ref Rebound p,usize old){assert(old<512 && p.mapping.owners.slots[old]!=0);return p.mapping.owners.slots[old]-1;}
fn Result<Mapping,u32> map(ref stamps::Snapshot saved,ref bindings::Bindings current,ref mut usize work){
    let count=stamps::count(saved);if(count>limits::source_files() || count!=bindings::count(current)){return .err(9);}
    var v::Vec<usize> modules=v::empty<usize>();if(!v::reserve<usize>(ref mut modules,count).is_ok){return .err(11);}var usize i=0;
    while(i<count){if(!v::push<usize>(ref mut modules,0).is_ok){return .err(11);}i+=1;}
    var usize[512] owners=[0;512];i=0;
    while(i<count){let old=stamps::module_at(saved,i);var usize j=0;while(j<bindings::count(current)){if(!budget::charge(work,1)){return .err(10);}let now=bindings::module_at(current,j);let ns=bindings::text(current,now.namespace);let path=bindings::text(current,now.path);if(bytes::equal(stamps::text(saved,old.namespace),ns) && bytes::equal(stamps::text(saved,old.path),path)){if(!budget::charge(work,ns.len+path.len+1)){return .err(10);}if(*v::at<usize>(ref modules,i)!=0 || owners[old.owner]!=0 && owners[old.owner]!=now.owner+1){return .err(9);}{let slot=v::at_mut<usize>(ref mut modules,i);*slot=j+1;}owners[old.owner]=now.owner+1;}j+=1;}if(*v::at<usize>(ref modules,i)==0){return .err(9);}i+=1;
    }return .ok(Mapping{.count,.modules,.owners=Owners{.slots=owners}});
}
fn Result<usize,u32> selected(ref identities::Snapshot ids,selection::Row old,ref indexing::Index index,ref source::Plan plan,ref Mapping mapping,usize module_index,usize declaration,u32 kind,ref mut usize work){
    if(old.node==0){return .ok(0);}let node=switch(indexing::find(index,plan,identities::identity(ids,old),work)){case .ok(p):{p}case .err(code):{return .err(code);}};let row=source::at(plan,node-1);let original=templates::template_at(prepared::templates(source::input(plan)),row.template_index);if(module_index>=mapping.count || module_slot(mapping,module_index)==0 || original.module_index!=module_slot(mapping,module_index)-1 || original.declaration+1!=declaration || original.kind!=kind){return .err(9);}return .ok(node);
}
// Authenticate the saved instance's ordered arguments against the freshly
// selected source node. Tuple handles and producer-local node numbers are not
// identities; aggregate arguments are the admitted scalar-only source subset.
fn u32 arguments(ref clean::Program context,usize id,ref source::Plan plan,usize node,ref Mapping mapping,ref mut usize work){
    let t=clean::type_at(context,id);let current=source::at(plan,node-1);if(t.first==0 || t.first>=id){return 9;}
    if(t.kind==13){var element=clean::type_at(context,t.first);if(element.kind==2 || element.kind==3){if(element.module_index>=mapping.count || module_slot(mapping,element.module_index)==0){return 9;}element.module_index=module_slot(mapping,element.module_index)-1;}let argument=switch(concrete::lookup_typed(prepared::types(source::input(plan)),element,work)){case .ok(p):{p}case .err(code):{return code;}};return if(current.count==1 && current.arguments[0]==argument){0}else{9};}
    if(t.kind!=20 && t.kind!=21){return 9;}var usize tail=t.first;var usize count=0;
    while(tail!=0){if(!budget::charge(work,1)){return 10;}if(tail>=id || count>=current.count || count>=8){return 9;}let tuple=clean::type_at(context,tail);if(tuple.kind!=22 || tuple.first==0 || tuple.first>=tail || tuple.second>=tail){return 9;}let scalar=clean::type_at(context,tuple.first);if(scalar.kind!=1){return 9;}let argument=switch(concrete::lookup_typed(prepared::types(source::input(plan)),scalar,work)){case .ok(p):{p}case .err(code):{return code;}};if(current.arguments[count]!=argument){return 9;}count+=1;tail=tuple.second;}
    return if(count==current.count){0}else{9};
}
fn Result<identities::Snapshot,u32> select(ref artifact::Restored saved,ref source::Plan plan,ref indexing::Index index,ref Mapping mapping,ref v::Vec<syntax::Parsed> parsed,ref mut usize work){let ids=artifact::source_identities(saved);let context=ir::cleanup_ir(artifact::program(saved));let raw=identities::names(ids);if(!budget::charge(work,raw.len+1)){return .err(10);}let names=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return .err(11);}};var v::Vec<selection::Row> types=v::empty<selection::Row>();var v::Vec<selection::Row> frames=v::empty<selection::Row>();var usize i=1;
    while(i<=identities::type_count(ids)){if(!budget::charge(work,1)){return .err(10);}let old=identities::type_at(ids,i);let t=clean::type_at(context,i);let node=switch(selected(ids,old,index,plan,mapping,t.module_index,t.declaration,if(t.kind==20){4}else{6},work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(node!=0){let code=arguments(context,i,plan,node,mapping,work);if(code!=0){return .err(code);}}
        if(!v::push<selection::Row>(ref mut types,selection::Row{.node,.id=old.id}).is_ok){return .err(11);}i+=1;
    }i=1;while(i<=identities::frame_count(ids)){if(!budget::charge(work,1)){return .err(10);}let old=identities::frame_at(ids,i);let frame=clean::frame_at(context,i);if(frame.module_index>=mapping.count || frame.declaration==0){return .err(9);}let p=v::at<syntax::Parsed>(parsed,module_slot(mapping,frame.module_index)-1);if(frame.declaration>syntax::declaration_count(p)){return .err(9);}let d=syntax::declaration_at(p,frame.declaration-1);if((old.node==0)!=(d.generic_parameters==0) || d.kind!=if(frame.kind==2){6}else{3}){return .err(9);}let node=switch(selected(ids,old,index,plan,mapping,frame.module_index,frame.declaration,d.kind,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(frame.kind==2 && (frame.receiver_type==0 || frame.receiver_type>v::len<selection::Row>(ref types) || node!=v::at<selection::Row>(ref types,frame.receiver_type-1).node)){return .err(9);}if(!v::push<selection::Row>(ref mut frames,selection::Row{.node,.id=old.id}).is_ok){return .err(11);}i+=1;
    }var usize measured=0;let result=switch(identities::assemble(names,types,frames,context,ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(10);}return .err(if(code==1002){10}else if(code==1003){11}else{9});}};if(!budget::charge(work,measured)){return .err(10);}return .ok(result);
}
pub fn Result<Rebound,u32> build(artifact::Restored saved,ref input::Input current,ref coordinates::Coordinates table,ref mut usize work){return build_sources(saved,input::sources(current),input::owners(current),input::bindings(current),table,work);}
pub fn Result<Rebound,u32> build_sources(artifact::Restored saved,ref v::Vec<syntax::Parsed> parsed,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table,ref mut usize work){
    return build_mode(saved,parsed,owners,imports,table,false,work);
}
pub fn Result<Rebound,u32> build_aggregate_sources(artifact::Restored saved,ref v::Vec<syntax::Parsed> parsed,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table,ref mut usize work){
    return build_mode(saved,parsed,owners,imports,table,true,work);
}
fn Result<Rebound,u32> build_mode(artifact::Restored saved,ref v::Vec<syntax::Parsed> parsed,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table,bool aggregates,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}var usize measured=0;let shape=artifact::check(artifact::program(ref saved),artifact::source_provenance(ref saved),artifact::source_identities(ref saved),ref mut measured);if(!budget::charge(work,measured)){return .err(10);}if(shape!=0){return .err(if(shape==1002){10}else if(shape==1003){11}else{9});}
    if(clean::aggregates(ir::cleanup_ir(artifact::program(ref saved)))!=aggregates){return .err(9);}
    let current=switch(sources::rebind(artifact::source_provenance(ref saved),parsed,owners,imports,table,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let plan=switch(if(aggregates){planning::build_aggregate_sources(parsed,owners,imports,table,work)}else{planning::build_sources(parsed,owners,imports,table,work)}){case .ok(p):{p}case .err(e):{return .err(e.code);}};let mapping=switch(map(artifact::source_provenance(ref saved),ref current,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let index=switch(indexing::build(ref plan,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let ids=switch(select(ref saved,ref plan,ref index,ref mapping,parsed,work)){case .ok(p):{p}case .err(code):{return .err(code);}};return .ok(Rebound{.artifact=saved,.provenance=current,.plan,.identities=ids,.mapping});
}
