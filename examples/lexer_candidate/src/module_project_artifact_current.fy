module module_project_artifact_current;
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

struct Mapping{usize count;usize[512] modules;usize[512] owners;}
// Current source facts and source-planner handles, still in the artifact's IR
// order. Whole-IR renumbering and completed Flow/interface admission come next.
pub resource Rebound{artifact::Restored artifact;bindings::Bindings provenance;source::Plan plan;identities::Snapshot identities;Mapping mapping;cleanup{}}
pub fn ref artifact::Restored restored(ref Rebound p){return ref p.artifact;}
pub fn ref bindings::Bindings provenance(ref Rebound p){return ref p.provenance;}
pub fn ref source::Plan source_plan(ref Rebound p){return ref p.plan;}
pub fn ref identities::Snapshot source_identities(ref Rebound p){return ref p.identities;}
pub fn usize module_count(ref Rebound p){return p.mapping.count;}
pub fn usize module_at(ref Rebound p,usize old){assert(old<p.mapping.count);return p.mapping.modules[old]-1;}
pub fn usize owner_at(ref Rebound p,usize old){assert(old<512 && p.mapping.owners[old]!=0);return p.mapping.owners[old]-1;}
fn Result<Mapping,u32> map(ref stamps::Snapshot saved,ref bindings::Bindings current,ref mut usize work){var Mapping result=Mapping{.count=stamps::count(saved),.modules=[0;512],.owners=[0;512]};var usize i=0;
    while(i<result.count){let old=stamps::module_at(saved,i);var usize j=0;while(j<bindings::count(current)){if(!budget::charge(work,1)){return .err(10);}let now=bindings::module_at(current,j);let ns=bindings::text(current,now.namespace);let path=bindings::text(current,now.path);if(bytes::equal(stamps::text(saved,old.namespace),ns) && bytes::equal(stamps::text(saved,old.path),path)){if(!budget::charge(work,ns.len+path.len+1)){return .err(10);}if(result.modules[i]!=0 || result.owners[old.owner]!=0 && result.owners[old.owner]!=now.owner+1){return .err(9);}result.modules[i]=j+1;result.owners[old.owner]=now.owner+1;}j+=1;}if(result.modules[i]==0){return .err(9);}i+=1;
    }return .ok(result);
}
fn Result<usize,u32> selected(ref identities::Snapshot ids,selection::Row old,ref indexing::Index index,ref source::Plan plan,ref Mapping mapping,usize module_index,usize declaration,u32 kind,ref mut usize work){
    if(old.node==0){return .ok(0);}let node=switch(indexing::find(index,plan,identities::identity(ids,old),work)){case .ok(p):{p}case .err(code):{return .err(code);}};let row=source::at(plan,node-1);let original=templates::template_at(prepared::templates(source::input(plan)),row.template_index);if(module_index>=mapping.count || mapping.modules[module_index]==0 || original.module_index!=mapping.modules[module_index]-1 || original.declaration+1!=declaration || original.kind!=kind){return .err(9);}return .ok(node);
}
fn Result<identities::Snapshot,u32> select(ref artifact::Restored saved,ref source::Plan plan,ref indexing::Index index,ref Mapping mapping,ref v::Vec<syntax::Parsed> parsed,ref mut usize work){let ids=artifact::source_identities(saved);let context=ir::cleanup_ir(artifact::program(saved));let raw=identities::names(ids);if(!budget::charge(work,raw.len+1)){return .err(10);}let names=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return .err(11);}};var v::Vec<selection::Row> types=v::empty<selection::Row>();var v::Vec<selection::Row> frames=v::empty<selection::Row>();var usize i=1;
    while(i<=identities::type_count(ids)){if(!budget::charge(work,1)){return .err(10);}let old=identities::type_at(ids,i);let t=clean::type_at(context,i);let node=switch(selected(ids,old,index,plan,mapping,t.module_index,t.declaration,6,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(node!=0){if(t.kind!=13 || t.first==0 || t.first>=i){return .err(9);}var element=clean::type_at(context,t.first);if(element.kind==2 || element.kind==3){if(element.module_index>=mapping.count){return .err(9);}element.module_index=mapping.modules[element.module_index]-1;}let argument=switch(concrete::lookup_typed(prepared::types(source::input(plan)),element,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let current=source::at(plan,node-1);if(current.count!=1 || current.arguments[0]!=argument){return .err(9);}}
        if(!v::push<selection::Row>(ref mut types,selection::Row{.node,.id=old.id}).is_ok){return .err(11);}i+=1;
    }i=1;while(i<=identities::frame_count(ids)){if(!budget::charge(work,1)){return .err(10);}let old=identities::frame_at(ids,i);let frame=clean::frame_at(context,i);if(frame.module_index>=mapping.count || frame.declaration==0){return .err(9);}let p=v::at<syntax::Parsed>(parsed,mapping.modules[frame.module_index]-1);if(frame.declaration>syntax::declaration_count(p)){return .err(9);}let d=syntax::declaration_at(p,frame.declaration-1);if((old.node==0)!=(d.generic_parameters==0) || d.kind!=if(frame.kind==2){6}else{3}){return .err(9);}let node=switch(selected(ids,old,index,plan,mapping,frame.module_index,frame.declaration,d.kind,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(frame.kind==2 && (frame.receiver_type==0 || frame.receiver_type>v::len<selection::Row>(ref types) || node!=v::at<selection::Row>(ref types,frame.receiver_type-1).node)){return .err(9);}if(!v::push<selection::Row>(ref mut frames,selection::Row{.node,.id=old.id}).is_ok){return .err(11);}i+=1;
    }var usize measured=0;let result=switch(identities::assemble(names,types,frames,context,ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(10);}return .err(if(code==1002){10}else if(code==1003){11}else{9});}};if(!budget::charge(work,measured)){return .err(10);}return .ok(result);
}
pub fn Result<Rebound,u32> build(artifact::Restored saved,ref input::Input current,ref coordinates::Coordinates table,ref mut usize work){return build_sources(saved,input::sources(current),input::owners(current),input::bindings(current),table,work);}
pub fn Result<Rebound,u32> build_sources(artifact::Restored saved,ref v::Vec<syntax::Parsed> parsed,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}var usize measured=0;let shape=artifact::check(artifact::program(ref saved),artifact::source_provenance(ref saved),artifact::source_identities(ref saved),ref mut measured);if(!budget::charge(work,measured)){return .err(10);}if(shape!=0){return .err(if(shape==1002){10}else if(shape==1003){11}else{9});}
    let current=switch(sources::rebind(artifact::source_provenance(ref saved),parsed,owners,imports,table,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let plan=switch(planning::build_sources(parsed,owners,imports,table,work)){case .ok(p):{p}case .err(e):{return .err(e.code);}};let mapping=switch(map(artifact::source_provenance(ref saved),ref current,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let index=switch(indexing::build(ref plan,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let ids=switch(select(ref saved,ref plan,ref index,ref mapping,parsed,work)){case .ok(p):{p}case .err(code):{return .err(code);}};return .ok(Rebound{.artifact=saved,.provenance=current,.plan,.identities=ids,.mapping});
}
