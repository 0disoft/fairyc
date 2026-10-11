module project_source_root;
import project_manifest as manifest;
import project_directory as directory;
import std::bytes as bytes;

// Phase 1 spans are in the supplied manifest; phase 2 spans are in its
// decoded package.source value. Reasons retain the owning leaf's codes.
pub struct Error{u32 phase;u32 reason;u32 field;u32 os_code;usize start;usize end;}
pub resource SourceRoot{manifest::Manifest package;bytes::Buffer path;bytes::Buffer document;cleanup{}}

@effects(filesystem)
pub fn Result<SourceRoot,Error> load(ref u8[] root,ref u8[] source,ref mut usize work){
    let manifest::Manifest package=switch(manifest::parse(source)){
        case .ok(value):{value}
        case .err(e):{return .err(Error{.phase=1,.reason=e.reason,.field=e.field,.os_code=0,.start=e.start,.end=e.end});}
    };
    let bytes::Buffer path=switch(directory::resolve(root,manifest::package_value(ref package,b"source"),work)){
        case .ok(value):{value}
        case .err(e):{return .err(Error{.phase=2,.reason=e.code,.field=7,.os_code=e.os_code,.start=e.start,.end=e.end});}
    };
    let bytes::Buffer document=switch(bytes::copy(source)){case .ok(value):{value}case .err(_):{return .err(Error{.phase=1,.reason=13,.field=0,.os_code=0,.start=0,.end=0});}};
    return .ok(SourceRoot{.package,.path,.document});
}
pub fn ref u8[] path(ref SourceRoot root){return bytes::view(ref root.path);}
pub fn ref u8[] package_value(ref SourceRoot root,ref u8[] key){return manifest::package_value(ref root.package,key);}
pub fn ref manifest::Manifest typed_manifest(ref SourceRoot root){return ref root.package;}
pub fn ref u8[] manifest_input(ref SourceRoot root){return bytes::view(ref root.document);}
