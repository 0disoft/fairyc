module project_atlas_header;
import project_atlas_facts as facts;
import project_atlas_write as w;
import std::bytes as bytes;
pub fn ref u8[] relations(){return b"allocates|borrows|calls|drops|effects|generic-specializations|generic-template-bodies|may-trap|moves|reads|returns-error|test-calls|uses-unsafe|writes";}
fn bool exact(ref u8[] relation,usize external,usize omitted_cleanups,bool borrowed,bool effects){return bytes::equal(relation,b"calls") && external==0 && omitted_cleanups==0 || bytes::equal(relation,b"drops") || bytes::equal(relation,b"may-trap") || bytes::equal(relation,b"uses-unsafe") || bytes::equal(relation,b"moves") || bytes::equal(relation,b"allocates") || bytes::equal(relation,b"returns-error") || bytes::equal(relation,b"generic-specializations") || borrowed && bytes::equal(relation,b"borrows") || effects && bytes::equal(relation,b"effects");}
fn ref u8[] status(ref u8[] relation,usize external,usize omitted_cleanups,bool borrowed,bool effects){if(bytes::equal(relation,b"calls") && (external!=0 || omitted_cleanups!=0)){return b"partial";}if(exact(relation,external,omitted_cleanups,borrowed,effects)){return b"exact";}return b"unavailable";}
fn ref u8[] reason(ref u8[] relation){if(bytes::equal(relation,b"generic-template-bodies")){return b"bodies-checked-only-after-specialization";}return b"candidate-atlas-not-collected";}
pub fn Result<bytes::Buffer,u32> completeness(usize resolved,usize external,ref mut usize meter){return completeness_generics(resolved,external,0,0,meter);}
pub fn Result<bytes::Buffer,u32> completeness_generics(usize resolved,usize external,usize templates,usize implementations,ref mut usize meter){return completeness_version(resolved,external,templates,implementations,3,meter);}
fn bool included(ref u8[] relation,u32 version){return version>1 || !bytes::equal(relation,b"generic-specializations") && !bytes::equal(relation,b"generic-template-bodies");}
pub fn Result<bytes::Buffer,u32> completeness_version(usize resolved,usize external,usize templates,usize implementations,u32 version,ref mut usize meter){return completeness_observed_version(resolved,external,templates,implementations,version,false,meter);}
pub fn Result<bytes::Buffer,u32> completeness_observed_version(usize resolved,usize external,usize templates,usize implementations,u32 version,bool borrowed,ref mut usize meter){
 return completeness_collected_version(resolved,external,0,templates,implementations,version,borrowed,false,meter);
}
// Effects describe retained declaration authority, not behavior beyond an FFI
// boundary or unchecked generic bodies. Legacy header factories have no proof
// that every declaration passed the source schema and leave effects unavailable.
pub fn Result<bytes::Buffer,u32> completeness_collected_version(usize resolved,usize external,usize omitted_cleanups,usize templates,usize implementations,u32 version,bool borrowed,bool effects,ref mut usize meter){
 if(version<1 || version>3 || version==1 && (templates!=0 || implementations!=0)){return .err(101);}
 var out=bytes::empty();discard w::raw(ref mut out,b"[",meter)?;let names=relations();var usize start=0;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let relation=names[start..end];if(!included(relation,version)){start=end+1;continue;}if(start!=0){discard w::raw(ref mut out,b",",meter)?;}
  discard w::raw(ref mut out,b"{",meter)?;discard w::pair(ref mut out,b"relation",relation,meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"status",status(relation,external,omitted_cleanups,borrowed,effects),meter)?;
  discard w::raw(ref mut out,b",\"resolved\":",meter)?;discard w::number(ref mut out,if(bytes::equal(relation,b"calls")){resolved}else if(bytes::equal(relation,b"generic-specializations")){implementations}else{0},meter)?;
  discard w::raw(ref mut out,b",\"external\":",meter)?;discard w::number(ref mut out,if(bytes::equal(relation,b"calls")){external}else{0},meter)?;
  discard w::raw(ref mut out,b",\"unknown\":",meter)?;discard w::number(ref mut out,if(bytes::equal(relation,b"calls")){omitted_cleanups}else if(bytes::equal(relation,b"generic-template-bodies")){templates}else{0},meter)?;discard w::raw(ref mut out,b",\"reasons\":[",meter)?;if(bytes::equal(relation,b"calls")){
   if(external!=0){discard w::string(ref mut out,b"external-ffi",meter)?;}
   if(omitted_cleanups!=0){if(external!=0){discard w::raw(ref mut out,b",",meter)?;}discard w::string(ref mut out,b"ordinary-cleanup-call-not-retained",meter)?;}
  }else if(!exact(relation,external,omitted_cleanups,borrowed,effects)){discard w::string(ref mut out,reason(relation),meter)?;}
  discard w::raw(ref mut out,b"]}",meter)?;start=end+1;
 }discard w::raw(ref mut out,b"]",meter)?;return .ok(out);
}
fn Result<bool,u32> omission(ref mut bytes::Buffer out,ref u8[] relation,ref u8[] status,ref u8[] reason,bool first,ref mut usize meter){
 if(!first){discard w::raw(out,b",",meter)?;}
 var name=bytes::empty();discard w::raw(ref mut name,relation,meter)?;discard w::raw(ref mut name,b":",meter)?;
 discard w::raw(ref mut name,status,meter)?;discard w::raw(ref mut name,b":",meter)?;discard w::raw(ref mut name,reason,meter)?;
 return w::string(out,bytes::view(ref name),meter);
}
fn Result<bool,u32> omissions(ref mut bytes::Buffer out,usize external,usize omitted_cleanups,u32 version,bool borrowed,bool effects,ref mut usize meter){
 discard w::raw(out,b"[",meter)?;let names=relations();var usize start=0;var bool first=true;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let relation=names[start..end];
  if(bytes::equal(relation,b"calls")){
   if(external!=0){discard omission(out,relation,b"partial",b"external-ffi",first,meter)?;first=false;}
   if(omitted_cleanups!=0){discard omission(out,relation,b"partial",b"ordinary-cleanup-call-not-retained",first,meter)?;first=false;}
  }else if(included(relation,version) && !exact(relation,external,omitted_cleanups,borrowed,effects)){
   discard omission(out,relation,b"unavailable",reason(relation),first,meter)?;first=false;
  }start=end+1;
 }return w::raw(out,b"]",meter);
}
pub fn Result<bytes::Buffer,u32> header(ref facts::Facts f,usize declarations,usize functions,usize calls,usize external,ref mut usize meter){return header_generics(f,declarations,functions,calls,external,0,meter);}
pub fn Result<bytes::Buffer,u32> header_generics(ref facts::Facts f,usize declarations,usize functions,usize calls,usize external,usize specializations,ref mut usize meter){return header_version(f,declarations,functions,calls,external,specializations,3,meter);}
pub fn Result<bytes::Buffer,u32> header_version(ref facts::Facts f,usize declarations,usize functions,usize calls,usize external,usize specializations,u32 version,ref mut usize meter){return header_observed_version(f,declarations,functions,calls,external,specializations,version,false,meter);}
pub fn Result<bytes::Buffer,u32> header_observed_version(ref facts::Facts f,usize declarations,usize functions,usize calls,usize external,usize specializations,u32 version,bool borrowed,ref mut usize meter){
 return header_collected_version(f,declarations,functions,calls,external,0,specializations,version,borrowed,false,meter);
}
pub fn Result<bytes::Buffer,u32> header_collected_version(ref facts::Facts f,usize declarations,usize functions,usize calls,usize external,usize omitted_cleanups,usize specializations,u32 version,bool borrowed,bool effects,ref mut usize meter){
 if(version<1 || version>3 || version==1 && (specializations!=0 || facts::value(f,b"generic_interface_hash").len!=0) || (version==3 || specializations>0) && facts::value(f,b"generic_interface_hash").len==0){return .err(101);}
 var logical=bytes::empty();discard w::raw(ref mut logical,b"fairyc.fymap-logical/v",meter)?;discard w::number(ref mut logical,version as usize,meter)?;
 var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;discard w::pair(ref mut out,b"logical_schema",bytes::view(ref logical),meter)?;discard w::raw(ref mut out,b",\"snapshot_model\":\"fairyc.semantic-snapshot/v0\",",meter)?;discard w::pair(ref mut out,b"compiler_build",facts::value(f,b"compiler_build"),meter)?;discard w::raw(ref mut out,b",\"compiler_build_algorithm\":\"fairyc.compiler-executable-sha256/v1\",",meter)?;
 let names=b"language|package_uid|package_instance|package_coordinate|package_source|package_source_kind|runtime_abi";var usize start=0;while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let name=names[start..end];discard w::pair(ref mut out,name,facts::value(f,name),meter)?;discard w::raw(ref mut out,b",",meter)?;start=end+1;}
 discard w::raw(ref mut out,b"\"target_scope\":\"target-independent\",\"target\":\"\",\"package_instance_algorithm\":\"fairyc.package-instance/v1\",\"package_coordinate_algorithm\":\"fairyc.package-coordinate/v1\",\"package_source_algorithm\":\"fairyc.package-source/v1\",\"source_file_algorithm\":\"fairyc.source-file/v1\",\"line_map_algorithm\":\"fairyc.source-line-map/v1\",\"symbol_algorithm\":\"fairyc.symbol-id/v1-sha256-128+256\",\"interface_algorithm\":\"fairyc.public-interface/v5\",",meter)?;discard w::pair(ref mut out,b"interface_hash",facts::value(f,b"interface_hash"),meter)?;if(facts::value(f,b"generic_interface_hash").len>0){discard w::raw(ref mut out,b",\"generic_interface_algorithm\":\"fairyc.generic-public-interface/v0\",",meter)?;discard w::pair(ref mut out,b"generic_interface_hash",facts::value(f,b"generic_interface_hash"),meter)?;}let arrays=b"features|dependencies|package_references";start=0;while(start<arrays.len){var usize end=start;while(end<arrays.len && arrays[end]!=124){end+=1;}let name=arrays[start..end];discard w::raw(ref mut out,b",",meter)?;discard w::string(ref mut out,name,meter)?;discard w::raw(ref mut out,b":",meter)?;discard w::raw(ref mut out,facts::array(f,name),meter)?;start=end+1;}
 discard w::raw(ref mut out,b",\"generation_status\":\"partial\",\"omissions\":",meter)?;discard omissions(ref mut out,external,omitted_cleanups,version,borrowed,effects,meter)?;discard w::raw(ref mut out,b",\"counts\":{",meter)?;if(specializations>0){discard w::raw(ref mut out,b"\"specializations\":",meter)?;discard w::number(ref mut out,specializations,meter)?;discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"\"files\":",meter)?;discard w::number(ref mut out,facts::files(f),meter)?;discard w::raw(ref mut out,b",\"modules\":",meter)?;discard w::number(ref mut out,facts::files(f),meter)?;discard w::raw(ref mut out,b",\"symbols\":",meter)?;discard w::number(ref mut out,declarations,meter)?;discard w::raw(ref mut out,b",\"functions\":",meter)?;discard w::number(ref mut out,functions,meter)?;discard w::raw(ref mut out,b",\"relations\":",meter)?;discard w::number(ref mut out,calls,meter)?;discard w::raw(ref mut out,b",\"completeness\":",meter)?;discard w::number(ref mut out,if(version==1){12}else{14},meter)?;discard w::raw(ref mut out,b"}}",meter)?;return .ok(out);}
