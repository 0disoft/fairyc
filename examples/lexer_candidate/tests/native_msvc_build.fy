module tests::native_msvc_build;
import native_msvc_build as build;
import native_packet_decode as packet;
import native_msvc_artifacts as artifact;
import native_runtime_assets as assets;
import windows_msvc_toolchain as tools;
import windows_host_paths as host;
import windows_run_capture as child;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::text as text;
import std::fs as fs;
import std::io as io;

@effects(console) fn void failed(ref build::Failure e){discard io::write_stderr("MSVC build failure ");discard io::write_stderr_u32(build::stage(e));discard io::write_stderr("/");discard io::write_stderr_u32(build::status(e));discard io::write_stderr("/");discard io::write_stderr_u32(build::os_code(e));discard io::write_stderr(" unit ");discard io::write_stderr_usize(build::unit(e));discard io::write_stderr("\n");discard io::write_stderr_bytes(build::diagnostics(e));assert(false);}
fn bytes::Buffer copied(ref u8[] data){return switch(bytes::copy(data)){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};}
@effects(filesystem) fn bytes::Buffer path(){var u8[4096] raw=[0;4096];let usize n=switch(process::arg(1,ref mut raw)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref raw;return copied(view[..n]);}
fn bytes::Buffer suffix(ref u8[] path,ref u8[] suffix){var bytes::Buffer out=copied(path);assert(bytes::append(ref mut out,suffix).is_ok);return out;}
@effects(filesystem) fn packet::Packet loaded(ref u8[] path){return switch(packet::read(path)){case .ok(p):{p}case .err(_):{assert(false);return loaded(b"");}};}
@effects(filesystem) fn bytes::Buffer assembled_runtime(ref u8[] path,ref packet::Packet p){let asset_path=suffix(path,b".runtime.pack");let raw=switch(artifact::read(bytes::view(ref asset_path))){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};let bundle=switch(assets::load(raw)){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};return switch(assets::source(ref bundle,packet::header(p),false)){case .ok(p):{p}case .err(_):{assert(false);return bytes::empty();}};}
@effects(filesystem) fn void save(ref u8[] path,ref u8[] data){let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{assert(false);return;}};var fs::File file=switch(fs::create_truncate_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(fs::write_all(ref mut file,data).is_ok);assert(fs::close(file).is_ok);}
@test @effects(filesystem,console,environment,clock) fn void compiles_links_and_executes_both_profiles_without_go_companion(){
    let location=path();let p=loaded(bytes::view(ref location));let runtime=assembled_runtime(bytes::view(ref location),ref p);let t=switch(tools::resolve()){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let executable=switch(host::executable()){case .ok(p):{p}case .err(_):{assert(false);return;}};let name=bytes::view(ref executable);let release=name.len>=11 && bytes::equal(name[name.len-11..],b"release.exe");let root=suffix(bytes::view(ref location),b".images/");var usize i=0;
    while(i<2){let result=switch(build::build(ref t,ref p,bytes::view(ref runtime),build::Options{.release=i==1,.timeout_ms=60000,.stack_reserve=65536})){case .ok(p):{p}case .err(e):{failed(ref e);return;}};assert(build::object_count(ref result)==packet::unit_count(ref p)+2 && artifact::executable(build::data(ref result)));
        let leaf=if(release){if(i==0){b"release-dev.exe"}else{b"release-opt.exe"}}else{if(i==0){b"development-dev.exe"}else{b"development-opt.exe"}};var bytes::Buffer output=copied(bytes::view(ref root));assert(bytes::append(ref mut output,leaf).is_ok);save(bytes::view(ref output),build::data(ref result));
        let cwd=switch(host::current()){case .ok(p):{p}case .err(_):{assert(false);return;}};var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();let executed=switch(child::run_input(bytes::view(ref output),ref args,bytes::view(ref cwd),5000,1,b"")){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(child::status(ref executed)==42 && child::trap(ref executed)==0 && bytes::equal(child::stdout(ref executed),b"built by Fairy\n") && child::stderr(ref executed).len==0);i+=1;
    }
}
@test @effects(filesystem,console,environment,clock) fn void separates_compile_link_and_option_failures(){
    let location=path();let original=loaded(bytes::view(ref location));let runtime=assembled_runtime(bytes::view(ref location),ref original);let t=switch(tools::resolve()){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize i=0;
    while(i<2){let name=suffix(bytes::view(ref location),if(i==0){b".compile-error"}else{b".link-error"});let p=loaded(bytes::view(ref name));switch(build::build(ref t,ref p,bytes::view(ref runtime),build::Options{.release=false,.timeout_ms=60000,.stack_reserve=0})){
        case .ok(_):{assert(false);}case .err(e):{assert(build::stage(ref e)==if(i==0){6}else{8});assert(build::status(ref e)!=0 && build::diagnostics(ref e).len>0);}
    };i+=1;}
    let p=loaded(bytes::view(ref location));switch(build::build(ref t,ref p,bytes::view(ref runtime),build::Options{.release=false,.timeout_ms=0,.stack_reserve=0})){case .ok(_):{assert(false);}case .err(e):{assert(build::stage(ref e)==1);}};
}
@test fn void rejects_invalid_options_and_artifact_headers(){
    assert(build::valid(build::Options{.release=false,.timeout_ms=1,.stack_reserve=0}) && build::valid(build::Options{.release=true,.timeout_ms=600000,.stack_reserve=16777216}));
    assert(!build::valid(build::Options{.release=false,.timeout_ms=600001,.stack_reserve=0}) && !build::valid(build::Options{.release=false,.timeout_ms=1,.stack_reserve=1}) && !build::valid(build::Options{.release=false,.timeout_ms=1,.stack_reserve=16842752}));
    assert(!artifact::object(b"") && !artifact::object(b"not COFF") && !artifact::executable(b"") && !artifact::executable(b"MZ"));
}
