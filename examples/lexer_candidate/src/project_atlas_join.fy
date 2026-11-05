module project_atlas_join;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub struct Symbol{usize package;usize token;}
pub struct Edge{Symbol source;Symbol target;usize source_implementation;usize target_implementation;usize call;bool resource_body;}
resource Packages{v::Vec<current::Ready> values;cleanup{}}
pub resource Graph{Packages packages;v::Vec<usize> order;v::Vec<Edge> edges;bool partial;cleanup{}}
fn ref validate::Partition part(ref Packages g,usize p){return current::partition(v::at<current::Ready>(ref g.values,p));}
fn ref u8[] name(ref Packages g,usize p){let row=part(g,p);return wire::string(validate::tree(row),validate::header(row),b"package_instance");}
pub fn ref validate::Partition partition(ref Graph g,usize p){return part(ref g.packages,p);}
pub fn usize package_count(ref Graph g){return v::len<usize>(ref g.order);}
pub fn usize package_at(ref Graph g,usize i){return *v::at<usize>(ref g.order,i);}
pub fn ref u8[] instance(ref Graph g,usize p){return name(ref g.packages,p);}
pub fn ref u8[] visible(ref Graph g,Symbol s){return wire::string(validate::tree(partition(g,s.package)),s.token,b"visible");}
pub fn bool equal(Symbol a,Symbol b){return a.package==b.package && a.token==b.token;}
pub fn bool before(ref Graph g,Symbol a,Symbol b){if(a.package!=b.package){return model::before(instance(g,a.package),instance(g,b.package));}return model::before(visible(g,a),visible(g,b));}
pub fn Result<bool,u32> contains(ref Graph g,Symbol s,ref mut usize meter){if(s.package>=package_count(g)){return .ok(false);}let part=partition(g,s.package);var usize i=0;while(i<validate::symbol_count(part)){if(!work::charge(meter,1)){return .err(3);}if(validate::symbol(part,i)==s.token){return .ok(true);}i+=1;}return .ok(false);}
pub fn bool partial(ref Graph g){return g.partial;}
pub fn usize edge_count(ref Graph g){return v::len<Edge>(ref g.edges);}
pub fn Edge edge(ref Graph g,usize i){return *v::at<Edge>(ref g.edges,i);}
fn usize find_package(ref Packages g,ref u8[] value,ref mut usize meter){let n=v::len<current::Ready>(ref g.values);var usize i=0;while(i<n){if(!work::charge(meter,value.len+name(g,i).len+1)){return n;}if(bytes::equal(name(g,i),value)){return i;}i+=1;}return n;}
fn bool key_equal(ref json::Tree a,usize x,ref json::Tree b,usize y,ref mut usize meter){let names=b"package_instance|module|kind|name";var usize start=0;while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let field=names[start..end];let left=wire::string(a,x,field);let right=wire::string(b,y,field);if(!work::charge(meter,left.len+right.len+1) || !bytes::equal(left,right)){return false;}start=end+1;}return true;}
fn Result<Symbol,u32> lookup(ref Packages g,usize p,ref json::Tree source,usize key,ref mut usize meter){let partition=part(g,p);let token=validate::symbol_for_key(partition,source,key,meter)?;return if(token<json::count(validate::tree(partition))){.ok(Symbol{.package=p,.token})}else{.err(9)};}
pub fn Result<Symbol,u32> resolve(ref Graph g,ref u8[] package_instance,ref u8[] symbol,ref mut usize meter){let p=find_package(ref g.packages,package_instance,meter);if(p==package_count(g)){return .err(9);}let partition=partition(g,p);let t=validate::tree(partition);var usize i=0;while(i<validate::symbol_count(partition)){let s=validate::symbol(partition,i);if(!work::charge(meter,symbol.len+256)){return .err(3);}if(bytes::equal(wire::string(t,s,b"visible"),symbol) || bytes::equal(wire::string(t,s,b"full"),symbol)){return .ok(Symbol{.package=p,.token=s});}i+=1;}return .err(9);}
pub fn ref u8[] source_specialization(ref Graph g,Edge edge){let p=partition(g,edge.source.package);return wire::string(validate::tree(p),edge.source_implementation,b"id");}
pub fn ref u8[] target_specialization(ref Graph g,Edge edge){let p=partition(g,edge.target.package);return wire::string(validate::tree(p),edge.target_implementation,b"id");}
// IDs are canonical and sorted by semantic admission; resolve against the exact
// target partition, then require its original declaration, not just ID presence.
fn Result<usize,u32> implementation(ref validate::Partition p,ref u8[] id,ref mut usize meter){
 let t=validate::tree(p);var usize lo=0;var usize hi=validate::specialization_count(p);
 while(lo<hi){if(!work::charge(meter,id.len+68)){return .err(3);}let mid=lo+(hi-lo)/2;let row=validate::specialization(p,mid);let value=wire::string(t,row,b"id");if(bytes::equal(value,id)){return .ok(row);}if(model::before(value,id)){lo=mid+1;}else{hi=mid;}}
 return .err(9);
}
fn Result<bool,u32> append_calls(ref Packages g,usize p,usize token,usize body,usize source_implementation,bool resource_body,usize maximum,ref mut v::Vec<Edge> edges,ref mut usize meter){
 let row=part(g,p);let t=validate::tree(row);let calls=model::rows(t,wire::field(t,body,b"calls"),65536,meter)?;var usize c=0;
 while(c<v::len<usize>(ref calls)){let call=*v::at<usize>(ref calls,c);
  if(bytes::equal(wire::string(t,call,b"target_kind"),b"resolved")){
   if(v::len<Edge>(edges)>=maximum){return .err(9);}let key=wire::field(t,call,b"target");let q=find_package(g,wire::string(t,key,b"package_instance"),meter);if(q==v::len<current::Ready>(ref g.values)){return .err(9);}
   let target=lookup(g,q,t,key,meter)?;let target_part=part(g,q);let target_tree=validate::tree(target_part);let target_key=wire::field(target_tree,target.token,b"declaration");let decl=validate::declaration(target_part,target_key,meter);
   if(p!=q && !wire::boolean(target_tree,decl,b"public")){return .err(9);}
   let id=wire::string(t,call,b"specialization");var usize target_implementation=0;
   if(id.len>0){target_implementation=implementation(target_part,id,meter)?;if(model::keys(target_tree,wire::field(target_tree,target_implementation,b"declaration"),target_key,meter)!=0){return .err(9);}}
   else if(wire::count(target_tree,wire::field(target_tree,decl,b"type_parameters"))>0){return .err(9);}
   let edge=Edge{.source=Symbol{.package=p,.token},.target,.source_implementation,.target_implementation,.call,.resource_body};if(!v::push<Edge>(edges,edge).is_ok){return .err(2);}
  }c+=1;
 }return .ok(true);
}
// Only Ready values, each carrying an independent current expectation, enter a join.
// Ordering is package instance, visible source id, then canonical call span/target.
pub fn Result<Graph,u32> join(v::Vec<current::Ready> packages,usize max_packages,usize max_calls,ref mut usize meter){
 let n=v::len<current::Ready>(ref packages);if(max_packages==0 || max_packages>65536 || max_calls>65536 || n==0 || n>max_packages){return .err(9);}
 let Packages g=Packages{.values=packages};var v::Vec<usize> order=v::empty<usize>();var v::Vec<Edge> edges=v::empty<Edge>();var usize p=0;
 while(p<n){if(!v::push<usize>(ref mut order,p).is_ok){return .err(2);}var usize at=p;while(at>0){let left=*v::at<usize>(ref order,at-1);if(!work::charge(meter,256)){return .err(3);}if(bytes::equal(name(ref g,left),name(ref g,p))){return .err(9);}if(model::before(name(ref g,left),name(ref g,p))){break;}{let slot=v::at_mut<usize>(ref mut order,at);*slot=left;}{let slot=v::at_mut<usize>(ref mut order,at-1);*slot=p;}at-=1;}p+=1;}
 var bool partial=false;var usize ordered=0;
 while(ordered<n){p=*v::at<usize>(ref order,ordered);let row=part(ref g,p);let t=validate::tree(row);let h=validate::header(row);if(validate::version(row)!=validate::version(part(ref g,0))){return .err(9);}
 let refs=model::rows(t,wire::field(t,h,b"package_references"),33,meter)?;var usize r=0;
 while(r<v::len<usize>(ref refs)){let row=*v::at<usize>(ref refs,r);let q=find_package(ref g,wire::string(t,row,b"instance"),meter);if(q==n){return .err(9);}let target=part(ref g,q);let target_tree=validate::tree(target);let target_header=validate::header(target);if(!work::charge(meter,wire::string(t,row,b"uid").len+wire::string(target_tree,target_header,b"package_uid").len+256)){return .err(3);}if(!bytes::equal(wire::string(t,row,b"uid"),wire::string(target_tree,target_header,b"package_uid")) || !bytes::equal(wire::string(t,row,b"coordinate"),wire::string(target_tree,target_header,b"package_coordinate"))){return .err(9);}r+=1;}
 if(!bytes::equal(wire::string(t,h,b"generation_status"),b"exact") || wire::count(t,wire::field(t,h,b"omissions"))!=0){partial=true;}
 r=0;while(r<validate::completeness_count(row)){let c=validate::completeness(row,r);if(!work::charge(meter,1)){return .err(3);}if((bytes::equal(wire::string(t,c,b"relation"),b"calls") || validate::version(row)==3 && (bytes::equal(wire::string(t,c,b"relation"),b"generic-specializations") || bytes::equal(wire::string(t,c,b"relation"),b"generic-template-bodies"))) && (!bytes::equal(wire::string(t,c,b"status"),b"exact") || wire::integer(t,c,b"unknown")!=0 || wire::integer(t,c,b"external")!=0)){partial=true;}r+=1;}
 var usize s=0;while(s<validate::symbol_count(row)){let token=validate::symbol(row,s);let key=wire::field(t,token,b"declaration");
  let summary=validate::function_for_key(row,key,meter)?;if(summary<json::count(t)){discard append_calls(ref g,p,token,summary,0,false,max_calls,ref mut edges,meter)?;}
  var usize f=0;while(f<validate::specialization_count(row)){let spec=validate::specialization(row,f);if(!work::charge(meter,1)){return .err(3);}if(model::keys(t,wire::field(t,spec,b"declaration"),key,meter)==0){let names=b"function|cleanup";var usize start=0;
   while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let body=wire::field(t,spec,names[start..end]);if(body<json::count(t)){discard append_calls(ref g,p,token,body,spec,start>0,max_calls,ref mut edges,meter)?;}start=end+1;}
  }f+=1;}
 s+=1;}
 ordered+=1;}
 return .ok(Graph{.packages=g,.order,.edges,.partial});
}
