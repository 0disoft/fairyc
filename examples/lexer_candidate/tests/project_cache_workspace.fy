module tests::project_cache_workspace;
import project_cache_workspace as facts;
import project_compiler_core as core;
import project_compiler_input as input;
import project_coordinate_graph as coordinates;
import project_module_visibility as visibility;
import project_source_set as sources;
import project_sdk_bundle as sdk;
import module_catalog as catalog;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::vec as v;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer p=bytes::empty();assert(bytes::append(ref mut p,root).is_ok && bytes::append_byte(ref mut p,47).is_ok && bytes::append(ref mut p,leaf).is_ok);return p;}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);return switch(files::load(bytes::view(ref path))){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] raw=ref path;return switch(files::manifest(raw[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn visibility::Resolved load(ref u8[] root,ref u8[] leaf){var v::Vec<sdk::File> files=v::empty<sdk::File>();var usize i=0;let vector=bytes::equal(leaf,b"Vec");while(i<if(vector){8}else{7}){assert(v::push<sdk::File>(ref mut files,sdk::file(i+1,read(root,sdk::path(i)))).is_ok);i+=1;}var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;
    let library=switch(sdk::assemble(files,false,ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return load(root,leaf);}};let path=join(root,leaf);return switch(visibility::load_sdk(bytes::view(ref path),library,ref mut budget,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return load(root,leaf);}};
}
@effects(console)
fn core::Complete complete(ref visibility::Resolved loaded){var usize work=0;let ids=switch(coordinates::collect(visibility::workspace_graph(loaded),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return complete(loaded);}};let projected=switch(input::project(loaded,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return complete(loaded);}};var usize source_work=0;var usize link_work=0;
    return switch(core::build_complete(ref projected,ref ids,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{discard io::write_stderr("complete phase/stage/code: ");discard io::write_stderr_u32(e.phase);discard io::write_stderr_u32(e.stage);discard io::write_stderr_u32(e.code);assert(false);return complete(loaded);}};
}
fn usize number(ref u8[] raw,ref mut usize at){assert(*at<=raw.len && raw.len-*at>=8);var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[*at+i] as usize);i+=1;}*at+=8;return n;}
fn ref u8[] field(ref u8[] raw,ref mut usize at){let n=number(raw,at);assert(*at<=raw.len && n<=raw.len-*at);let start=*at;*at+=n;return raw[start..*at];}
fn usize find(ref facts::Facts p,ref u8[] id){var usize i=0;while(i<facts::count(p)){if(bytes::equal(facts::instance(facts::at(p,i)),id)){return i;}i+=1;}return facts::count(p);}
@effects(filesystem)
fn void oracle(ref u8[] root,ref u8[] leaf,ref facts::Facts actual){let path=join(root,leaf);let fixture=read(bytes::view(ref path),b"facts.bin");let raw=bytes::view(ref fixture);assert(bytes::equal(raw[..8],b"FYCWORK1"));var usize at=8;let total=number(raw,ref mut at);assert(total==facts::count(actual));var bool[512] seen=[false;512];var usize i=0;
    while(i<total){let id=field(raw,ref mut at);let owner=find(actual,id);assert(owner<total && !seen[owner]);seen[owner]=true;let p=facts::at(actual,owner);assert(bytes::equal(facts::coordinate(p),field(raw,ref mut at)) && bytes::equal(facts::source(p),field(raw,ref mut at)));let features=number(raw,ref mut at);assert(features==facts::feature_count(p));var usize j=0;while(j<features){assert(bytes::equal(facts::feature(p,j),field(raw,ref mut at)));j+=1;}let dependencies=number(raw,ref mut at);assert(dependencies==facts::dependency_count(p));var bool[512] targets=[false;512];j=0;
        while(j<dependencies){let target=find(actual,field(raw,ref mut at));assert(target<total && !targets[target]);targets[target]=true;j+=1;}j=0;while(j<dependencies){let target=facts::dependency(p,j);assert(target<total && targets[target] && (j==0 || facts::dependency(p,j-1)<target));j+=1;}i+=1;
    }assert(bytes::equal(facts::sdk_instance(actual),field(raw,ref mut at)) && at==raw.len);
}
@test @effects(filesystem,console)
fn void matches_go_source_sdk_features_and_deduplicated_dependencies(){let path=root();var usize i=0;while(i<7){let leaf=if(i==0){b"Base"}else if(i==1){b"Edited"}else if(i==2){b"Reordered"}else if(i==3){b"NoSDK"}else if(i==4){b"Vec"}else if(i==5){b"Version"}else{b"Feature"};let loaded=load(bytes::view(ref path),leaf);let checked=complete(ref loaded);var usize work=0;let actual=switch(facts::collect(ref loaded,ref checked,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};oracle(bytes::view(ref path),leaf,ref actual);assert(work>0 && work<=16777216);i+=1;}}
@effects(filesystem,console)
fn facts::Facts detached(ref u8[] root){let loaded=load(root,b"NoSDK");let checked=complete(ref loaded);var usize work=0;return switch(facts::collect(ref loaded,ref checked,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return detached(root);}};}
@test @effects(filesystem,console)
fn void owns_facts_after_graph_and_complete_close_and_honors_work(){let path=root();let owned=detached(bytes::view(ref path));oracle(bytes::view(ref path),b"NoSDK",ref owned);assert(facts::sdk_instance(ref owned).len==0);let loaded=load(bytes::view(ref path),b"NoSDK");let checked=complete(ref loaded);var usize cost=0;let measured=switch(facts::collect(ref loaded,ref checked,ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(cost>0 && cost<16777216);var usize work=16777216-cost;let exact=switch(facts::collect(ref loaded,ref checked,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(work==16777216);oracle(bytes::view(ref path),b"NoSDK",ref exact);assert(facts::count(ref measured)==facts::count(ref exact));work=16777216-cost+1;assert(switch(facts::collect(ref loaded,ref checked,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});work=16777217;assert(switch(facts::collect(ref loaded,ref checked,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work==16777217);}
@test @effects(filesystem,console)
fn void rejects_stale_source_and_coordinate_without_partial_facts(){let path=root();let base=load(bytes::view(ref path),b"Base");let checked=complete(ref base);var usize i=0;while(i<3){let loaded=load(bytes::view(ref path),if(i==0){b"Edited"}else if(i==1){b"Version"}else{b"Feature"});var usize work=0;assert(switch(facts::collect(ref loaded,ref checked,ref mut work)){case .ok(_):{false}case .err(code):{if(i==0){code==12}else{code==9}}});i+=1;}}
@test @effects(filesystem,console)
fn void rejects_checked_ir_with_a_different_import_provider(){let path=root();let loaded=load(bytes::view(ref path),b"NoSDK");var usize work=0;let ids=switch(coordinates::collect(visibility::workspace_graph(ref loaded),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};let projected=switch(input::project(ref loaded,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize first=input::count(ref projected);var usize second=first;var usize i=0;
    while(i<input::count(ref projected)){let parsed=input::source(ref projected,i);let d=syntax::declaration_at(parsed,0);if(bytes::equal(syntax::source_view(parsed)[d.name_start..d.name_end],b"common")){if(first==input::count(ref projected)){first=i;}else{assert(second==input::count(ref projected));second=i;}}i+=1;}assert(first<input::count(ref projected) && second<input::count(ref projected));
    var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();i=0;var usize changed=0;while(i<v::len<catalog::Binding>(input::bindings(ref projected))){var binding=*v::at<catalog::Binding>(input::bindings(ref projected),i);if(binding.target==first){binding.target=second;changed+=1;}else if(binding.target==second){binding.target=first;changed+=1;}assert(v::push<catalog::Binding>(ref mut imports,binding).is_ok);i+=1;}assert(changed==2);var usize source_work=0;var usize link_work=0;
    let other=switch(core::build_complete_sources(input::sources(ref projected),input::owners(ref projected),ref imports,ref ids,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;assert(switch(facts::collect(ref loaded,ref other,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});
}
