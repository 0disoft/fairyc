module project_compiler_identity;
import project_compiler_image_hash as image;
import windows_host_paths as paths;
import project_cache_encoding as budget;
import std::bytes as bytes;

// Windows obtains the actual module filename through its existing host adapter,
// never argv[0], PATH, a companion executable or a caller-supplied build label.
@effects(filesystem,environment)
pub fn Result<image::Identity,image::Problem> current(ref mut usize work){
    if(!budget::charge(work,0)){return .err(image::Problem{.stage=5,.status=0,.os_code=0,.completed=0});}
    let path=switch(paths::executable()){case .ok(p):{p}case .err(e):{return .err(image::Problem{.stage=7,.status=e.kind,.os_code=e.os_code,.completed=0});}};
    return image::read_path(bytes::view(ref path),work);
}
