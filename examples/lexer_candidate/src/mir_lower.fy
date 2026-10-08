module mir_lower;
import scalar_views as slices;
import pointer_types as pointers;
import reference_types as refs;
import array_types as arrays;
import type_state as ids;
import mir_arrays as array;
import mir_places as places;
import hir_model as h;
import hir_view_check as views;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import mir_ownership_snapshot as snapshots;
import mir_targets as targets_lower;
import mir_math as math;
import mir_cleanup as finalize;
import mir_resource_lower as lifetime;
import resource_identity as resources;
import mir_verify as verify;
import mir_nominal as nominal;
import mir_propagation as propagation;
import std::bytes as bytes;
import std::vec as v;

// Scalar lowering includes conditionals, loops and bounded lexical defer.

fn bool prepare_place(ref h::Program input, ref mut v::Vec<places::Step> path, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize depth) {
    var usize at=v::len<places::Step>(path);
    while(at!=0){at-=1;let places::Step step=*v::at<places::Step>(path,at);let e=h::expression_at(input,step.expression);
        if(h::expression_code(e.kind)==13){let usize raw=expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1);if(c.current==0 || c.problem.code!=0){return false;}
            let usize index=array::index(locals,blocks,operations,c,h::type_code(h::expression_at(input,e.left).type),raw,e.start,e.end);if(index==0){return false;}
            let ref mut places::Step slot=v::at_mut<places::Step>(path,at);slot.index=index;
        }
    }return true;
}

