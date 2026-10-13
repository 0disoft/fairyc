module module_identified_c;
import module_identified_ir as identified;
import module_unit_c_emit as emit;
import module_unit_cleanup_mir_model as clean;
import std::bytes as bytes;

// Derive names from the same checked owner before transactional C rendering.
pub fn clean::Problem render(ref identified::Program program,usize entry,ref mut bytes::Buffer out){return emit::render_identified(program,entry,out);}
