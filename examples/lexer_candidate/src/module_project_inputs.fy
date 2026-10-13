module module_project_inputs;
import module_project_identified_compile as named;
import module_identified_ir as identified;
import module_unit_body_ir_model as ir;
import std::vec as v;

// Count disjoint input domains; return borrows only from one named owner.
// Workers select their local source without storing a borrow in an aggregate.
pub fn usize count(ref v::Vec<ir::Program> ordinary,ref named::Plan source){assert(v::len<ir::Program>(ordinary)==0 || named::count(source)==0);return v::len<ir::Program>(ordinary)+named::count(source);}
pub fn ref ir::Program source_at(ref named::Plan source,usize index){assert(index<named::count(source));return identified::ir_program(named::at(source,index));}