// Project addresses into original storage, never into by-value field snapshots.
fn usize address(ref h::Program p,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<usize> edges,ref mut bytes::Buffer payload,ref mut state::Context c,usize id,bool mutable,usize depth){
    let e=h::expression_at(p,id);if(depth>=64){return state::fail(c,2,e.start,e.end);}let u32 type=h::type_code(e.type);let u32 kind=h::expression_code(e.kind);
    if(kind==4 && h::operator_code(e.operator)==5){let usize parent=expression(p,locals,blocks,ops,edges,payload,c,e.left,depth+1);if(parent==0){return 0;}let u32 old=v::at<m::Local>(locals,parent-1).type;if(mutable || !refs::mutable(old)){return parent;}let u32 shared=refs::make(type,false);let usize result=state::local(locals,c,shared,0,false,false,e.start,e.end);return if(state::emit(ops,blocks,c,state::operation(34,result,shared,old,0,parent,0,e.start,e.end))!=0){result}else{0};}
    if(kind==3 && refs::valid(type)){return expression(p,locals,blocks,ops,edges,payload,c,id,depth+1);}
    let u32 borrowed=refs::make(type,mutable);if(borrowed==0){return state::fail(c,4,e.start,e.end);}var usize owner=0;var usize index=0;var u32 op=28;var u32 field=0;var u32 owner_type=type;
    if(kind==3){owner=c.locals[e.binding];}
    else if(kind==10){owner=address(p,locals,blocks,ops,edges,payload,c,e.left,mutable,depth+1);if(owner==0){return 0;}owner_type=v::at<m::Local>(locals,owner-1).type;op=32;field=ids::small_id(e.binding);}
    else if(kind==13 && h::expression_code(h::expression_at(p,e.left).kind)==3 && arrays::valid(h::type_code(h::expression_at(p,e.left).type))){owner=c.locals[h::expression_at(p,e.left).binding];owner_type=h::type_code(h::expression_at(p,e.left).type);let usize raw=expression(p,locals,blocks,ops,edges,payload,c,e.right,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}index=array::index(locals,blocks,ops,c,owner_type,raw,e.start,e.end);op=33;}
    else{return state::fail(c,4,e.start,e.end);}if(owner==0){return state::fail(c,4,e.start,e.end);}let usize result=state::local(locals,c,borrowed,0,false,false,e.start,e.end);
    return if(state::emit(ops,blocks,c,state::operation(op,result,borrowed,owner_type,field,owner,index,e.start,e.end))!=0){result}else{0};
}
fn usize expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    if (id == 0 || c.current == 0 || c.problem.code != 0) { return 0; }
    let h::Expression e = h::expression_at(input,id); let u32 kind = h::expression_code(e.kind); let u32 type = h::type_code(e.type); let u32 operator = h::operator_code(e.operator);
    if (depth >= 64) { return state::fail(c,2,e.start,e.end); }
    // Keep aggregate temporaries on the selected lane, not on every recursive
    // expression frame. Helpers preserve the original source-depth accounting.
    if(kind>=18 && kind!=25){return extended_expression(input,locals,blocks,operations,edges,payload,c,id,e,kind,type,operator,depth);}
    if(kind<=3 || kind==15 || kind==16 || kind==17 || kind==25 ||
       kind==4 && operator==5 && refs::valid(h::type_code(h::expression_at(input,e.left).type)) ||
       (kind==13 || kind==14) && h::type_code(h::expression_at(input,e.left).type)<256){return view_expression(input,locals,blocks,operations,edges,payload,c,id,e,kind,type,operator,depth);}
    if(kind==4 || kind==5 || kind==9 || kind==10 || kind==12 || kind==13 || kind==14){return aggregate_expression(input,locals,blocks,operations,edges,payload,c,id,e,kind,type,operator,depth);}
    return call_expression(input,locals,blocks,operations,edges,payload,c,id,e,kind,type,operator,depth);
}
fn usize extended_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Expression e, u32 kind, u32 type, u32 operator, usize depth) {
    if(kind==30){
        let usize left=if(e.left==0){0}else{expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1)};if(c.current==0 || c.problem.code!=0){return 0;}
        let usize right=if(e.right==0){0}else{expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1)};if(c.current==0 || c.problem.code!=0){return 0;}
        let usize length=if(e.alternative==0){0}else{expression(input,locals,blocks,operations,edges,payload,c,e.alternative,depth+1)};if(c.current==0 || c.problem.code!=0){return 0;}
        let usize start=v::len<usize>(edges);if(length!=0 && !state::edge(edges,length,c)){return 0;}
        let usize result=if(type==7){0}else{state::local(locals,c,type,0,false,false,e.start,e.end)};
        var m::Operation o=state::operation(39,result,type,h::storage_element(e.kind),h::storage_code(e.kind),left,right,e.start,e.end);o.boolean=true;
        if(length!=0){o.argument_start=start;o.argument_count=1;}
        return if(state::emit(operations,blocks,c,o)!=0){result}else{0};
    }
    if(kind==29){
        let usize owner=expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}
        let usize data=expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}
        let usize length=expression(input,locals,blocks,operations,edges,payload,c,e.alternative,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}
        let usize start=v::len<usize>(edges);if(!state::edge(edges,length,c)){return 0;}
        let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o=state::operation(38,result,type,h::type_code(h::expression_at(input,e.left).type),0,owner,data,e.start,e.end);o.boolean=e.boolean;o.argument_start=start;o.argument_count=1;
        return if(state::emit(operations,blocks,c,o)!=0){result}else{0};
    }
    if(kind==28){let usize owner=expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);var m::Operation o=state::operation(37,result,type,h::type_code(h::expression_at(input,e.left).type),0,owner,0,e.start,e.end);o.boolean=e.boolean;return if(state::emit(operations,blocks,c,o)!=0){result}else{0};}
    if(kind==26){let usize owner=expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(c.current==0 || c.problem.code!=0){return 0;}let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);return if(state::emit(operations,blocks,c,state::operation(35,result,type,h::type_code(h::expression_at(input,e.left).type),0,owner,0,e.start,e.end))!=0){result}else{0};}
    if(kind==24){
        let usize donor=c.locals[e.binding];if(!shapes::needs_drop(type) || donor==0 || c.live[e.binding]!=1){return state::fail(c,4,e.start,e.end);}
        let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);c.live[e.binding]=0;
        if(state::emit(operations,blocks,c,state::operation(24,result,type,type,0,donor,0,e.start,e.end))==0){return 0;}return result;
    }
    if (kind == 23) {
        let usize subject = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner = h::type_code(h::expression_at(input,e.left).type);
        let usize result = state::local(locals,c,6,0,false,false,e.start,e.end);
        if (result == 0 || state::emit(operations,blocks,c,state::operation(22,result,6,owner,0,subject,0,e.start,e.end)) == 0) { return 0; }
        return result;
    }
    if (kind == 22) {
        let usize subject = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        return propagation::lower(input,e,subject,locals,blocks,operations,edges,c);
    }
    if (kind == 18) {
        let shapes::Field member=shapes::field_at(h::nominal(input),e.binding);
        var usize[16] values=[0;16];var usize i=0;
        // Evaluate in written order, then arrange frozen operands by declaration.
        while(i<e.edge_count){let initializer=h::expression_at(input,h::edge_at(input,e.edge_start+i));
            let usize value=expression(input,locals,blocks,operations,edges,payload,c,initializer.left,depth+1);
            if(c.current==0 || c.problem.code!=0){return 0;}values[initializer.binding-member.payload_start-1]=value;i+=1;
        }
        let usize start=if(e.edge_count==0){0}else{v::len<usize>(edges)};i=0;
        while(i<e.edge_count){let usize value=values[i];if(!state::edge(edges,value,c)){return 0;}state::release(c,value);i+=1;}
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        let u32 tag=ids::small_id(shapes::member_tag(h::nominal(input),type,e.binding));
        var m::Operation o=state::operation(21,result,type,type,tag,0,0,e.start,e.end);o.argument_start=start;o.argument_count=e.edge_count;
        if(state::emit(operations,blocks,c,o)==0){return 0;}return result;
    }
    if (kind == 19) { return switch_expression(input,locals,blocks,operations,edges,payload,c,e,type,depth); }
    if(kind==27){let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);return if(state::emit(operations,blocks,c,state::operation(36,result,type,type,0,0,0,e.start,e.end))!=0){result}else{0};}
    return state::fail(c,4,e.start,e.end);
}
fn usize view_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Expression e, u32 kind, u32 type, u32 operator, usize depth) {
    if (kind == 1 || kind == 2) {
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o = state::operation(1,result,type,type,0,0,0,e.start,e.end); o.boolean = e.boolean;
        if (kind == 1) {
            let ref u8[] text = h::integer_text(input,id);
            if (text.len > 688128 - bytes::len(payload)) { return state::fail(c,2,e.start,e.end); }
            o.payload_start = bytes::len(payload); if (!bytes::append(payload,text).is_ok) { return state::fail(c,3,e.start,e.end); } o.payload_end = bytes::len(payload);
        }
        if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if (kind == 3) {
        let usize original = c.locals[e.binding];
        if (original == 0) { return state::fail(c,4,e.start,e.end); }
        if(shapes::needs_drop(type)){return original;}
        // Freeze source reads before later child expressions can assign/call.
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(2,result,type,type,0,original,0,e.start,e.end)) == 0) { return 0; } return result;
    }
    if(kind==4 && operator==5 && refs::valid(h::type_code(h::expression_at(input,e.left).type))){
        if(shapes::needs_drop(type)){return state::fail(c,4,e.start,e.end);}let usize owner=expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);let u32 owner_type=h::type_code(h::expression_at(input,e.left).type);return if(state::emit(operations,blocks,c,state::operation(29,result,type,owner_type,0,owner,0,e.start,e.end))!=0){result}else{0};
    }
    if(kind==25){let usize owner=expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1);if(owner==0){return 0;}let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);return if(state::emit(operations,blocks,c,state::operation(34,result,type,h::type_code(h::expression_at(input,e.left).type),0,owner,0,e.start,e.end))!=0){result}else{0};}
    if (kind == 16) {
        if(refs::valid(type)){return address(input,locals,blocks,operations,edges,payload,c,e.left,e.boolean,depth+1);}
        let h::Expression source=h::expression_at(input,e.left);
        let bool projected=h::expression_code(source.kind)==10;
        let usize owner=if(projected){address(input,locals,blocks,operations,edges,payload,c,e.left,e.boolean,depth+1)}else{c.locals[source.binding]};
        if (owner == 0) { return state::fail(c,4,e.start,e.end); }
        let u32 array_type = h::type_code(h::expression_at(input,e.left).type);
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(18,result,type,if(projected){v::at<m::Local>(locals,owner-1).type}else{array_type},0,owner,0,e.start,e.end)) == 0) { return 0; } return result;
    }
    if (kind == 17) {
        let usize owner = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1); if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize raw = expression(input,locals,blocks,operations,edges,payload,c,e.right,depth + 1); if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize start = state::local(locals,c,5,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(2,start,5,5,0,raw,0,e.start,e.end)) == 0) { return 0; }
        var usize end = 0;
        if (e.boolean) { end = state::local(locals,c,5,0,false,false,e.start,e.end); if (state::emit(operations,blocks,c,state::operation(15,end,5,type,0,owner,0,e.start,e.end)) == 0) { return 0; } }
        else { end = expression(input,locals,blocks,operations,edges,payload,c,e.alternative,depth + 1); if (c.current == 0 || c.problem.code != 0) { return 0; } }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end); var m::Operation o = state::operation(19,result,type,type,0,owner,start,e.start,e.end);
        o.argument_start = v::len<usize>(edges); o.argument_count = 1;
        if (!state::edge(edges,end,c) || state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if (kind == 15) {
        let ref u8[] data = h::literal_bytes(input,id);
        if (bytes::len(payload) > 688128 || data.len > 688128 - bytes::len(payload)) { return state::fail(c,2,e.start,e.end); }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end); var m::Operation o = state::operation(14,result,type,type,0,0,0,e.start,e.end);
        o.payload_start = bytes::len(payload); if (!bytes::append(payload,data).is_ok) { return state::fail(c,3,e.start,e.end); } o.payload_end = bytes::len(payload);
        if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if ((kind == 13 || kind == 14) && h::type_code(h::expression_at(input,e.left).type) < 256) {
        let u32 owner_type = h::type_code(h::expression_at(input,e.left).type);
        let usize owner = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1); if (c.current == 0 || c.problem.code != 0) { return 0; }
        if (kind == 14) { let usize result = state::local(locals,c,type,0,false,false,e.start,e.end); var m::Operation o = state::operation(15,result,type,owner_type,0,owner,0,e.start,e.end); o.boolean = e.boolean; if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result; }
        let usize raw = expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1); if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize index = state::local(locals,c,5,0,false,false,e.start,e.end); if (state::emit(operations,blocks,c,state::operation(16,index,5,owner_type,0,owner,raw,e.start,e.end)) == 0) { return 0; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end); if (state::emit(operations,blocks,c,state::operation(17,result,type,owner_type,0,owner,index,e.start,e.end)) == 0) { return 0; } return result;
    }
    return state::fail(c,4,e.start,e.end);
}
fn usize aggregate_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Expression e, u32 kind, u32 type, u32 operator, usize depth) {
    if (kind == 12) {
        var v::Vec<usize> values = v::empty<usize>(); var usize i = 0;
        while (i < e.edge_count) { let usize value = expression(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,e.edge_start+i),depth+1); if (c.current == 0 || c.problem.code != 0) { return 0; } if (!v::push<usize>(ref mut values,value).is_ok) { return state::fail(c,3,e.start,e.end); } i += 1; }
        let usize first = v::len<usize>(edges); i = 0;
        while (i < e.edge_count) { if (!state::edge(edges,*v::at<usize>(ref values,i),c)) { return 0; } i += 1; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o = state::operation(9,result,type,type,0,0,0,e.start,e.end); o.argument_start=first; o.argument_count=e.edge_count; o.boolean=e.boolean;
        if (state::emit(operations,blocks,c,o)==0) { return 0; } return result;
    }
    if (kind == 13 && places::root(input,id,false)!=0) {
        var v::Vec<places::Step> path=switch(places::collect(input,id)){case .ok(p):{p}case .err(code):{return state::fail(c,code,e.start,e.end);}};
        if(!prepare_place(input,ref mut path,locals,blocks,operations,edges,payload,c,depth)){return 0;}
        return places::read(input,ref path,c.locals[places::root(input,id,false)],locals,blocks,operations,c,e.start,e.end);
    }
    if (kind == 13 || kind == 14) {
        let h::Expression root = h::expression_at(input,e.left);
        let usize owner = if (kind == 13 && h::expression_code(root.kind) == 3) { c.locals[root.binding] } else { expression(input,locals,blocks,operations,edges,payload,c,e.left,depth+1) };
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner_type = h::type_code(h::expression_at(input,e.left).type);
        if (kind == 14) { let usize result = state::local(locals,c,5,0,false,false,e.start,e.end); if (state::emit(operations,blocks,c,state::operation(13,result,5,owner_type,0,owner,0,e.start,e.end))==0) { return 0; } return result; }
        let usize raw = expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize index = array::index(locals,blocks,operations,c,owner_type,raw,e.start,e.end);
        return array::read(locals,blocks,operations,c,owner_type,type,owner,index,e.start,e.end);
    }
    if (kind == 10) {
        let usize owner = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 receiver_type=h::type_code(h::expression_at(input,e.left).type);if(refs::valid(receiver_type)){
            if(shapes::needs_drop(type)){return state::fail(c,4,e.start,e.end);}let usize result=state::local(locals,c,type,0,false,false,e.start,e.end);return if(state::emit(operations,blocks,c,state::operation(31,result,type,receiver_type,ids::small_id(e.binding),owner,0,e.start,e.end))!=0){result}else{0};}
        return nominal::read(locals,blocks,operations,c,owner,type,e.binding,e.start,e.end);
    }
    if (kind == 9) {
        let ref shapes::Table table = h::nominal(input); let usize index = shapes::find(table,type);
        if (index == 0 || e.edge_count > 65536 - v::len<usize>(edges)) { return state::fail(c,4,e.start,e.end); }
        let shapes::Shape shape = shapes::at(table,index - 1);
        var v::Vec<usize> values = v::empty<usize>(); var usize i = 0;
        while (i < shape.field_count) { if (!v::push<usize>(ref mut values,0).is_ok) { return state::fail(c,3,e.start,e.end); } i += 1; }
        i = 0;
        while (i < e.edge_count) {
            let h::Expression initializer = h::expression_at(input,h::edge_at(input,e.edge_start + i));
            let usize value = expression(input,locals,blocks,operations,edges,payload,c,initializer.left,depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return 0; }
            { let ref mut usize slot = v::at_mut<usize>(ref mut values,initializer.binding - shape.field_start - 1); *slot = value; }
            i += 1;
        }
        let usize start = v::len<usize>(edges); i = 0;
        while (i < shape.field_count) { let usize value=*v::at<usize>(ref values,i);if (!state::edge(edges,value,c)) { return 0; }state::release(c,value);i += 1; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        var m::Operation o = state::operation(6,result,type,type,0,0,0,e.start,e.end); o.argument_start = start; o.argument_count = shape.field_count;
        if (state::emit(operations,blocks,c,o) == 0) { return 0; } return result;
    }
    if (kind == 4 || kind == 5) {
        let usize left = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner_type = h::type_code(h::expression_at(input,e.left).type);
        if (kind == 5) {
            if (operator == 19 || operator == 20) { return logical_expression(input,locals,blocks,operations,edges,payload,c,e,operator,left,depth); }
            let usize right = expression(input,locals,blocks,operations,edges,payload,c,e.right,depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return 0; }
            return math::binary(locals,blocks,operations,c,type,owner_type,operator,left,right,e.start,e.end);
        }
        let u32 mask = math::trap_mask(operator,owner_type,true);
        if (mask != 0 && !state::guard(operations,blocks,c,owner_type,operator,left,0,mask,e.start,e.end)) { return 0; }
        let usize result = state::local(locals,c,type,0,false,false,e.start,e.end);
        if (state::emit(operations,blocks,c,state::operation(3,result,type,owner_type,operator,left,0,e.start,e.end)) == 0) { return 0; } return result;
    }
    return state::fail(c,4,e.start,e.end);
}
fn usize call_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, h::Expression e, u32 kind, u32 type, u32 operator, usize depth) {
    if (kind == 6) {
        let usize callee = c.targets[e.binding]; if (callee == 0) { return state::fail(c,4,e.start,e.end); }
        // Materialize ordered arguments before publishing their contiguous
        // range. An argument may return, leaving no orphan parent-call edges.
        if (e.edge_count > 65536 - v::len<usize>(edges)) { return state::fail(c,2,e.start,e.end); }
        var v::Vec<usize> arguments = v::empty<usize>(); var usize j = 0;
        while (j < e.edge_count) {
            let usize argument = expression(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,e.edge_start + j),depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return 0; }
            if (!v::push<usize>(ref mut arguments,argument).is_ok) { return state::fail(c,3,e.start,e.end); } j += 1;
        }
        let usize start = v::len<usize>(edges); j = 0;
        while (j < e.edge_count) {
            let usize argument = *v::at<usize>(ref arguments,j);
            if (!state::edge(edges,argument,c)) { return 0; }state::release(c,argument);j += 1;
        }
        let usize destination = if (type == 7) { 0 } else { state::local(locals,c,type,0,false,false,e.start,e.end) };
        let usize continuation = state::new_block(blocks,c,e.start,e.end);
        var m::Block term = state::block(c.function,e.start,e.end); term.kind = 5; term.callee = callee; term.unsafe_call=e.boolean;
        term.destination = destination; term.argument_start = start; term.argument_count = e.edge_count; term.target = continuation;
        if (!state::terminate(blocks,c,term)) { return 0; } c.current = continuation;if(shapes::needs_drop(type)){state::hold(c,destination);}return destination;
    }
    if (kind == 7) { return block(input,locals,blocks,operations,edges,payload,c,e.block,depth + 1); }
    if (kind == 8) { return if_expression(input,locals,blocks,operations,edges,payload,c,e,type,depth); }
    return state::fail(c,4,e.start,e.end);
}
// Large ownership snapshots live only on actual branch frames, not on every
// recursive scalar/field node. Helpers retain the same source-depth accounting.
fn usize switch_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, h::Expression e, u32 type, usize depth) {
        // Evaluate the subject once. The final exhaustive arm needs no test.
        let usize subject = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let u32 owner = h::type_code(h::expression_at(input,e.left).type);
        let shapes::Shape shape = shapes::at(h::nominal(input),shapes::find(h::nominal(input),owner) - 1);
        let usize result = if (type == 7) { 0 } else { state::local(locals,c,type,0,false,false,e.start,e.end) };
        let u8[1153] incoming=c.live;let u8[4096] incoming_pending=c.pending;var u8[1153] merged=[0;1153];var bool continuing=false;
        var v::Vec<usize> ends = v::empty<usize>(); var usize i = 0;
        while (i < e.edge_count) {
            c.live=incoming;c.pending=incoming_pending;
            let h::Expression arm = h::expression_at(input,h::edge_at(input,e.edge_start + i)); var usize next = 0;
            if (i + 1 < e.edge_count) {
                let usize condition = state::local(locals,c,6,0,false,false,arm.start,arm.end);
                let u32 tag = ids::small_id(shapes::member_tag(h::nominal(input),owner,arm.binding));
                if (state::emit(operations,blocks,c,state::operation(22,condition,6,owner,tag,subject,0,arm.start,arm.end)) == 0) { return 0; }
                let usize body = state::new_block(blocks,c,arm.start,arm.end); next = state::new_block(blocks,c,e.start,e.end);
                if (!state::branch(blocks,c,condition,body,next,arm.start,arm.end)) { return 0; } c.current = body;
            }
            var usize capture=0;while(capture<arm.edge_count){let field=h::expression_at(input,h::edge_at(input,arm.edge_start+capture));
                if(field.edge_count==1){let usize binding=h::edge_at(input,field.edge_start);let usize local=c.locals[binding];let u32 field_type=h::type_code(field.type);
                    if(local==0 || state::emit(operations,blocks,c,state::operation(23,local,field_type,owner,ids::small_id(field.binding),subject,0,field.start,field.end))==0){return 0;}
                    if(shapes::needs_drop(owner)){state::release(c,subject);}if(shapes::needs_drop(field_type)){c.live[binding]=1;}
                }else if(shapes::needs_drop(owner)){
                    let u32 field_type=h::type_code(field.type);let usize ignored=state::local(locals,c,field_type,0,false,false,field.start,field.end);
                    if(state::emit(operations,blocks,c,state::operation(23,ignored,field_type,owner,ids::small_id(field.binding),subject,0,field.start,field.end))==0){return 0;}
                    state::release(c,subject);if(shapes::needs_drop(field_type)){state::in_scope(c,ignored,h::block_at(input,arm.block).scope);}
                }capture+=1;
            }
            if(shapes::needs_drop(owner)){var bool extracted=arm.edge_count!=0;var usize j=0;while(j<arm.edge_count){extracted=extracted || h::expression_at(input,h::edge_at(input,arm.edge_start+j)).edge_count==1;j+=1;}
                if(!extracted && !lifetime::drop(input,locals,blocks,operations,edges,c,subject,arm.start,arm.end)){return 0;}}
            let usize value = block(input,locals,blocks,operations,edges,payload,c,arm.block,depth + 1);
            if (c.problem.code != 0) { return 0; }
            if (c.current != 0) {
                if (result != 0 && state::emit(operations,blocks,c,state::operation(if(shapes::needs_drop(type)){24}else{2},result,type,type,0,value,0,arm.start,arm.end)) == 0) { return 0; }
                if(!continuing){merged=c.live;continuing=true;}
                if (!v::push<usize>(ref mut ends,c.current).is_ok) { return state::fail(c,3,arm.start,arm.end); }
            }
            c.current = next; i += 1;
        }
        if (v::len<usize>(ref ends) == 0) { c.current = 0; return 0; }
        let usize join = state::new_block(blocks,c,e.start,e.end); i = 0;
        while (i < v::len<usize>(ref ends)) { c.current = *v::at<usize>(ref ends,i); if (!state::jump(blocks,c,join,e.start,e.end)) { return 0; } i += 1; }
        c.live=merged;c.current = join; return result;
}
fn usize if_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, h::Expression e, u32 type, usize depth) {
        let usize condition = expression(input,locals,blocks,operations,edges,payload,c,e.left,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return 0; }
        let usize consequent = state::new_block(blocks,c,e.start,e.end); let usize alternative = state::new_block(blocks,c,e.start,e.end);
        if (!state::branch(blocks,c,condition,consequent,alternative,e.start,e.end)) { return 0; }
        let usize result = if (type == 7) { 0 } else { state::local(locals,c,type,0,false,false,e.start,e.end) };
        let bytes::Buffer incoming=snapshots::capture(c,e.start,e.end);if(c.problem.code!=0){return 0;}
        c.current = consequent; let usize yes = block(input,locals,blocks,operations,edges,payload,c,e.block,depth + 1);
        if (c.problem.code != 0) { return 0; }
        if (c.current != 0 && result != 0 && state::emit(operations,blocks,c,state::operation(if(shapes::needs_drop(type)){24}else{2},result,type,type,0,yes,0,e.start,e.end)) == 0) { return 0; }
        let usize yes_end = c.current;
        let bytes::Buffer yes_state=snapshots::capture(c,e.start,e.end);if(c.problem.code!=0){return 0;}snapshots::restore(c,ref incoming);
        c.current = alternative; let usize no = expression(input,locals,blocks,operations,edges,payload,c,e.alternative,depth + 1);
        if (c.problem.code != 0) { return 0; }
        if (c.current != 0 && result != 0 && state::emit(operations,blocks,c,state::operation(if(shapes::needs_drop(type)){24}else{2},result,type,type,0,no,0,e.start,e.end)) == 0) { return 0; }
        let usize no_end = c.current;
        if(yes_end!=0){snapshots::restore(c,ref yes_state);}
        // Allocate a join only for live fallthrough; both returning arms leave
        // no orphan unfinished block and suppress enclosing RHS/store/return.
        if (yes_end == 0 && no_end == 0) { c.current = 0; return 0; }
        let usize join = state::new_block(blocks,c,e.start,e.end);
        if (yes_end != 0) { c.current = yes_end; if (!state::jump(blocks,c,join,e.start,e.end)) { return 0; } }
        if (no_end != 0) { c.current = no_end; if (!state::jump(blocks,c,join,e.start,e.end)) { return 0; } }
        c.current = join; return result;
}
fn usize logical_expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, h::Expression e, u32 operator, usize left, usize depth) {
                // Default result is the frozen left value. Only the required
                // RHS path overwrites it; its return/trap never reaches join.
                let usize result = state::local(locals,c,6,0,false,false,e.start,e.end);
                if (state::emit(operations,blocks,c,state::operation(2,result,6,6,0,left,0,e.start,e.end)) == 0) { return 0; }
                let usize rhs = state::new_block(blocks,c,e.start,e.end); let usize join = state::new_block(blocks,c,e.start,e.end);
                let usize consequent = if (operator == 19) { rhs } else { join };
                let usize alternative = if (operator == 19) { join } else { rhs };
                if (!state::branch(blocks,c,left,consequent,alternative,e.start,e.end)) { return 0; }
                let u8[1153] skipped=c.live;let u8[4096] skipped_pending=c.pending;
                c.current = rhs; let usize right = expression(input,locals,blocks,operations,edges,payload,c,e.right,depth + 1);
                if (c.problem.code != 0) { return 0; }
                if (c.current != 0) {
                    if (state::emit(operations,blocks,c,state::operation(2,result,6,6,0,right,0,e.start,e.end)) == 0) { return 0; }
                    // RHS-only owned temporaries cannot cross the skipped edge.
                    // Freeze the boolean, then discharge those values in reverse
                    // creation order before both paths meet with identical states.
                    var usize temporary=v::len<m::Local>(locals);
                    while(temporary>c.base){if(state::pending(c,temporary)==1 && skipped_pending[temporary-c.base-1]==0){if(!lifetime::drop(input,locals,blocks,operations,edges,c,temporary,e.start,e.end)){return 0;}}temporary-=1;}
                    if (!state::jump(blocks,c,join,e.start,e.end)) { return 0; }
                }
                c.live=skipped;c.pending=skipped_pending;c.current = join; return result;
}
fn bool statement(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    let h::Statement s = h::statement_at(input,id); let u32 kind = h::statement_code(s.kind);
    if (depth >= 64) { discard state::fail(c,2,s.start,s.end); return false; }
    if (kind == 11) {
        let h::Expression call = h::expression_at(input,s.expression);
        let usize callee = c.targets[call.binding];
        let u32 target_kind=h::binding_code(h::binding_at(input,call.binding).kind);
        if (h::expression_code(call.kind) != 6 || h::type_code(call.type) != 7 || callee == 0 || (target_kind!=1 && target_kind!=11)) { discard state::fail(c,4,s.start,s.end); return false; }
        if (c.defer_count >= 256) { discard state::fail(c,2,s.start,s.end); return false; }
        let usize captured = c.capture_count; var usize j = 0;
        while (j < call.edge_count) {
            let usize value = expression(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,call.edge_start + j),depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
            if (c.capture_count >= 4096) { discard state::fail(c,2,s.start,s.end); return false; }
            let u32 capture_type=v::at<m::Local>(locals,value-1).type;if(capture_type<1 || capture_type>6){discard state::fail(c,4,s.start,s.end);return false;}
            let usize slot = c.capture_count; c.captures[slot] = value; c.capture_count += 1; j += 1;
        }
        let usize at = c.defer_count; c.defer_count += 1;
        c.deferred_unsafe[at]=call.boolean; c.deferred_targets[at] = callee; c.deferred_starts[at] = s.start; c.deferred_ends[at] = s.end;
        c.deferred_arg_starts[at] = captured; c.deferred_arg_counts[at] = call.edge_count;
        return true;
    }
    if (kind == 7 || kind == 8) {
        if (c.loop_depth == 0 || c.loop_ids[c.loop_depth - 1] != s.loop_target) { discard state::fail(c,4,s.start,s.end); return false; }
        let usize frame = c.loop_depth - 1;
        if (kind == 8 && c.continues[frame] == 0) {
            // A for-post block is created only when a continue or fallthrough
            // reaches it, so an unconditional break/return leaves no orphan.
            let h::Statement loop = h::statement_at(input,s.loop_target);
            let usize post = state::new_block(blocks,c,loop.start,loop.end); c.continues[frame] = post;
        }
        let usize target = if (kind == 7) { c.breaks[frame] } else { c.continues[frame] };
        let usize from = c.loop_defer_base[frame];
        let h::Statement owner=h::statement_at(input,s.loop_target);
        if(!finalize::exit(input,locals,blocks,operations,edges,c,from,h::block_at(input,owner.body).scope,true,s.start,s.end)){return false;}
        return state::jump(blocks,c,target,s.start,s.end);
    }
    if(kind==12){let e=h::expression_at(input,s.expression);let usize binding=e.binding;let usize local=c.locals[binding];if(h::expression_code(e.kind)!=24 || local==0 || c.live[binding]==0){discard state::fail(c,4,s.start,s.end);return false;}c.live[binding]=0;state::release(c,local);let u32 type=h::type_code(e.type);return state::emit(operations,blocks,c,state::operation(27,0,type,type,0,local,0,s.start,s.end))!=0;}
    if (kind == 5 || kind == 6) {
        if (s.initial != 0 && !statement(input,locals,blocks,operations,edges,payload,c,s.initial,depth + 1)) { return false; }
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        let usize header = state::new_block(blocks,c,s.start,s.end);
        if (!state::jump(blocks,c,header,s.start,s.end)) { return false; }
        c.current = header;
        var usize condition = 0;
        if (s.expression != 0) { condition = expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1); }
        else {
            condition = state::local(locals,c,6,0,false,false,s.start,s.end);
            var m::Operation truth = state::operation(1,condition,6,6,0,0,0,s.start,s.end); truth.boolean = true;
            if (state::emit(operations,blocks,c,truth) == 0) { return false; }
        }
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        let usize body = state::new_block(blocks,c,s.start,s.end);
        let usize exit = state::new_block(blocks,c,s.start,s.end);
        if (!state::branch(blocks,c,condition,body,exit,s.start,s.end)) { return false; }
        if (c.loop_depth >= 64) { discard state::fail(c,2,s.start,s.end); return false; }
        let usize frame = c.loop_depth; c.loop_depth += 1;
        let u8[1153] incoming=c.live;let u8[4096] incoming_pending=c.pending;
        c.loop_ids[frame] = id; c.breaks[frame] = exit;
        let usize deferred_base = c.defer_count; c.loop_defer_base[frame] = deferred_base;
        c.continues[frame] = if (s.post == 0) { header } else { 0 };
        c.current = body;
        discard block(input,locals,blocks,operations,edges,payload,c,s.body,depth + 1);
        if (c.problem.code != 0) { return false; }
        if (c.current != 0) {
            if (c.continues[frame] == 0) { let usize post = state::new_block(blocks,c,s.start,s.end); c.continues[frame] = post; }
            let usize target = c.continues[frame];
            if (!state::jump(blocks,c,target,s.start,s.end)) { return false; }
        }
        let usize post = c.continues[frame]; c.loop_depth -= 1;
        if (s.post != 0 && post != 0) {
            c.current = post;
            if (!statement(input,locals,blocks,operations,edges,payload,c,s.post,depth + 1)) { return false; }
            if (c.current != 0 && !state::jump(blocks,c,header,s.start,s.end)) { return false; }
        }
        c.live=incoming;c.pending=incoming_pending;c.current = exit; return true;
    }
    if(kind==3){let target=h::expression_at(input,s.target);let usize root=places::root(input,s.target,false);let bool deref=h::expression_code(target.kind)==4 && h::operator_code(target.operator)==5;
        if(deref || root!=0 && refs::mutable(h::type_code(h::binding_at(input,root).type)) && s.target!=0 && h::expression_code(target.kind)==10){
            let usize pointer=if(deref){expression(input,locals,blocks,operations,edges,payload,c,target.left,depth+1)}else{address(input,locals,blocks,operations,edges,payload,c,s.target,true,depth+1)};if(pointer==0 || c.current==0){return c.problem.code==0;}
            let u32 type=h::type_code(target.type);let u32 owner_type=v::at<m::Local>(locals,pointer-1).type;let u32 op=h::operator_code(s.operator);var usize prior=0;
            if(op!=21){prior=state::local(locals,c,type,0,false,false,s.start,s.end);if(state::emit(operations,blocks,c,state::operation(29,prior,type,owner_type,0,pointer,0,s.start,s.end))==0){return false;}}
            var usize value=expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth+1);if(c.current==0 || c.problem.code!=0){return c.problem.code==0;}if(op!=21){value=math::binary(locals,blocks,operations,c,type,type,op,prior,value,s.start,s.end);}
            return state::emit(operations,blocks,c,state::operation(30,0,type,owner_type,0,pointer,value,s.start,s.end))!=0;
        }
    }
    if (kind == 3 && h::expression_code(h::expression_at(input,s.target).kind) == 13 && slices::mutable(h::type_code(h::expression_at(input,h::expression_at(input,s.target).left).type))) {
        let target = h::expression_at(input,s.target); let u32 operator = h::operator_code(s.operator);let u32 owner_type=h::type_code(h::expression_at(input,target.left).type);let u32 type=h::type_code(target.type);
        // Freeze the descriptor, then evaluate and check the index before RHS.
        let usize owner = expression(input,locals,blocks,operations,edges,payload,c,target.left,depth+1);
        let usize raw = expression(input,locals,blocks,operations,edges,payload,c,target.right,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        let usize index = state::local(locals,c,5,0,false,false,s.start,s.end);
        if (state::emit(operations,blocks,c,state::operation(16,index,5,owner_type,0,owner,raw,s.start,s.end)) == 0) { return false; }
        var usize prior = 0;
        if (operator != 21) { prior = state::local(locals,c,type,0,false,false,s.start,s.end); if (state::emit(operations,blocks,c,state::operation(17,prior,type,owner_type,0,owner,index,s.start,s.end)) == 0) { return false; } }
        var usize value = expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth+1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (operator != 21) { value = math::binary(locals,blocks,operations,c,type,type,operator,prior,value,s.start,s.end); }
        var m::Operation o = state::operation(20,0,type,owner_type,0,owner,index,s.start,s.end); o.argument_start=v::len<usize>(edges); o.argument_count=1;
        return state::edge(edges,value,c) && state::emit(operations,blocks,c,o) != 0;
    }
    if (kind == 3 && h::expression_code(h::expression_at(input,s.target).kind) == 13) {
        let usize binding=places::root(input,s.target,true);let usize destination=c.locals[binding];
        if(binding==0 || destination==0){discard state::fail(c,4,s.start,s.end);return false;}
        var v::Vec<places::Step> path=switch(places::collect(input,s.target)){case .ok(p):{p}case .err(code):{discard state::fail(c,code,s.start,s.end);return false;}};
        if(!prepare_place(input,ref mut path,locals,blocks,operations,edges,payload,c,depth)){return c.problem.code==0;}
        let u32 type=h::type_code(h::expression_at(input,s.target).type);let u32 operator=h::operator_code(s.operator);var usize prior=0;
        if(operator!=21){prior=places::read(input,ref path,destination,locals,blocks,operations,c,s.start,s.end);}
        var usize value=expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth+1);if(c.current==0 || c.problem.code!=0){return c.problem.code==0;}
        if(operator!=21){value=math::binary(locals,blocks,operations,c,type,type,operator,prior,value,s.start,s.end);}
        value=places::store(input,ref path,destination,value,locals,blocks,operations,edges,c,s.start,s.end);if(value==0){return false;}
        let u32 owner=v::at<m::Local>(locals,destination-1).type;
        return state::emit(operations,blocks,c,state::operation(2,destination,owner,owner,0,value,0,s.start,s.end))!=0;
    }
    if (kind == 1 || kind == 3) {
        let usize binding = if (kind == 1) { s.binding } else { nominal::root(input,s.target) };
        let usize destination = c.locals[binding]; if (destination == 0) { discard state::fail(c,4,s.start,s.end); return false; }
        let u32 type = if (kind == 1) { v::at<m::Local>(locals,destination - 1).type } else { h::type_code(h::expression_at(input,s.target).type) }; let u32 operator = h::operator_code(s.operator);
        var usize left = 0;
        if (kind == 3 && operator != 21) {
            let h::Expression target = h::expression_at(input,s.target);
            left = expression(input,locals,blocks,operations,edges,payload,c,s.target,depth + 1);
            if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        }
        var usize value = expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (kind == 3 && operator != 21) { value = math::binary(locals,blocks,operations,c,type,type,operator,left,value,s.start,s.end); }
        if (kind == 3) { value = nominal::store(input,s.target,destination,value,locals,blocks,operations,c,s.start,s.end); if (value == 0) { return false; } }
        let u32 destination_type = v::at<m::Local>(locals,destination - 1).type;
        if(shapes::needs_drop(destination_type)){
            if(kind==3 && h::expression_code(h::expression_at(input,s.target).kind)==10){return value==destination;}
            // The complete RHS is frozen above. A move of the old root has
            // already discharged it; otherwise retire its previous generation.
            if(c.live[binding]!=0){
                if(c.live[binding]!=1){discard state::fail(c,1104,s.start,s.end);return false;}
                c.live[binding]=0;if(!lifetime::drop(input,locals,blocks,operations,edges,c,destination,s.start,s.end)){return false;}
            }
            c.live[binding]=1;return state::emit(operations,blocks,c,state::operation(24,destination,destination_type,destination_type,0,value,0,s.start,s.end))!=0;
        }
        return state::emit(operations,blocks,c,state::operation(2,destination,destination_type,destination_type,0,value,0,s.start,s.end)) != 0;
    }
    if (kind == 2) {
        let usize value = expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        c.keep=value;
        if (!finalize::exit(input,locals,blocks,operations,edges,c,0,0,false,s.start,s.end)) { return false; }
        var m::Block term = state::block(c.function,s.start,s.end); term.kind = 1; term.value = value;
        return state::terminate(blocks,c,term);
    }
    if (kind == 4 || kind == 9 || kind == 10) {
        let usize value = expression(input,locals,blocks,operations,edges,payload,c,s.expression,depth + 1);
        if (c.current == 0 || c.problem.code != 0) { return c.problem.code == 0; }
        if (kind == 9) {
            if (!state::guard(operations,blocks,c,6,0,value,0,16,s.start,s.end)) { return false; }
            let h::Expression assertion = h::expression_at(input,s.expression);
            if (h::expression_code(assertion.kind) == 2 && !assertion.boolean) {
                var m::Block term = state::block(c.function,s.start,s.end); term.kind = 7;
                return state::terminate(blocks,c,term);
            }
            return true;
        }
        if(value!=0 && shapes::needs_drop(v::at<m::Local>(locals,value-1).type)){return lifetime::drop(input,locals,blocks,operations,edges,c,value,s.start,s.end);}
        return true;
    }
    discard state::fail(c,4,s.start,s.end); return false;
}
fn usize block(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    let h::Block b = h::block_at(input,id);
    if (depth >= 64) { return state::fail(c,2,b.start,b.end); }
    let usize deferred_base = c.defer_count; let usize capture_base = c.capture_count;let usize previous_scope=c.scope;c.scope=b.scope;
    var usize i = 0;
    while (i < b.edge_count && c.current != 0 && c.problem.code == 0) {
        if (!statement(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,b.edge_start + i),depth + 1)) { return 0; } i += 1;
    }
    let usize value = expression(input,locals,blocks,operations,edges,payload,c,b.tail,depth + 1);
    let usize kept=c.keep;c.keep=value;
    if (c.current != 0 && c.problem.code == 0) { if(!finalize::exit(input,locals,blocks,operations,edges,c,deferred_base,b.scope,false,b.start,b.end)){return 0;} }
    c.defer_count = deferred_base; c.capture_count = capture_base;c.keep=kept;c.scope=previous_scope;
    if(c.current!=0 && value!=0 && shapes::needs_drop(h::type_code(h::expression_at(input,b.tail).type))){state::hold(c,value);}
    return value;
}

