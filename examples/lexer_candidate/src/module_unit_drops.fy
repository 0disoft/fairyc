module module_unit_drops;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_types as types;
import module_ast_join as join;
import std::vec as v;

// Type-level drop requirements. Ownership decides which obligations execute.
pub struct Node {usize type;usize context;usize edge_start;usize edge_count;u32 own_effects;u32 effects;bool needed;bool droppable;bool complete;}
// kind: 1 field, 2 Option, 3 Result success, 4 Result failure, 5 Vec element.
pub struct Edge {usize owner;usize target;u32 kind;usize module_index;usize declaration;usize source_index;usize start;usize end;}
pub resource Plan {v::Vec<Node> nodes;v::Vec<Edge> edges;usize work;cleanup{}}
fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
fn bool charge(ref mut usize work,usize amount){if(*work>1048576 || amount>1048576-*work){return false;};*work=*work+amount;return true;}
fn Node empty(usize type){return Node{.type,.context=0,.edge_start=0,.edge_count=0,.own_effects=0,.effects=0,.needed=false,.droppable=true,.complete=true};}
fn join::Problem edge(ref mut v::Vec<Edge> edges,usize owner,usize target,u32 kind,usize module_index,usize declaration,usize source_index,usize start,usize end,ref mut usize work){
    if(!charge(work,1) || v::len<Edge>(edges)>=16384){return fail(10,module_index,start,end);}
    if(!v::push<Edge>(edges,Edge{.owner,.target,.kind,.module_index,.declaration,.source_index,.start,.end}).is_ok){return fail(11,module_index,start,end);}return fail(0,0,0,0);
}
pub fn Result<Plan,join::Problem> build(ref interfaces::Interfaces imported){
    var v::Vec<Node> nodes=v::empty<Node>();var v::Vec<Edge> edges=v::empty<Edge>();var usize work=0;var usize id=1;
    // Keep canonical IDs unchanged, including graph entries with no drop work.
    while(id<=interfaces::type_count(imported)){if(!charge(ref mut work,1)){return .err(fail(10,0,0,0));}var node=empty(id);let t=interfaces::type_at(imported,id);node.complete=t.kind!=2 && t.kind!=3 && t.kind!=4 && t.kind!=13;if(!v::push<Node>(ref mut nodes,node).is_ok){return .err(fail(11,0,0,0));}id+=1;}
    var usize i=0;while(i<interfaces::shape_count(imported)){let s=interfaces::shape_at(imported,i);let t=interfaces::type_at(imported,s.type);var node=empty(s.type);node.edge_start=v::len<Edge>(ref edges);
        if(t.kind==3 || t.kind==13){node.needed=true;node.droppable=(s.abilities&1)!=0;node.own_effects=s.cleanup_effects;node.effects=s.cleanup_effects;var usize c=1;
            while(c<=generics::cleanup_count(interfaces::specializations(imported))){if(!charge(ref mut work,1)){return .err(fail(10,s.module_index,s.start,s.end));}if(generics::cleanup_at(interfaces::specializations(imported),c).type==s.type){node.context=c;break;}c+=1;}
            if(node.context==0){return .err(fail(9,s.module_index,s.start,s.end));}
        }
        if(t.kind==13){let p=edge(ref mut edges,s.type,t.first,5,s.module_index,s.declaration,0,s.start,s.end,ref mut work);if(p.code!=0){return .err(p);}}
        else{var usize f=0;while(f<s.field_count){if(!charge(ref mut work,1)){return .err(fail(10,s.module_index,s.start,s.end));}let field=interfaces::field_at(imported,s.field_start+f);
            if(field.type!=0){let p=edge(ref mut edges,s.type,field.type,1,field.module_index,field.declaration,field.source_index,field.start,field.end,ref mut work);if(p.code!=0){return .err(p);}}f+=1;}}
        node.edge_count=v::len<Edge>(ref edges)-node.edge_start;let slot=v::at_mut<Node>(ref mut nodes,s.type-1);*slot=node;i+=1;
    }
    id=1;while(id<=interfaces::type_count(imported)){if(!charge(ref mut work,1)){return .err(fail(10,0,0,0));}let t=interfaces::type_at(imported,id);
        if(t.kind==5 || t.kind==6 || t.kind==16){var node=*v::at<Node>(ref nodes,id-1);node.edge_start=v::len<Edge>(ref edges);let p=edge(ref mut edges,id,t.first,if(t.kind==5){2}else if(t.kind==6){3}else{1},0,0,0,0,0,ref mut work);if(p.code!=0){return .err(p);}
            if(t.kind==6){let p=edge(ref mut edges,id,t.second,4,0,0,0,0,0,ref mut work);if(p.code!=0){return .err(p);}}node.edge_count=v::len<Edge>(ref edges)-node.edge_start;let slot=v::at_mut<Node>(ref mut nodes,id-1);*slot=node;
        }id+=1;
    }
    // Monotone propagation also terminates for heap-backed recursive Vec shapes.
    // References and pointers have no outgoing edges and never inherit drops.
    var bool changed=true;while(changed){changed=false;id=1;while(id<=v::len<Node>(ref nodes)){if(!charge(ref mut work,1)){return .err(fail(10,0,0,0));}var node=*v::at<Node>(ref nodes,id-1);let before=node;var usize n=0;
        while(n<node.edge_count){if(!charge(ref mut work,1)){return .err(fail(10,0,0,0));}let e=*v::at<Edge>(ref edges,node.edge_start+n);if(e.target==0 || e.target>v::len<Node>(ref nodes)){return .err(fail(9,e.module_index,e.start,e.end));}let child=*v::at<Node>(ref nodes,e.target-1);node.effects|=child.effects;node.needed=node.needed || child.needed;node.droppable=node.droppable && child.droppable;node.complete=node.complete && child.complete;n+=1;}
        if(node.effects!=before.effects || node.needed!=before.needed || node.droppable!=before.droppable || node.complete!=before.complete){changed=true;let slot=v::at_mut<Node>(ref mut nodes,id-1);*slot=node;}id+=1;
    }}i=0;while(i<interfaces::shape_count(imported)){let s=interfaces::shape_at(imported,i);if(!v::at<Node>(ref nodes,s.type-1).complete){return .err(fail(9,s.module_index,s.start,s.end));}i+=1;}return .ok(Plan{.nodes,.edges,.work});
}
pub fn usize count(ref Plan p){return v::len<Node>(ref p.nodes);}
pub fn Node at(ref Plan p,usize type){assert(type>0 && type<=count(p));return *v::at<Node>(ref p.nodes,type-1);}
pub fn Result<Node,u32> required(ref Plan p,usize type){if(type==0 || type>count(p)){return .err(9);}let n=at(p,type);return if(n.complete){.ok(n)}else{.err(9)};}
pub fn usize edge_count(ref Plan p){return v::len<Edge>(ref p.edges);}
pub fn Edge edge_at(ref Plan p,usize index){return *v::at<Edge>(ref p.edges,index);}
pub fn usize work(ref Plan p){return p.work;}
