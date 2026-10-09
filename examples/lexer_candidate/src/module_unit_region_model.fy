module module_unit_region_model;
import module_unit_value_model as m;
import std::vec as v;

// IDs are local to one typed value context. Source/type/local IDs retain their
// original meaning after the parser and specialization owners are dropped.
// source_mode: 1 direct ref-mut Arena value local; 2 named/aliased borrow;
// 3 evaluated source with no statically named owner. Runtime admission is later.
pub struct Scope {usize parent;usize capability;usize scope;usize statement;usize body;usize source;usize source_binding;usize arena_owner;usize arena_type;usize source_type;u32 source_mode;usize start;usize end;usize body_start;usize body_end;}
pub struct Placement {usize region_id;usize capability;usize expression;usize argument;usize owner_type;usize borrow_type;usize result_type;usize error_type;usize start;usize end;}
// kind: 1 fallthrough, 2 return, 3 Result propagation, 4 break, 5 continue.
// Resets of one group are ordered inner-first. The region body's span is the
// cleanup boundary; executable move-aware drop actions are not inferred here.
pub struct Exit {usize region_id;u32 kind;usize group;usize order;usize loop;usize start;usize end;}
pub resource Plan {v::Vec<Scope> regions;v::Vec<Placement> placements;v::Vec<Exit> exits;cleanup{}}
pub fn Plan empty(){return Plan{.regions=v::empty<Scope>(),.placements=v::empty<Placement>(),.exits=v::empty<Exit>()};}
pub fn Plan assemble(v::Vec<Scope> regions,v::Vec<Placement> placements,v::Vec<Exit> exits){return Plan{.regions,.placements,.exits};}
pub fn usize region_count(ref Plan p){return v::len<Scope>(ref p.regions);}
pub fn Scope region_at(ref Plan p,usize index){return *v::at<Scope>(ref p.regions,index);}
pub fn usize placement_count(ref Plan p){return v::len<Placement>(ref p.placements);}
pub fn Placement placement_at(ref Plan p,usize index){return *v::at<Placement>(ref p.placements,index);}
pub fn usize exit_count(ref Plan p){return v::len<Exit>(ref p.exits);}
pub fn Exit exit_at(ref Plan p,usize index){return *v::at<Exit>(ref p.exits,index);}
pub struct Loop {usize statement;usize region_depth;}
pub struct Context {m::Frame frame;usize work;usize groups;}
pub fn m::Problem fail(ref Context c,u32 code,usize start,usize end){return m::fail(c.frame,code,start,end,0,0);}
pub fn Result<usize,m::Problem> resets(u32 kind,usize first,usize loop,usize start,usize end,ref v::Vec<usize> active,ref mut v::Vec<Exit> exits,ref mut Context c){
    if(first>v::len<usize>(active)){return .err(fail(c,12,start,end));}if(first==v::len<usize>(active)){return .ok(0);}c.groups+=1;var usize i=v::len<usize>(active);var usize order=0;while(i>first){i-=1;if(!m::charge(ref mut c.work,1) || v::len<Exit>(exits)>=16384){return .err(fail(c,10,start,end));}if(!v::push<Exit>(exits,Exit{.region_id=*v::at<usize>(active,i),.kind,.group=c.groups,.order,.loop,.start,.end}).is_ok){return .err(fail(c,11,start,end));}order+=1;}return .ok(order);
}
