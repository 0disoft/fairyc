module project_cache_profile;
import module_unit_body_ir_model as ir;
import project_sdk_embedded as embedded;
import project_sdk_bundle as sdk;
import project_instance_identity as identity;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// This private profile describes checked candidate IR, not native C objects.
// The existing candidate driver selects Windows MSVC; callers cannot substitute
// a target, library digest, ABI string or backend for the current profile.
pub resource Profile{identity::Identity standard_library;cleanup{}}
pub fn ref u8[] language(){return sdk::language();}
pub fn ref u8[] runtime_abi(){return sdk::runtime_abi();}
pub fn ref u8[] mir_version(){return ir::cache_revision();}
pub fn ref u8[] target(){return b"x86_64-windows-msvc";}
pub fn ref u8[] target_schema(){return b"fairyc.target-spec/v0";}
pub fn ref u8[] abi_profile(){return b"fairyc.target-abi-profile/0;x86_64-windows-msvc;llp64;microsoft-x64;msvc-x64-llp64;microsoft-x64;coff-x86-64;windows-x64-pdata-xdata";}
pub fn ref u8[] backend(){return b"verified-candidate-owned-body-ir";}
pub fn ref u8[] toolchain(){return b"not-applicable:verified-candidate-owned-body-ir";}
pub fn ref u8[] standard_library(ref Profile p){return identity::digest(ref p.standard_library);}
// Like Go's semantic-cache environment, this always identifies the complete
// ordinary bundled SDK, even when the loaded graph omits it or selects Vec.
pub fn Result<Profile,u32> current(ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}let library=switch(embedded::load(1,work)){case .ok(p):{p}case .err(e):{return .err(if(e.code==5){10}else if(e.code==4){11}else{9});}};
    let features=v::empty<bytes::Buffer>();let id=switch(identity::compute(sdk::uid(),sdk::source_identity(ref library),sdk::version(),ref features,work)){case .ok(p):{p}case .err(code):{return .err(if(code==6){10}else if(code==5){11}else{9});}};
    return .ok(Profile{.standard_library=id});
}
