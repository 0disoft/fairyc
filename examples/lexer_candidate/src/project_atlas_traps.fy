module project_atlas_traps;
import module_unit_body_ir_model as ir;
import project_atlas_drop_sites as observed;
import project_atlas_trap_kinds as kinds;
import std::vec as v;

// Independent trap-only entry points retain their own count policy. The shared
// owner also offers a combined drop/trap batch with one proof and traversal.
pub resource Collection{v::Vec<u32> frames;cleanup{}}
pub fn usize frame_count(ref Collection p){return v::len<u32>(ref p.frames);}
pub fn u32 at(ref Collection p,usize index){return *v::at<u32>(ref p.frames,index);}
pub fn ref u8[] category(usize index){return kinds::category(index);}
pub fn bool contains(u32 mask,usize index){return kinds::contains(mask,index);}
pub fn Result<u32,u32> collect(ref ir::Program program,usize frame,usize maximum,ref mut usize meter){return observed::collect_traps(program,frame,maximum,meter);}
pub fn Result<Collection,u32> collect_all(ref ir::Program program,usize maximum,ref mut usize meter){let frames=observed::collect_all_traps(program,maximum,meter)?;return .ok(Collection{.frames});}
