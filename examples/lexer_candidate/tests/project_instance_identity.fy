module tests::project_instance_identity;
import project_instance_identity as identity;
import project_workspace_input as workspace;
import project_source_set as sources;
import project_source_file as files;
import file_source as arguments;
import std::bytes as bytes;
import std::vec as v;
import std::text as text;
import std::process as process;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem,console)
fn bytes::Buffer root(){var u8[4096] arg=[0;4096];let usize length=switch(process::arg(1,ref mut arg)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref arg;return switch(arguments::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){var usize meter=0;return switch(files::read(root,leaf,8388608,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(console)
fn void require(bool condition,ref u8[] leaf){if(!condition){discard io::write_stderr_bytes(leaf);discard io::write_stderr("\n");assert(false);}}
@effects(filesystem,console)
fn void check(ref u8[] root,ref u8[] leaf,usize count){let path=join(root,leaf);let uid=read(bytes::view(ref path),b"uid.bin");let source=read(bytes::view(ref path),b"source.bin");let version=read(bytes::view(ref path),b"version.bin");var v::Vec<bytes::Buffer> features=v::empty<bytes::Buffer>();if(count>0){let f=read(bytes::view(ref path),b"feature0.bin");assert(v::push<bytes::Buffer>(ref mut features,f).is_ok);}if(count>1){let f=read(bytes::view(ref path),b"feature1.bin");assert(v::push<bytes::Buffer>(ref mut features,f).is_ok);}assert(count<=2);
    let expected=read(bytes::view(ref path),b"oracle.txt");var bytes::Buffer actual=bytes::empty();var usize meter=0;
    switch(identity::compute(bytes::view(ref uid),bytes::view(ref source),bytes::view(ref version),ref features,ref mut meter)){case .ok(value):{let encoded=read(bytes::view(ref path),b"encoding.bin");require(bytes::equal(identity::canonical(ref value),bytes::view(ref encoded)) && identity::valid_text(identity::digest(ref value)) && identity::compare(ref value,ref value)==1,leaf);assert(bytes::append(ref mut actual,b"pass\n").is_ok && bytes::append(ref mut actual,identity::digest(ref value)).is_ok && bytes::append_byte(ref mut actual,10).is_ok);}case .err(code):{assert(bytes::append(ref mut actual,b"reject\n").is_ok && bytes::append(ref mut actual,text::as_bytes(identity::reason(code))).is_ok && bytes::append_byte(ref mut actual,10).is_ok);}};
    require(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)),leaf);
}
@test
@effects(filesystem,console)
fn void matches_normative_and_go_utf8_boundary_and_large_vectors(){let root=root();let ref u8[] path=bytes::view(ref root);
    check(path,b"pass.identity.empty-features",0);check(path,b"pass.identity.two-features",2);check(path,b"pass.identity.component-boundary-left",0);check(path,b"pass.identity.component-boundary-right",0);check(path,b"fail.identity.unsorted-features",2);check(path,b"fail.identity.duplicate-features",2);check(path,b"fail.identity.empty-feature",1);
    check(path,b"composed",2);check(path,b"decomposed",2);check(path,b"nul-bytes",0);check(path,b"empty-uid",0);check(path,b"empty-source",0);check(path,b"empty-version",0);check(path,b"utf8-uid",0);check(path,b"utf8-source",0);check(path,b"utf8-version",0);check(path,b"utf8-feature",1);check(path,b"mixed-empty-order",2);check(path,b"mixed-utf8-order",2);check(path,b"large",0);
}
@effects(filesystem,console)
fn void expected(ref identity::Identity value,ref u8[] root,ref u8[] id_file,ref u8[] encoding_file){let digest=read(root,id_file);let encoded=read(root,encoding_file);require(bytes::equal(identity::digest(value),bytes::view(ref digest)) && bytes::equal(identity::canonical(value),bytes::view(ref encoded)),id_file);}
@test
@effects(filesystem,console)
fn void binds_local_features_and_checks_tuple_collisions(){let root=root();let path=join(bytes::view(ref root),b"Local");var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;let loaded=switch(workspace::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    let root_id=switch(identity::root(ref loaded,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let a=switch(identity::dependency(ref loaded,0,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let b=switch(identity::dependency(ref loaded,1,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let twin=switch(identity::dependency(ref loaded,2,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    expected(ref root_id,bytes::view(ref path),b"root.id",b"root.encoding.bin");expected(ref a,bytes::view(ref path),b"edge0.id",b"edge0.encoding.bin");expected(ref b,bytes::view(ref path),b"edge1.id",b"edge1.encoding.bin");expected(ref twin,bytes::view(ref path),b"edge2.id",b"edge2.encoding.bin");
    assert(identity::compare(ref a,ref twin)==1 && identity::compare(ref a,ref b)==0 && identity::compare_record(ref a,identity::digest(ref a),identity::canonical(ref b))==2 && identity::compare_record(ref a,b"fypi1-sha256-x",identity::canonical(ref a))==3);
    let ref u8[] id=identity::digest(ref a);var bytes::Buffer modified=switch(bytes::copy(id)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref mut u8[] mutable=bytes::view_mut(ref mut modified);mutable[13]=65;assert(!identity::valid_text(bytes::view(ref modified)) && !identity::valid_text(id[..76]) && !identity::valid_text(b"fypi2-sha256-0000000000000000000000000000000000000000000000000000000000000000"));
}
