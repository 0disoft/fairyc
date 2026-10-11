module tests::project_source_identity;
import project_source_identity as identity;
import project_source_set as sources;
import project_package_input as packages;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem)
fn void check(ref u8[] root,ref u8[] leaf,bool parsed){
    let path=join(root,leaf);let expected_path=join(bytes::view(ref path),b"digest.txt");let expected=switch(files::manifest(bytes::view(ref expected_path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};
    var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;
    if(parsed){let loaded=switch(packages::load_manifest(bytes::view(ref path),ref mut budget,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};let digest=switch(packages::source_digest(ref loaded)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};assert(bytes::equal(bytes::view(ref digest),bytes::view(ref expected)));}
    else{let loaded=switch(sources::load_manifest(bytes::view(ref path),ref mut budget,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};let digest=switch(identity::compute(ref loaded)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};assert(bytes::equal(bytes::view(ref digest),bytes::view(ref expected)));let repeated=switch(identity::compute(ref loaded)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};assert(bytes::equal(bytes::view(ref digest),bytes::view(ref repeated)));}
}
@test
@effects(filesystem,console)
fn void matches_go_exact_paths_bytes_order_and_large_input(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;let root=switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};
    check(bytes::view(ref root),b"One",true);check(bytes::view(ref root),b"Three",true);check(bytes::view(ref root),b"OtherRoot",true);check(bytes::view(ref root),b"CRLF",false);check(bytes::view(ref root),b"Binary",false);check(bytes::view(ref root),b"Large",false);check(bytes::view(ref root),b"ModuleOrder",true);check(bytes::view(ref root),b"Renamed",false);
}
