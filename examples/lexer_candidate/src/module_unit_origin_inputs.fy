module module_unit_origin_inputs;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_origin_shape as shape;
import module_types as types;
import std::vec as v;
import module_unit_loan_live_model as live;

// Unit 0 is the caller-owned primary plan; provider unit IDs are one-based.
// Frame/owner IDs are analysis-local. Provider canonical type IDs stay in their
// own value graph, and generic instance IDs are never compared across units.
pub struct Frame {bool tagged_return;usize unit;usize context;m::Frame value;usize expression_start;usize expression_count;usize local_start;usize local_count;usize call_start;usize call_count;usize parameter_start;usize parameter_count;}
pub struct Owner {usize frame;usize local;usize scope;u32 kind;bool borrowed;usize start;usize end;}
pub struct Allocation {usize frame;usize module_index;usize start;usize end;}
pub resource Inputs {v::Vec<live::Plan> liveness;v::Vec<Allocation> allocations;v::Vec<usize> placements;v::Vec<values::Plan> providers;v::Vec<usize> modules;v::Vec<Frame> frames;v::Vec<Owner> owners;v::Vec<usize> storage;v::Vec<usize> parameters;v::Vec<usize> targets;v::Vec<usize> statics;usize expressions;usize work;cleanup{}}
fn m::Problem fail(m::Frame f,u32 code,usize start,usize end){return m::fail(f,code,start,end,0,0);}
pub fn usize frame_count(ref Inputs p){return v::len<Frame>(ref p.frames);}
pub fn Frame frame(ref Inputs p,usize id){return *v::at<Frame>(ref p.frames,id-1);}
pub fn Owner owner(ref Inputs p,usize id){return *v::at<Owner>(ref p.owners,id-1);}
pub fn usize owner_count(ref Inputs p){return v::len<Owner>(ref p.owners);}
pub fn usize expression_count(ref Inputs p){return p.expressions;}
pub fn usize provider_count(ref Inputs p){return v::len<values::Plan>(ref p.providers);}
pub fn usize work(ref Inputs p){return p.work;}
pub fn ref m::Values body(ref Inputs p,ref values::Plan primary,usize id){let f=frame(p,id);if(f.unit==0){return values::at(primary,f.context);}return values::at(v::at<values::Plan>(ref p.providers,f.unit-1),f.context);}
pub fn types::Type type_at(ref Inputs p,ref values::Plan primary,usize id,usize type){let f=frame(p,id);return if(f.unit==0){values::type_at(primary,type)}else{values::type_at(v::at<values::Plan>(ref p.providers,f.unit-1),type)};}
pub fn Result<u32,m::Problem> borrow_leaves(ref Inputs p,ref values::Plan primary,usize id,usize type,ref mut usize work){let f=frame(p,id);return if(f.unit==0){shape::leaves(primary,type,f.value,work,0)}else{shape::leaves(v::at<values::Plan>(ref p.providers,f.unit-1),type,f.value,work,0)};}
pub fn usize parameter(ref Inputs p,usize id,usize ordinal){let f=frame(p,id);return *v::at<usize>(ref p.parameters,f.parameter_start+ordinal);}
pub fn usize target(ref Inputs p,usize id,usize call){let f=frame(p,id);return *v::at<usize>(ref p.targets,f.call_start+call);}
// Static owners use kind 11 and retain a real module-local expression span.
// Equal source expressions across generic instances share one static owner.
pub fn Result<usize,m::Problem> static_owner(ref Inputs p,ref values::Plan primary,usize id,usize original,ref mut usize work){
    if(id==0 || id>frame_count(p)){return .err(m::Problem{.code=12,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}let f=frame(p,id);let b=body(p,primary,id);var usize low=0;var usize high=f.expression_count;
    while(low<high){if(!m::charge(work,1)){return .err(fail(f.value,10,0,0));}let usize mid=low+(high-low)/2;if(m::expression_at(b,mid+1).source_index<original){low=mid+1;}else{high=mid;}}
    if(low>=f.expression_count || m::expression_at(b,low+1).source_index!=original){return .err(fail(f.value,12,0,0));}let e=m::expression_at(b,low+1);let usize owner=*v::at<usize>(ref p.statics,f.expression_start+low);return if(owner==0){.err(fail(f.value,12,e.start,e.end))}else{.ok(owner)};
}
// Convert the stable lexical binding, never the projected/forwarded place root.
pub fn Result<usize,m::Problem> binding_owner(ref Inputs p,usize id,m::Expression expression){
    if(id==0 || id>frame_count(p)){return .err(m::Problem{.code=12,.module_index=0,.start=expression.start,.end=expression.end,.expected=0,.actual=0});}let f=frame(p,id);
    if(expression.binding==0){return .ok(0);}if(expression.kind!=2 || expression.binding>f.local_count){return .err(fail(f.value,12,expression.start,expression.end));}return .ok(f.local_start+expression.binding);
}
// Borrowing the wrapper's storage differs from forwarding its borrowed payload.
pub fn Result<usize,m::Problem> storage_owner(ref Inputs p,usize id,m::Expression expression){let owner=switch(binding_owner(p,id,expression)){case .ok(n):{n}case .err(p):{return .err(p);}};if(owner==0){return .ok(0);}if(owner>v::len<usize>(ref p.storage)){return .err(fail(frame(p,id).value,12,expression.start,expression.end));}return .ok(*v::at<usize>(ref p.storage,owner-1));}

pub fn usize allocation_count(ref Inputs p){return v::len<Allocation>(ref p.allocations);}
pub fn Allocation allocation(ref Inputs p,usize id){return *v::at<Allocation>(ref p.allocations,id-1);}
pub fn Result<usize,m::Problem> placement(ref Inputs p,ref values::Plan primary,usize id,usize original,ref mut usize work){
    let f=frame(p,id);let b=body(p,primary,id);var usize low=0;var usize high=f.expression_count;while(low<high){if(!m::charge(work,1)){return .err(fail(f.value,10,0,0));}let mid=low+(high-low)/2;if(m::expression_at(b,mid+1).source_index<original){low=mid+1;}else{high=mid;}}if(low>=f.expression_count || m::expression_at(b,low+1).source_index!=original){return .err(fail(f.value,12,0,0));}let e=m::expression_at(b,low+1);let site=*v::at<usize>(ref p.placements,f.expression_start+low);return if(site==0 || site>allocation_count(p)){.err(fail(f.value,12,e.start,e.end))}else{.ok(site)};
}

pub fn ref live::Plan liveness(ref Inputs p,usize frame){return v::at<live::Plan>(ref p.liveness,frame-1);}

// Compiler-private ownership handoff from the checked origin-input factory.
// This owner carries analysis data; it never grants project artifact admission.
pub fn Inputs assemble(v::Vec<live::Plan> liveness,v::Vec<Allocation> allocations,v::Vec<usize> placements,v::Vec<values::Plan> providers,v::Vec<usize> modules,v::Vec<Frame> frames,v::Vec<Owner> owners,v::Vec<usize> storage,v::Vec<usize> parameters,v::Vec<usize> targets,v::Vec<usize> statics,usize expressions,usize work){return Inputs{.liveness,.allocations,.placements,.providers,.modules,.frames,.owners,.storage,.parameters,.targets,.statics,.expressions,.work};}
