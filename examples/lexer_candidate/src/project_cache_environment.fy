module project_cache_environment;
import windows_host_paths as paths;
import project_compiler_image_hash as image;
import project_cache_profile as profile;
import project_cache_key as key;
import project_cache_encoding as budget;
import std::bytes as bytes;

pub resource Environment{image::Identity compiler;profile::Profile profile;cleanup{}}
pub struct Problem{u32 phase;u32 code;image::Problem image;}
fn Problem error(u32 phase,u32 code){return Problem{.phase,.code,.image=image::Problem{.stage=0,.status=0,.os_code=0,.completed=0}};}
pub fn ref u8[] compiler_build(ref Environment p){return image::value(ref p.compiler);}
pub fn ref u8[] standard_library(ref Environment p){return profile::standard_library(ref p.profile);}
@effects(filesystem,environment)
pub fn Result<Environment,Problem> current(ref mut usize work){
    if(!budget::charge(work,0)){return .err(error(1,10));}let build=switch(compiler_current(work)){case .ok(p):{p}case .err(e):{return .err(Problem{.phase=1,.code=if(e.stage==5){10}else if(e.stage==6){11}else{9},.image=e});}};
    let selected=switch(profile::current(work)){case .ok(p):{p}case .err(code):{return .err(error(2,code));}};return .ok(Environment{.compiler=build,.profile=selected});
}
fn Result<bytes::Buffer,u32> copy(ref u8[] value,ref mut usize work){if(!budget::charge(work,value.len+1)){return .err(10);}return switch(bytes::copy(value)){case .ok(p):{.ok(p)}case .err(_):{.err(11)}};}
// Detached compatibility facts for the existing v4 encoder. This does not read,
// validate or admit any cached package; artifact authority remains separate.
pub fn Result<key::Environment,u32> key_environment(ref Environment p,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}let build=switch(copy(compiler_build(p),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let language=switch(copy(profile::language(),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let mir=switch(copy(profile::mir_version(),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let target=switch(copy(profile::target_schema(),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let abi=switch(copy(profile::abi_profile(),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let toolchain=switch(copy(profile::toolchain(),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let backend=switch(copy(profile::backend(),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let runtime=switch(copy(profile::runtime_abi(),work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    return .ok(key::environment_of(build,language,mir,target,abi,toolchain,backend,runtime));
}

// Windows obtains the actual module filename through its existing host adapter,
// never argv[0], PATH, a companion executable or a caller-supplied build label.
@effects(filesystem,environment)
pub fn Result<image::Identity,image::Problem> compiler_current(ref mut usize work){
    if(!budget::charge(work,0)){return .err(image::Problem{.stage=5,.status=0,.os_code=0,.completed=0});}
    let path=switch(paths::executable()){case .ok(p):{p}case .err(e):{return .err(image::Problem{.stage=7,.status=e.kind,.os_code=e.os_code,.completed=0});}};
    return image::read_path(bytes::view(ref path),work);
}
