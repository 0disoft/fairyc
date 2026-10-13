module project_generic_cache_key;
import project_cache_key as base;
import project_cache_encoding as enc;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

pub struct Dependency{u8[32] coordinate;u8[32] interface_hash;}
struct Interface{u8[32] hash;}
pub resource Inputs{base::Inputs ordinary;Interface current_interface;v::Vec<Dependency> dependencies;v::Vec<bytes::Buffer> specializations;cleanup{}}
pub resource Identity{bytes::Buffer canonical;bytes::Buffer digest;cleanup{}}
pub fn Inputs assemble(base::Inputs ordinary,u8[32] current_interface,v::Vec<Dependency> dependencies,v::Vec<bytes::Buffer> specializations){return Inputs{.ordinary,.current_interface=Interface{.hash=current_interface},.dependencies,.specializations};}
pub fn ref u8[] canonical(ref Identity value){return bytes::view(ref value.canonical);}
pub fn ref u8[] digest(ref Identity value){return bytes::view(ref value.digest);}
pub fn bool specialization_text(ref u8[] value){if(value.len!=67 || !bytes::equal(value[..3],b"g0_")){return false;}var usize at=3;while(at<67){let c=value[at];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}at+=1;}return true;}
fn u32 validate(ref Inputs p,ref mut usize work){
    // A named local array supplies the raw view without borrowing resource fields.
    let current=p.current_interface.hash;var u32 code=enc::digest_check(ref current,work);if(code!=0){return code;}
    let count=v::len<Dependency>(ref p.dependencies);if(!enc::charge(work,count+1)){return 5;}if(count!=base::dependency_count(ref p.ordinary)){return 3;}var usize at=0;
    while(at<count){let row=v::at<Dependency>(ref p.dependencies,at);let expected=base::dependency_at(ref p.ordinary,at);code=enc::digest_check(ref row.interface_hash,work);if(code!=0){return code;}if(!enc::charge(work,65)){return 5;}if(!bytes::equal(ref row.coordinate,ref expected.coordinate)){return 3;}at+=1;}
    let size=v::len<bytes::Buffer>(ref p.specializations);if(!enc::charge(work,size+1)){return 5;}at=0;
    while(at<size){let id=bytes::view(v::at<bytes::Buffer>(ref p.specializations,at));if(id.len>=16777216 || !enc::charge(work,id.len+1)){return 5;}if(!specialization_text(id)){return 1;}
        if(at!=0){let previous=bytes::view(v::at<bytes::Buffer>(ref p.specializations,at-1));if(!enc::charge(work,previous.len+id.len+1)){return 5;}if(!enc::before(previous,id)){return 3;}}at+=1;}
    return 0;
}
// Exact Go generic-package key v0: the complete ordinary v4 tuple is embedded,
// followed by source-generic interfaces and sorted IDs actually emitted here.
// This fact encoder does not validate a cached artifact or current body owner.
pub fn Result<Identity,u32> compute(ref Inputs p,ref mut usize work){
    let ordinary=switch(base::encode(ref p.ordinary,work)){case .ok(b):{b}case .err(code):{return .err(code);}};
    var u32 code=validate(p,work);if(code!=0){return .err(code);}var bytes::Buffer out=bytes::empty();code=enc::string(ref mut out,b"fairyc.generic-package-cache-key/v0",work);if(code!=0){return .err(code);}
    code=enc::length(ref mut out,bytes::len(ref ordinary),work);if(code!=0){return .err(code);}code=enc::raw(ref mut out,bytes::view(ref ordinary),work);if(code!=0){return .err(code);}
    let current=p.current_interface.hash;code=enc::raw(ref mut out,ref current,work);if(code!=0){return .err(code);}let count=v::len<Dependency>(ref p.dependencies);code=enc::length(ref mut out,count,work);if(code!=0){return .err(code);}var usize at=0;
    while(at<count){let row=v::at<Dependency>(ref p.dependencies,at);code=enc::raw(ref mut out,ref row.coordinate,work);if(code!=0){return .err(code);}code=enc::raw(ref mut out,ref row.interface_hash,work);if(code!=0){return .err(code);}at+=1;}
    code=enc::strings(ref mut out,ref p.specializations,work);if(code!=0){return .err(code);}if(!enc::charge(work,bytes::len(ref out)+1+82)){return .err(5);}var sha::State state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref out))){return .err(5);}var bytes::Buffer key=bytes::empty();if(!bytes::append(ref mut key,b"fycacheg0-sha256-").is_ok || !sha::append_hex(ref state,ref mut key)){return .err(4);}return .ok(Identity{.canonical=out,.digest=key});
}
pub fn bool valid_text(ref u8[] value){if(value.len!=81 || !bytes::equal(value[..17],b"fycacheg0-sha256-")){return false;}var usize at=17;while(at<81){let c=value[at];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}at+=1;}return true;}
