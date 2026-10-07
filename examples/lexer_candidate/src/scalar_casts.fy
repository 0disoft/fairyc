module scalar_casts;
// Exact subset of hir.ValuePreservingIntegerCast for candidate scalar IDs.
// c_int and raw-pointer casts belong to the separate C ABI/unsafe admission.
pub fn bool allowed(u32 source,u32 target){return source==3 && (target==1 || target==2 || target==4 || target==5) || source==1 && target==2 || source==4 && (target==2 || target==5);}
