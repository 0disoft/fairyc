module module_state_graph_frame_assets;
import module_state_graph_frame_pin as pin;
import project_cache_encoding as budget;
import sha256 as sha;
import std::bytes as bytes;

// Explicit detached distribution bytes. Only this module can construct or
// mutate the validated owner. Borrowing content never allocates or copies.
pub resource Bundle{bytes::Buffer data;cleanup{}}

// Private errors: 1001 invalid pin, 1002 bounds/work, 1003 allocation.
pub fn Result<Bundle,u32> load(bytes::Buffer data,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}
    let input=bytes::view(ref data);
    if(input.len>1048576){return .err(1002);}
    if(input.len!=pin::size()){return .err(1001);}
    if(pin::content_start()>pin::content_end() || pin::content_end()!=input.len || pin::content_end()-pin::content_start()>8388608){return .err(1002);}
    // Admit the complete hash and independent owned copy before doing either.
    if(!budget::charge(work,input.len*2+1)){return .err(1002);}
    if(!sha::matches(input,pin::digest())){return .err(1001);}
    let owned=switch(bytes::copy(input)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    return .ok(Bundle{.data=owned});
}

// Exact raw frame bytes, excluding the pack envelope. This immutable borrow
// lives only as long as its Bundle; the parent charges its own hash/emission.
pub fn ref u8[] content(ref Bundle bundle){return bytes::view(ref bundle.data)[pin::content_start()..pin::content_end()];}
