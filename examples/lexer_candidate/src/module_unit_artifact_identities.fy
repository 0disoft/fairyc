module module_unit_artifact_identities;
import module_identified_ir as unit;
import module_identity_artifact_model as model;
import module_source_specializations as selection;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_program_artifact_wire as budget;
import std::bytes as bytes;
import std::vec as v;

// Reattach current original-source identities before comparing saved IR. Copy
// each identity separately so the detached snapshot has exact byte coverage.
pub fn Result<model::Snapshot,u32> extract(ref unit::Program p,ref mut usize work){
    let context=ir::cleanup_ir(unit::ir_program(p));var bytes::Buffer names=bytes::empty();var v::Vec<selection::Row> types=v::empty<selection::Row>();var v::Vec<selection::Row> frames=v::empty<selection::Row>();var usize i=1;
    while(i<=clean::type_count(context)+unit::frame_count(p)){if(!budget::charge(work,1)){return .err(1002);}var row=if(i<=clean::type_count(context)){unit::type_at(p,i)}else{unit::frame_at(p,i-clean::type_count(context))};
        if(row.node!=0){let text=unit::identity(p,row);if(!budget::charge(work,text.len+1)){return .err(1002);}let start=bytes::len(ref names);if(!bytes::append(ref mut names,text).is_ok){return .err(1003);}row.id.start=start;row.id.end=bytes::len(ref names);}
        if(i<=clean::type_count(context)){if(!v::push<selection::Row>(ref mut types,row).is_ok){return .err(1003);}}else{if(!v::push<selection::Row>(ref mut frames,row).is_ok){return .err(1003);}}i+=1;
    }return model::assemble(names,types,frames,context,work);
}