pub fn Result<m::Program,m::Problem> lower(h::Program input) {
    let h::Problem proof = views::check(ref input); if (proof.code != 0) { return .err(m::Problem { .code=proof.code,.start=proof.start,.end=proof.end }); }
    let shapes::Table table = switch(shapes::copy(h::nominal(ref input))) { case .ok(value): { value } case .err(code): { return .err(m::Problem { .code, .start = 0, .end = 0 }); } };
    var state::Context c = state::empty(); var bytes::Buffer source = bytes::empty(); var bytes::Buffer payload = bytes::empty();
    var v::Vec<m::Target> targets = v::empty<m::Target>(); var v::Vec<m::Signature> signatures = v::empty<m::Signature>();
    var v::Vec<m::Drop> drops=v::empty<m::Drop>();
    var v::Vec<m::Function> functions = v::empty<m::Function>(); var v::Vec<m::Local> locals = v::empty<m::Local>();
    var v::Vec<m::Block> blocks = v::empty<m::Block>(); var v::Vec<m::Operation> operations = v::empty<m::Operation>(); var v::Vec<usize> edges = v::empty<usize>();
    if (!targets_lower::prepare(ref input,ref mut targets,ref mut signatures,ref mut c)) { return .err(c.problem); }
    var usize dropping=0;while(dropping<h::drop_count(ref input)){let d=h::drop_at(ref input,dropping);if(!v::push<m::Drop>(ref mut drops,m::Drop{.abilities=d.abilities,.type=d.type,.target=if(d.nonempty){c.targets[d.function]}else{0},.nonempty=d.nonempty}).is_ok){return .err(m::Problem{.code=3,.start=d.start,.end=d.end});}dropping+=1;}
    if (!bytes::append(ref mut source,h::source_view(ref input)).is_ok) { return .err(m::Problem { .code = 3, .start = 0, .end = 0 }); }
    var usize index = 0;
    while (index < h::function_count(ref input)) {
        let h::Function f = h::function_at(ref input,index); c.function = index + 1;c.live=[0;1153];c.pending=[0;4096];c.keep=0;c.scope=f.scope;
        let usize local_start = v::len<m::Local>(ref locals);c.base=local_start; let usize block_start = v::len<m::Block>(ref blocks);
        var usize id = 1;
        while (id <= h::binding_count(ref input)) {
            let h::Binding b = h::binding_at(ref input,id);
            if (b.owner == f.binding) { let usize local = state::local(ref mut locals,ref mut c,h::type_code(b.type),id,h::binding_code(b.kind) == 2 || h::binding_code(b.kind)==10,h::binding_code(b.kind) == 4 || h::binding_code(b.kind)==10,b.start,b.end); c.locals[id] = local;if(h::binding_code(b.kind)==2 && shapes::needs_drop(h::type_code(b.type))){c.live[id]=1;} }
            id += 1;
        }
        let usize parameter_start = v::len<usize>(ref edges); var usize j = 0;
        while (j < f.parameter_count) { let usize parameter = c.locals[h::edge_at(ref input,f.parameter_start + j)]; if (!state::edge(ref mut edges,parameter,ref mut c)) { return .err(c.problem); } j += 1; }
        let usize entry = state::new_block(ref mut blocks,ref mut c,f.start,f.end); c.current = entry;
        discard block(ref input,ref mut locals,ref mut blocks,ref mut operations,ref mut edges,ref mut payload,ref mut c,f.body,0);
        if (c.problem.code != 0) { return .err(c.problem); }
        if (c.current != 0) {
            if(!lifetime::scope(ref input,ref mut locals,ref mut blocks,ref mut operations,ref mut edges,ref mut c,0,f.start,f.end)){return .err(c.problem);}
            if (h::type_code(h::binding_at(ref input,f.binding).type) != 7) { return .err(m::Problem { .code = 4, .start = f.start, .end = f.end }); }
            var m::Block term = state::block(c.function,f.start,f.end); term.kind = 1;
            if (!state::terminate(ref mut blocks,ref mut c,term)) { return .err(c.problem); }
        }
        if (!v::push<m::Function>(ref mut functions,m::Function { .target = c.targets[f.binding], .entry, .local_start, .local_count = v::len<m::Local>(ref locals) - local_start, .block_start, .block_count = v::len<m::Block>(ref blocks) - block_start, .parameter_start, .parameter_count = f.parameter_count, .start = f.start, .end = f.end, .public = f.public }).is_ok) { return .err(m::Problem { .code = 3, .start = f.start, .end = f.end }); }
        index += 1;
    }
    let m::Program result = switch(m::assemble_resources(source,payload,targets,signatures,functions,locals,blocks,operations,edges,h::module_index(ref input),table,drops)) {
        case .ok(value): { value } case .err(problem): { return .err(problem); }
    };
    let m::Problem problem = verify::check(ref result);
    if (problem.code != 0) { return .err(problem); }
    return .ok(result);
}
