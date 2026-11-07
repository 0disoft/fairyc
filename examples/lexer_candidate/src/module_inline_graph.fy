module module_inline_graph;
import module_inline_identity as identity;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_callable_index as contracts;
import module_unit_c_frame_index as targets;
import module_unit_value_model as budget;
import candidate_limits as limits;

// Each root gets its own distinct-reachability set. A finished shared target
// costs one slot and is never expanded again; an active target is a cycle.
struct Walk {usize[64] frames;u32[64] states;usize count;}
fn u32 visit(ref ir::Program p,ref contracts::Index signatures,ref targets::Index frames,usize frame,usize module_index,ref mut Walk walk,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}var usize slot=0;while(slot<walk.count){if(!budget::charge(work,1)){return 1002;}if(walk.frames[slot]==frame){return if(walk.states[slot]==2){0}else{1001};}slot+=1;}
    if(walk.count>=64 || frame==0 || frame>ir::count(p)){return 1001;}walk.frames[slot]=frame;walk.states[slot]=1;walk.count+=1;
    let f=clean::frame_at(ir::cleanup_ir(p),frame);let b=ir::at(p,frame);if(f.kind!=3 || f.module_index!=module_index || f.effects!=0 || identity::zero(ir::inline_hash(b))){return 1001;}
    var u32 code=0;let signature=contracts::find(p,signatures,f.module_index,f.declaration,work,ref mut code);if(code!=0){return code;}
    // The callable table contains invocation targets, not every declaration.
    // An unused exported root can lack a signature; every reached target is
    // required to have one below, before recursion enters its body.
    if(signature!=0){let c=ir::callable_at(p,signature-1);if(c.template || c.foreign || c.unsafe_call || c.effects!=0 || c.result!=f.result || c.parameter_count>4 || c.parameter_start>ir::parameter_count(p) || c.parameter_count>ir::parameter_count(p)-c.parameter_start){return 1001;}
        var usize arity=0;while(arity<ir::local_count(b) && ir::local_at(b,arity+1).kind==1){if(!budget::charge(work,1)){return 1002;}if(arity>=c.parameter_count || ir::local_at(b,arity+1).type!=ir::parameter_at(p,c.parameter_start+arity)){return 1001;}arity+=1;}if(arity!=c.parameter_count){return 1001;}}
    var usize i=0;while(i<ir::call_count(b)){if(!budget::charge(work,1)){return 1002;}let call=ir::call_at(b,i);if(call.kind!=1 || !call.checked || call.instance!=0 || call.target_module!=module_index || call.expression==0 || call.expression>ir::node_count(b) || call.argument_start>ir::argument_count(b) || call.argument_count>ir::argument_count(b)-call.argument_start){return 1001;}
        let target=targets::find(p,frames,call.target_module,call.target_declaration,0,false,work,ref mut code);if(code!=0){return code;}if(target==0){return 1001;}let tf=clean::frame_at(ir::cleanup_ir(p),target);if(tf.result!=ir::node_at(b,call.expression-1).value.type){return 1001;}
        let tc=contracts::find(p,signatures,call.target_module,call.target_declaration,work,ref mut code);if(code!=0){return code;}if(tc==0){return 1001;}let signature=ir::callable_at(p,tc-1);if(signature.parameter_count!=call.argument_count || signature.parameter_start>ir::parameter_count(p) || signature.parameter_count>ir::parameter_count(p)-signature.parameter_start){return 1001;}
        var usize j=0;while(j<call.argument_count){if(!budget::charge(work,1)){return 1002;}let a=ir::argument_at(b,call.argument_start+j);if(a.expected!=ir::parameter_at(p,signature.parameter_start+j) || a.actual!=a.expected || a.expression==0 || a.expression>ir::node_count(b) || ir::node_at(b,a.expression-1).value.type!=a.actual){return 1001;}j+=1;}
        let nested=visit(p,signatures,frames,target,module_index,walk,work);if(nested!=0){return nested;}i+=1;
    }walk.states[slot]=2;return 0;
}
// Called only after every body passed ordinary and inline identity validation.
// Source authentication additionally proves public visibility and declaration
// name-to-coordinate identity, which detached callable metadata does not carry.
pub fn u32 check(ref ir::Program p,ref mut usize work,ref mut usize total){
    var bool needed=false;var usize i=1;while(i<=ir::count(p)){if(!budget::charge(work,1)){return 1002;}if(!identity::zero(ir::inline_hash(ir::at(p,i))) && ir::call_count(ir::at(p,i))!=0){needed=true;}i+=1;}if(!needed){return 0;}
    let signatures=switch(contracts::build(p,work)){case .ok(p):{p}case .err(code):{return code;}};let frames=switch(targets::build(p,ref signatures,work)){case .ok(p):{p}case .err(code):{return code;}};
    let project=clean::project(ir::cleanup_ir(p));if(!checkpoint(project,work,total)){return 1002;}
    i=1;while(i<=ir::count(p)){if(!budget::charge(work,1)){return 1002;}let b=ir::at(p,i);if(!identity::zero(ir::inline_hash(b)) && ir::call_count(b)!=0){var Walk walk=Walk{.frames=[0;64],.states=[0;64],.count=0};let code=visit(p,ref signatures,ref frames,i,clean::frame_at(ir::cleanup_ir(p),i).module_index,ref mut walk,work);if(code!=0){return code;}}if(!checkpoint(project,work,total)){return 1002;}i+=1;}return 0;
}
fn bool checkpoint(bool project,ref mut usize work,ref mut usize total){if(!project){return true;}if(!limits::project_charge(total,*work)){return false;};*work=0;return true;}
