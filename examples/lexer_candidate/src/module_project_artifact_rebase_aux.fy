module module_project_artifact_rebase_aux;
import module_project_artifact_current as current;
import module_state_layout as state;
import module_unit_sdk_sources as sdk;
import module_program_artifact_wire as work;
import std::bytes as bytes;
import std::vec as v;
pub fn Result<state::Plan,u32> states(ref state::Plan p,ref current::Rebound mapping,ref mut usize used){if(!work::charge(used,state::names(p).len+1)){return .err(1002);}let names=switch(bytes::copy(state::names(p))){case .ok(p):{p}case .err(_):{return .err(1003);}};var v::Vec<state::Layout> layouts=v::empty<state::Layout>();var v::Vec<state::Field> fields=v::empty<state::Field>();var usize i=0;
    while(i<state::count(p)){if(!work::charge(used,1)){return .err(1002);}var row=state::at(p,i);if(row.module_index>=current::module_count(mapping)){return .err(1001);}row.module_index=current::module_at(mapping,row.module_index);row.owner=current::owner_at(mapping,row.owner);if(!v::push<state::Layout>(ref mut layouts,row).is_ok){return .err(1003);}i+=1;}i=0;while(i<state::field_count(p)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<state::Field>(ref mut fields,state::field_at(p,i)).is_ok){return .err(1003);}i+=1;}
    let result=state::assemble(names,layouts,fields);var usize measured=0;let checked=state::validate(ref result,ref mut measured);if(!work::charge(used,measured)){return .err(1002);}if(checked!=0){return .err(if(checked==10){1002}else if(checked==11){1003}else{1001});}return .ok(result);
}
// SDK rows and their owned byte ranges are rebuilt in current module order.
pub fn Result<sdk::Sources,u32> sdk_sources(ref sdk::Sources p,ref current::Rebound mapping,ref mut usize used){if(sdk::module_count(p)!=current::module_count(mapping)){return .err(1001);}var bytes::Buffer data=bytes::empty();var v::Vec<sdk::Unit> units=v::empty<sdk::Unit>();var usize module_index=0;
    while(module_index<current::module_count(mapping)){var usize i=0;while(i<sdk::count(p)){if(!work::charge(used,1)){return .err(1002);}let old=sdk::at(p,i);if(old.module_index>=current::module_count(mapping)){return .err(1001);}if(current::module_at(mapping,old.module_index)==module_index){let raw=sdk::source_at(p,i);if(!work::charge(used,raw.len+1)){return .err(1002);}let start=bytes::len(ref data);if(!bytes::append(ref mut data,raw).is_ok || !v::push<sdk::Unit>(ref mut units,sdk::Unit{.module_index,.sdk_id=old.sdk_id,.start,.end=bytes::len(ref data)}).is_ok){return .err(1003);}}i+=1;}module_index+=1;}
    var usize measured=0;let result=switch(sdk::assemble_artifact(current::module_count(mapping),units,data,ref mut measured)){case .ok(p):{p}case .err(e):{if(!work::charge(used,measured)){return .err(1002);}return .err(e.code);}};if(!work::charge(used,measured)){return .err(1002);}return .ok(result);
}
