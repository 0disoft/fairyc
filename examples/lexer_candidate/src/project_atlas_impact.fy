module project_atlas_impact;
import project_atlas_join as join;
import project_path as work;
import std::vec as v;

pub struct Node{join::Symbol subject;usize depth;}
pub resource Impact{join::Symbol subject;v::Vec<Node> nodes;bool partial;bool depth_cut;bool symbol_cut;cleanup{}}
pub fn usize count(ref Impact r){return v::len<Node>(ref r.nodes);}
pub fn Node node(ref Impact r,usize i){return *v::at<Node>(ref r.nodes,i);}
pub fn bool partial(ref Impact r){return r.partial;}
pub fn bool complete(ref Impact r){return !r.depth_cut && !r.symbol_cut;}
pub fn bool depth_cut(ref Impact r){return r.depth_cut;}
pub fn bool symbol_cut(ref Impact r){return r.symbol_cut;}
pub fn join::Symbol subject(ref Impact r){return r.subject;}
fn Result<bool,u32> seen(ref v::Vec<Node> nodes,join::Symbol s,ref mut usize meter){var usize i=0;while(i<v::len<Node>(nodes)){if(!work::charge(meter,1)){return .err(3);}if(join::equal(v::at<Node>(nodes,i).subject,s)){return .ok(true);}i+=1;}return .ok(false);}
// Reverse compiler-resolved calls only. No safe-change claim or inferred callbacks.
pub fn Result<Impact,u32> explore(ref join::Graph g,join::Symbol root,usize max_depth,usize max_symbols,ref mut usize meter){
 if(max_depth>64 || max_symbols==0 || max_symbols>4096 || !join::contains(g,root,meter)?){return .err(9);}
 var v::Vec<Node> nodes=v::empty<Node>();var bool depth_cut=false;var bool symbol_cut=false;if(!v::push<Node>(ref mut nodes,Node{.subject=root,.depth=0}).is_ok){return .err(2);}
 var usize next=0;while(next<v::len<Node>(ref nodes)){let current=*v::at<Node>(ref nodes,next);var usize i=0;while(i<join::edge_count(g)){if(!work::charge(meter,1)){return .err(3);}let edge=join::edge(g,i);if(join::equal(edge.target,current.subject) && !seen(ref nodes,edge.source,meter)?){if(current.depth>=max_depth){depth_cut=true;}else if(v::len<Node>(ref nodes)>=max_symbols){symbol_cut=true;}else{if(!v::push<Node>(ref mut nodes,Node{.subject=edge.source,.depth=current.depth+1}).is_ok){return .err(2);}}}i+=1;}next+=1;}

 var usize i=1;while(i<v::len<Node>(ref nodes)){let current=*v::at<Node>(ref nodes,i);var usize at=i;while(at>0){let prior=*v::at<Node>(ref nodes,at-1);if(!work::charge(meter,256)){return .err(3);}if(prior.depth<current.depth || prior.depth==current.depth && !join::before(g,current.subject,prior.subject)){break;}{let slot=v::at_mut<Node>(ref mut nodes,at);*slot=prior;}{let slot=v::at_mut<Node>(ref mut nodes,at-1);*slot=current;}at-=1;}i+=1;}
 return .ok(Impact{.subject=root,.nodes,.partial=join::partial(g) || depth_cut || symbol_cut,.depth_cut,.symbol_cut});
}
