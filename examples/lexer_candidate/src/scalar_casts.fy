module scalar_casts;
import c_abi_types as c_types;
// Exact subset of hir.ValuePreservingIntegerCast for candidate scalar IDs.
// Exact i32/c::int conversion additionally requires the emitted signed-32 ABI check.
pub fn bool allowed(u32 source,u32 target){return c_types::allowed(source,target);}
