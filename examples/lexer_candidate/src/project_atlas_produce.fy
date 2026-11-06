module project_atlas_produce;
import project_atlas_write as w;
import project_atlas_facts as facts;
import project_atlas_declarations as declarations;
import project_atlas_calls as calls;
import project_atlas_instances as instances;
import project_atlas_closures as closures;
import project_atlas_header as header;
import project_atlas_current as current;
import project_atlas_model as model;
import project_compiler_input as input;
import project_compiler_core as compiler;
import module_project_identified_ir as identified;
import module_borrow_observations as observations;
import project_atlas_source_version as source_version;
import project_path as work;
import module_unit_body_ir_model as ir;
import project_atlas_schema as schema;
import module_signatures as signatures;
import module_types as types;
import module_public_type_text as coordinates;
import std::bytes as bytes;
import std::vec as v;
pub resource Package{u32 version;facts::Facts facts;v::Vec<bytes::Buffer> declarations;v::Vec<bytes::Buffer> keys;v::Vec<bytes::Buffer> full;calls::Summaries summaries;cleanup{}}
pub fn usize call_count(ref Package p){return calls::count(ref p.summaries);}
pub fn usize external_count(ref Package p){return calls::external_count(ref p.summaries);}
pub fn ref u8[] uid(ref Package p){return facts::value(ref p.facts,b"package_uid");}
pub fn ref u8[] instance(ref Package p){return facts::value(ref p.facts,b"package_instance");}
pub fn Result<Package,u32> package(facts::Facts facts,ref input::Input source,ref ir::Program program,ref coordinates::Coordinates table,ref signatures::Program metadata,ref mut v::Vec<types::Type> type_rows,usize owner,ref mut usize meter){let context=instances::empty();return package_generics(facts,source,program,table,metadata,type_rows,ref context,owner,meter);}
pub fn Result<Package,u32> package_generics(facts::Facts facts,ref input::Input source,ref ir::Program program,ref coordinates::Coordinates table,ref signatures::Program metadata,ref mut v::Vec<types::Type> type_rows,ref instances::Instances context,usize owner,ref mut usize meter){let absent=observations::absent();return package_mode(facts,source,program,table,metadata,type_rows,context,ref absent,owner,3,meter);}
pub fn Result<Package,u32> package_complete(facts::Facts facts,ref input::Input source,ref coordinates::Coordinates table,ref compiler::Complete complete,ref signatures::Program metadata,ref mut v::Vec<types::Type> type_rows,usize owner,ref mut usize meter){let context=instances::empty();return package_version(facts,source,table,complete,metadata,type_rows,ref context,owner,3,meter);}
// Explicit source-backed schemas bind the executable program to Complete here.
// Existing package/package_generics factories retain their v3-only behavior.
pub fn Result<Package,u32> package_version(facts::Facts facts,ref input::Input source,ref coordinates::Coordinates table,ref compiler::Complete complete,ref signatures::Program metadata,ref mut v::Vec<types::Type> type_rows,ref instances::Instances context,usize owner,u32 version,ref mut usize meter){
 discard source_version::admit(source,table,complete,metadata,version,meter)?;
 if(version==1 && (instances::template_count(context)!=0 || instances::count(context)!=0)){return .err(101);}
 return package_mode(facts,source,identified::ir_program(compiler::checked_program(complete)),table,metadata,type_rows,context,identified::borrows(compiler::checked_program(complete)),owner,version,meter);
}
fn Result<Package,u32> package_mode(facts::Facts facts,ref input::Input source,ref ir::Program program,ref coordinates::Coordinates table,ref signatures::Program metadata,ref mut v::Vec<types::Type> type_rows,ref instances::Instances context,ref observations::Table borrowed,usize owner,u32 version,ref mut usize meter){
 let generic=version==3 || version==2 && instances::template_count(context)>0;
 if(version<1 || version>3 || generic!=(facts::value(ref facts,b"generic_interface_hash").len>0)){return .err(101);}
 let rows=if(instances::aggregate_types(context)){declarations::collect_aggregates(source,owner,meter)?}else if(instances::vector_types(context)){declarations::collect_sdk_vectors(source,owner,meter)?}else if(instances::template_count(context)>0){declarations::collect_generics(source,owner,meter)?}else{declarations::collect_all(source,owner,meter)?};
 let projected=closures::build(source,program,context,ref rows,meter)?;if(closures::count(ref projected)==0){return .err(104);}
 var keys=v::empty<bytes::Buffer>();var decl=v::empty<bytes::Buffer>();var hashes=v::empty<bytes::Buffer>();var usize i=0;
 while(i<closures::count(ref projected)){let row=closures::at(ref projected,i);let declaration=schema::declaration_named(source,table,metadata,type_rows,row.declaration,closures::name(ref projected,row),meter)?;var key=bytes::empty();discard closures::key(ref mut key,source,ref projected,row,meter)?;
  if(!v::push<bytes::Buffer>(ref mut keys,key).is_ok){return .err(2);}let hash=declarations::digest(ref facts,bytes::view(ref declaration),meter)?;
  if(!v::push<bytes::Buffer>(ref mut decl,declaration).is_ok){return .err(2);}if(!v::push<bytes::Buffer>(ref mut hashes,hash).is_ok){return .err(2);}i+=1;
 }
 let summaries=if(observations::ready(borrowed)){calls::summaries_observed(source,program,metadata,table,context,ref rows,ref projected,borrowed,meter)?}else{calls::summaries_projected(source,program,metadata,table,context,ref rows,ref projected,meter)?};return .ok(Package{.version,.facts,.declarations=decl,.keys,.full=hashes,.summaries});
}
// Every Package declaration is serialized by schema::declaration_named above.
// This collects declared function/drop effects even for unused templates and
// foreign declarations; it does not count them as executable relations.
pub fn Result<bytes::Buffer,u32> encode(ref Package p,usize total_resolved,usize total_external,ref mut usize meter){let count=v::len<bytes::Buffer>(ref p.declarations);let h=header::header_collected_version(ref p.facts,count,calls::function_count(ref p.summaries),call_count(p)+calls::drop_count(ref p.summaries)+calls::trap_count(ref p.summaries)+calls::unsafe_count(ref p.summaries)+calls::move_count(ref p.summaries)+calls::allocation_count(ref p.summaries)+calls::return_error_count(ref p.summaries)+calls::borrow_count(ref p.summaries),total_external,calls::specialization_count(ref p.summaries),p.version,calls::borrows_observed(ref p.summaries),true,meter)?;let complete=header::completeness_collected_version(total_resolved,total_external,calls::template_count(ref p.summaries),calls::implementation_count(ref p.summaries),p.version,calls::borrows_observed(ref p.summaries),true,meter)?;var bytes::Buffer map=bytes::empty();discard w::raw(ref mut map,b"{",meter)?;if(calls::specialization_count(ref p.summaries)>0){discard w::raw(ref mut map,b"\"specializations\":",meter)?;discard w::raw(ref mut map,calls::specializations(ref p.summaries),meter)?;discard w::raw(ref mut map,b",",meter)?;}discard w::raw(ref mut map,b"\"header\":",meter)?;discard w::raw(ref mut map,bytes::view(ref h),meter)?;discard w::raw(ref mut map,b",\"files\":",meter)?;discard w::raw(ref mut map,facts::array(ref p.facts,b"files"),meter)?;discard w::raw(ref mut map,b",\"modules\":",meter)?;discard w::raw(ref mut map,facts::modules(ref p.facts),meter)?;discard w::raw(ref mut map,b",\"symbols\":[",meter)?;
 var v::Vec<usize> ordered=v::empty<usize>();var usize i=0;while(i<count){if(!v::push<usize>(ref mut ordered,i).is_ok){return .err(2);}var usize at=i;while(at>0){let old=*v::at<usize>(ref ordered,at-1);let left=bytes::view(v::at<bytes::Buffer>(ref p.full,i));let right=bytes::view(v::at<bytes::Buffer>(ref p.full,old));if(!work::charge(meter,left.len+right.len+1)){return .err(3);}if(!model::before(left,right)){break;}{let slot=v::at_mut<usize>(ref mut ordered,at);*slot=old;}{let slot=v::at_mut<usize>(ref mut ordered,at-1);*slot=i;}at-=1;}i+=1;}
 i=0;while(i<count){let at=*v::at<usize>(ref ordered,i);let full=bytes::view(v::at<bytes::Buffer>(ref p.full,at));var bytes::Buffer visible=bytes::empty();discard w::raw(ref mut visible,b"fysym1-sha256-128-",meter)?;discard w::raw(ref mut visible,full[18..50],meter)?;if(i>0){discard w::raw(ref mut map,b",",meter)?;}discard w::raw(ref mut map,b"{\"declaration\":",meter)?;discard w::raw(ref mut map,bytes::view(v::at<bytes::Buffer>(ref p.keys,at)),meter)?;discard w::raw(ref mut map,b",",meter)?;discard w::pair(ref mut map,b"visible",bytes::view(ref visible),meter)?;discard w::raw(ref mut map,b",",meter)?;discard w::pair(ref mut map,b"full",full,meter)?;discard w::raw(ref mut map,b"}",meter)?;i+=1;}
 discard w::raw(ref mut map,b"],\"declarations\":[",meter)?;i=0;while(i<count){if(i>0){discard w::raw(ref mut map,b",",meter)?;}discard w::raw(ref mut map,bytes::view(v::at<bytes::Buffer>(ref p.declarations,i)),meter)?;i+=1;}discard w::raw(ref mut map,b"],\"functions\":",meter)?;discard w::raw(ref mut map,calls::text(ref p.summaries),meter)?;discard w::raw(ref mut map,b",\"completeness\":",meter)?;discard w::raw(ref mut map,bytes::view(ref complete),meter)?;discard w::raw(ref mut map,b"}",meter)?;return w::envelope_version(bytes::view(ref map),p.version,meter);
}
pub fn Result<current::Ready,u32> ready(ref Package p,usize total_resolved,usize total_external,ref mut usize meter){let encoded=encode(p,total_resolved,total_external,meter)?;let expected=switch(bytes::copy(model::raw(facts::expected(ref p.facts),0))){case .ok(p):{p}case .err(_):{return .err(2);}};return if(p.version==2){current::read_v2(encoded,expected,meter)}else if(calls::template_count(ref p.summaries)>0){current::read_generics(encoded,expected,meter)}else{current::read(encoded,expected,meter)};
}
