module native_test_result;
import std::bytes as bytes;
// Platform-independent retained test outcome. OS exit/control checks precede construction.
pub resource Captured{u32 category;bytes::Buffer output;bytes::Buffer error;cleanup{}}
pub fn u32 trap(ref Captured value){return value.category;}
pub fn ref u8[] stdout(ref Captured value){return bytes::view(ref value.output);}
pub fn ref u8[] stderr(ref Captured value){return bytes::view(ref value.error);}
pub fn Result<Captured,u32> copy(u32 category,ref u8[] output,ref u8[] error){
    let out=switch(bytes::copy(output)){case .ok(p):{p}case .err(_):{return .err(4);}};
    let err=switch(bytes::copy(error)){case .ok(p):{p}case .err(_):{return .err(4);}};
    return .ok(Captured{.category,.output=out,.error=err});
}
