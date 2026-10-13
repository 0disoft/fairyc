module module_project_c_owner;
import module_project_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import project_compiler_core as compiler;
import std::vec as v;

// One owner retains ordinary, identified or complete IR through all C partitions.
// Both borrow branches originate from this owner; no resource field is moved out.
pub resource Program{v::Vec<ir::Program> ordinary;v::Vec<identified::Program> identified;v::Vec<compiler::Complete> complete;cleanup{}}
fn clean::Problem failed(){return clean::Problem{.code=1003,.module_index=0,.start=0,.end=0};}
pub fn Result<Program,clean::Problem> plain(ir::Program program){var v::Vec<ir::Program> ordinary=v::empty<ir::Program>();if(!v::push<ir::Program>(ref mut ordinary,program).is_ok){return .err(failed());}return .ok(Program{.ordinary,.identified=v::empty<identified::Program>(),.complete=v::empty<compiler::Complete>()});}
pub fn Result<Program,clean::Problem> named(identified::Program program){var v::Vec<identified::Program> identified=v::empty<identified::Program>();if(!v::push<identified::Program>(ref mut identified,program).is_ok){return .err(failed());}return .ok(Program{.ordinary=v::empty<ir::Program>(),.identified,.complete=v::empty<compiler::Complete>()});}
pub fn Result<Program,clean::Problem> complete(compiler::Complete program){var v::Vec<compiler::Complete> complete=v::empty<compiler::Complete>();if(!v::push<compiler::Complete>(ref mut complete,program).is_ok){return .err(failed());}return .ok(Program{.ordinary=v::empty<ir::Program>(),.identified=v::empty<identified::Program>(),.complete});}
pub fn bool has_complete(ref Program p){return v::len<compiler::Complete>(ref p.complete)==1;}
pub fn ref compiler::Complete complete_project(ref Program p){assert(has_complete(p));return v::at<compiler::Complete>(ref p.complete,0);}
pub fn ref ir::Program ir_program(ref Program p){assert(v::len<ir::Program>(ref p.ordinary)+v::len<identified::Program>(ref p.identified)+v::len<compiler::Complete>(ref p.complete)==1);return if(v::len<ir::Program>(ref p.ordinary)==1){v::at<ir::Program>(ref p.ordinary,0)}else if(has_complete(p)){identified::ir_program(compiler::checked_program(complete_project(p)))}else{identified::ir_program(v::at<identified::Program>(ref p.identified,0))};}
