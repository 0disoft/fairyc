module tests::project_lock_match;
import project_lock_match as matching;
import project_lock_read_file as locked;
import project_lock_document as document;
import project_module_visibility as visibility;
import project_source_set as sources;
import project_source_file as source_files;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,root).is_ok && bytes::append_byte(ref mut output,47).is_ok && bytes::append(ref mut output,leaf).is_ok);return output;}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] path=ref argument;return switch(files::manifest(path[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){var usize meter=0;return switch(source_files::read(root,leaf,8388608,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] input,ref mut usize at){assert(*at<=input.len && input.len-*at>=8);var usize value=0;var usize index=0;while(index<8){value=value*256+(input[*at+index] as usize);index+=1;}*at+=8;return value;}
fn ref u8[] field(ref u8[] input,ref mut usize at){let usize size=number(input,at);assert(*at<=input.len && size<=input.len-*at);let usize start=*at;*at+=size;return input[start..*at];}
@test
@effects(filesystem,console)
fn void physical_lock_and_fresh_workspace_matching_agree_with_go_without_writes(){let owned=root();let path=bytes::view(ref owned);let data=read(path,b"matches.bin");let input=bytes::view(ref data);assert(input.len>=16 && bytes::equal(input[..8],b"FYLMATCH"));var usize at=8;let usize count=number(input,ref mut at);assert(count==26);var usize index=0;var usize current=0;
    while(index<count){let leaf=field(input,ref mut at);let bool candidate=number(input,ref mut at)==1;let bool expected=number(input,ref mut at)==1;let parent=join(path,leaf);let workspace_root=bytes::view(ref parent);let before=read(workspace_root,b"fairy.lock");var usize meter=0;
        let actual=switch(locked::read(workspace_root,ref mut meter)){case .ok(value):{value}case .err(p):{discard io::write_stderr_bytes(leaf);discard io::write_stderr_u32(p.stage);discard io::write_stderr_u32(p.code);assert(false);return;}};
        var sources::Budget budget=sources::Budget{.files=17,.bytes=31};let workspace=switch(visibility::load_embedded(workspace_root,candidate,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{discard io::write_stderr_bytes(leaf);assert(false);return;}};let snapshot=budget;
        let same=switch(matching::matches(ref actual,ref workspace,ref mut meter)){case .ok(value):{value}case .err(p):{discard io::write_stderr_bytes(leaf);discard io::write_stderr_u32(p.stage);discard io::write_stderr_u32(p.code);assert(false);return;}};if(same!=expected){discard io::write_stderr_bytes(leaf);assert(false);}if(same){current+=1;}assert(budget.files==snapshot.files && budget.bytes==snapshot.bytes);
        let after=read(workspace_root,b"fairy.lock");assert(bytes::equal(bytes::view(ref before),bytes::view(ref after)));meter=16777216;assert(switch(matching::matches(ref actual,ref workspace,ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==1 && p.code==1}});index+=1;
    }assert(at==input.len && current==18);
}
@effects(filesystem)
fn void bad(ref u8[] root,ref u8[] leaf,u32 stage,u32 code,u32 reader_stage){let path=join(root,leaf);var usize meter=0;assert(switch(locked::read(bytes::view(ref path),ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==stage && p.code==code && p.reader_stage==reader_stage}});}
@test
@effects(filesystem)
fn void regular_file_bounds_and_strict_read_failures_keep_their_categories(){let owned=root();let path=bytes::view(ref owned);bad(path,b"faults/Missing",1,2,0);bad(path,b"faults/Directory",1,9,0);bad(path,b"faults/Oversized",1,3,0);bad(path,b"faults/Reparse",1,8,0);bad(path,b"faults/Empty",2,1,1);bad(path,b"faults/Malformed",2,1,1);bad(path,b"faults/NullDependencies",2,9,2);let valid=join(path,b"None");var usize meter=16777216;assert(switch(locked::read(bytes::view(ref valid),ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==1 && p.code==5}});}
@test
@effects(filesystem)
fn void private_candidate_projection_is_an_error_instead_of_stale(){let owned=root();let path=join(bytes::view(ref owned),b"CandidateCore");var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;let candidate=switch(visibility::load_embedded(bytes::view(ref path),true,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let plain_path=join(bytes::view(ref owned),b"None");let actual=switch(locked::read(bytes::view(ref plain_path),ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(switch(matching::matches(ref actual,ref candidate,ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==1 && p.code==3}});}
