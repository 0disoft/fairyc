module mir_lower;
import mir_value_lower as values;
import mir_extended_lower as extended;
import mir_view_lower as view;
import mir_statement_lower as statements;
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

pub fn bool prepare_place(ref h::Program input, ref mut v::Vec<places::Step> path, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize depth) {
    var usize at=v::len<places::Step>(path);
    while(at!=0){at-=1;let places::Step step=*v::at<places::Step>(path,at);let e=h::expression_at(input,step.expression);
        if(h::expression_code(e.kind)==13){let usize raw=expression(input,locals,blocks,operations,edges,payload,c,e.right,depth+1);if(c.current==0 || c.problem.code!=0){return false;}
            let usize index=array::index(locals,blocks,operations,c,h::type_code(h::expression_at(input,e.left).type),raw,e.start,e.end);if(index==0){return false;}
            let ref mut places::Step slot=v::at_mut<places::Step>(path,at);slot.index=index;
        }
    }return true;
}

// Project addresses into original storage, never into by-value field snapshots.
pub fn usize address(ref h::Program p,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<usize> edges,ref mut bytes::Buffer payload,ref mut state::Context c,usize id,bool mutable,usize depth){
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
pub fn usize expression(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    if (id == 0 || c.current == 0 || c.problem.code != 0) { return 0; }
    let h::Expression e = h::expression_at(input,id); let u32 kind = h::expression_code(e.kind); let u32 type = h::type_code(e.type); let u32 operator = h::operator_code(e.operator);
    if (depth >= 64) { return state::fail(c,2,e.start,e.end); }
    // Keep aggregate temporaries on the selected lane, not on every recursive
    // expression frame. Helpers preserve the original source-depth accounting.
    if(kind>=18 && kind!=25){return extended::extended_expression(input,locals,blocks,operations,edges,payload,c,id,e,kind,type,operator,depth);}
    if(kind<=3 || kind==15 || kind==16 || kind==17 || kind==25 ||
       kind==4 && operator==5 && refs::valid(h::type_code(h::expression_at(input,e.left).type)) ||
       (kind==13 || kind==14) && h::type_code(h::expression_at(input,e.left).type)<256){return view::view_expression(input,locals,blocks,operations,edges,payload,c,id,e,kind,type,operator,depth);}
    if(kind==4 || kind==5 || kind==9 || kind==10 || kind==12 || kind==13 || kind==14){return values::aggregate_expression(input,locals,blocks,operations,edges,payload,c,id,e,kind,type,operator,depth);}
    return values::call_expression(input,locals,blocks,operations,edges,payload,c,id,e,kind,type,operator,depth);
}
// Large ownership snapshots live only on actual branch frames, not on every
// recursive scalar/field node. Helpers retain the same source-depth accounting.




pub fn usize block(ref h::Program input, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> operations, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, ref mut state::Context c, usize id, usize depth) {
    let h::Block b = h::block_at(input,id);
    if (depth >= 64) { return state::fail(c,2,b.start,b.end); }
    let usize deferred_base = c.defer_count; let usize capture_base = c.capture_count;let usize previous_scope=c.scope;c.scope=b.scope;
    var usize i = 0;
    while (i < b.edge_count && c.current != 0 && c.problem.code == 0) {
        if (!statements::statement(input,locals,blocks,operations,edges,payload,c,h::edge_at(input,b.edge_start + i),depth + 1)) { return 0; } i += 1;
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
