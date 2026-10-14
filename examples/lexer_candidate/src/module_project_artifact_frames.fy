module module_project_artifact_frames;
import module_project_artifact_types as types;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_value_model as body;
import module_identity_artifact_model as identities;
import module_source_specializations as selection;
import module_public_interface_model as text;
import generic_source_plan as source;
import module_signatures as signatures;
import effect_declarations as effects;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

struct Row {clean::Frame frame;u32 category;text::Text id;}
pub resource Index {types::Index types;v::Vec<Row> rows;v::Vec<usize> slots;bytes::Buffer names;cleanup{}}
// Frame/type maps and current specialization selection precede body/call and
// whole-project coverage checking. They do not authorize Complete/cache reuse.
pub resource Mapping {types::Mapping types;selection::Selection selected;v::Vec<usize> frames;v::Vec<u8> used;cleanup{}}
pub fn usize count(ref Index p){return v::len<Row>(ref p.rows);}
pub fn clean::Frame frame_at(ref Index p,usize id){assert(id>0 && id<=count(p));return v::at<Row>(ref p.rows,id-1).frame;}
pub fn ref types::Index type_index(ref Index p){return ref p.types;}
pub fn usize unit_count(ref Mapping p){return v::len<usize>(ref p.frames);}
pub fn usize target_count(ref Mapping p){return v::len<u8>(ref p.used);}
pub fn usize mapped(ref Mapping p,usize id){return if(id==0 || id>unit_count(p)){0}else{*v::at<usize>(ref p.frames,id-1)};}
pub fn bool used(ref Mapping p,usize id){return id>0 && id<=target_count(p) && *v::at<u8>(ref p.used,id-1)!=0;}
pub fn ref types::Mapping type_mapping(ref Mapping p){return ref p.types;}
pub fn ref selection::Selection selected(ref Mapping p){return ref p.selected;}
fn ref u8[] identity(ref Index p,Row row){return bytes::view(ref p.names)[row.id.start..row.id.end];}
fn usize bucket(usize module_index,usize declaration,u32 category,ref u8[] id,usize mask){var usize result=(module_index*104729 ^ declaration*3266489917 ^ category as usize*31)&mask;var usize i=0;while(i<id.len){result=(result*33+(id[i] as usize))&mask;i+=1;}return result;}
fn bool same_row(ref bytes::Buffer names,Row row,usize module_index,usize declaration,u32 category,ref u8[] id){return row.frame.module_index==module_index && row.frame.declaration==declaration && row.category==category && bytes::equal(bytes::view(names)[row.id.start..row.id.end],id);}
fn u32 translated(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
pub fn Result<Index,u32> build(ref ir::Program program,ref identities::Snapshot ids,ref mut usize work){
    let indexed=switch(types::build(program,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var usize measured=0;let valid=identities::check(ids,ir::cleanup_ir(program),ref mut measured);if(!budget::charge(work,measured)){return .err(1002);}if(valid!=0){return .err(valid);}
    var usize size=1;while(size<ir::count(program)*2){size*=2;}var v::Vec<usize> slots=v::empty<usize>();var usize i=0;while(i<size){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut slots,0).is_ok){return .err(1003);}i+=1;}
    var v::Vec<Row> rows=v::empty<Row>();var bytes::Buffer names=bytes::empty();i=1;
    while(i<=ir::count(program)){let frame=clean::frame_at(ir::cleanup_ir(program),i);let original=identities::frame_at(ids,i);let id=identities::identity(ids,original);let u32 category=if(frame.kind==2){3}else if(original.node!=0){2}else{1};if(!budget::charge(work,id.len+1)){return .err(1002);}let start=bytes::len(ref names);if(!bytes::append(ref mut names,id).is_ok){return .err(1003);}let row=Row{.frame,.category,.id=text::Text{.start,.end=bytes::len(ref names)}};
        var usize slot=bucket(frame.module_index,frame.declaration,category,id,size-1);var usize probes=0;while(*v::at<usize>(ref slots,slot)!=0){if(probes>=size || !budget::charge(work,id.len+1)){return .err(1002);}if(same_row(ref names,*v::at<Row>(ref rows,*v::at<usize>(ref slots,slot)-1),frame.module_index,frame.declaration,category,id)){return .err(1001);}slot=(slot+1)&(size-1);probes+=1;}
        if(!v::push<Row>(ref mut rows,row).is_ok){return .err(1003);}let cell=v::at_mut<usize>(ref mut slots,slot);*cell=i;i+=1;
    }return .ok(Index{.types=indexed,.rows,.slots,.names});
}
fn Result<usize,u32> find(ref Index p,body::Frame frame,u32 category,ref u8[] id,ref mut usize work){let size=v::len<usize>(ref p.slots);var usize slot=bucket(frame.module_index,frame.declaration,category,id,size-1);var usize probes=0;
    while(probes<size){if(!budget::charge(work,id.len+1)){return .err(1002);}let found=*v::at<usize>(ref p.slots,slot);if(found==0){return .err(1001);}if(same_row(ref p.names,*v::at<Row>(ref p.rows,found-1),frame.module_index,frame.declaration,category,id)){return .ok(found);}slot=(slot+1)&(size-1);probes+=1;}return .err(1002);
}
pub fn Result<Mapping,u32> match(ref Index p,ref checking::Plan checked,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    let mapped_types=switch(types::match(ref p.types,checked,sources,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var usize measured=0;let chosen=switch(selection::build(plan,sources,metadata,checking::checked_interfaces(checked),ref mut measured)){case .ok(p):{p}case .err(e):{if(!budget::charge(work,measured)){return .err(1002);}return .err(translated(e.code));}};if(!budget::charge(work,measured)){return .err(1002);}
    var v::Vec<usize> frames=v::empty<usize>();var v::Vec<u8> used=v::empty<u8>();var usize i=0;while(i<count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<u8>(ref mut used,0).is_ok){return .err(1003);}i+=1;}let typed=checking::checked_values(checked);i=0;
    while(i<values::count(typed)){if(!budget::charge(work,1)){return .err(1002);}let value=values::at(typed,i);let frame=body::frame(value);if(frame.module_index>=v::len<syntax::Parsed>(sources) || frame.declaration==0 || frame.declaration>syntax::declaration_count(v::at<syntax::Parsed>(sources,frame.module_index))){return .err(1001);}var selection::Row row=selection::Row{.node=0,.id=text::Text{.start=0,.end=0}};
        if(frame.kind==1){if(frame.context==0 || frame.context>selection::function_count(ref chosen)){return .err(1001);}row=selection::function_at(ref chosen,frame.context);}else if(frame.kind==2){if(frame.context==0 || frame.context>selection::cleanup_count(ref chosen)){return .err(1001);}row=selection::cleanup_at(ref chosen,frame.context);}else if(frame.kind!=3){return .err(1001);}
        let parsed=v::at<syntax::Parsed>(sources,frame.module_index);let declaration=syntax::declaration_at(parsed,frame.declaration-1);if((row.node!=0)!=(declaration.generic_parameters!=0) || declaration.kind!=if(frame.kind==2){6}else{3}){return .err(1001);}if(!budget::charge(work,declaration.end-declaration.start+1)){return .err(1002);}let authority=effects::read(parsed,frame.declaration-1);if(authority.code!=0){return .err(1001);}
        var usize receiver=0;var usize receiver_type=0;if(frame.kind==2){var usize j=1;while(j<=body::local_count(value)){if(!budget::charge(work,1)){return .err(1002);}let local=body::local_at(value,j);if(local.kind==10){if(receiver!=0){return .err(1001);}receiver=j;receiver_type=types::mapped(ref mapped_types,local.type);if(local.type==0 || local.type>selection::type_count(ref chosen) || selection::type_at(ref chosen,local.type).node!=row.node){return .err(1001);}}j+=1;}if(receiver==0 || receiver_type==0){return .err(1001);}}
        let u32 category=if(frame.kind==2){3}else if(row.node!=0){2}else{1};let target=switch(find(p,frame,category,selection::identity(ref chosen,row),work)){case .ok(id):{id}case .err(code):{return .err(code);}};let old=frame_at(p,target);let result=types::mapped(ref mapped_types,frame.result);if(result==0 || old.result!=result || old.effects!=authority.bits || old.receiver!=receiver || old.receiver_type!=receiver_type || *v::at<u8>(ref used,target-1)!=0){return .err(1001);}
        if(!v::push<usize>(ref mut frames,target).is_ok){return .err(1003);}let marked=v::at_mut<u8>(ref mut used,target-1);*marked=1;i+=1;
    }return .ok(Mapping{.types=mapped_types,.selected=chosen,.frames,.used});
}
