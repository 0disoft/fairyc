module module_project_calls;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_aggregate as aggregate;
import module_unit_value_model as budget;

fn clean::Problem fail(u32 code,clean::Frame f,usize start,usize end){return clean::Problem{.code,.module_index=f.module_index,.start,.end};}
fn usize callable(ref ir::Program p,usize module_index,usize declaration,ref mut usize work){var usize i=0;var usize found=0;while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return 0;}let c=ir::callable_at(p,i);if(c.module_index==module_index && c.declaration==declaration){if(found!=0){return 0;}found=i+1;}i+=1;}return found;}
pub fn clean::Problem check(ref ir::Program p,ref mut usize work){var usize frame=1;while(frame<=ir::count(p)){let b=ir::at(p,frame);let caller=clean::frame_at(ir::cleanup_ir(p),frame);var usize i=0;while(i<ir::call_count(b)){if(!budget::charge(work,1)){return fail(1002,caller,0,0);}let c=ir::call_at(b,i);let e=ir::node_at(b,c.expression-1).value;if(!c.checked || c.kind!=1 && c.kind!=2){return fail(1001,caller,c.start,c.end);}let contract=callable(p,c.target_module,c.target_declaration,work);var usize target=0;var usize j=1;
        while(j<=ir::count(p)){if(!budget::charge(work,1)){return fail(1002,caller,c.start,c.end);}let f=clean::frame_at(ir::cleanup_ir(p),j);let ordinary=f.kind==3 || f.kind==1 && contract!=0 && !ir::callable_at(p,contract-1).template;
            if(f.module_index==c.target_module && f.declaration==c.target_declaration && (c.kind==1 && ordinary || c.kind==2 && f.kind==1 && f.context==c.instance)){if(target!=0 || f.result!=e.type || (caller.effects&f.effects)!=f.effects){return fail(1001,caller,c.start,c.end);}target=j;}j+=1;
        }
        if(target==0){if(contract==0){return fail(if(*work>=1048576){1002}else{1001},caller,c.start,c.end);}let f=ir::callable_at(p,contract-1);if(c.kind!=1 || !f.foreign || f.template || f.result!=e.type || f.parameter_count!=c.argument_count || (caller.effects&f.effects)!=f.effects){return fail(1001,caller,c.start,c.end);}j=0;while(j<c.argument_count){if(!budget::charge(work,1)){return fail(1002,caller,c.start,c.end);}let a=ir::argument_at(b,c.argument_start+j);let type=ir::parameter_at(p,f.parameter_start+j);if(!a.checked || a.expected!=type || a.actual!=type || a.shared){return fail(1001,caller,a.start,a.end);}j+=1;}}
        else{let body=ir::at(p,target);var usize parameter=0;j=1;while(j<=ir::local_count(body)){if(!budget::charge(work,1)){return fail(1002,caller,c.start,c.end);}let local=ir::local_at(body,j);if(local.kind==1){if(parameter>=c.argument_count){return fail(1001,caller,c.start,c.end);}let a=ir::argument_at(b,c.argument_start+parameter);if(!a.checked || a.expected!=local.type || a.actual!=ir::node_at(b,a.expression-1).value.type || a.actual!=local.type && (!a.shared || !aggregate::shared(p,local.type,a.actual)) || a.actual==local.type && a.shared){return fail(1001,caller,a.start,a.end);}parameter+=1;}j+=1;}if(parameter!=c.argument_count){return fail(1001,caller,c.start,c.end);}}
        if(contract!=0 && ir::callable_at(p,contract-1).unsafe_call && !proof::unsafe_scope(p,b,frame,e,work)){return fail(if(*work>=1048576){1002}else{1001},caller,c.start,c.end);}i+=1;
    }frame+=1;}return clean::Problem{.code=0,.module_index=0,.start=0,.end=0};
}
