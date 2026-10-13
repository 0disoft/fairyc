module module_project_complete_c;
import module_unit_c_emit as c;
import module_project_c_owner as owner;
import module_unit_c_names as names;
import module_unit_cleanup_mir_model as clean;
import project_compiler_core as compiler;
import candidate_limits as limits;

// Keep the checked IR and detached interfaces together without adding the
// complete publication adapters to the large per-body C emitter source unit.
fn Result<c::Bundle,clean::Problem> partition_scoped(compiler::Complete program,usize entry,usize max_frames,bool entry_only){if(max_frames==0 || max_frames>limits::project_frames()){return .err(clean::Problem{.code=1001,.module_index=0,.start=0,.end=0});}var usize work=0;let named=switch(names::build_project(compiler::checked_program(ref program),ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};let owned=switch(owner::complete(program)){case .ok(p):{p}case .err(e):{return .err(e);}};return c::partition_scoped(owned,named,entry,max_frames,entry_only);}
pub fn Result<c::Bundle,clean::Problem> partition(compiler::Complete program,usize entry,usize max_frames){return partition_scoped(program,entry,max_frames,false);}
pub fn Result<c::Bundle,clean::Problem> partition_entry(compiler::Complete program,usize entry,usize max_frames){return partition_scoped(program,entry,max_frames,true);}
pub fn bool has_complete(ref c::Bundle p){return owner::has_complete(c::project_owner(p));}
pub fn ref compiler::Complete complete(ref c::Bundle p){return owner::complete_project(c::project_owner(p));}
