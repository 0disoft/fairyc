module tests::project_workspace_catalog;
import project_workspace_catalog as catalog;
import project_package_input as packages;
import project_source_modules as modules;
import project_source_set as sources;
import project_manifest as manifest;
import project_directory as directory;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
import std::text as text;
import std::io as io;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(console)
fn void require(bool condition,ref u8[] leaf,ref u8[] stage){if(!condition){discard io::write_stderr_bytes(leaf);discard io::write_stderr(": ");discard io::write_stderr_bytes(stage);discard io::write_stderr("\n");assert(false);}}
@effects(filesystem,console)
fn void oracle(ref u8[] path,bool accepted,ref str reason){let file=join(path,b"oracle.txt");let expected=switch(files::manifest(bytes::view(ref file))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};var bytes::Buffer actual=bytes::empty();assert(bytes::append(ref mut actual,if(accepted){b"true\n"}else{b"false\n"}).is_ok && bytes::append(ref mut actual,text::as_bytes(reason)).is_ok && bytes::append_byte(ref mut actual,10).is_ok);require(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)),path,b"Go acceptance/reason");}
@effects(filesystem,console)
fn void accepted(ref u8[] root,ref u8[] leaf,usize count,usize bytes_count){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=4096-count,.bytes=67108864-bytes_count};var usize meter=0;
    let loaded=switch(catalog::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(value):{value}case .err(e):{let p=catalog::error(ref e);discard io::write_stderr_u32(p.code);discard io::write_stderr_u32(p.reason);require(false,leaf,b"unexpected load error");return;}};
    require(catalog::count(ref loaded)==count && budget.files==4096 && budget.bytes==67108864 && catalog::find(ref loaded,b".")==1 && catalog::find(ref loaded,b"absent")==0 && catalog::find_uid(ref loaded,b"example.root")==1,leaf,b"counts/budget/index");
    require(bytes::equal(catalog::path(ref loaded,0),b".") && modules::count(catalog::input(ref loaded,0))==1 && catalog::digest(ref loaded,0).len==78,leaf,b"retained root");
    oracle(bytes::view(ref path),true,"");let expected_path=join(bytes::view(ref path),b"catalog.txt");let expected=switch(files::manifest(bytes::view(ref expected_path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};var bytes::Buffer actual=bytes::empty();var usize at=0;while(at<count){assert(bytes::append(ref mut actual,catalog::path(ref loaded,at)).is_ok && bytes::append_byte(ref mut actual,10).is_ok && bytes::append(ref mut actual,manifest::package_value(catalog::package(ref loaded,at),b"uid")).is_ok && bytes::append_byte(ref mut actual,10).is_ok && bytes::append(ref mut actual,catalog::digest(ref loaded,at)).is_ok && bytes::append_byte(ref mut actual,10).is_ok);at+=1;}assert(bytes::equal(bytes::view(ref actual),bytes::view(ref expected)));
    if(bytes::equal(leaf,b"Valid")){assert(catalog::find_uid(ref loaded,b"example.zed")==2 && catalog::find_uid(ref loaded,b"example.aaa")==3 && catalog::find(ref loaded,b"a")==2 && catalog::find(ref loaded,b"b")==3 && bytes::equal(catalog::digest(ref loaded,1),catalog::digest(ref loaded,2)) && bytes::equal(manifest::package_value(catalog::package(ref loaded,2),b"uid"),b"example.aaa"));}
}
@effects(filesystem,console)
fn void rejected(ref u8[] root,ref u8[] leaf,u32 code,u32 reason,usize at){let path=join(root,leaf);var sources::Budget budget=sources::Budget{.files=9,.bytes=17};var usize meter=0;
    let failure=switch(catalog::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(_):{require(false,leaf,b"unexpected success");return;}case .err(e):{e}};let p=catalog::error(ref failure);if(p.code!=code || p.package_index!=at){discard io::write_stderr_u32(p.code);discard io::write_stderr_usize(p.package_index);}require(p.code==code && p.package_index==at && budget.files==9 && budget.bytes==17 && catalog::error_path(ref failure).len>0,leaf,b"rejection/budget");
    if(code==7){assert(p.reason==reason);oracle(bytes::view(ref path),false,directory::reason(p.reason));}else if(code==8){oracle(bytes::view(ref path),false,if(reason==0){"invalid-project-manifest"}else{"invalid-source-syntax"});}else{oracle(bytes::view(ref path),false,catalog::reason(code));}
    if(code==8){assert(catalog::has_input_error(ref failure));let e=catalog::input_error(ref failure);assert(packages::error(e).phase==reason);if(reason==6){assert(bytes::equal(packages::error_input(e),b"module other;\n") && bytes::equal(packages::error_path(e),b"lib.fy") && packages::error(e).start==7 && packages::error(e).end==12);}}else{assert(!catalog::has_input_error(ref failure));}
    if(code==2 || code==3 || code==5){assert(catalog::related_path(ref failure).len>0);}
}
@test
@effects(filesystem,console)
fn void owns_explicit_catalog_and_rejects_identity_path_and_budget_failures(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;let root=switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};
    accepted(bytes::view(ref root),b"Single",1,13);accepted(bytes::view(ref root),b"Valid",3,37);accepted(bytes::view(ref root),b"NearNames",3,37);accepted(bytes::view(ref root),b"Undeclared",1,13);
    rejected(bytes::view(ref root),b"InterveningOverlap",2,0,3);rejected(bytes::view(ref root),b"CaseCollision",3,0,2);rejected(bytes::view(ref root),b"Nested",4,0,1);rejected(bytes::view(ref root),b"DuplicateUID",5,0,1);rejected(bytes::view(ref root),b"MemberLanguage",6,0,1);rejected(bytes::view(ref root),b"RootLanguage",6,0,0);rejected(bytes::view(ref root),b"MissingManifest",8,0,1);rejected(bytes::view(ref root),b"BadMemberSyntax",8,6,1);rejected(bytes::view(ref root),b"WrongCase",7,3,1);rejected(bytes::view(ref root),b"TooMany",1,0,1024);
}
