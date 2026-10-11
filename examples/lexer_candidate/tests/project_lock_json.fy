module tests::project_lock_json;
import project_lock_json as json;
import project_json_string as strings;
import project_lock_document as lock;
import project_module_visibility as visibility;
import project_source_set as sources;
import file_source as files;
import project_source_file as source_files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){var usize meter=0;return switch(source_files::read(root,leaf,1048576,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;return switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn void check(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);let expected=read(bytes::view(ref path),b"lock.json");let flags=read(bytes::view(ref path),b"candidate.txt");var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;let loaded=switch(visibility::load_embedded(bytes::view(ref path),bytes::equal(bytes::view(ref flags),b"true"),ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let document=switch(lock::project(ref loaded,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let before=budget;let encoded=switch(json::marshal(ref document,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref encoded),bytes::view(ref expected)) && budget.files==before.files && budget.bytes==before.bytes);meter=16777216;assert(switch(json::marshal(ref document,ref mut meter)){case .ok(_):{false}case .err(code):{code==1}});}
@test
@effects(filesystem)
fn void canonical_json_matches_every_go_lock_byte(){let owned=root();let path=bytes::view(ref owned);check(path,b"None");check(path,b"CandidateUnused");check(path,b"Core");check(path,b"AliasCore");check(path,b"Vector");check(path,b"ChildOnly");check(path,b"Both");check(path,b"ChildVector");check(path,b"UnreachableVector");check(path,b"FeatureSplit");check(path,b"InstanceAliases");check(path,b"LongTrivia");check(path,b"SimilarNamespaces");}
@effects(filesystem,console)
fn void scalar(ref u8[] root,ref u8[] leaf){let input_path=join(b"strings",leaf);let actual=read(root,bytes::view(ref input_path));var bytes::Buffer expected_path=bytes::empty();assert(bytes::append(ref mut expected_path,b"strings/").is_ok && bytes::append(ref mut expected_path,leaf[..leaf.len-4]).is_ok && bytes::append(ref mut expected_path,b".json").is_ok);let expected=read(root,bytes::view(ref expected_path));var usize meter=0;let quoted=switch(strings::quote(bytes::view(ref actual),ref mut meter)){case .ok(value):{value}case .err(_):{discard io::write_stderr_bytes(leaf);discard io::write_stderr(" quote error\n");assert(false);return;}};if(!bytes::equal(bytes::view(ref quoted),bytes::view(ref expected))){discard io::write_stderr_bytes(leaf);discard io::write_stderr("\nactual: ");discard io::write_stderr_bytes(bytes::view(ref quoted));discard io::write_stderr("\nexpected: ");discard io::write_stderr_bytes(bytes::view(ref expected));discard io::write_stderr("\n");assert(false);}}
@test
@effects(filesystem,console)
fn void json_strings_match_go_control_html_unicode_and_large_spelling(){let owned=root();let path=bytes::view(ref owned);scalar(path,b"Empty.bin");scalar(path,b"ASCII.bin");scalar(path,b"Controls.bin");scalar(path,b"Escapes.bin");scalar(path,b"HTML.bin");scalar(path,b"Unicode.bin");scalar(path,b"DELHigh.bin");scalar(path,b"Large.bin");}
@test
fn void json_string_invalid_utf8_and_bounds_return_no_partial_output(){var usize meter=0;assert(switch(strings::quote(b"\xff",ref mut meter)){case .ok(_):{false}case .err(code):{code==2}});meter=16777216;assert(switch(strings::quote(b"value",ref mut meter)){case .ok(_):{false}case .err(code):{code==1}});var bytes::Buffer huge=bytes::empty();assert(bytes::resize_zeroed(ref mut huge,8388609).is_ok);meter=0;assert(switch(strings::quote(bytes::view(ref huge),ref mut meter)){case .ok(_):{false}case .err(code):{code==1}});assert(meter==0);}
