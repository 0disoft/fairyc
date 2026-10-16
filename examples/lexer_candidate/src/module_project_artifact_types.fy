module module_project_artifact_types;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as cleanup_ir;
import module_unit_cleanup_mir_types as identity;
import module_unit_checked_bodies as checking;
import module_signatures as signatures;
import module_unit_values as values;
import module_types as types;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::vec as v;

// This detached index describes verified IR in current module coordinates.
// A successful unit map proves only its checked type descriptors. Body, call,
// layout and whole-project coverage checks still precede Complete/cache reuse.
pub resource Index {v::Vec<types::Type> types;v::Vec<usize> slots;cleanup{}}
pub resource Mapping {v::Vec<usize> ids;v::Vec<u8> used;cleanup{}}
pub fn usize type_count(ref Index p){return v::len<types::Type>(ref p.types);}
pub fn types::Type type_at(ref Index p,usize id){assert(id>0 && id<=type_count(p));return *v::at<types::Type>(ref p.types,id-1);}
pub fn usize count(ref Mapping p){return v::len<usize>(ref p.ids);}
pub fn usize target_count(ref Mapping p){return v::len<u8>(ref p.used);}
pub fn usize mapped(ref Mapping p,usize id){return if(id==0 || id>count(p)){0}else{*v::at<usize>(ref p.ids,id-1)};}
pub fn bool used(ref Mapping p,usize id){return id>0 && id<=target_count(p) && *v::at<u8>(ref p.used,id-1)!=0;}
pub fn Result<Index,u32> build(ref ir::Program p,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}var usize measured=0;let valid=proof::check_metered(p,ref mut measured);if(!budget::charge(work,measured)){return .err(1002);}if(valid.code!=0){return .err(valid.code);}
    var v::Vec<types::Type> rows=v::empty<types::Type>();var v::Vec<usize> slots=v::empty<usize>();var usize i=0;
    while(i<8192){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut slots,0).is_ok){return .err(1003);}i+=1;}i=1;
    while(i<=cleanup_ir::type_count(ir::cleanup_ir(p))){if(!budget::charge(work,1)){return .err(1002);}let row=cleanup_ir::type_at(ir::cleanup_ir(p),i);var usize slot=identity::bucket(row);var usize probes=0;
        while(*v::at<usize>(ref slots,slot)!=0){if(probes>=8192 || !budget::charge(work,1)){return .err(1002);}if(identity::same(row,*v::at<types::Type>(ref rows,*v::at<usize>(ref slots,slot)-1))){return .err(1001);}slot=(slot+1)&8191;probes+=1;}
        if(!v::push<types::Type>(ref mut rows,row).is_ok){return .err(1003);}let cell=v::at_mut<usize>(ref mut slots,slot);*cell=i;i+=1;
    }return .ok(Index{.types=rows,.slots});
}
fn Result<usize,u32> find(ref Index p,types::Type row,ref mut usize work){var usize slot=identity::bucket(row);var usize probes=0;
    while(probes<8192){if(!budget::charge(work,1)){return .err(1002);}let id=*v::at<usize>(ref p.slots,slot);if(id==0){return .err(1001);}if(identity::same(row,type_at(p,id))){return .ok(id);}slot=(slot+1)&8191;probes+=1;}return .err(1002);
}
pub fn Result<Mapping,u32> match(ref Index p,ref checking::Plan checked,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}var usize measured=0;let current=checking::validate_sources_catalog(checked,sources,metadata,ref mut measured);if(!budget::charge(work,measured)){return .err(1002);}if(current.code!=0){return .err(if(current.code==10){1002}else if(current.code==11){1003}else{1001});}
    let typed=checking::checked_values(checked);if(values::type_count(typed)==0 || values::type_count(typed)>4096){return .err(1001);}var v::Vec<usize> ids=v::empty<usize>();var v::Vec<u8> used=v::empty<u8>();var usize i=0;
    while(i<type_count(p)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<u8>(ref mut used,0).is_ok){return .err(1003);}i+=1;}i=1;
    while(i<=values::type_count(typed)){if(!budget::charge(work,1)){return .err(1002);}var row=values::type_at(typed,i);
        if(row.first!=0){if(row.first>=i){return .err(1001);}row.first=*v::at<usize>(ref ids,row.first-1);}
        // Result's second component is a type. Arrays retain their exact length;
        // pointer scalar bits, source nominal coordinates and Vec origins stay.
        if(row.kind==6){if(row.second==0 || row.second>=i){return .err(1001);}row.second=*v::at<usize>(ref ids,row.second-1);}
        let id=switch(find(p,row,work)){case .ok(id):{id}case .err(code):{return .err(code);}};if(!v::push<usize>(ref mut ids,id).is_ok){return .err(1003);}let marked=v::at_mut<u8>(ref mut used,id-1);*marked=1;i+=1;
    }return .ok(Mapping{.ids,.used});
}
