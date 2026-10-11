module tests::project_sdk_selection;
import project_instance_graph as graph;
import project_source_set as sources;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);return switch(files::manifest(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn void check(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;let loaded=switch(graph::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let candidate_input=read(bytes::view(ref path),b"candidate.txt");let bool candidate=bytes::equal(bytes::view(ref candidate_input),b"true");let before=budget;
    let selected=switch(graph::select_sdk(ref loaded,candidate,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};var bytes::Buffer actual=bytes::empty();let ref u8[] profiles=b"0123";let usize profile=graph::sdk_profile(ref selected) as usize;assert(profile<4 && bytes::append_byte(ref mut actual,profiles[profile]).is_ok && bytes::append_byte(ref mut actual,10).is_ok);var usize at=0;while(at<graph::sdk_owner_count(ref selected)){let usize owner=graph::sdk_owner(ref selected,at);assert(owner<graph::node_count(ref loaded) && (at==0 || graph::sdk_owner(ref selected,at-1)<owner));assert(bytes::append(ref mut actual,graph::node_id(ref loaded,owner)).is_ok && bytes::append_byte(ref mut actual,10).is_ok);at+=1;}let expected=read(bytes::view(ref path),b"selection.txt");assert(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)) && budget.files==before.files && budget.bytes==before.bytes);
    meter=16777216;let rejected=switch(graph::select_sdk(ref loaded,candidate,ref mut meter)){case .ok(_):{assert(false);return;}case .err(value):{value}};assert(rejected.code==4);
}
@test
@effects(filesystem)
fn void selects_sdk_only_for_exact_reachable_instance_imports(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;let root=switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref u8[] path=bytes::view(ref root);
    check(path,b"None");check(path,b"CandidateUnused");check(path,b"Core");check(path,b"AliasCore");check(path,b"Vector");check(path,b"CandidateCore");check(path,b"ChildOnly");check(path,b"Both");check(path,b"ChildVector");check(path,b"UnreachableVector");check(path,b"FeatureSplit");check(path,b"InstanceAliases");check(path,b"LongTrivia");check(path,b"SimilarNamespaces");
}
