module hir_expression_lower;
import pointer_types as pointers;
import reference_types as refs;
import hir_model as h;
import tagged_types as tagged;
import hir_state as state;
import hir_slices as slices;
import hir_shape_lower as shape_lower;
import type_check as typing;
import literal_types as literals;
import name_resolution as names;
import syntax as syntax;
import raw_view_intrinsics as raw;
import storage_intrinsics as storage;
import std::bytes as bytes;
import std::vec as v;
import hir_lower as lowering;

pub fn usize expression(ref typing::Typed input, ref v::Vec<h::Binding> bindings, ref v::Vec<state::Link> links, ref mut state::Context c, ref mut v::Vec<h::Expression> expressions, ref mut v::Vec<h::Statement> statements, ref mut v::Vec<h::Block> blocks, ref mut v::Vec<usize> edges, ref mut bytes::Buffer payload, usize id, usize depth) {
    if (id == 0 || c.problem.code != 0) { return 0; }
    let ref syntax::Parsed parsed = names::parsed(typing::resolved(input));
    if (id > syntax::expression_count(parsed)) { return state::fail(c,4,0,0); }
    let syntax::Expression node = syntax::expression_at(parsed,id);
    if (depth >= 64) { return state::fail(c,2,node.start,node.end); }
    if (node.kind == 3) { return expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1); }
    var h::Expression e = state::expression(node.start,node.end);
    let h::Type type = state::type(typing::expression_type(input,id));
    var h::ExpressionKind kind = .Integer; var h::Operator operator = .None;
    if (h::type_code(type) == h::type_code(.Invalid)) { return state::fail(c,4,node.start,node.end); }
    if (node.kind == 1 || (node.kind == 4 && typing::signed_minimum(input,id))) {
        if(pointers::valid(h::type_code(type))){kind=.Null;}else if (literals::valid(h::type_code(type))) {
            kind = .Literal; let ref u8[] cooked = syntax::literal_cooked(parsed,id);
            e.payload_start = bytes::len(payload); let u32 code = literals::decode(payload,cooked);
            if (code != 0) { return state::fail(c,code,node.start,node.end); } e.payload_end = bytes::len(payload);
        } else if (h::type_code(type) == h::type_code(.Bool)) { kind = .Boolean; e.boolean = bytes::equal(syntax::expression_text(parsed,id),b"true"); }
        else {
            kind = .Integer; e.payload_start = bytes::len(payload);
            let usize literal = if (node.kind == 4) { node.left } else { id };
            if (!state::integer(payload,syntax::expression_text(parsed,literal),node.kind == 4,c)) { return 0; }
            e.payload_end = bytes::len(payload);
        }
    } else if (node.kind == 30) {
        kind = .Propagate; e.binding = c.owner;
        e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth+1);
    } else if (node.kind == 2) {
        kind = if(typing::moving(input,id)){.Move}else{.Reference}; e.binding = state::use(links,id);
        if (e.binding == 0) { return state::fail(c,4,node.start,node.end); }
        let h::Binding b = *v::at<h::Binding>(bindings,e.binding - 1);
        if (!state::visible(c,b) || h::type_code(b.type) != h::type_code(type) || h::binding_code(b.kind) == h::binding_code(.Function) || h::binding_code(b.kind) == h::binding_code(.External)) { return state::fail(c,4,node.start,node.end); }
    } else if (node.kind == 4 || node.kind == 5) {
        kind = if (node.kind == 4) { .Unary } else { .Binary }; operator = state::operator(syntax::expression_text(parsed,id));
        if (h::operator_code(operator) == h::operator_code(.None)) { return state::fail(c,4,node.start,node.end); }
        e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
        if (node.kind == 5) { e.right = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.right,depth + 1); }
    } else if (node.kind == 6 && tagged::builtin(h::type_code(type)) && syntax::expression_at(parsed,node.left).kind==28) {
        kind=.Variant;e.binding=shape_lower::field_id(parsed,typing::field_target(input,id));e.edge_count=1;e.edge_start=state::reserve_edges(edges,1,c);
        let usize arg=syntax::call_argument(parsed,id,0);let a=syntax::expression_at(parsed,arg);
        let usize value=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,arg,depth+1);
        let bool first=typing::field_target(input,id)==tagged::member(parsed,h::type_code(type),true);
        let h::Expression initializer=h::Expression{.kind=.Initializer,.type=state::type(typing::expression_type(input,arg)),.operator=.None,.start=a.start,.end=a.end,.left=value,.right=0,.alternative=0,.block=0,.binding=shape_lower::field_id(parsed,tagged::payload_for(parsed,h::type_code(type),first)),.edge_start=0,.edge_count=0,.payload_start=0,.payload_end=0,.boolean=false};
        let usize child=state::push_expression(expressions,initializer,c);if(child==0){return 0;}state::set_edge(edges,e.edge_start,child);
    } else if (storage::call(parsed,id)!=0) {
        let u32 code=storage::call(parsed,id);kind=.Storage{.code,.element=storage::element(parsed,id)};e.boolean=true;
        let usize count=storage::arguments(code);
        if(count!=0){e.left=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::call_argument(parsed,id,0),depth+1);}
        if(count>=2){e.right=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::call_argument(parsed,id,1),depth+1);}
        if(count==3){e.alternative=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::call_argument(parsed,id,2),depth+1);}
    } else if (raw::call(parsed,id)!=0) {
        kind=.RawView;e.boolean=true;
        e.left=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::call_argument(parsed,id,0),depth+1);
        e.right=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::call_argument(parsed,id,1),depth+1);
        e.alternative=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::call_argument(parsed,id,2),depth+1);
    } else if (node.kind == 6) {
        kind = .Call; e.binding = state::use(links,node.left);
        if (e.binding == 0) { return state::fail(c,4,node.start,node.end); }
        let h::Binding callee = *v::at<h::Binding>(bindings,e.binding - 1);e.boolean=callee.unsafe_call;
        if ((h::binding_code(callee.kind) != h::binding_code(.Function) && h::binding_code(callee.kind) != h::binding_code(.External) && h::binding_code(callee.kind)!=11) || h::type_code(callee.type) != h::type_code(type)) { return state::fail(c,4,node.start,node.end); }
        e.edge_count = syntax::call_argument_count(parsed,id);
        if (callee.signature_count != e.edge_count) { return state::fail(c,4,node.start,node.end); }
        e.edge_start = state::reserve_edges(edges,e.edge_count,c); if (c.problem.code != 0) { return 0; }
        var usize i = 0;
        while (i < e.edge_count) {
            let usize argument = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,syntax::call_argument(parsed,id,i),depth + 1);
            if (argument == 0) { return 0; } state::set_edge(edges,e.edge_start + i,argument); i += 1;
        }
    } else if(node.kind==23){kind=if(pointers::valid(h::type_code(type))){.PointerCast}else{.Cast};e.boolean=pointers::valid(h::type_code(type));e.left=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth+1);
    } else if (node.kind == 24 || node.kind == 29) {
        kind = .Array; e.boolean = node.kind == 29;
        e.edge_count = if (e.boolean) { 1 } else { syntax::array_element_count(parsed,id) };
        e.edge_start = state::reserve_edges(edges,e.edge_count,c); if (c.problem.code != 0) { return 0; }
        var usize i = 0; while (i < e.edge_count) {
            let usize child = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,if (e.boolean) { node.left } else { syntax::array_element(parsed,id,i) },depth + 1);
            if (child == 0) { return 0; } state::set_edge(edges,e.edge_start + i,child); i += 1;
        }
    } else if (node.kind == 21 || node.kind == 22 || node.kind==26) {
        let bool borrowed=node.kind!=26;
        let selected=if(borrowed){syntax::expression_at(parsed,node.left)}else{node};
        if(borrowed && selected.kind!=26){
            kind=.Borrow;e.boolean=node.kind==22;e.left=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth+1);
        }else{
            kind=.Slice;
            let usize owner=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,selected.left,depth+1);
            var syntax::Expression address=node;address.end=syntax::expression_at(parsed,selected.left).end;
            let usize view=if(borrowed){slices::borrow(address,type,owner,node.kind==22,expressions,c)}else{owner};
            let bounds=syntax::expression_at(parsed,selected.right);
            let usize low=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,bounds.left,depth+1);
            let usize high=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,bounds.right,depth+1);
            e=slices::bounds(node,type,view,low,high,expressions,payload,c);
        }
    } else if (node.kind == 9) {
        kind = .Index; e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
        e.right = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.right,depth + 1);
    } else if (node.kind == 8 && typing::result_is_ok(input,id)) {
        kind = .IsOk; e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
    } else if (node.kind == 8 && typing::field_target(input,id) == 0) {
        kind = .Length; e.boolean = bytes::equal(syntax::expression_text(parsed,id),b"is_empty"); e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
    } else if (node.kind == 8) {
        kind = .Field; e.binding = shape_lower::field_id(parsed,typing::field_target(input,id));
        e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
    } else if (node.kind == 10) {
        let bool variant=syntax::expression_at(parsed,node.left).kind==28;
        kind = if(variant){.Variant}else{.Construct}; e.binding = if(variant){shape_lower::field_id(parsed,typing::field_target(input,id))}else{state::use(links,node.left)};
        if (e.binding == 0 || !variant && h::binding_code(v::at<h::Binding>(bindings,e.binding - 1).kind) != 6 && h::binding_code(v::at<h::Binding>(bindings,e.binding - 1).kind) != 8) { return state::fail(c,4,node.start,node.end); }
        e.edge_count = syntax::record_field_count(parsed,id);
        e.edge_start = state::reserve_edges(edges,e.edge_count,c); if (c.problem.code != 0) { return 0; }
        var usize i = 0;
        while (i < e.edge_count) {
            let usize entry = syntax::record_field(parsed,id,i); let syntax::Expression f = syntax::expression_at(parsed,entry);
            let usize value = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,f.left,depth + 1);
            let h::Expression initializer = h::Expression { .kind = .Initializer, .type = state::type(typing::expression_type(input,entry)), .operator = .None, .start = f.start, .end = f.end, .left = value, .right = 0, .alternative = 0, .block = 0, .binding = shape_lower::field_id(parsed,typing::field_target(input,entry)), .edge_start = 0, .edge_count = 0, .payload_start = 0, .payload_end = 0, .boolean = false };
            let usize child = state::push_expression(expressions,initializer,c);
            if (child == 0) { return 0; } state::set_edge(edges,e.edge_start + i,child); i += 1;
        }
    } else if (node.kind == 28) {
        kind=.Variant;e.binding=shape_lower::field_id(parsed,typing::field_target(input,id));
    } else if (node.kind == 15) {
        kind=.Switch;e.left=expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth+1);
        var usize link=node.right;while(link!=0){e.edge_count+=1;link=syntax::expression_at(parsed,link).right;}
        e.edge_start=state::reserve_edges(edges,e.edge_count,c);if(c.problem.code!=0){return 0;}
        link=node.right;var usize remaining=e.edge_count;
        while(link!=0){let usize arm_id=syntax::expression_at(parsed,link).left;let arm=syntax::expression_at(parsed,arm_id);
            let pattern=syntax::expression_at(parsed,arm.left);let usize count=if(pattern.kind==32){syntax::pattern_field_count(parsed,arm.left)}else if(pattern.left!=0){1}else{0};
            let usize captures=state::reserve_edges(edges,count,c);if(c.problem.code!=0){return 0;}
            var usize n=0;while(n<count){let usize field_id=if(pattern.kind==32){syntax::pattern_field(parsed,arm.left,n)}else{pattern.left};let f=syntax::expression_at(parsed,field_id);let usize name=if(pattern.kind==32){f.left}else{field_id};let usize binding=state::use(links,name);
                let usize start=if(binding==0){0}else{state::reserve_edges(edges,1,c)};if(binding!=0){state::set_edge(edges,start,binding);}
                let h::Expression capture=h::Expression{.kind=.Capture,.type=state::type(typing::expression_type(input,name)),.operator=.None,.start=f.start,.end=f.end,.left=0,.right=0,.alternative=0,.block=0,.binding=shape_lower::field_id(parsed,typing::field_target(input,field_id)),.edge_start=start,.edge_count=if(binding==0){0}else{1},.payload_start=0,.payload_end=0,.boolean=false};
                let usize id=state::push_expression(expressions,capture,c);if(id==0){return 0;}state::set_edge(edges,captures+n,id);n+=1;
            }
            let usize body=lowering::block(input,bindings,links,c,expressions,statements,blocks,edges,payload,arm.right,depth+1);
            let h::Expression child=h::Expression{.kind=.Arm,.type=state::type(typing::expression_type(input,arm_id)),.operator=.None,.start=arm.start,.end=arm.end,.block=body,.binding=shape_lower::field_id(parsed,typing::field_target(input,arm_id)),.left=0,.right=0,.alternative=0,.edge_start=captures,.edge_count=count,.payload_start=0,.payload_end=0,.boolean=false};
            let usize lowered=state::push_expression(expressions,child,c);if(lowered==0){return 0;}remaining-=1;state::set_edge(edges,e.edge_start+remaining,lowered);link=syntax::expression_at(parsed,link).right;
        }
    } else if ((node.kind==12 || node.kind==31)) {
        kind = .Block; e.block = lowering::block(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
    } else if (node.kind == 13) {
        kind = .If; e.left = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,node.left,depth + 1);
        let syntax::Expression branches = syntax::expression_at(parsed,node.right);
        e.block = lowering::block(input,bindings,links,c,expressions,statements,blocks,edges,payload,branches.left,depth + 1);
        e.alternative = expression(input,bindings,links,c,expressions,statements,blocks,edges,payload,branches.right,depth + 1);
    } else { return state::fail(c,4,node.start,node.end); }
    let usize retained=state::push_expression(expressions,h::Expression { .kind, .type, .operator, .start = e.start, .end = e.end, .left = e.left, .right = e.right, .alternative = e.alternative, .block = e.block, .binding = e.binding, .edge_start = e.edge_start, .edge_count = e.edge_count, .payload_start = e.payload_start, .payload_end = e.payload_end, .boolean = e.boolean },c);
    if(retained==0 || !typing::shared_argument(input,id)){return retained;}
    return state::push_expression(expressions,h::Expression{.kind=.Reborrow,.type=state::type(refs::make(refs::inner(h::type_code(type)),false)),.operator=.None,.start=e.start,.end=e.end,.left=retained,.right=0,.alternative=0,.block=0,.binding=0,.edge_start=0,.edge_count=0,.payload_start=0,.payload_end=0,.boolean=false},c);
}
