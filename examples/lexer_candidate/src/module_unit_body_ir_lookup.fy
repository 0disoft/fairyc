module module_unit_body_ir_lookup;
import module_unit_body_ir_model as ir;
// Compatibility imports for authored tests. Production uses the model directly.
pub fn usize node(ref ir::Body b,usize id){return ir::node(b,id);}
pub fn usize statement(ref ir::Body b,usize id){return ir::statement(b,id);}
pub fn usize local(ref ir::Body b,usize source_index){return ir::local(b,source_index);}
