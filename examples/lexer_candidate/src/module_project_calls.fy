module module_project_calls;
import candidate_limits as limits;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_aggregate as aggregate;
import std::vec as v;

fn clean::Problem fail(u32 code,clean::Frame f,usize start,usize end){return clean::Problem{.code,.module_index=f.module_index,.start,.end};}
resource Index {v::Vec<usize> contracts;v::Vec<usize> ordinary;v::Vec<usize> instances;cleanup{}}
fn clean::Problem fault(u32 code,usize module_index){return clean::Problem{.code,.module_index,.start=0,.end=0};}
fn Result<usize,clean::Problem> slot(ref ir::Program p,ref v::Vec<usize> table,usize module_index,usize declaration,usize context,bool contract,ref mut usize work){
    var usize bucket=(module_index*104729 ^ declaration*3266489917 ^ context*668265263)&8191;
    while(true){if(!limits::project_charge(work,1)){return .err(fault(1002,module_index));}let id=*v::at<usize>(table,bucket);if(id==0){return .ok(bucket);}if(contract){let c=ir::callable_at(p,id-1);if(c.module_index==module_index && c.declaration==declaration){return .ok(bucket);}}else{let f=clean::frame_at(ir::cleanup_ir(p),id);if(f.module_index==module_index && f.declaration==declaration && (context==0 || f.context==context)){return .ok(bucket);}}bucket=(bucket+1)&8191;
    }return .err(fault(1001,module_index));
}
fn Result<usize,clean::Problem> find(ref ir::Program p,ref v::Vec<usize> table,usize module_index,usize declaration,usize context,bool contract,ref mut usize work){return switch(slot(p,table,module_index,declaration,context,contract,work)){case .ok(n):{.ok(*v::at<usize>(table,n))}case .err(e):{.err(e)}};}
fn clean::Problem put(ref ir::Program p,ref mut v::Vec<usize> table,usize module_index,usize declaration,usize context,bool contract,usize id,ref mut usize work){let bucket=switch(slot(p,table,module_index,declaration,context,contract,work)){case .ok(n):{n}case .err(e):{return e;}};let ref mut usize target=v::at_mut<usize>(table,bucket);if(*target!=0){return fault(1001,module_index);};*target=id;return fault(0,0);}
fn Result<Index,clean::Problem> index(ref ir::Program p,ref mut usize work){
    var v::Vec<usize> contracts=v::empty<usize>();var v::Vec<usize> ordinary=v::empty<usize>();var v::Vec<usize> instances=v::empty<usize>();var usize i=0;
    while(i<8192){if(!limits::project_charge(work,3)){return .err(fault(1002,0));}if(!v::push<usize>(ref mut contracts,0).is_ok || !v::push<usize>(ref mut ordinary,0).is_ok || !v::push<usize>(ref mut instances,0).is_ok){return .err(fault(1003,0));}i+=1;}
    i=0;while(i<ir::callable_count(p)){let c=ir::callable_at(p,i);let e=put(p,ref mut contracts,c.module_index,c.declaration,0,true,i+1,work);if(e.code!=0){return .err(e);}i+=1;}
    i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);var bool plain=f.kind==3;
        if(f.kind==1){let contract=switch(find(p,ref contracts,f.module_index,f.declaration,0,true,work)){case .ok(n):{n}case .err(e):{return .err(e);}};plain=contract!=0 && !ir::callable_at(p,contract-1).template;let e=put(p,ref mut instances,f.module_index,f.declaration,f.context,false,i,work);if(e.code!=0){return .err(e);}}
        if(plain){let e=put(p,ref mut ordinary,f.module_index,f.declaration,0,false,i,work);if(e.code!=0){return .err(e);}}i+=1;
    }return .ok(Index{.contracts,.ordinary,.instances});
}
pub fn clean::Problem check(ref ir::Program p,ref mut usize work){let lookup=switch(index(p,work)){case .ok(p):{p}case .err(e):{return e;}};var usize frame=1;while(frame<=ir::count(p)){let b=ir::at(p,frame);let caller=clean::frame_at(ir::cleanup_ir(p),frame);var usize i=0;while(i<ir::call_count(b)){if(!limits::project_charge(work,1)){return fail(1002,caller,0,0);}let c=ir::call_at(b,i);let e=ir::node_at(b,c.expression-1).value;if(!c.checked || c.kind!=1 && c.kind!=2){return fail(1001,caller,c.start,c.end);}let contract=switch(find(p,ref lookup.contracts,c.target_module,c.target_declaration,0,true,work)){case .ok(n):{n}case .err(e):{return e;}};let target=switch(find(p,if(c.kind==1){ref lookup.ordinary}else{ref lookup.instances},c.target_module,c.target_declaration,if(c.kind==1){0}else{c.instance},false,work)){case .ok(n):{n}case .err(e):{return e;}};var usize j=1;
        if(target!=0){let f=clean::frame_at(ir::cleanup_ir(p),target);if(f.result!=e.type || (caller.effects&f.effects)!=f.effects){return fail(1001,caller,c.start,c.end);}}
        if(target==0){if(contract==0){return fail(if(*work>=limits::project_work()){1002}else{1001},caller,c.start,c.end);}let f=ir::callable_at(p,contract-1);if(c.kind!=1 || !f.foreign || f.template || f.result!=e.type || f.parameter_count!=c.argument_count || (caller.effects&f.effects)!=f.effects){return fail(1001,caller,c.start,c.end);}j=0;while(j<c.argument_count){if(!limits::project_charge(work,1)){return fail(1002,caller,c.start,c.end);}let a=ir::argument_at(b,c.argument_start+j);let type=ir::parameter_at(p,f.parameter_start+j);if(!a.checked || a.expected!=type || a.actual!=type || a.shared){return fail(1001,caller,a.start,a.end);}j+=1;}}
        else{let body=ir::at(p,target);var usize parameter=0;j=1;while(j<=ir::local_count(body)){if(!limits::project_charge(work,1)){return fail(1002,caller,c.start,c.end);}let local=ir::local_at(body,j);if(local.kind==1){if(parameter>=c.argument_count){return fail(1001,caller,c.start,c.end);}let a=ir::argument_at(b,c.argument_start+parameter);if(!a.checked || a.expected!=local.type || a.actual!=ir::node_at(b,a.expression-1).value.type || a.actual!=local.type && (!a.shared || !aggregate::shared(p,local.type,a.actual)) || a.actual==local.type && a.shared){return fail(1001,caller,a.start,a.end);}parameter+=1;}j+=1;}if(parameter!=c.argument_count){return fail(1001,caller,c.start,c.end);}}
        if(contract!=0 && ir::callable_at(p,contract-1).unsafe_call){var usize scoped=0;let permitted=proof::unsafe_scope(p,b,frame,e,ref mut scoped);if(!limits::project_charge(work,scoped)){return fail(1002,caller,c.start,c.end);}if(!permitted){return fail(if(scoped>=1048576){1002}else{1001},caller,c.start,c.end);}}i+=1;
    }frame+=1;}return clean::Problem{.code=0,.module_index=0,.start=0,.end=0};
}
