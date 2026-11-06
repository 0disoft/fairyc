module module_state_graph_native_bodies;
import module_state_graph_program as program;
import module_state_graph_model as m;
import module_state_graph_native_plan as plan;
import module_state_graph_native_operations as ops;
import module_state_graph_native_text as w;
import std::bytes as bytes;
import std::vec as v;

pub fn ref m::Graph body(ref program::Verified proof,usize index){return if(index==0){program::root(proof)}else{program::helper(proof,index-1)};}
fn bool exit(ref plan::Plan p,ref m::Graph g,m::Terminator term,ref mut bytes::Buffer out,ref mut w::State s){
    let category=m::data(g)[term.category_start..term.category_end];
    let trap=switch(plan::trap_id(p,category,ref mut s.work)){case .ok(n):{n}case .err(code):{s.code=code;return false;}};
    let usize kind=if(term.kind==3){0}else if(term.kind==4){1}else if(term.kind==5){2}else{3};
    return w::text(out,s,b"    *exit_kind = ") && w::number(out,s,kind) && w::text(out,s,b"u; *trap_kind = ") && w::number(out,s,trap) && w::text(out,s,b"u; goto finished;\n");
}
fn bool edge(ref plan::Plan p,usize index,ref m::Graph g,usize target,usize start,usize count,ref mut bytes::Buffer out,ref mut w::State s){
    let block=*v::at<m::Block>(m::blocks(g),target);var usize i=0;
    // Verified DAG destination parameters cannot alias predecessor values.
    while(i<count){let parameter=*v::at<m::Value>(m::parameters(g),block.parameter_start+i);let argument=*v::at<usize>(m::arguments(g),start+i);
        if(!w::text(out,s,b"    ") || !w::value(out,s,p,index,parameter.id) || !w::text(out,s,b" = ") || !w::value(out,s,p,index,argument) || !w::text(out,s,b";\n")){return false;}i+=1;
    }
    return w::text(out,s,b"    goto block_") && w::number(out,s,target) && w::text(out,s,b";\n");
}
fn bool terminator(ref plan::Plan p,usize index,ref m::Graph g,m::Terminator term,ref mut bytes::Buffer out,ref mut w::State s){
    if(!w::checked(out,s) || !w::text(out,s,b"fycs_graph_step_v1(context,1u,0u)") || !w::end(out,s)){return false;}s.term+=1;
    let site=2147483648+s.term;
    if(s.audit && (!w::checked(out,s) || !w::text(out,s,b"observe(user,context->operation,") || !w::number(out,s,site) || !w::text(out,s,b"u,NULL,NULL,0u)") || !w::end(out,s))){return false;}
    if(term.kind==1){return edge(p,index,g,term.target,term.argument_start,term.argument_count,out,s);}
    if(term.kind!=2){return exit(p,g,term,out,s);}
    if(!w::text(out,s,b"    if (") || !w::value(out,s,p,index,term.condition) || !w::text(out,s,b".length != 1u || ") || !w::value(out,s,p,index,term.condition) || !w::text(out,s,b".bytes[0] > 1u) { status = FYCS_OV_CODEC; goto finished; }\n    if (") || !w::value(out,s,p,index,term.condition) || !w::text(out,s,b".bytes[0] != 0u) {\n")){return false;}
    return edge(p,index,g,term.target,term.argument_start,term.argument_count,out,s) && w::text(out,s,b"    } else {\n") && edge(p,index,g,term.alternative,term.alternate_start,term.alternate_count,out,s) && w::text(out,s,b"    }\n");
}
pub fn bool declaration(usize index,ref mut bytes::Buffer out,ref mut w::State s){return w::text(out,s,b"static int32_t fygr_body_") && w::number(out,s,index) && w::text(out,s,b"(fycs_graph_v1 *, size_t, fycs_graph_observer_v1, void *, uint8_t *, uint32_t *);\n");}
pub fn bool emit(ref program::Verified proof,ref plan::Plan p,usize index,ref mut bytes::Buffer out,ref mut w::State s){
    let g=body(proof,index);
    if(!w::text(out,s,b"static int32_t fygr_body_") || !w::number(out,s,index) || !w::text(out,s,b"(fycs_graph_v1 *context, size_t max_key, fycs_graph_observer_v1 observe, void *user, uint8_t *exit_kind, uint32_t *trap_kind) {\n    int32_t status = 0; size_t mark = context->used;\n    (void)max_key; (void)observe; (void)user; (void)trap_kind; (void)fygr_bool_v1;\n")){return false;}
    let values=plan::value_count(p,index);let borrows=plan::borrow_count(p,index);
    if(values!=0 && (!w::text(out,s,b"    fycs_graph_value_v1 values[") || !w::number(out,s,values) || !w::text(out,s,b"] = {{0}};\n"))){return false;}
    if(borrows!=0 && (!w::text(out,s,b"    fycs_graph_borrow_v1 borrows[") || !w::number(out,s,borrows) || !w::text(out,s,b"] = {{0}};\n"))){return false;}
    if(!w::text(out,s,b"    goto block_") || !w::number(out,s,m::header(g).entry) || !w::text(out,s,b";\n")){return false;}
    var usize i=0;while(i<v::len<m::Block>(m::blocks(g))){let block=*v::at<m::Block>(m::blocks(g),i);
        if(!w::text(out,s,b"block_") || !w::number(out,s,block.id) || !w::text(out,s,b":;\n")){return false;}
        var usize j=0;while(j<block.operation_count){if(!ops::operation(proof,p,index,g,*v::at<m::Operation>(m::operations(g),block.operation_start+j),out,s)){return false;}j+=1;}
        if(!terminator(p,index,g,block.term,out,s)){return false;}i+=1;
    }
    return w::text(out,s,b"finished:\n    context->used = mark; return status;\n}\n");
}
pub fn bool entry(ref mut bytes::Buffer out,ref mut w::State s){return w::text(out,s,b"\nint32_t fycs_graph_execute_v1(fycs_graph_v1 *context, size_t max_key,\n    fycs_graph_observer_v1 observe, void *user, uint8_t *exit_kind, uint32_t *trap_kind) {\n    int32_t status;\n    if (context == NULL || context->transaction == NULL || observe == NULL ||\n        exit_kind == NULL || trap_kind == NULL || max_key == 0u) { return FYCS_OV_INVALID; }\n    if (context->transaction->state != 1u) { return FYCS_OV_CLOSED; }\n    *exit_kind = 3u; *trap_kind = 0u;\n    status = fygr_body_0(context,max_key,observe,user,exit_kind,trap_kind);\n    if (status == 0 && *exit_kind == 0u) { return fycs_overlay_commit_v1(context->transaction); }\n    (void)fycs_overlay_abort_v1(context->transaction);\n    if (status != 0) { *exit_kind = 0u; *trap_kind = 0u; return status; }\n    return *exit_kind == 1u || *exit_kind == 2u ? FYCS_OV_OK : FYCS_OV_INVALID;\n}\n");}
