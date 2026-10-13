module project_cache_key;
import project_cache_encoding as enc;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

pub resource Environment{bytes::Buffer compiler_build;bytes::Buffer language_revision;bytes::Buffer mir_version;bytes::Buffer target_schema;bytes::Buffer abi_profile;bytes::Buffer toolchain_fingerprint;bytes::Buffer backend;bytes::Buffer runtime_abi;cleanup{}}
pub struct Facts{u8[32] standard_library;u8[32] package;u8[32] source;u8[32] current_interface;u8[32] current_body;}
pub struct Dependency{u8[32] coordinate;u8[32] interface_hash;}
pub resource Inputs{Environment environment;v::Vec<bytes::Buffer> profile_flags;Facts facts;v::Vec<Dependency> dependencies;v::Vec<bytes::Buffer> features;cleanup{}}
pub resource Identity{bytes::Buffer canonical;bytes::Buffer digest;cleanup{}}
pub fn Environment environment_of(bytes::Buffer compiler_build,bytes::Buffer language_revision,bytes::Buffer mir_version,bytes::Buffer target_schema,bytes::Buffer abi_profile,bytes::Buffer toolchain_fingerprint,bytes::Buffer backend,bytes::Buffer runtime_abi){return Environment{.compiler_build,.language_revision,.mir_version,.target_schema,.abi_profile,.toolchain_fingerprint,.backend,.runtime_abi};}
pub fn ref u8[] environment_field(ref Environment p,usize index){assert(index<8);return if(index==0){bytes::view(ref p.compiler_build)}else if(index==1){bytes::view(ref p.language_revision)}else if(index==2){bytes::view(ref p.mir_version)}else if(index==3){bytes::view(ref p.target_schema)}else if(index==4){bytes::view(ref p.abi_profile)}else if(index==5){bytes::view(ref p.toolchain_fingerprint)}else if(index==6){bytes::view(ref p.backend)}else{bytes::view(ref p.runtime_abi)};}
pub fn Inputs assemble(Environment environment,v::Vec<bytes::Buffer> profile_flags,Facts facts,v::Vec<Dependency> dependencies,v::Vec<bytes::Buffer> features){return Inputs{.environment,.profile_flags,.facts,.dependencies,.features};}
pub fn ref u8[] canonical(ref Identity value){return bytes::view(ref value.canonical);}
pub fn ref u8[] digest(ref Identity value){return bytes::view(ref value.digest);}
pub fn usize dependency_count(ref Inputs p){return v::len<Dependency>(ref p.dependencies);}
pub fn Dependency dependency_at(ref Inputs p,usize at){return *v::at<Dependency>(ref p.dependencies,at);}
fn u32 environment_check(ref Environment e,ref mut usize work){
    var u32 code=enc::string_check(bytes::view(ref e.compiler_build),work);if(code!=0){return code;}
    code=enc::string_check(bytes::view(ref e.language_revision),work);if(code!=0){return code;}
    code=enc::string_check(bytes::view(ref e.mir_version),work);if(code!=0){return code;}
    code=enc::string_check(bytes::view(ref e.target_schema),work);if(code!=0){return code;}
    code=enc::string_check(bytes::view(ref e.abi_profile),work);if(code!=0){return code;}
    code=enc::string_check(bytes::view(ref e.toolchain_fingerprint),work);if(code!=0){return code;}
    code=enc::string_check(bytes::view(ref e.backend),work);if(code!=0){return code;}
    return enc::string_check(bytes::view(ref e.runtime_abi),work);
}
fn u32 facts_check(ref Facts f,ref mut usize work){
    var u32 code=enc::digest_check(ref f.standard_library,work);if(code!=0){return code;}
    code=enc::digest_check(ref f.package,work);if(code!=0){return code;}
    code=enc::digest_check(ref f.source,work);if(code!=0){return code;}
    code=enc::digest_check(ref f.current_interface,work);if(code!=0){return code;}
    return enc::digest_check(ref f.current_body,work);
}
fn u32 validate(ref Inputs p,ref mut usize work){
    var u32 code=environment_check(ref p.environment,work);if(code!=0){return code;}
    code=facts_check(ref p.facts,work);if(code!=0){return code;}
    code=enc::set_check(ref p.profile_flags,work);if(code!=0){return code;}
    code=enc::set_check(ref p.features,work);if(code!=0){return code;}
    let count=v::len<Dependency>(ref p.dependencies);if(!enc::charge(work,count+1)){return 5;}var usize at=0;
    while(at<count){let current=v::at<Dependency>(ref p.dependencies,at);code=enc::digest_check(ref current.coordinate,work);if(code!=0){return code;}code=enc::digest_check(ref current.interface_hash,work);if(code!=0){return code;}
        if(at!=0){let previous=v::at<Dependency>(ref p.dependencies,at-1);if(!enc::charge(work,65)){return 5;}if(!enc::before(ref previous.coordinate,ref current.coordinate)){return 3;}}at+=1;}
    return 0;
}
fn ref u8[] marker(usize at){if(at==0){return b"fairyc.package-instance/v1";}if(at==2){return b"fairyc.package-coordinate/v1";}if(at==4){return b"fairyc.package-source/v1";}if(at==6){return b"fairyc.public-interface/v5";}if(at==8){return b"fairyc.contract-state-package-body/v1";}return b"sha256";}
fn u32 environment(ref mut bytes::Buffer out,ref Environment e,ref mut usize work){
    var u32 code=enc::string(out,bytes::view(ref e.compiler_build),work);if(code!=0){return code;}
    code=enc::string(out,bytes::view(ref e.language_revision),work);if(code!=0){return code;}
    code=enc::string(out,bytes::view(ref e.mir_version),work);if(code!=0){return code;}
    code=enc::string(out,bytes::view(ref e.target_schema),work);if(code!=0){return code;}
    code=enc::string(out,bytes::view(ref e.abi_profile),work);if(code!=0){return code;}
    code=enc::string(out,bytes::view(ref e.toolchain_fingerprint),work);if(code!=0){return code;}
    return enc::string(out,bytes::view(ref e.backend),work);
}
fn u32 facts(ref mut bytes::Buffer out,ref Facts f,ref mut usize work){
    var u32 code=enc::raw(out,ref f.standard_library,work);if(code!=0){return code;}
    code=enc::raw(out,ref f.package,work);if(code!=0){return code;}
    code=enc::raw(out,ref f.source,work);if(code!=0){return code;}
    code=enc::raw(out,ref f.current_interface,work);if(code!=0){return code;}
    return enc::raw(out,ref f.current_body,work);
}
// Exact Go packagecache v4 bytes, including the independently versioned state
// body identity. A successful key is a fact, not a checked artifact or hit.
pub fn Result<bytes::Buffer,u32> encode(ref Inputs p,ref mut usize work){
    var u32 code=validate(p,work);if(code!=0){return .err(code);}var bytes::Buffer out=bytes::empty();
    code=enc::raw(ref mut out,b"fairyc.package-cache-key\x00\x04",work);if(code!=0){return .err(code);}var usize at=0;
    while(at<10){code=enc::string(ref mut out,marker(at),work);if(code!=0){return .err(code);}at+=1;}
    code=environment(ref mut out,ref p.environment,work);if(code!=0){return .err(code);}
    code=enc::strings(ref mut out,ref p.profile_flags,work);if(code!=0){return .err(code);}
    code=enc::string(ref mut out,bytes::view(ref p.environment.runtime_abi),work);if(code!=0){return .err(code);}
    code=facts(ref mut out,ref p.facts,work);if(code!=0){return .err(code);}
    let count=v::len<Dependency>(ref p.dependencies);code=enc::length(ref mut out,count,work);if(code!=0){return .err(code);}at=0;
    while(at<count){let row=v::at<Dependency>(ref p.dependencies,at);code=enc::raw(ref mut out,ref row.coordinate,work);if(code!=0){return .err(code);}code=enc::raw(ref mut out,ref row.interface_hash,work);if(code!=0){return .err(code);}at+=1;}
    code=enc::strings(ref mut out,ref p.features,work);if(code!=0){return .err(code);}
    return .ok(out);
}
pub fn Result<Identity,u32> compute(ref Inputs p,ref mut usize work){
    let out=switch(encode(p,work)){case .ok(b):{b}case .err(code):{return .err(code);}};
    if(!enc::charge(work,bytes::len(ref out)+1+81)){return .err(5);}var sha::State state=sha::empty();if(!sha::update(ref mut state,bytes::view(ref out))){return .err(5);}var bytes::Buffer key=bytes::empty();
    if(!bytes::append(ref mut key,b"fycache4-sha256-").is_ok || !sha::append_hex(ref state,ref mut key)){return .err(4);}return .ok(Identity{.canonical=out,.digest=key});
}
pub fn bool valid_text(ref u8[] value){if(value.len!=80 || !bytes::equal(value[..16],b"fycache4-sha256-")){return false;}var usize at=16;while(at<80){let c=value[at];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}at+=1;}return true;}
