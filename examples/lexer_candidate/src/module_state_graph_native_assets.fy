module module_state_graph_native_assets;
import module_state_graph_native_asset_pin as pin;
import project_cache_encoding as budget;
import sha256 as sha;
import std::bytes as bytes;

// Explicit distribution bytes only. The private owner has no mutable accessor,
// source lookup, fallback, process invocation, or global selector admission.
pub resource Bundle{bytes::Buffer data;cleanup{}}

// Private errors: 1001 invalid pin, 1002 bounds/work, 1003 allocation.
pub fn Result<Bundle,u32> load(bytes::Buffer data,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}
    let input=bytes::view(ref data);
    if(input.len>1048576){return .err(1002);}
    if(input.len!=pin::size()){return .err(1001);}
    // Charge both hash input and the independent owned copy before doing either.
    if(!budget::charge(work,input.len*2+1)){return .err(1002);}
    if(!sha::matches(input,pin::digest())){return .err(1001);}
    let owned=switch(bytes::copy(input)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    return .ok(Bundle{.data=owned});
}

fn Result<bytes::Buffer,u32> section(ref Bundle bundle,usize start,usize end,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}
    let input=bytes::view(ref bundle.data);
    // Fixed offsets are used only inside the hash-validated, immutable owner.
    if(start>end || end>input.len || end-start>8388608){return .err(1002);}
    if(!budget::charge(work,end-start+1)){return .err(1002);}
    return switch(bytes::copy(input[start..end])){case .ok(p):{.ok(p)}case .err(_):{.err(1003)}};
}
pub fn Result<bytes::Buffer,u32> declarations(ref Bundle bundle,ref mut usize work){return section(bundle,pin::declarations_start(),pin::declarations_end(),work);}
pub fn Result<bytes::Buffer,u32> runtime(ref Bundle bundle,ref mut usize work){return section(bundle,pin::runtime_start(),pin::runtime_end(),work);}
