module hir_validate_expr_basic;
import hir_model as h;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;

pub fn h::Problem check(ref bytes::Buffer source, ref bytes::Buffer payload, ref v::Vec<h::Function> functions, ref v::Vec<h::Binding> bindings, ref v::Vec<h::Expression> expressions, ref v::Vec<h::Block> blocks, ref v::Vec<usize> edges, ref v::Vec<h::Signature> signatures, ref shapes::Table nominal, h::Expression e, usize i){
    let usize ns=bytes::len(source);
    let usize np=bytes::len(payload);
    let usize nf=v::len<h::Function>(functions);
    let usize nb=v::len<h::Binding>(bindings);
    let usize ne=v::len<h::Expression>(expressions);
    let usize nk=v::len<h::Block>(blocks);
    let usize na=v::len<usize>(edges);
        let u32 kind = h::expression_code(e.kind); let u32 type = h::type_code(e.type);
        let bool is_field = kind == 10 || kind == 11 || kind == 18 || kind == 20 || kind == 21;
        if (e.start > e.end || e.end > ns || !h::represented_type(e.type) || !shapes::valid_type(nominal,type,true) || e.left > ne || e.right > ne || e.alternative > ne || e.block > nk || (!is_field && e.binding > nb) || e.edge_start > na || e.edge_count > na - e.edge_start || e.payload_start > e.payload_end || e.payload_end > np) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        // h::Expression links are postorder and cannot create cycles.
        if (e.left > i || e.right > i || e.alternative > i) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if (h::expression_code(e.kind) == h::expression_code(.Integer) && (e.payload_start == e.payload_end || type < 1 || type > 5)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if (h::expression_code(e.kind) == h::expression_code(.Boolean) && h::type_code(e.type) != h::type_code(.Bool)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if ((kind==3 || kind==24 || kind==6) && e.binding == 0) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if ((h::expression_code(e.kind) == h::expression_code(.Unary) || h::expression_code(e.kind) == h::expression_code(.Binary)) && (e.left == 0 || h::operator_code(e.operator) == h::operator_code(.None))) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if (h::expression_code(e.kind) == h::expression_code(.Binary) && e.right == 0) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if ((h::expression_code(e.kind) == h::expression_code(.Block) || h::expression_code(e.kind) == h::expression_code(.If)) && e.block == 0) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if (h::expression_code(e.kind) == h::expression_code(.If) && (e.left == 0 || h::type_code(v::at<h::Expression>(expressions,e.left - 1).type) != h::type_code(.Bool))) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if (kind == 3 || kind == 24 || kind == 6) {
            let h::Binding b = *v::at<h::Binding>(bindings,e.binding - 1);
            if (h::type_code(e.type) != h::type_code(b.type) || ((kind == 3 || kind == 24) && (h::function_kind(b.kind) || (h::binding_code(b.kind) == h::binding_code(.External) || h::binding_code(b.kind)==11) || h::binding_code(b.kind) == h::binding_code(.Struct) || h::binding_code(b.kind) == 7 || h::binding_code(b.kind)==8)) || (h::expression_code(e.kind) == h::expression_code(.Call) && h::binding_code(b.kind) != h::binding_code(.Function) && h::binding_code(b.kind) != h::binding_code(.External) && h::binding_code(b.kind)!=11)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
            if (h::expression_code(e.kind) == h::expression_code(.Call)) {
                var bool attached=false;var usize owner_index=0;
                while(owner_index<nf){let f=*v::at<h::Function>(functions,owner_index);if(f.start<=e.start && e.end<=f.end){attached=true;if((v::at<h::Binding>(bindings,f.binding-1).effects&b.effects)!=b.effects){return h::Problem{.code=7102,.start=e.start,.end=e.end};}}owner_index+=1;}
                if(!attached){return h::Problem{.code=4,.start=e.start,.end=e.end};}
                if (e.edge_count != b.signature_count) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
                var usize j = 0;
                while (j < e.edge_count) {
                    let usize id = *v::at<usize>(edges,e.edge_start + j);
                    if (id == 0 || id > i || h::type_code(v::at<h::Expression>(expressions,id - 1).type) != h::type_code(v::at<h::Signature>(signatures,b.signature_start + j).type)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; } j += 1;
                }
            }
        }
    return h::Problem{.code=0,.start=0,.end=0};
}
