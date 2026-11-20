module project_atlas_reverse;
import project_atlas_join as join;
import project_atlas_validate as validate;
import project_path as work;
import std::vec as v;

struct Links{usize declaration;usize implementation;}
pub resource Incoming{v::Vec<usize> bases;v::Vec<usize> heads;v::Vec<Links> links;cleanup{}}

// Layout uses dense ordinals from the validated symbol and specialization
// tables. Their JSON token positions rise in table order, even if sparse.
fn Result<usize,u32> slot(ref join::Graph g,ref v::Vec<usize> bases,join::Symbol subject,usize implementation,ref mut usize meter){
 let p=join::partition(g,subject.package);
 let base=*v::at<usize>(bases,subject.package);
 let concrete=implementation!=0;
 let token=if(concrete){implementation}else{subject.token};
 let count=if(concrete){validate::specialization_count(p)}else{validate::symbol_count(p)};
 var usize lo=0;
 var usize hi=count;
 while(lo<hi){
  if(!work::charge(meter,1)){return .err(3);}
  let mid=lo+(hi-lo)/2;
  let row=if(concrete){validate::specialization(p,mid)}else{validate::symbol(p,mid)};
  if(row==token){return .ok(base+mid+if(concrete){validate::symbol_count(p)}else{0});}
  if(row<token){lo=mid+1;}else{hi=mid;}
 }
 return .err(9);
}

fn usize prepend(ref mut v::Vec<usize> heads,usize slot,usize edge){
 let head=v::at_mut<usize>(heads,slot);
 let old=*head;
 *head=edge;
 return old;
}

pub fn Result<Incoming,u32> build(ref join::Graph g,ref mut usize meter){
 var bases=v::empty<usize>();
 var heads=v::empty<usize>();
 var links=v::empty<Links>();
 let edges=join::edge_count(g);
 if(edges==0){return .ok(Incoming{.bases,.heads,.links});}
 var usize p=0;
 while(p<join::package_count(g)){
  let partition=join::partition(g,p);
  let count=validate::symbol_count(partition)+validate::specialization_count(partition);
  if(!work::charge(meter,count+1)){return .err(3);}
  if(!v::push<usize>(ref mut bases,v::len<usize>(ref heads)).is_ok){return .err(2);}
  var usize i=0;
  while(i<count){if(!v::push<usize>(ref mut heads,0).is_ok){return .err(2);}i+=1;}
  p+=1;
 }
 if(!work::charge(meter,edges)){return .err(3);}
 var usize i=0;
 while(i<edges){if(!v::push<Links>(ref mut links,Links{.declaration=0,.implementation=0}).is_ok){return .err(2);}i+=1;}

 // Prepending in reverse preserves the joined edge order, including ties
 // that decide which callers fit in a limited traversal. Zero is end-of-list.
 while(i>0){
  let edge=join::edge(g,i-1);
  let declaration=slot(g,ref bases,edge.target,0,meter)?;
  let decl_next=prepend(ref mut heads,declaration,i);
  var usize impl_next=0;
  if(edge.target_implementation!=0){
   let impl=slot(g,ref bases,edge.target,edge.target_implementation,meter)?;
   impl_next=prepend(ref mut heads,impl,i);
  }
  i-=1;
  let row=v::at_mut<Links>(ref mut links,i);
  *row=Links{.declaration=decl_next,.implementation=impl_next};
 }
 return .ok(Incoming{.bases,.heads,.links});
}

pub fn Result<usize,u32> first(ref Incoming incoming,ref join::Graph g,join::Symbol subject,usize implementation,ref mut usize meter){
 if(v::len<Links>(ref incoming.links)==0){return .ok(0);}
 let at=slot(g,ref incoming.bases,subject,implementation,meter)?;
 return .ok(*v::at<usize>(ref incoming.heads,at));
}

pub fn usize next(ref Incoming incoming,usize edge,bool concrete){
 let link=v::at<Links>(ref incoming.links,edge-1);
 return if(concrete){link.implementation}else{link.declaration};
}
