module project_atlas_specializations;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_atlas_binders as binders;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Checks claimed observations, not current-source/compiler authorship. Cross-
// package implementation existence belongs to the complete workspace join.
fn bool equal(ref json::Tree t,usize a,usize b,ref u8[] field,ref mut usize meter){
 let x=wire::field(t,a,field);let y=wire::field(t,b,field);if(x==json::count(t) || y==json::count(t)){return false;}
 let left=model::raw(t,x);let right=model::raw(t,y);return work::charge(meter,left.len+right.len+1) && bytes::equal(left,right);
}
fn bool contains(ref json::Tree t,usize outer,usize inner,ref mut usize meter){
 return work::charge(meter,256) && model::eq(t,outer,inner,b"package_instance") && model::eq(t,outer,inner,b"source") && wire::integer(t,outer,b"start_byte")<=wire::integer(t,inner,b"start_byte") && wire::integer(t,inner,b"start_byte")<=wire::integer(t,inner,b"end_byte") && wire::integer(t,inner,b"end_byte")<=wire::integer(t,outer,b"end_byte");
}
fn bool scalar(ref json::Tree t,usize at,ref mut usize meter){
 if(!model::type(t,at,0,meter) || !bytes::equal(wire::string(t,at,b"kind"),b"builtin")){return false;}
 let name=wire::string(t,at,b"name");let names=b"bool|i8|i16|i32|i64|isize|u8|u16|u32|u64|usize|f32|f64";var usize start=0;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}if(bytes::equal(name,names[start..end])){return true;}start=end+1;}return false;
}
fn bool substituted(ref json::Tree t,usize source,usize actual,usize args,usize depth,ref mut usize meter){
 if(depth>=64 || !work::charge(meter,1)){return false;}
 if(bytes::equal(wire::string(t,source,b"kind"),b"parameter")){
  let index=switch(wire::number(wire::string(t,source,b"name"))){case .ok(p):{p}case .err(_):{return false;}};
  if(index>=wire::count(t,args)){return false;}let x=model::raw(t,wire::item(t,args,index));let y=model::raw(t,actual);return work::charge(meter,x.len+y.len+1) && bytes::equal(x,y);
 }
 if(!equal(t,source,actual,b"kind",meter) || !equal(t,source,actual,b"name",meter) || !equal(t,source,actual,b"package_coordinate",meter) || !equal(t,source,actual,b"mutable",meter) || !equal(t,source,actual,b"length",meter)){return false;}
 let left=switch(model::rows(t,wire::field(t,source,b"arguments"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};let right=switch(model::rows(t,wire::field(t,actual,b"arguments"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};
 if(v::len<usize>(ref left)!=v::len<usize>(ref right)){return false;}var usize i=0;
 while(i<v::len<usize>(ref left)){if(!substituted(t,*v::at<usize>(ref left,i),*v::at<usize>(ref right,i),args,depth+1,meter)){return false;}i+=1;}return true;
}
fn bool cleanup_spans(ref json::Tree t,usize cleanup_body,usize source,ref mut usize meter){
 let names=b"calls|unsafe_sites|move_sites|drop_sites|allocation_sites|return_error_sites|borrows";var usize start=0;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}
  let rows=switch(model::rows(t,wire::field(t,cleanup_body,names[start..end]),65536,meter)){case .ok(p):{p}case .err(_):{return false;}};var usize i=0;
  while(i<v::len<usize>(ref rows)){let row=*v::at<usize>(ref rows,i);let span=if(start==0 || bytes::equal(names[start..end],b"borrows")){wire::field(t,row,b"span")}else{row};if(!contains(t,source,span,meter)){return false;}i+=1;}start=end+1;
 }return true;
}
fn bool aggregate(ref json::Tree t,usize item,usize source,ref mut usize meter){
 let actual=wire::field(t,item,b"aggregate");let args=wire::field(t,item,b"type_arguments");let cleanup_body=wire::field(t,item,b"cleanup");let kind=wire::string(t,wire::field(t,item,b"declaration"),b"kind");
 if(actual==json::count(t) || !bytes::equal(wire::string(t,actual,b"provenance"),b"compiler-observed") || wire::field(t,actual,b"type_parameters")<json::count(t)){return false;}
 let names=b"key|span|public|unsafe|foreign|abi|link_name|owner|receiver|parameters|effects|variants|repr_c";var usize start=0;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}if(!equal(t,source,actual,names[start..end],meter)){return false;}start=end+1;}
 if(!substituted(t,wire::field(t,source,b"type"),wire::field(t,actual,b"type"),args,0,meter)){return false;}
 let left=switch(model::rows(t,wire::field(t,source,b"fields"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};let right=switch(model::rows(t,wire::field(t,actual,b"fields"),4096,meter)){case .ok(p):{p}case .err(_):{return false;}};
 if(v::len<usize>(ref left)!=v::len<usize>(ref right)){return false;}var usize i=0;
 while(i<v::len<usize>(ref left)){let a=*v::at<usize>(ref left,i);let b=*v::at<usize>(ref right,i);
  if(!equal(t,a,b,b"name",meter) || !substituted(t,wire::field(t,a,b"type"),wire::field(t,b,b"type"),args,0,meter) || !contains(t,wire::field(t,b,b"span"),wire::field(t,a,b"span"),meter) || !contains(t,wire::field(t,source,b"span"),wire::field(t,b,b"span"),meter)){return false;}i+=1;
 }
 let drop=wire::string(t,actual,b"drop_kind");
 if(bytes::equal(kind,b"struct")){return cleanup_body==json::count(t) && drop.len==0 && wire::count(t,wire::field(t,actual,b"drop_effects"))==0;}
 if(!bytes::equal(kind,b"resource") || !bytes::equal(drop,b"empty") && !bytes::equal(drop,b"non-empty") || bytes::equal(drop,b"non-empty")!=(cleanup_body<json::count(t)) || !model::sorted(t,wire::field(t,actual,b"drop_effects"),meter)){return false;}
 return cleanup_body==json::count(t) || wire::string(t,source,b"drop_kind").len>0 && model::keys(t,wire::field(t,cleanup_body,b"function"),wire::field(t,item,b"declaration"),meter)==0 && cleanup_spans(t,cleanup_body,wire::field(t,source,b"span"),meter);
}
pub fn bool records(ref json::Tree t,usize h,ref v::Vec<usize> declarations,ref v::Vec<usize> records,ref v::Vec<usize> completeness,ref mut usize meter){
 var usize i=0;var usize templates=0;
 while(i<v::len<usize>(declarations)){if(wire::count(t,wire::field(t,*v::at<usize>(declarations,i),b"type_parameters"))>0){templates+=1;}i+=1;}
 i=0;while(i<v::len<usize>(records)){let r=*v::at<usize>(records,i);let key=wire::field(t,r,b"declaration");let source=model::find_key(t,declarations,b"key",key,meter);
  if(!wire::hex(wire::string(t,r,b"id"),b"g0_",64,false) || i>0 && !model::before(wire::string(t,*v::at<usize>(records,i-1),b"id"),wire::string(t,r,b"id")) || source==json::count(t) || !model::eq(t,key,h,b"package_instance")){return false;}
  let arity=switch(binders::validate(t,source,meter)){case .ok(p):{p}case .err(_):{return false;}};let args=wire::field(t,r,b"type_arguments");
  let rows=switch(model::rows(t,args,4096,meter)){case .ok(p):{p}case .err(_):{return false;}};if(arity==0 || arity!=v::len<usize>(ref rows)){return false;}var usize j=0;
  while(j<arity){if(!scalar(t,*v::at<usize>(ref rows,j),meter)){return false;}j+=1;}
  j=0;while(j<i){let prior=*v::at<usize>(records,j);if(model::keys(t,key,wire::field(t,prior,b"declaration"),meter)==0 && equal(t,r,prior,b"type_arguments",meter)){return false;}j+=1;}
  let body=wire::field(t,r,b"function");
  if(bytes::equal(wire::string(t,key,b"kind"),b"function")){if(body==json::count(t) || model::keys(t,key,wire::field(t,body,b"function"),meter)!=0){return false;}}
  else{if(!aggregate(t,r,source,meter)){return false;}}i+=1;
 }
 if(templates>0){
  let bodies=model::find(t,completeness,b"relation",b"generic-template-bodies",meter);let concrete=model::find(t,completeness,b"relation",b"generic-specializations",meter);
  if(wire::string(t,h,b"generic_interface_hash").len==0 || !bytes::equal(wire::string(t,bodies,b"status"),b"unavailable") || wire::integer(t,bodies,b"unknown")<templates || !bytes::equal(wire::string(t,concrete,b"status"),b"exact") || wire::integer(t,concrete,b"resolved")<v::len<usize>(records)){return false;}
 }return true;
}
pub fn bool calls(ref json::Tree t,usize h,usize body,ref v::Vec<usize> declarations,ref v::Vec<usize> records,ref mut usize meter){
 let calls=switch(model::rows(t,wire::field(t,body,b"calls"),65536,meter)){case .ok(p):{p}case .err(_):{return false;}};var usize i=0;
 while(i<v::len<usize>(ref calls)){let call=*v::at<usize>(ref calls,i);let id=wire::string(t,call,b"specialization");let target=wire::field(t,call,b"target");let local=model::eq(t,h,target,b"package_instance");
  if(id.len>0){if(!wire::hex(id,b"g0_",64,false) || !bytes::equal(wire::string(t,call,b"target_kind"),b"resolved")){return false;}if(local){let r=model::find(t,records,b"id",id,meter);if(r==json::count(t) || model::keys(t,wire::field(t,r,b"declaration"),target,meter)!=0){return false;}}}
  else if(local){let decl=model::find_key(t,declarations,b"key",target,meter);if(decl<json::count(t) && wire::count(t,wire::field(t,decl,b"type_parameters"))>0){return false;}}i+=1;
 }return true;
}
