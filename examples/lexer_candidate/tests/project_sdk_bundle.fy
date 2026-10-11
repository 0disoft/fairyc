module tests::project_sdk_bundle;
import project_sdk_bundle as sdk;
import project_source_set as sources;
import project_import_path as paths;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::text as text;
import std::process as process;
import std::vec as v;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem)
fn bytes::Buffer read(ref u8[] root,ref u8[] leaf){let path=join(root,leaf);return switch(files::load(bytes::view(ref path))){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;return switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem)
fn v::Vec<sdk::File> input(ref u8[] root,usize count,u32 mode){var v::Vec<sdk::File> input=v::empty<sdk::File>();var usize index=0;while(index<count){let usize id=if(mode==1 && index==0){0}else if(mode==2 && index==0){9}else if(mode==3 && index==0){2}else{count-index};var bytes::Buffer body=read(root,sdk::path(if(mode==3){id-1}else{count-index-1}));if(mode==4 && index==0){assert(bytes::append_byte(ref mut body,32).is_ok);}assert(v::push<sdk::File>(ref mut input,sdk::file(id,body)).is_ok);index+=1;}return input;}
fn void line(ref mut bytes::Buffer output,ref u8[] value){assert(bytes::append(output,value).is_ok && bytes::append_byte(output,10).is_ok);}
@effects(filesystem)
fn void profile(ref u8[] root,usize count,bool candidate,ref u8[] oracle){let owned=input(root,count,0);var usize total=0;var usize index=0;while(index<count){total+=sdk::file_size(v::at<sdk::File>(ref owned,index));index+=1;}var sources::Budget budget=sources::Budget{.files=4096-count,.bytes=67108864-total};var usize meter=0;
    let bundle=switch(sdk::assemble(owned,candidate,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(budget.files==4096 && budget.bytes==67108864 && sdk::count(ref bundle)==count && sdk::vector(ref bundle)==(count==8));var bytes::Buffer record=bytes::empty();line(ref mut record,sdk::uid());line(ref mut record,sdk::package_name());line(ref mut record,sdk::version());line(ref mut record,sdk::language());line(ref mut record,sdk::runtime_abi());line(ref mut record,sdk::package_path());line(ref mut record,sdk::sdk_identity(ref bundle));line(ref mut record,sdk::node_id());line(ref mut record,sdk::source_identity(ref bundle));line(ref mut record,sdk::digest(ref bundle));
    index=0;while(index<count){let original=read(root,sdk::path(index));assert(bytes::equal(sdk::source(ref bundle,index),bytes::view(ref original)) && sdk::find(ref bundle,sdk::name(index))==index+1);line(ref mut record,sdk::path(index));line(ref mut record,sdk::name(index));let tree=sdk::parsed(ref bundle,index);var usize declaration=1;while(declaration<syntax::declaration_count(tree)){if(syntax::declaration_at(tree,declaration).kind==2){let path=switch(paths::read(tree,declaration,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};line(ref mut record,paths::name(ref path));}declaration+=1;}line(ref mut record,b"");index+=1;}
    assert(sdk::find(ref bundle,b"std")==0 && sdk::find(ref bundle,b"std::missing")==0 && sdk::find(ref bundle,b"Std::io")==0 && sdk::find(ref bundle,b"std::vec")==if(count==8){8}else{0});let expected=read(root,oracle);assert(bytes::equal(bytes::view(ref record),bytes::view(ref expected)));
}
@test
@effects(filesystem)
fn void authenticates_exact_core_and_vector_profiles(){let owned=root();let path=bytes::view(ref owned);profile(path,7,false,b"core.txt");profile(path,8,false,b"vector.txt");profile(path,8,true,b"candidate.txt");}
@effects(filesystem)
fn void rejected(ref u8[] root,usize count,bool candidate,u32 mode,u32 code){let owned=input(root,count,mode);var sources::Budget budget=sources::Budget{.files=if(mode==5){4090}else if(mode==6){4097}else{17},.bytes=if(mode==7){67108864}else if(mode==8){67108865}else{31}};let sources::Budget before=budget;var usize meter=if(mode==9){16777216}else if(mode==10){16777217}else{0};let problem=switch(sdk::assemble(owned,candidate,ref mut budget,ref mut meter)){case .ok(_):{assert(false);return;}case .err(value):{value}};assert(problem.code==code && budget.files==before.files && budget.bytes==before.bytes);}
@test
@effects(filesystem)
fn void rejects_changed_incomplete_and_over_budget_bundles(){let owned=root();let path=bytes::view(ref owned);rejected(path,6,false,0,2);rejected(path,7,true,0,2);rejected(path,7,false,1,1);rejected(path,7,false,2,1);rejected(path,7,false,3,1);rejected(path,7,false,4,3);rejected(path,7,false,5,5);rejected(path,7,false,6,5);rejected(path,7,false,7,5);rejected(path,7,false,8,5);rejected(path,7,false,9,5);rejected(path,7,false,10,5);}
