module module_project_artifact_rebase_cleanup;
import module_project_artifact_current as current;
import module_unit_cleanup_mir_model as clean;
import module_types as types;
import module_program_artifact_wire as work;
import std::vec as v;
pub fn Result<clean::Program,u32> build(ref clean::Program p,ref current::Rebound mapping,ref mut usize used){var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<clean::Frame> frames=v::empty<clean::Frame>();var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();var v::Vec<clean::Field> fields=v::empty<clean::Field>();var v::Vec<clean::Operand> operands=v::empty<clean::Operand>();var v::Vec<clean::Drop> drops=v::empty<clean::Drop>();var usize i=1;
    while(i<=clean::type_count(p)){if(!work::charge(used,1)){return .err(1002);}var t=clean::type_at(p,i);if(t.kind==2 || t.kind==3 || t.kind==4 || t.kind==13){if(t.module_index>=current::module_count(mapping)){return .err(1001);}t.module_index=current::module_at(mapping,t.module_index);}if(!v::push<types::Type>(ref mut graph,t).is_ok || !v::push<clean::Glue>(ref mut glues,clean::glue_at(p,i)).is_ok){return .err(1003);}i+=1;}
    i=1;while(i<=clean::frame_count(p)){if(!work::charge(used,1)){return .err(1002);}var f=clean::frame_at(p,i);if(f.module_index>=current::module_count(mapping)){return .err(1001);}f.module_index=current::module_at(mapping,f.module_index);if(!v::push<clean::Frame>(ref mut frames,f).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<clean::field_count(p)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<clean::Field>(ref mut fields,clean::field_at(p,i)).is_ok){return .err(1003);}i+=1;}i=1;while(i<=clean::operand_count(p)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<clean::Operand>(ref mut operands,clean::operand_at(p,i)).is_ok){return .err(1003);}i+=1;}i=0;while(i<clean::drop_count(p)){if(!work::charge(used,1)){return .err(1002);}if(!v::push<clean::Drop>(ref mut drops,clean::drop_at(p,i)).is_ok){return .err(1003);}i+=1;}
    var usize measured=0;let result=switch(clean::assemble_artifact(graph,frames,glues,fields,operands,drops,clean::project(p),ref mut measured)){case .ok(p):{p}case .err(e):{if(!work::charge(used,measured)){return .err(1002);}return .err(e.code);}};if(!work::charge(used,measured)){return .err(1002);}return .ok(result);
}
