module project_package_input;
import project_source_set as sources;
import project_source_modules as modules;
import project_source_file as files;
import project_manifest as manifest;
import project_source_identity as identity;
import std::bytes as bytes;

// Phases 0..5 preserve the source loader's codes and span scopes. Phase 6
// carries a module-binding problem over the original source file. Phase 7 is
// diagnostic snapshot allocation failure. File/byte budgets commit only once.
pub struct Problem{u32 phase;u32 reason;u32 diagnostic;i32 status;u32 field;u32 os_code;usize file;usize start;usize end;}
pub resource Error{Problem problem;bytes::Buffer path;bytes::Buffer input;cleanup{}}
fn Problem allocation(){return Problem{.phase=7,.reason=1,.diagnostic=0,.status=3,.field=0,.os_code=0,.file=0,.start=0,.end=0};}
fn Error capture(Problem problem,ref u8[] path,ref u8[] input){
    let bytes::Buffer owned_path=switch(bytes::copy(path)){case .ok(value):{value}case .err(_):{return Error{.problem=allocation(),.path=bytes::empty(),.input=bytes::empty()};}};
    let bytes::Buffer owned_input=switch(bytes::copy(input)){case .ok(value):{value}case .err(_):{return Error{.problem=allocation(),.path=owned_path,.input=bytes::empty()};}};return Error{.problem,.path=owned_path,.input=owned_input};
}
fn Error source_error(ref sources::Error error,ref u8[] document){let p=sources::error(error);return capture(Problem{.phase=p.phase,.reason=p.reason,.diagnostic=0,.status=0,.field=p.field,.os_code=p.os_code,.file=0,.start=p.start,.end=p.end},sources::error_path(error),if(p.phase==1 || p.phase==2){document}else{b""});}
fn Error module_error(ref modules::Error error){let p=modules::error(error);let original=modules::error_sources(error);return capture(Problem{.phase=6,.reason=p.code,.diagnostic=p.diagnostic,.status=p.status,.field=0,.os_code=0,.file=p.file,.start=p.start,.end=p.end},sources::path(original,p.file),sources::source(original,p.file));}
pub fn Problem error(ref Error error){return error.problem;}
pub fn ref u8[] error_path(ref Error error){return bytes::view(ref error.path);}
pub fn ref u8[] error_input(ref Error error){return bytes::view(ref error.input);}
pub fn ref manifest::Manifest package(ref modules::Modules input){return sources::typed_manifest(modules::source_set(input));}
pub fn Result<bytes::Buffer,u32> source_digest(ref modules::Modules input){return identity::compute(modules::source_set(input));}

@effects(filesystem)
pub fn Result<modules::Modules,Error> load(ref u8[] root,ref u8[] document,ref mut sources::Budget budget,ref mut usize work){
    var sources::Budget staged=*budget;let sources::Sources loaded=switch(sources::load(root,document,ref mut staged,work)){case .ok(value):{value}case .err(e):{return .err(source_error(ref e,document));}};
    let modules::Modules bound=switch(modules::bind(loaded,work)){case .ok(value):{value}case .err(e):{return .err(module_error(ref e));}};
    *budget=staged;return .ok(bound);
}
@effects(filesystem)
pub fn Result<modules::Modules,Error> load_manifest(ref u8[] root,ref mut sources::Budget budget,ref mut usize work){
    let bytes::Buffer document=switch(files::read(root,b"fairy.toml",1048576,work)){case .ok(value):{value}case .err(e):{return .err(capture(Problem{.phase=0,.reason=e.code,.diagnostic=0,.status=0,.field=0,.os_code=e.os_code,.file=0,.start=e.start,.end=e.end},b"fairy.toml",b""));}};
    return load(root,bytes::view(ref document),budget,work);
}
