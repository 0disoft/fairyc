module tests::project_lock_document;
import project_lock_document as lock;
import project_module_visibility as visibility;
import project_source_set as sources;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);return switch(files::load(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;return switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn u8 octet(usize value){let ref u8[] digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16]<<4)|digits[value%16];}
fn void number(ref mut bytes::Buffer output,usize value){var u8[8] encoded=[0;8];var usize index=0;var usize remaining=value;while(index<8){encoded[7-index]=octet(remaining%256);remaining/=256;index+=1;}let ref u8[] input=ref encoded;assert(remaining==0 && bytes::append(output,input).is_ok);}
fn void text(ref mut bytes::Buffer output,ref u8[] value){number(output,value.len);assert(bytes::append(output,value).is_ok);}
fn bytes::Buffer observation(ref lock::Document document){var bytes::Buffer output=bytes::empty();var usize field=0;while(field<5){text(ref mut output,lock::header(document,field));field+=1;}number(ref mut output,lock::package_count(document));var usize at=0;while(at<lock::package_count(document)){field=0;while(field<10){text(ref mut output,lock::package_field(document,at,field));field+=1;}number(ref mut output,lock::feature_count(document,at));var usize index=0;while(index<lock::feature_count(document,at)){text(ref mut output,lock::feature(document,at,index));index+=1;}at+=1;}number(ref mut output,lock::dependency_count(document));at=0;while(at<lock::dependency_count(document)){text(ref mut output,lock::package_field(document,lock::owner(document,at),0));text(ref mut output,lock::binding(document,at));text(ref mut output,lock::package_field(document,lock::target(document,at),0));at+=1;}return output;}
@effects(filesystem)
fn lock::Document retained(ref u8[] path,bool candidate){var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;let loaded=switch(visibility::load_embedded(path,candidate,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return retained(path,candidate);}};let before=budget;let document=switch(lock::project(ref loaded,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return retained(path,candidate);}};assert(budget.files==before.files && budget.bytes==before.bytes);meter=16777216;let rejected=switch(lock::project(ref loaded,ref mut meter)){case .ok(_):{assert(false);return retained(path,candidate);}case .err(value):{value}};assert(rejected.code==1);return document;}
@effects(filesystem)
fn void check(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);let flags=read(bytes::view(ref path),b"candidate.txt");let bool candidate=bytes::equal(bytes::view(ref flags),b"true");let expected=read(bytes::view(ref path),b"projection.bin");let document=retained(bytes::view(ref path),candidate);let actual=observation(ref document);assert(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)));let repeated=observation(ref document);assert(bytes::equal(bytes::view(ref actual),bytes::view(ref repeated)));}
@test
@effects(filesystem)
fn void projects_owned_sorted_lock_fields_and_edges_matching_go(){let owned=root();let path=bytes::view(ref owned);check(path,b"None");check(path,b"CandidateUnused");check(path,b"Core");check(path,b"AliasCore");check(path,b"Vector");check(path,b"ChildOnly");check(path,b"Both");check(path,b"ChildVector");check(path,b"UnreachableVector");check(path,b"FeatureSplit");check(path,b"InstanceAliases");check(path,b"LongTrivia");check(path,b"SimilarNamespaces");}
@test
@effects(filesystem)
fn void private_candidate_sdk_cannot_be_normalized_into_a_public_lock(){let owned=root();let path=join(bytes::view(ref owned),b"CandidateCore");let oracle=read(bytes::view(ref path),b"lock-accepted.txt");assert(bytes::equal(bytes::view(ref oracle),b"false"));var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;let loaded=switch(visibility::load_embedded(bytes::view(ref path),true,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let before=budget;let rejected=switch(lock::project(ref loaded,ref mut meter)){case .ok(_):{assert(false);return;}case .err(value):{value}};assert(rejected.code==3 && budget.files==before.files && budget.bytes==before.bytes);}
