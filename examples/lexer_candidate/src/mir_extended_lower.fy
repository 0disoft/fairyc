module mir_extended_lower;
import type_state as ids;
import hir_model as h;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_propagation as propagation;
import std::bytes as bytes;
import std::vec as v;
import mir_lower as lowering;
import mir_control_lower as control;

// Shared owned context and source-depth budgets remain with the coordinator.
pub fn usize extended_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Expression e, u32 kind, u32 type, u32 operator, usize depth) {
    if(kind==30){
        let usize left=if(e.left==0){0}else{lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1)};if(c.current==0 || c.problem.code!=0){return 0;}
        let usize right=if(e.right==0){0}else{lowering::expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1)};if(c.current==0 || c.problem.code!=0){return 0;}
        let usize length=if(e.alternative==0){0}else{lowering::expression(input,locals,blocks,operations,edges,payload,c,e.alternative,depth+1)};if(c.current==0 || c.problem.code!=0){return 0;}
        let usize start=v::len<usize>(edges);if(length!=0 && !state::edge(edges,length,c)){return 0;}
        let usize result=if(type==7){0}else{state::local(locals,c,type,0,false,false,e.start,e.end)};
        var m::Operation o=state::operation(39,result,type,h::storage_element(e.kind),h::storage_code(e.kind),left,right,e.start,e.end);o.boolean=true;
        if(length!=0){o.argument_start=start;o.argument_count=1;}
        return if(state::emit(operations,blocks,c,o)!=0){result}else{0};
    }
    if(kind==29){
        let usize owner=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}
        let usize data=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}
        let usize length=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.alternative,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}
        let usize start=v::len<usize>(edges);if(!state::edge(edges,length,c)){return 0;}
        let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o=state::operation(38,result,type,h::type_code(h::expression_at(input,e.left).type),0,owner,data,e.start,e.end);o.boolean=e.boolean;o.argument_start=start;o.argument_count=1;
        return if(state::emit(operations,blocks,c,o)!=0){result}else{0};
    }
    if(kind==28){let usize owner=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);var m::Operation o=state::operation(37,result,type,h::type_code(h::expression_at(input,e.left).type),0,owner,0,e.start,e.end);o.boolean=e.boolean;return if(state::emit(operations,blocks,c,o)!=0){result}else{0};}
    if(kind==26){let usize owner=lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);return if(state::emit(operations,blocks,c,state::operation(35,result,type,h::type_code(h::expression_at(input,e.left).type),0,owner,0,e.start,e.end))!=0){result}else{0};}
    if(kind==24){
        let usize donor=c.locals[e.binding];if(!shapes::needs_drop(type) || donor==0 || c.live[e.binding]!=1){return state::fail(c,4,e.start,e.end);}
        let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);c.live[e.binding]=0;
        if(state::emit(operations,blocks,c,state::operation(24,result,type,type,0,donor,0,e.start,e.end))==0){return 0;}return result;
    }
    if (kind == 23) {
        let usize subject = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner = h::type_code(h::expression_at(input,e.left).type);
        let usize result = state::local(locals,c,6,0,false,false,e.start,e.end);
        if (result == 0 || state::emit(operations,blocks,c,state::operation(22,result,6,owner,0,subject,0,e.start,e.end)) == 0) { return 0; }
        return result;
    }
    if (kind == 22) {
        let usize subject = lowering::expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        return propagation::lower(input,e,subject,locals,blocks,operations,edges,c);
    }
    if (kind == 18) {
        let shapes::Field member=shapes::field_at(h::nominal(input),e.binding);
        var usize[16] values=[0;16];var usize i=0;
        // Evaluate in written order, then arrange frozen operands by declaration.
        while(i<e.edge_count){let initializer=h::expression_at(input,h::edge_at(input,e.edge_start+i));
            let usize value=lowering::expression(input,locals,blocks,operations,edges,payload,c,initializer.left,depth+1);
            if(c.current==0 || c.problem.code!=0){return 0;}values[initializer.binding-member.payload_start-1]=value;i+=1;
        }
        let usize start=if(e.edge_count==0){0}else{v::len<usize>(edges)};i=0;
        while(i<e.edge_count){let usize value=values[i];if(!state::edge(edges,value,c)){return 0;}state::release(c,value);i+=1;}
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        let u32 tag=ids::small_id(shapes::member_tag(h::nominal(input),type,e.binding));
        var m::Operation o=state::operation(21,result,type,type,tag,0,0,e.start,e.end);o.argument_start=start;o.argument_count=e.edge_count;
        if(state::emit(operations,blocks,c,o)==0){return 0;}return result;
    }
    if (kind == 19) { return control::switch_expression(input,locals,blocks,operations,edges,payload,c,e,type,depth); }
    if(kind==27){let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);return if(state::emit(operations,blocks,c,state::operation(36,result,type,type,0,0,0,e.start,e.end))!=0){result}else{0};}
    return state::fail(c,4,e.start,e.end);
}
