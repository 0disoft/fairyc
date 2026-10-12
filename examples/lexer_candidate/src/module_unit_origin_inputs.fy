module module_unit_origin_inputs;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_value_builtin as borrows;
import module_unit_origin_shape as shape;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import module_types as types;
import syntax as syntax;
import std::vec as v;
import module_unit_loan_live_model as live;
import module_unit_loan_liveness as live_builder;

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
struct Budget {usize expressions;usize work;}
struct Static {usize module_index;usize expression;usize owner;}
fn ref m::Values unit_body(ref v::Vec<values::Plan> providers,ref values::Plan primary,Frame f){if(f.unit==0){return values::at(primary,f.context);}return values::at(v::at<values::Plan>(providers,f.unit-1),f.context);}
fn types::Type unit_type(ref v::Vec<values::Plan> providers,ref values::Plan primary,Frame f,usize type){return if(f.unit==0){values::type_at(primary,type)}else{values::type_at(v::at<values::Plan>(providers,f.unit-1),type)};}
fn Result<v::Vec<usize>,m::Problem> storage_inputs(ref v::Vec<values::Plan> providers,ref values::Plan primary,ref v::Vec<Frame> frames,ref mut v::Vec<Owner> owners,ref mut Budget budget){
    var v::Vec<usize> storage=v::empty<usize>();var usize i=0;while(i<v::len<Frame>(frames)){let f=*v::at<Frame>(frames,i);let body=unit_body(providers,primary,f);var usize j=1;while(j<=f.local_count){let usize id=f.local_start+j;let o=*v::at<Owner>(owners,id-1);let local=m::local_at(body,j);if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,o.start,o.end));}var usize owner=id;
        if(o.borrowed && o.kind!=13 && o.kind!=14 && !borrows::borrowed(unit_type(providers,primary,f,local.type))){if(v::len<Owner>(owners)>=16384){return .err(fail(f.value,10,o.start,o.end));}if(!v::push<Owner>(owners,Owner{.frame=o.frame,.local=o.local,.scope=o.scope,.kind=12,.borrowed=false,.start=o.start,.end=o.end}).is_ok){return .err(fail(f.value,11,o.start,o.end));}owner=v::len<Owner>(owners);}
        if(v::len<usize>(ref storage)!=id-1){return .err(fail(f.value,12,o.start,o.end));}if(!v::push<usize>(ref mut storage,owner).is_ok){return .err(fail(f.value,11,o.start,o.end));}j+=1;
    }i+=1;}return .ok(storage);
}
fn Result<v::Vec<usize>,m::Problem> static_inputs(ref v::Vec<values::Plan> providers,ref values::Plan primary,ref v::Vec<Frame> frames,ref mut v::Vec<Owner> owners,ref mut Budget budget){
    var usize capacity=8;while(capacity<budget.expressions*2){capacity*=2;}var v::Vec<Static> cache=v::empty<Static>();var v::Vec<usize> statics=v::empty<usize>();var usize i=0;
    while(i<capacity){if(!m::charge(ref mut budget.work,1)){return .err(m::Problem{.code=10,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}if(!v::push<Static>(ref mut cache,Static{.module_index=0,.expression=0,.owner=0}).is_ok){return .err(m::Problem{.code=11,.module_index=0,.start=0,.end=0,.expected=0,.actual=0});}i+=1;}
    i=0;while(i<v::len<Frame>(frames)){let f=*v::at<Frame>(frames,i);var usize j=1;while(j<=f.expression_count){let e=m::expression_at(unit_body(providers,primary,f),j);if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,e.start,e.end));}var usize owner=0;
            if(e.type!=0 && borrows::borrowed(unit_type(providers,primary,f,e.type)) && (e.kind==1 || e.kind==6 && e.builtin.kind==200)){var usize slot=(f.value.module_index*4161+e.source_index)%capacity;var bool found=false;
                while(!found){if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,e.start,e.end));}let row=*v::at<Static>(ref cache,slot);if(row.owner==0){if(v::len<Owner>(owners)>=16384){return .err(fail(f.value,10,e.start,e.end));}if(!v::push<Owner>(owners,Owner{.frame=i+1,.local=0,.scope=0,.kind=11,.borrowed=true,.start=e.start,.end=e.end}).is_ok){return .err(fail(f.value,11,e.start,e.end));}owner=v::len<Owner>(owners);let cell=v::at_mut<Static>(ref mut cache,slot);*cell=Static{.module_index=f.value.module_index,.expression=e.source_index,.owner};found=true;}else if(row.module_index==f.value.module_index && row.expression==e.source_index){owner=row.owner;found=true;}else{slot=(slot+1)%capacity;}}
            }if(!v::push<usize>(ref mut statics,owner).is_ok){return .err(fail(f.value,11,e.start,e.end));}j+=1;
        }i+=1;}return .ok(statics);
}
// Retain only required provider frames; canonical context indices still address
// the original checked value plan. Unrelated provider bodies are never executed
// by origin/loan analysis, and a later borrowed call can append its exact frame.
fn m::Problem append_frame(ref values::Plan unit,usize i,usize unit_id,ref mut v::Vec<Frame> frames,ref mut v::Vec<Owner> owners,ref mut v::Vec<usize> parameters,ref mut v::Vec<usize> targets,ref mut Budget budget){
    let b=values::at(unit,i);let f=m::frame(b);let usize expressions=m::expression_count(b);let usize locals=m::local_count(b);let usize calls=m::call_count(b);
        if(!m::charge(ref mut budget.work,1) || v::len<Frame>(frames)>=1024 || expressions>16384-budget.expressions || locals>16384-v::len<Owner>(owners) || calls>16384-v::len<usize>(targets)){return fail(f,10,0,0);}let usize id=v::len<Frame>(frames)+1;let usize local_start=v::len<Owner>(owners);let usize parameter_start=v::len<usize>(parameters);var usize j=1;
        while(j<=locals){let l=m::local_at(b,j);if(!m::charge(ref mut budget.work,1)){return fail(f,10,l.start,l.end);}let leaves=switch(shape::leaves(unit,l.type,f,ref mut budget.work,0)){case .ok(n):{n}case .err(p):{return p;}};if(leaves>1){return fail(f,2,l.start,l.end);}var usize start=l.start;var usize end=l.end;if(l.kind==6){var usize k=0;var bool found=false;while(k<m::consumer_count(b)){let c=m::consumer_at(b,k);if(!m::charge(ref mut budget.work,1)){return fail(f,10,l.start,l.end);}if(c.kind==17 && c.start<=l.start && l.end<=c.end && (!found || c.start>start)){start=c.start;end=c.end;found=true;}k+=1;}if(!found){return fail(f,12,l.start,l.end);}}if(!v::push<Owner>(owners,Owner{.frame=id,.local=j,.scope=l.scope,.kind=if(l.kind==6){14}else if(l.kind==1 && values::type_at(unit,l.type).kind==19){13}else{l.kind},.borrowed=leaves==1 || l.kind==6 || l.kind==1 && values::type_at(unit,l.type).kind==19,.start,.end}).is_ok){return fail(f,11,l.start,l.end);}if(l.kind==1){if(v::len<usize>(parameters)-parameter_start>=128){return fail(f,10,l.start,l.end);}if(!v::push<usize>(parameters,v::len<Owner>(owners)).is_ok){return fail(f,11,l.start,l.end);}}j+=1;}
        let usize call_start=v::len<usize>(targets);j=0;while(j<calls){if(!m::charge(ref mut budget.work,1)){return fail(f,10,0,0);}if(!v::push<usize>(targets,0).is_ok){return fail(f,11,0,0);}j+=1;}
        let result=if(f.result==0){types::Type{.kind=0,.scalar=0,.module_index=0,.declaration=0,.first=0,.second=0}}else{values::type_at(unit,f.result)};if(!v::push<Frame>(frames,Frame{.tagged_return=result.kind==5 || result.kind==6,.unit=unit_id,.context=i,.value=f,.expression_start=budget.expressions,.expression_count=expressions,.local_start,.local_count=locals,.call_start,.call_count=calls,.parameter_start,.parameter_count=v::len<usize>(parameters)-parameter_start}).is_ok){return fail(f,11,0,0);}budget.expressions+=expressions;return m::Problem{.code=0,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};
}
fn m::Problem append(ref values::Plan unit,usize unit_id,ref mut v::Vec<Frame> frames,ref mut v::Vec<Owner> owners,ref mut v::Vec<usize> parameters,ref mut v::Vec<usize> targets,ref mut Budget budget){
    var usize i=0;while(i<values::count(unit)){let p=append_frame(unit,i,unit_id,frames,owners,parameters,targets,budget);if(p.code!=0){return p;}i+=1;}return m::Problem{.code=0,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};
}
fn m::Problem append_callee(ref values::Plan unit,usize unit_id,m::Invocation call,m::Frame caller,ref mut v::Vec<Frame> frames,ref mut v::Vec<Owner> owners,ref mut v::Vec<usize> parameters,ref mut v::Vec<usize> targets,ref mut Budget budget){
    var usize i=0;while(i<values::count(unit)){if(!m::charge(ref mut budget.work,1)){return fail(caller,10,call.start,call.end);}let f=m::frame(values::at(unit,i));
        if(f.module_index==call.target_module && f.declaration==call.target_declaration && (call.kind==1 && (f.kind==3 || f.kind==1) || call.kind==2 && f.kind==1 && f.context==call.instance)){return append_frame(unit,i,unit_id,frames,owners,parameters,targets,budget);}i+=1;
    }return fail(caller,12,call.start,call.end);
}
fn Result<usize,m::Problem> find(ref v::Vec<Frame> frames,Frame caller,m::Invocation call,ref mut usize work){
    var usize i=0;while(i<v::len<Frame>(frames)){if(!m::charge(work,1)){return .err(fail(caller.value,10,call.start,call.end));}let f=*v::at<Frame>(frames,i);if(call.kind==2){if(f.unit==caller.unit && f.value.kind==1 && f.value.context==call.instance && f.value.module_index==call.target_module && f.value.declaration==call.target_declaration){return .ok(i+1);}}
        // Imported ordinary bodies share the concrete owned lane; call kind 1
        // still names an exact non-template source declaration.
        else if(call.kind==1 && (f.value.kind==3 || f.value.kind==1) && f.value.module_index==call.target_module && f.value.declaration==call.target_declaration){return .ok(i+1);}i+=1;
    }return .ok(0);
}
// Provider type checking enforces its own value-phase allowance, just like the
// caller-owned primary plan. Do not charge that completed phase a second time
// against origin/liveness traversal; retained frames and work remain bounded.
fn Result<values::Plan,m::Problem> provider(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize module_index){
    let bound=switch(binding::bind(sources,metadata,module_index,module_index+1)){case .ok(p):{p}case .err(p):{return .err(m::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};
    let ready=switch(closure::build_unit(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(m::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,metadata,ref ready);
}
pub fn Result<Inputs,m::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref values::Plan primary){
    var v::Vec<values::Plan> providers=v::empty<values::Plan>();var v::Vec<usize> modules=v::empty<usize>();var v::Vec<Frame> frames=v::empty<Frame>();var v::Vec<Owner> owners=v::empty<Owner>();var v::Vec<usize> parameters=v::empty<usize>();var v::Vec<usize> targets=v::empty<usize>();var Budget budget=Budget{.expressions=0,.work=0};let initial=append(primary,0,ref mut frames,ref mut owners,ref mut parameters,ref mut targets,ref mut budget);if(initial.code!=0){return .err(initial);}var usize id=0;
    while(id<v::len<Frame>(ref frames)){let f=*v::at<Frame>(ref frames,id);var usize j=0;while(j<f.call_count){let call=m::call_at(unit_body(ref providers,primary,f),j);let e=m::expression_at(unit_body(ref providers,primary,f),call.expression);
        if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,call.start,call.end));}let leaves=switch(shape::leaves(if(f.unit==0){primary}else{v::at<values::Plan>(ref providers,f.unit-1)},e.type,f.value,ref mut budget.work,0)){case .ok(n):{n}case .err(p):{return .err(p);}};if(leaves>1){return .err(fail(f.value,2,e.start,e.end));}if(leaves==1){var usize found=switch(find(ref frames,f,call,ref mut budget.work)){case .ok(n):{n}case .err(p):{return .err(p);}};
            if(found==0 && call.kind==1){if(call.target_module>=v::len<syntax::Parsed>(sources) || call.target_declaration==0 || call.target_declaration>syntax::declaration_count(v::at<syntax::Parsed>(sources,call.target_module))){return .err(fail(f.value,12,call.start,call.end));}let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,call.target_module),call.target_declaration-1);
                if(d.kind==3 && d.generic_parameters==0 && d.body!=0){var usize k=0;var usize provider_id=0;while(k<v::len<usize>(ref modules)){if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,call.start,call.end));}if(*v::at<usize>(ref modules,k)==call.target_module){provider_id=k+1;break;}k+=1;}
                    if(provider_id==0){if(v::len<values::Plan>(ref providers)>=1024){return .err(fail(f.value,10,call.start,call.end));}let unit=switch(provider(sources,metadata,call.target_module)){case .ok(u):{u}case .err(p):{return .err(p);}};if(!v::push<usize>(ref mut modules,call.target_module).is_ok){return .err(fail(f.value,11,call.start,call.end));}if(!v::push<values::Plan>(ref mut providers,unit).is_ok){return .err(fail(f.value,11,call.start,call.end));}provider_id=v::len<values::Plan>(ref providers);}
                    let added=append_callee(v::at<values::Plan>(ref providers,provider_id-1),provider_id,call,f.value,ref mut frames,ref mut owners,ref mut parameters,ref mut targets,ref mut budget);if(added.code!=0){return .err(added);}found=switch(find(ref frames,f,call,ref mut budget.work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(found==0){return .err(fail(f.value,12,call.start,call.end));}
                }
            }else if(found==0 && call.kind==2){if(f.unit==0 || f.unit>v::len<values::Plan>(ref providers)){return .err(fail(f.value,12,call.start,call.end));}let added=append_callee(v::at<values::Plan>(ref providers,f.unit-1),f.unit,call,f.value,ref mut frames,ref mut owners,ref mut parameters,ref mut targets,ref mut budget);if(added.code!=0){return .err(added);}found=switch(find(ref frames,f,call,ref mut budget.work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(found==0){return .err(fail(f.value,12,call.start,call.end));}}
            let slot=v::at_mut<usize>(ref mut targets,f.call_start+j);*slot=found;
        }j+=1;
    }id+=1;}let storage=switch(storage_inputs(ref providers,primary,ref frames,ref mut owners,ref mut budget)){case .ok(p):{p}case .err(p):{return .err(p);}};let statics=switch(static_inputs(ref providers,primary,ref frames,ref mut owners,ref mut budget)){case .ok(p):{p}case .err(p):{return .err(p);}};var v::Vec<Allocation> allocations=v::empty<Allocation>();var v::Vec<usize> placements=v::empty<usize>();id=0;while(id<v::len<Frame>(ref frames)){let f=*v::at<Frame>(ref frames,id);let b=unit_body(ref providers,primary,f);var usize j=1;while(j<=f.expression_count){let e=m::expression_at(b,j);if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,e.start,e.end));}var usize site=0;if(e.kind==6 && e.builtin.kind==300){if(v::len<Allocation>(ref allocations)>=16384){return .err(fail(f.value,10,e.start,e.end));}if(!v::push<Allocation>(ref mut allocations,Allocation{.frame=id+1,.module_index=f.value.module_index,.start=e.start,.end=e.end}).is_ok){return .err(fail(f.value,11,e.start,e.end));}site=v::len<Allocation>(ref allocations);}if(!v::push<usize>(ref mut placements,site).is_ok){return .err(fail(f.value,11,e.start,e.end));}j+=1;}id+=1;}var v::Vec<live::Plan> live_frames=v::empty<live::Plan>();id=0;while(id<v::len<Frame>(ref frames)){let f=*v::at<Frame>(ref frames,id);var v::Vec<bool> flags=v::empty<bool>();var usize local=1;while(local<=f.local_count){if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,0,0));}let owner=*v::at<Owner>(ref owners,f.local_start+local-1);if(!v::push<bool>(ref mut flags,owner.borrowed && owner.kind!=13 && owner.kind!=14).is_ok){return .err(fail(f.value,11,owner.start,owner.end));}local+=1;}let data=switch(live_builder::build(v::at<syntax::Parsed>(sources,f.value.module_index),unit_body(ref providers,primary,f),ref flags,ref mut budget.work)){case .ok(p):{p}case .err(p):{return .err(p);}};if(!v::push<live::Plan>(ref mut live_frames,data).is_ok){return .err(fail(f.value,11,0,0));}id+=1;}return .ok(Inputs{.liveness=live_frames,.allocations,.placements,.providers,.modules,.frames,.owners,.storage,.parameters,.targets,.statics,.expressions=budget.expressions,.work=budget.work});
}

pub fn usize allocation_count(ref Inputs p){return v::len<Allocation>(ref p.allocations);}
pub fn Allocation allocation(ref Inputs p,usize id){return *v::at<Allocation>(ref p.allocations,id-1);}
pub fn Result<usize,m::Problem> placement(ref Inputs p,ref values::Plan primary,usize id,usize original,ref mut usize work){
    let f=frame(p,id);let b=body(p,primary,id);var usize low=0;var usize high=f.expression_count;while(low<high){if(!m::charge(work,1)){return .err(fail(f.value,10,0,0));}let mid=low+(high-low)/2;if(m::expression_at(b,mid+1).source_index<original){low=mid+1;}else{high=mid;}}if(low>=f.expression_count || m::expression_at(b,low+1).source_index!=original){return .err(fail(f.value,12,0,0));}let e=m::expression_at(b,low+1);let site=*v::at<usize>(ref p.placements,f.expression_start+low);return if(site==0 || site>allocation_count(p)){.err(fail(f.value,12,e.start,e.end))}else{.ok(site)};
}

pub fn ref live::Plan liveness(ref Inputs p,usize frame){return v::at<live::Plan>(ref p.liveness,frame-1);}
