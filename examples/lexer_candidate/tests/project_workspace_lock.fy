module tests::project_workspace_lock;
import project_workspace_lock as loader;
import project_lock_policy as policy;
import project_source_set as sources;
import project_source_file as files;
import project_workspace_graph as graph;
import project_module_visibility as visibility;
import file_source as fixture;
import std::bytes as bytes;
import std::text as text;
import std::fs as fs;
import std::process as process;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer output=bytes::empty();assert(bytes::append(ref mut output,root).is_ok && bytes::append_byte(ref mut output,47).is_ok && bytes::append(ref mut output,leaf).is_ok);return output;}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] path=[0;4096];let count=switch(process::arg(1,ref mut path)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] view=ref path;return switch(fixture::manifest(view[..count])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){var usize meter=0;return switch(files::read(root,leaf,8388608,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
fn usize number(ref u8[] input,ref mut usize at){assert(*at<=input.len && input.len-*at>=8);var usize result=0;var usize i=0;while(i<8){result=result*256+(input[*at+i] as usize);i+=1;}*at+=8;return result;}
fn ref u8[] field(ref u8[] input,ref mut usize at){let count=number(input,at);assert(*at<=input.len && count<=input.len-*at);let start=*at;*at+=count;return input[start..*at];}
@effects(filesystem)
fn void reset(ref u8[] root,usize kind,ref u8[] before){
    if(kind==2){return;}let owned=join(root,b"fairy.lock");let path=switch(text::validate_utf8(bytes::view(ref owned))){case .ok(value):{value}case .err(_):{assert(false);return;}};
    if(kind==0){let c=switch(text::cstring_from_text(path)){case .ok(value):{value}case .err(_):{assert(false);return;}};discard fs::remove_file(text::cstring_as_cstr(ref c));return;}
    var usize meter=0;let same=switch(files::read(root,b"fairy.lock",8388608,ref mut meter)){case .ok(value):{bytes::equal(bytes::view(ref value),before)}case .err(_):{false}};if(same){return;}
    var fs::File file=switch(fs::create_truncate_utf8(path)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(fs::write_all(ref mut file,before).is_ok);assert(fs::close(file).is_ok);
}
@test
@effects(filesystem,console)
fn void owning_go_cli_lock_states_bytes_failures_and_budget_commit_match(){let parent=root();let data=read(bytes::view(ref parent),b"policies.bin");let input=bytes::view(ref data);assert(input.len>=16 && bytes::equal(input[..8],b"FYLPOL01"));var usize at=8;let count=number(input,ref mut at);assert(count==24);var usize index=0;
    while(index<count){let name=field(input,ref mut at);let locked=number(input,ref mut at)==1;let expected=number(input,ref mut at);let phase=number(input,ref mut at);let stage=number(input,ref mut at);let kind=number(input,ref mut at);let file_count=number(input,ref mut at);let file_bytes=number(input,ref mut at);let before=field(input,ref mut at);let after=field(input,ref mut at);let path=join(bytes::view(ref parent),name);let base=bytes::view(ref path);reset(base,kind,before);
        var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;
        switch(loader::load(base,locked,ref mut budget,ref mut meter)){
            case .ok(loaded):{if(expected==0 || policy::state_code(loader::state(ref loaded)) as usize!=expected){discard io::write_stderr_bytes(name);assert(false);}assert(budget.files==17+file_count && budget.bytes==31+file_bytes);let code=policy::state_code(loader::state(ref loaded));assert(bytes::equal(text::as_bytes(policy::label(loader::state(ref loaded))),if(code==1){b"unchanged"}else if(code==2){b"updated"}else{b"verified"}));assert(graph::node_count(visibility::workspace_graph(loader::workspace(ref loaded)))>=1);}
            case .err(error):{if(expected!=0 || loader::phase(ref error) as usize!=phase){discard io::write_stderr_bytes(name);discard io::write_stderr_u32(loader::phase(ref error));assert(false);}assert(budget.files==17 && budget.bytes==31);if(phase==2){let p=loader::policy_error(ref error);if(p.stage as usize!=stage){discard io::write_stderr_bytes(name);discard io::write_stderr_u32(p.stage);assert(false);}if(stage==3){assert(p.read_stage!=0);}if(stage==6){assert(p.read_stage==9 && p.os_code!=0 && p.cleanup_code==0);}}else{let retained=loader::input_error(ref error);assert(visibility::load_phase(retained)!=0);}}
        };
        if(kind==2){let sentinel=read(base,b"fairy.lock/sentinel");assert(bytes::equal(bytes::view(ref sentinel),b"DIRECTORY"));}
        else if(kind==0 && expected==0){meter=0;assert(switch(files::read(base,b"fairy.lock",8388608,ref mut meter)){case .ok(_):{false}case .err(p):{p.code==2}});}
        else{let actual=read(base,b"fairy.lock");assert(bytes::equal(bytes::view(ref actual),after));}index+=1;
    }assert(at==input.len);
}
@test
@effects(filesystem)
fn void private_projection_and_bounds_never_publish_or_commit_source_counters(){let parent=root();let path=join(bytes::view(ref parent),b"Private");let base=bytes::view(ref path);var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;
    assert(switch(loader::load_embedded(base,true,false,ref mut budget,ref mut meter)){case .ok(_):{false}case .err(e):{if(loader::phase(ref e)!=2){false}else{let p=loader::policy_error(ref e);p.stage==1 && p.code==3}}});assert(budget.files==17 && budget.bytes==31);let before=read(base,b"fairy.lock");assert(bytes::equal(bytes::view(ref before),b"OLD"));
    meter=16777216;assert(switch(loader::load(base,false,ref mut budget,ref mut meter)){case .ok(_):{false}case .err(e):{loader::phase(ref e)==1}});assert(budget.files==17 && budget.bytes==31);let after=read(base,b"fairy.lock");assert(bytes::equal(bytes::view(ref before),bytes::view(ref after)));
    let plain=join(bytes::view(ref parent),b"Current");meter=0;let loaded=switch(loader::load(bytes::view(ref plain),true,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};let snapshot=budget;meter=16777216;assert(switch(policy::apply(bytes::view(ref plain),loader::workspace(ref loaded),false,ref mut meter)){case .ok(_):{false}case .err(p):{p.stage==1 && p.code==1}});assert(budget.files==snapshot.files && budget.bytes==snapshot.bytes);
}
@test
@effects(filesystem)
fn void missing_import_retains_visibility_error_and_preserves_lock(){let parent=root();let path=join(bytes::view(ref parent),b"Import");let before=read(bytes::view(ref path),b"fairy.lock");var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;
    assert(switch(loader::load(bytes::view(ref path),false,ref mut budget,ref mut meter)){case .ok(_):{false}case .err(e):{if(loader::phase(ref e)!=1){false}else{let retained=loader::input_error(ref e);if(visibility::load_phase(retained)!=2){false}else{visibility::error(visibility::visibility_error(retained)).code==1}}}});assert(budget.files==17 && budget.bytes==31);let after=read(bytes::view(ref path),b"fairy.lock");assert(bytes::equal(bytes::view(ref before),bytes::view(ref after)));
}
