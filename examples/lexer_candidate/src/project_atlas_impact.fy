module project_atlas_impact;
import project_atlas_join as join;
import project_atlas_options as options;
import project_atlas_reverse as reverse;
import project_path as work;
import std::vec as v;

pub struct Node{join::Symbol subject;usize depth;}
pub resource Impact{join::Symbol subject;usize implementation;v::Vec<Node> nodes;bool partial;bool depth_cut;bool symbol_cut;cleanup{}}
pub fn usize count(ref Impact r){return v::len<Node>(ref r.nodes);}
pub fn Node node(ref Impact r,usize i){return *v::at<Node>(ref r.nodes,i);}
pub fn bool partial(ref Impact r){return r.partial;}
pub fn bool complete(ref Impact r){return !r.depth_cut && !r.symbol_cut;}
pub fn bool depth_cut(ref Impact r){return r.depth_cut;}
pub fn bool symbol_cut(ref Impact r){return r.symbol_cut;}
pub fn join::Symbol subject(ref Impact r){return r.subject;}
pub fn usize subject_implementation(ref Impact r){return r.implementation;}
struct Step{Node node;usize implementation;}
struct Visit{bool declaration;bool implementation;}
fn Result<Visit,u32> visited(ref v::Vec<Step> queue,join::Symbol subject,usize implementation,ref mut usize meter){
 var bool declaration=false;
 var usize i=0;
 while(i<v::len<Step>(queue)){
  if(!work::charge(meter,1)){return .err(3);}
  let step=*v::at<Step>(queue,i);
  if(join::equal(step.node.subject,subject)){
   declaration=true;
   if(step.implementation==implementation){return .ok(Visit{.declaration=true,.implementation=true});}
  }
  i+=1;
 }
 return .ok(Visit{.declaration,.implementation=false});
}

// Reverse compiler-resolved calls only. A bare selector covers the whole root
// declaration. Concrete steps follow the implementation on the preceding edge.
pub fn Result<Impact,u32> explore(ref join::Graph g,ref u8[] instance,ref u8[] selector,usize max_depth,usize max_symbols,ref mut usize meter){
 if(max_depth>64 || max_symbols==0 || max_symbols>4096){return .err(9);}
 let end=options::selector_end(selector)?;
 let root=join::resolve(g,instance,selector[..end],meter)?;
 var usize implementation=0;
 if(end<selector.len){implementation=join::resolve_specialization(g,root,selector[end+1..],meter)?;}
 var nodes=v::empty<Node>();
 var queue=v::empty<Step>();
 var bool depth_cut=false;
 var bool symbol_cut=false;
 let subject=Node{.subject=root,.depth=0};
 if(!v::push<Node>(ref mut nodes,subject).is_ok ||
    !v::push<Step>(ref mut queue,Step{.node=subject,.implementation}).is_ok){return .err(2);}

 let incoming=reverse::build(g,meter)?;
 var usize next=0;
 while(next<v::len<Step>(ref queue)){
  let current=*v::at<Step>(ref queue,next);
  var usize i=reverse::first(ref incoming,g,current.node.subject,current.implementation,meter)?;
  while(i!=0){
   if(!work::charge(meter,1)){return .err(3);}
   let edge=join::edge(g,i-1);
   i=reverse::next(ref incoming,i,current.implementation!=0);
   if(implementation==0 && join::equal(edge.source,root)){continue;}

   let seen=visited(ref queue,edge.source,edge.source_implementation,meter)?;
   if(seen.implementation){continue;}
   if(current.node.depth>=max_depth){depth_cut=true;continue;}
   if(!seen.declaration && v::len<Node>(ref nodes)>=max_symbols){symbol_cut=true;continue;}

   let node=Node{.subject=edge.source,.depth=current.node.depth+1};
   if(!seen.declaration && !v::push<Node>(ref mut nodes,node).is_ok){return .err(2);}
   if(!v::push<Step>(ref mut queue,Step{.node,.implementation=edge.source_implementation}).is_ok){return .err(2);}
  }
  next+=1;
 }

 var usize i=1;while(i<v::len<Node>(ref nodes)){let current=*v::at<Node>(ref nodes,i);var usize at=i;while(at>0){let prior=*v::at<Node>(ref nodes,at-1);if(!work::charge(meter,256)){return .err(3);}if(prior.depth<current.depth || prior.depth==current.depth && !join::before(g,current.subject,prior.subject)){break;}{let slot=v::at_mut<Node>(ref mut nodes,at);*slot=prior;}{let slot=v::at_mut<Node>(ref mut nodes,at-1);*slot=current;}at-=1;}i+=1;}
 return .ok(Impact{.subject=root,.implementation,.nodes,.partial=join::partial(g) || depth_cut || symbol_cut,.depth_cut,.symbol_cut});
}
