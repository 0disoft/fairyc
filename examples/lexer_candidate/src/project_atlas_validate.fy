module project_atlas_validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_atlas_symbols as symbols;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub resource Partition{wire::Document document;v::Vec<usize> files;v::Vec<usize> modules;v::Vec<usize> declarations;v::Vec<usize> symbols;v::Vec<usize> functions;v::Vec<usize> completeness;cleanup{}}
pub fn ref json::Tree tree(ref Partition p){return wire::tree(ref p.document);}
pub fn usize header(ref Partition p){return wire::header(ref p.document);}
pub fn usize root(ref Partition p){return wire::map(ref p.document);}
pub fn u32 version(ref Partition p){return wire::version(ref p.document);}
pub fn usize symbol_count(ref Partition p){return v::len<usize>(ref p.symbols);}
pub fn usize symbol(ref Partition p,usize n){return *v::at<usize>(ref p.symbols,n);}
pub fn usize declaration(ref Partition p,usize key,ref mut usize meter){return model::find_key(tree(p),ref p.declarations,b"key",key,meter);}
pub fn usize file_count(ref Partition p){return v::len<usize>(ref p.files);}
pub fn usize function_count(ref Partition p){return v::len<usize>(ref p.functions);}
pub fn usize function(ref Partition p,usize n){return *v::at<usize>(ref p.functions,n);}
pub fn usize completeness_count(ref Partition p){return v::len<usize>(ref p.completeness);}
pub fn usize completeness(ref Partition p,usize n){return *v::at<usize>(ref p.completeness,n);}
fn bool equal(ref json::Tree t,usize at,ref u8[] name,ref u8[] expected){return bytes::equal(wire::string(t,at,name),expected);}
fn bool header_valid(ref json::Tree t,usize h,u32 version,ref mut usize meter){
 if(!equal(t,h,b"snapshot_model",b"fairyc.semantic-snapshot/v0")){return false;}
 if(!equal(t,h,b"compiler_build_algorithm",b"fairyc.compiler-executable-sha256/v1")){return false;}
 if(!equal(t,h,b"package_instance_algorithm",b"fairyc.package-instance/v1")){return false;}
 if(!equal(t,h,b"package_coordinate_algorithm",b"fairyc.package-coordinate/v1")){return false;}
 if(!equal(t,h,b"package_source_algorithm",b"fairyc.package-source/v1")){return false;}
 if(!equal(t,h,b"source_file_algorithm",b"fairyc.source-file/v1")){return false;}
 if(!equal(t,h,b"line_map_algorithm",b"fairyc.source-line-map/v1")){return false;}
 if(!equal(t,h,b"symbol_algorithm",b"fairyc.symbol-id/v1-sha256-128+256")){return false;}
 if(!equal(t,h,b"interface_algorithm",b"fairyc.public-interface/v5")){return false;}
 if(!equal(t,h,b"target_scope",b"target-independent")){return false;}
 if(!equal(t,h,b"target",b"")){return false;}
 if(!wire::hex(wire::string(t,h,b"compiler_build"),b"fairyc.compiler-executable-sha256/v1:",64,true)){return false;}
 if(!wire::hex(wire::string(t,h,b"package_instance"),b"fypi1-sha256-",64,true)){return false;}
 if(!wire::hex(wire::string(t,h,b"package_coordinate"),b"fypc1-sha256-",64,true)){return false;}
 if(!wire::hex(wire::string(t,h,b"package_source"),b"fysrc1-sha256-",64,true)){return false;}
 if(!wire::hex(wire::string(t,h,b"interface_hash"),b"fyif5-sha256-",64,true)){return false;}
 if(!model::text(wire::string(t,h,b"language")) || !model::text(wire::string(t,h,b"package_uid")) || !model::text(wire::string(t,h,b"package_source_kind"))){return false;}
 if(wire::string(t,h,b"runtime_abi").len!=0 && !model::text(wire::string(t,h,b"runtime_abi"))){return false;}
 let g=wire::field(t,h,b"generic_interface_algorithm");if(g<json::count(t) && (version!=3 || !equal(t,h,b"generic_interface_algorithm",b"fairyc.generic-public-interface/v0") || !wire::hex(wire::string(t,h,b"generic_interface_hash"),b"fygif0-sha256-",64,true))){return false;}
 if(!model::sorted(t,wire::field(t,h,b"features"),meter) || !model::sorted(t,wire::field(t,h,b"dependencies"),meter) || !model::sorted(t,wire::field(t,h,b"omissions"),meter)){return false;}
 let deps=switch(model::rows(t,wire::field(t,h,b"dependencies"),32,meter)){case .ok(p):{p}case .err(_):{return false;}};let refs=switch(model::rows(t,wire::field(t,h,b"package_references"),33,meter)){case .ok(p):{p}case .err(_):{return false;}};
 if(v::len<usize>(ref refs)!=v::len<usize>(ref deps)+1){return false;}var usize i=0;var bool self=false;
 while(i<v::len<usize>(ref deps)){let d=json::value(t,*v::at<usize>(ref deps,i));if(!wire::hex(d,b"fypi1-sha256-",64,true) || bytes::equal(d,wire::string(t,h,b"package_instance"))){return false;}i+=1;}
 i=0;while(i<v::len<usize>(ref refs)){let r=*v::at<usize>(ref refs,i);let instance=wire::string(t,r,b"instance");let coordinate=wire::string(t,r,b"coordinate");if(!work::charge(meter,256) || !model::text(wire::string(t,r,b"uid")) || !wire::hex(instance,b"fypi1-sha256-",64,true) || !wire::hex(coordinate,b"fypc1-sha256-",64,true) || i>0 && !model::before(wire::string(t,*v::at<usize>(ref refs,i-1),b"instance"),instance)){return false;}
  if(bytes::equal(instance,wire::string(t,h,b"package_instance"))){self=equal(t,r,b"uid",wire::string(t,h,b"package_uid")) && bytes::equal(coordinate,wire::string(t,h,b"package_coordinate"));}
  else{var usize j=0;var bool found=false;while(j<v::len<usize>(ref deps)){if(bytes::equal(json::value(t,*v::at<usize>(ref deps,j)),instance)){found=true;}j+=1;}if(!found){return false;}}
  var usize j=0;while(j<i){let prior=*v::at<usize>(ref refs,j);if(equal(t,prior,b"coordinate",coordinate) && !model::eq(t,prior,r,b"uid")){return false;}j+=1;}i+=1;
 }return self;
}
fn bool fields_valid(ref json::Tree t,usize at,ref u8[] instance,ref v::Vec<usize> files,ref mut usize meter){let rows=switch(model::rows(t,at,4096,meter)){case .ok(p):{p}case .err(_):{return false;}};var usize i=0;while(i<v::len<usize>(ref rows)){let r=*v::at<usize>(ref rows,i);if(!model::identifier(wire::string(t,r,b"name")) || !model::type(t,wire::field(t,r,b"type"),0,meter) || !model::span(t,wire::field(t,r,b"span"),instance,files,meter)){return false;}var usize j=0;while(j<i){if(!work::charge(meter,128) || model::eq(t,r,*v::at<usize>(ref rows,j),b"name")){return false;}j+=1;}i+=1;}return true;}
fn bool declaration_valid(ref json::Tree t,usize at,ref u8[] instance,ref v::Vec<usize> files,ref v::Vec<usize> modules,ref mut usize meter){
 let key=wire::field(t,at,b"key");if(!equal(t,key,b"package_instance",instance) || !model::kind(wire::string(t,key,b"kind")) || !model::identifier(wire::string(t,key,b"name")) || model::find(t,modules,b"path",wire::string(t,key,b"module"),meter)==json::count(t) || !equal(t,at,b"provenance",b"compiler-declared") || !model::span(t,wire::field(t,at,b"span"),instance,files,meter) || !model::type(t,wire::field(t,at,b"type"),0,meter)){return false;}
 let owner=wire::field(t,at,b"owner");if(json::token(t,owner).kind!=11 && !model::type(t,owner,0,meter)){return false;}
 if(!model::sorted(t,wire::field(t,at,b"effects"),meter) || !model::sorted(t,wire::field(t,at,b"drop_effects"),meter) || !fields_valid(t,wire::field(t,at,b"fields"),instance,files,meter)){return false;}
 let params=switch(model::rows(t,wire::field(t,at,b"parameters"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};var usize i=0;while(i<v::len<usize>(ref params)){let r=*v::at<usize>(ref params,i);if(!model::identifier(wire::string(t,r,b"name")) || !model::type(t,wire::field(t,r,b"type"),0,meter) || !model::span(t,wire::field(t,r,b"span"),instance,files,meter)){return false;}i+=1;}
 let variants=switch(model::rows(t,wire::field(t,at,b"variants"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};i=0;while(i<v::len<usize>(ref variants)){let r=*v::at<usize>(ref variants,i);if(!model::identifier(wire::string(t,r,b"name")) || !model::span(t,wire::field(t,r,b"span"),instance,files,meter) || !fields_valid(t,wire::field(t,r,b"fields"),instance,files,meter)){return false;}var usize j=0;while(j<i){if(!work::charge(meter,128) || model::eq(t,r,*v::at<usize>(ref variants,j),b"name")){return false;}j+=1;}i+=1;}return true;
}
fn bool spans_valid(ref json::Tree t,usize at,ref u8[] instance,ref v::Vec<usize> files,ref mut usize meter){let rows=switch(model::rows(t,at,4096,meter)){case .ok(p):{p}case .err(_):{return false;}};var usize i=0;while(i<v::len<usize>(ref rows)){let r=*v::at<usize>(ref rows,i);if(!model::span(t,r,instance,files,meter) || i>0 && model::spans(t,*v::at<usize>(ref rows,i-1),r,meter)>=0){return false;}i+=1;}return true;}
fn bool function_valid(ref json::Tree t,usize h,usize at,ref u8[] instance,ref v::Vec<usize> files,ref v::Vec<usize> declarations,ref mut usize meter){
 let key=wire::field(t,at,b"function");if(!equal(t,key,b"package_instance",instance) || !equal(t,key,b"kind",b"function") || model::find_key(t,declarations,b"key",key,meter)==json::count(t) || !equal(t,at,b"provenance",b"compiler-observed") || !model::sorted(t,wire::field(t,at,b"trap_categories"),meter)){return false;}
 let refs=switch(model::rows(t,wire::field(t,h,b"package_references"),33,meter)){case .ok(p):{p}case .err(_):{return false;}};
 let calls=switch(model::rows(t,wire::field(t,at,b"calls"),65536,meter)){case .ok(p):{p}case .err(_):{return false;}};var usize i=0;
 while(i<v::len<usize>(ref calls)){let call=*v::at<usize>(ref calls,i);let target=wire::field(t,call,b"target");let span=wire::field(t,call,b"span");if(!model::span(t,span,instance,files,meter) || !wire::hex(wire::string(t,target,b"package_instance"),b"fypi1-sha256-",64,false) || !model::qualified(wire::string(t,target,b"module")) || !model::identifier(wire::string(t,target,b"name")) || !model::kind(wire::string(t,target,b"kind")) || model::find(t,ref refs,b"instance",wire::string(t,target,b"package_instance"),meter)==json::count(t)){return false;}
 if(i>0){let previous=*v::at<usize>(ref calls,i-1);let order=model::spans(t,wire::field(t,previous,b"span"),span,meter);if(order>0 || order==0 && model::keys(t,wire::field(t,previous,b"target"),target,meter)>=0){return false;}}
 if(equal(t,call,b"target_kind",b"resolved")){if(!equal(t,call,b"certainty",b"exact") || wire::string(t,call,b"abi").len!=0 || wire::string(t,call,b"link_name").len!=0 || !equal(t,target,b"kind",b"function")){return false;}if(equal(t,target,b"package_instance",instance) && model::find_key(t,declarations,b"key",target,meter)==json::count(t)){return false;}}
 else if(equal(t,call,b"target_kind",b"external")){if(!equal(t,call,b"certainty",b"opaque") || !model::text(wire::string(t,call,b"abi")) || !model::text(wire::string(t,call,b"link_name")) || !equal(t,target,b"package_instance",instance) || !equal(t,target,b"kind",b"foreign-function") || model::find_key(t,declarations,b"key",target,meter)==json::count(t)){return false;}}
 else{return false;}i+=1;
 }
 if(!spans_valid(t,wire::field(t,at,b"unsafe_sites"),instance,files,meter)){return false;}
 if(!spans_valid(t,wire::field(t,at,b"move_sites"),instance,files,meter)){return false;}
 if(!spans_valid(t,wire::field(t,at,b"drop_sites"),instance,files,meter)){return false;}
 if(!spans_valid(t,wire::field(t,at,b"allocation_sites"),instance,files,meter)){return false;}
 if(!spans_valid(t,wire::field(t,at,b"return_error_sites"),instance,files,meter)){return false;}
 let borrows=switch(model::rows(t,wire::field(t,at,b"borrows"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};i=0;while(i<v::len<usize>(ref borrows)){let r=*v::at<usize>(ref borrows,i);if(!model::span(t,wire::field(t,r,b"span"),instance,files,meter) || !equal(t,r,b"kind",b"shared") && !equal(t,r,b"kind",b"mutable")){return false;}if(i>0){let old=*v::at<usize>(ref borrows,i-1);let order=model::spans(t,wire::field(t,old,b"span"),wire::field(t,r,b"span"),meter);if(order>0 || order==0 && (!model::eq(t,old,r,b"kind") && !model::before(wire::string(t,old,b"kind"),wire::string(t,r,b"kind")) || model::eq(t,old,r,b"kind") && (wire::boolean(t,old,b"escapes") || !wire::boolean(t,r,b"escapes")))){return false;}}i+=1;}return true;
}
fn bool completeness_valid(ref json::Tree t,usize h,ref v::Vec<usize> records,ref mut usize meter){if(v::len<usize>(records)==0){return false;}var bool partial=false;var v::Vec<bytes::Buffer> omissions=v::empty<bytes::Buffer>();var usize i=0;
 while(i<v::len<usize>(records)){let r=*v::at<usize>(records,i);let relation=wire::string(t,r,b"relation");let status=wire::string(t,r,b"status");if(!model::text(relation) || i>0 && !model::before(wire::string(t,*v::at<usize>(records,i-1),b"relation"),relation) || wire::integer(t,r,b"resolved")>9223372036854775807 || wire::integer(t,r,b"external")>9223372036854775807 || wire::integer(t,r,b"unknown")>9223372036854775807){return false;}
 let reasons=switch(model::rows(t,wire::field(t,r,b"reasons"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};
 if(bytes::equal(status,b"exact")){if(v::len<usize>(ref reasons)!=0){return false;}}
 else{if(!bytes::equal(status,b"partial") && !bytes::equal(status,b"unavailable") || v::len<usize>(ref reasons)==0){return false;}partial=true;var usize j=0;while(j<v::len<usize>(ref reasons)){let reason=json::value(t,*v::at<usize>(ref reasons,j));if(!model::text(reason) || !work::charge(meter,relation.len+status.len+reason.len+3)){return false;}var bytes::Buffer value=bytes::empty();if(!bytes::append(ref mut value,relation).is_ok || !bytes::append_byte(ref mut value,58).is_ok || !bytes::append(ref mut value,status).is_ok || !bytes::append_byte(ref mut value,58).is_ok || !bytes::append(ref mut value,reason).is_ok){return false;}if(!v::push<bytes::Buffer>(ref mut omissions,value).is_ok){return false;}j+=1;}}
 i+=1;
 }
 if(!equal(t,h,b"generation_status",if(partial){b"partial"}else{b"exact"})){return false;}
 let expected=switch(model::rows(t,wire::field(t,h,b"omissions"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};if(v::len<usize>(ref expected)!=v::len<bytes::Buffer>(ref omissions)){return false;}
 // Header omissions are already sorted/unique. Compare sets without moving owned buffers.
 i=0;while(i<v::len<usize>(ref expected)){let value=json::value(t,*v::at<usize>(ref expected,i));var usize matches=0;var usize j=0;while(j<v::len<bytes::Buffer>(ref omissions)){let actual=bytes::view(v::at<bytes::Buffer>(ref omissions,j));if(!work::charge(meter,value.len+actual.len+1)){return false;}if(bytes::equal(value,actual)){matches+=1;}j+=1;}if(matches!=1){return false;}i+=1;}return true;
}
// Semantically checked partition; current-source expectations are a separate gate.
pub fn Result<Partition,u32> validate(wire::Document document,ref mut usize meter){
 let t=wire::tree(ref document);let h=wire::header(ref document);let map=wire::map(ref document);if(!header_valid(t,h,wire::version(ref document),meter)){return .err(7);}
 let files=model::rows(t,wire::field(t,map,b"files"),4096,meter)?;let modules=model::rows(t,wire::field(t,map,b"modules"),4096,meter)?;let declarations=model::rows(t,wire::field(t,map,b"declarations"),4096,meter)?;let ids=model::rows(t,wire::field(t,map,b"symbols"),4096,meter)?;let functions=model::rows(t,wire::field(t,map,b"functions"),4096,meter)?;let complete=model::rows(t,wire::field(t,map,b"completeness"),4096,meter)?;
 if(v::len<usize>(ref files)==0 || v::len<usize>(ref files)!=v::len<usize>(ref modules) || v::len<usize>(ref declarations)==0 || v::len<usize>(ref ids)!=v::len<usize>(ref declarations)){return .err(7);}
 var usize token=0;while(token<json::count(t)){if(json::value(t,token).len>4096){return .err(3);}token+=1;}
 let instance=wire::string(t,h,b"package_instance");var usize i=0;
 while(i<v::len<usize>(ref files)){let r=*v::at<usize>(ref files,i);if(!equal(t,r,b"package_instance",instance) || !wire::hex(wire::string(t,r,b"digest"),b"fyfile1-sha256-",64,true) || !wire::hex(wire::string(t,r,b"line_map"),b"fylm1-sha256-",64,true) || !equal(t,r,b"kind",b"source") && !equal(t,r,b"kind",b"test") || i>0 && !model::before(wire::string(t,*v::at<usize>(ref files,i-1),b"path"),wire::string(t,r,b"path"))){return .err(7);}i+=1;}
 i=0;while(i<v::len<usize>(ref modules)){let r=*v::at<usize>(ref modules,i);if(!equal(t,r,b"package_instance",instance) || !model::module_path(wire::string(t,r,b"source"),wire::string(t,r,b"path")) || model::find(t,ref files,b"path",wire::string(t,r,b"source"),meter)==json::count(t) || i>0 && !model::before(wire::string(t,*v::at<usize>(ref modules,i-1),b"path"),wire::string(t,r,b"path"))){return .err(7);}var usize j=0;while(j<i){if(!work::charge(meter,wire::string(t,r,b"source").len+wire::string(t,*v::at<usize>(ref modules,j),b"source").len+1) || model::eq(t,r,*v::at<usize>(ref modules,j),b"source")){return .err(7);}j+=1;}i+=1;}
 i=0;while(i<v::len<usize>(ref declarations)){let r=*v::at<usize>(ref declarations,i);if(!declaration_valid(t,r,instance,ref files,ref modules,meter) || i>0 && model::keys(t,wire::field(t,*v::at<usize>(ref declarations,i-1),b"key"),wire::field(t,r,b"key"),meter)>=0){return .err(7);}i+=1;}
 i=0;while(i<v::len<usize>(ref ids)){let r=*v::at<usize>(ref ids,i);let decl=model::find_key(t,ref declarations,b"key",wire::field(t,r,b"declaration"),meter);if(decl==json::count(t) || !symbols::matches(t,h,decl,r,meter) || i>0 && !model::before(wire::string(t,*v::at<usize>(ref ids,i-1),b"full"),wire::string(t,r,b"full"))){return .err(7);}var usize j=0;while(j<i){let old=*v::at<usize>(ref ids,j);if(!work::charge(meter,256) || model::eq(t,r,old,b"visible") || model::keys(t,wire::field(t,r,b"declaration"),wire::field(t,old,b"declaration"),meter)==0){return .err(7);}j+=1;}i+=1;}
 var usize relations=0;i=0;while(i<v::len<usize>(ref functions)){let r=*v::at<usize>(ref functions,i);if(!function_valid(t,h,r,instance,ref files,ref declarations,meter) || i>0 && model::keys(t,wire::field(t,*v::at<usize>(ref functions,i-1),b"function"),wire::field(t,r,b"function"),meter)>=0){return .err(7);}
 relations+=wire::count(t,wire::field(t,r,b"calls"));
 relations+=wire::count(t,wire::field(t,r,b"trap_categories"));
 relations+=wire::count(t,wire::field(t,r,b"unsafe_sites"));
 relations+=wire::count(t,wire::field(t,r,b"move_sites"));
 relations+=wire::count(t,wire::field(t,r,b"drop_sites"));
 relations+=wire::count(t,wire::field(t,r,b"allocation_sites"));
 relations+=wire::count(t,wire::field(t,r,b"return_error_sites"));
 relations+=wire::count(t,wire::field(t,r,b"borrows"));
 i+=1;}
 i=0;while(i<v::len<usize>(ref declarations)){let r=*v::at<usize>(ref declarations,i);let key=wire::field(t,r,b"key");if(equal(t,key,b"kind",b"function") && model::find_key(t,ref functions,b"function",key,meter)==json::count(t)){return .err(7);}i+=1;}
 let counts=wire::field(t,h,b"counts");if(wire::integer(t,counts,b"files")!=v::len<usize>(ref files) || wire::integer(t,counts,b"modules")!=v::len<usize>(ref modules) || wire::integer(t,counts,b"symbols")!=v::len<usize>(ref ids) || wire::integer(t,counts,b"functions")!=v::len<usize>(ref functions) || wire::integer(t,counts,b"relations")!=relations || wire::integer(t,counts,b"completeness")!=v::len<usize>(ref complete) || !completeness_valid(t,h,ref complete,meter)){return .err(7);}
 return .ok(Partition{.document,.files,.modules,.declarations,.symbols=ids,.functions,.completeness=complete});
}
