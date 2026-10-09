module module_unit_origin_inputs;
import module_unit_values as values;
import module_unit_value_model as m;
import module_unit_value_builtin as borrows;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import module_types as types;
import syntax as syntax;
import std::vec as v;

// Unit 0 is the caller-owned primary plan; provider unit IDs are one-based.
// Frame/owner IDs are analysis-local. Provider canonical type IDs stay in their
// own value graph, and generic instance IDs are never compared across units.
pub struct Frame {usize unit;usize context;m::Frame value;usize expression_start;usize expression_count;usize local_start;usize local_count;usize call_start;usize call_count;usize parameter_start;usize parameter_count;}
pub struct Owner {usize frame;usize local;usize scope;u32 kind;bool borrowed;usize start;usize end;}
pub resource Inputs {v::Vec<values::Plan> providers;v::Vec<usize> modules;v::Vec<Frame> frames;v::Vec<Owner> owners;v::Vec<usize> parameters;v::Vec<usize> targets;usize expressions;usize work;cleanup{}}
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
pub fn usize parameter(ref Inputs p,usize id,usize ordinal){let f=frame(p,id);return *v::at<usize>(ref p.parameters,f.parameter_start+ordinal);}
pub fn usize target(ref Inputs p,usize id,usize call){let f=frame(p,id);return *v::at<usize>(ref p.targets,f.call_start+call);}
// Convert the stable lexical binding, never the projected/forwarded place root.
pub fn Result<usize,m::Problem> binding_owner(ref Inputs p,usize id,m::Expression expression){
    if(id==0 || id>frame_count(p)){return .err(m::Problem{.code=12,.module_index=0,.start=expression.start,.end=expression.end,.expected=0,.actual=0});}let f=frame(p,id);
    if(expression.binding==0){return .ok(0);}if(expression.kind!=2 || expression.binding>f.local_count){return .err(fail(f.value,12,expression.start,expression.end));}return .ok(f.local_start+expression.binding);
}
struct Budget {usize expressions;usize work;}
fn ref m::Values unit_body(ref v::Vec<values::Plan> providers,ref values::Plan primary,Frame f){if(f.unit==0){return values::at(primary,f.context);}return values::at(v::at<values::Plan>(providers,f.unit-1),f.context);}
fn types::Type unit_type(ref v::Vec<values::Plan> providers,ref values::Plan primary,Frame f,usize type){return if(f.unit==0){values::type_at(primary,type)}else{values::type_at(v::at<values::Plan>(providers,f.unit-1),type)};}
fn m::Problem append(ref values::Plan unit,usize unit_id,ref mut v::Vec<Frame> frames,ref mut v::Vec<Owner> owners,ref mut v::Vec<usize> parameters,ref mut v::Vec<usize> targets,ref mut Budget budget){
    var usize i=0;while(i<values::count(unit)){let b=values::at(unit,i);let f=m::frame(b);let usize expressions=m::expression_count(b);let usize locals=m::local_count(b);let usize calls=m::call_count(b);
        if(!m::charge(ref mut budget.work,1) || v::len<Frame>(frames)>=1024 || expressions>16384-budget.expressions || locals>16384-v::len<Owner>(owners) || calls>16384-v::len<usize>(targets)){return fail(f,10,0,0);}let usize id=v::len<Frame>(frames)+1;let usize local_start=v::len<Owner>(owners);let usize parameter_start=v::len<usize>(parameters);var usize j=1;
        while(j<=locals){let l=m::local_at(b,j);if(!m::charge(ref mut budget.work,1)){return fail(f,10,l.start,l.end);}if(!v::push<Owner>(owners,Owner{.frame=id,.local=j,.scope=l.scope,.kind=l.kind,.borrowed=l.type!=0 && borrows::borrowed(values::type_at(unit,l.type)),.start=l.start,.end=l.end}).is_ok){return fail(f,11,l.start,l.end);}if(l.kind==1){if(v::len<usize>(parameters)-parameter_start>=128){return fail(f,10,l.start,l.end);}if(!v::push<usize>(parameters,v::len<Owner>(owners)).is_ok){return fail(f,11,l.start,l.end);}}j+=1;}
        let usize call_start=v::len<usize>(targets);j=0;while(j<calls){if(!m::charge(ref mut budget.work,1)){return fail(f,10,0,0);}if(!v::push<usize>(targets,0).is_ok){return fail(f,11,0,0);}j+=1;}
        if(!v::push<Frame>(frames,Frame{.unit=unit_id,.context=i,.value=f,.expression_start=budget.expressions,.expression_count=expressions,.local_start,.local_count=locals,.call_start,.call_count=calls,.parameter_start,.parameter_count=v::len<usize>(parameters)-parameter_start}).is_ok){return fail(f,11,0,0);}budget.expressions+=expressions;i+=1;
    }return m::Problem{.code=0,.module_index=0,.start=0,.end=0,.expected=0,.actual=0};
}
fn Result<usize,m::Problem> find(ref v::Vec<Frame> frames,Frame caller,m::Invocation call,ref mut usize work){
    var usize i=0;while(i<v::len<Frame>(frames)){if(!m::charge(work,1)){return .err(fail(caller.value,10,call.start,call.end));}let f=*v::at<Frame>(frames,i);if(call.kind==2){if(f.unit==caller.unit && f.value.kind==1 && f.value.context==call.instance && f.value.module_index==call.target_module && f.value.declaration==call.target_declaration){return .ok(i+1);}}
        else if(call.kind==1 && f.value.kind==3 && f.value.module_index==call.target_module && f.value.declaration==call.target_declaration){return .ok(i+1);}i+=1;
    }return .ok(0);
}
fn Result<values::Plan,m::Problem> provider(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize module_index){
    let bound=switch(binding::bind(sources,metadata,module_index,module_index+1)){case .ok(p):{p}case .err(p):{return .err(m::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};
    let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{return .err(m::Problem{.code=p.code,.module_index=p.module_index,.start=p.start,.end=p.end,.expected=0,.actual=0});}};return values::build(sources,metadata,ref ready);
}
pub fn Result<Inputs,m::Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref values::Plan primary){
    var v::Vec<values::Plan> providers=v::empty<values::Plan>();var v::Vec<usize> modules=v::empty<usize>();var v::Vec<Frame> frames=v::empty<Frame>();var v::Vec<Owner> owners=v::empty<Owner>();var v::Vec<usize> parameters=v::empty<usize>();var v::Vec<usize> targets=v::empty<usize>();var Budget budget=Budget{.expressions=0,.work=0};let initial=append(primary,0,ref mut frames,ref mut owners,ref mut parameters,ref mut targets,ref mut budget);if(initial.code!=0){return .err(initial);}var usize id=0;
    while(id<v::len<Frame>(ref frames)){let f=*v::at<Frame>(ref frames,id);var usize j=0;while(j<f.call_count){let call=m::call_at(unit_body(ref providers,primary,f),j);let e=m::expression_at(unit_body(ref providers,primary,f),call.expression);
        if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,call.start,call.end));}if(e.type!=0 && borrows::borrowed(unit_type(ref providers,primary,f,e.type))){var usize found=switch(find(ref frames,f,call,ref mut budget.work)){case .ok(n):{n}case .err(p):{return .err(p);}};
            if(found==0 && call.kind==1){if(call.target_module>=v::len<syntax::Parsed>(sources) || call.target_declaration==0 || call.target_declaration>syntax::declaration_count(v::at<syntax::Parsed>(sources,call.target_module))){return .err(fail(f.value,12,call.start,call.end));}let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,call.target_module),call.target_declaration-1);
                if(d.kind==3 && d.generic_parameters==0 && d.body!=0){var usize k=0;var bool cached=false;while(k<v::len<usize>(ref modules)){if(!m::charge(ref mut budget.work,1)){return .err(fail(f.value,10,call.start,call.end));}cached=cached || *v::at<usize>(ref modules,k)==call.target_module;k+=1;}if(!cached){if(v::len<values::Plan>(ref providers)>=1024){return .err(fail(f.value,10,call.start,call.end));}let unit=switch(provider(sources,metadata,call.target_module)){case .ok(u):{u}case .err(p):{return .err(p);}};if(!m::charge(ref mut budget.work,values::work(ref unit))){return .err(fail(f.value,10,call.start,call.end));}let added=append(ref unit,v::len<values::Plan>(ref providers)+1,ref mut frames,ref mut owners,ref mut parameters,ref mut targets,ref mut budget);if(added.code!=0){return .err(added);}if(!v::push<usize>(ref mut modules,call.target_module).is_ok){return .err(fail(f.value,11,call.start,call.end));}if(!v::push<values::Plan>(ref mut providers,unit).is_ok){return .err(fail(f.value,11,call.start,call.end));}}
                    found=switch(find(ref frames,f,call,ref mut budget.work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(found==0){return .err(fail(f.value,12,call.start,call.end));}
                }
            }else if(found==0 && call.kind==2){return .err(fail(f.value,12,call.start,call.end));}let slot=v::at_mut<usize>(ref mut targets,f.call_start+j);*slot=found;
        }j+=1;
    }id+=1;}return .ok(Inputs{.providers,.modules,.frames,.owners,.parameters,.targets,.expressions=budget.expressions,.work=budget.work});
}
