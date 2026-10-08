module hir_validate_expr_values;
import hir_model as h;
import scalar_views as views;
import reference_types as refs;
import array_types as arrays;
import resource_identity as resources;
import literal_types as literals;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;
import hir_validate_enums as enum_check;

pub fn h::Problem check(ref bytes::Buffer payload, ref v::Vec<h::Binding> bindings, ref v::Vec<h::Expression> expressions, ref v::Vec<h::Statement> statements, ref v::Vec<h::Block> blocks, ref v::Vec<usize> edges, ref shapes::Table nominal, h::Expression e, usize i){
        let u32 kind=h::expression_code(e.kind);let u32 type=h::type_code(e.type);
        if (kind == 12) {
            if (!arrays::valid(type) || e.edge_count != if (e.boolean) { 1 } else { arrays::length(type) } || e.left != 0 || e.right != 0 || e.binding != 0) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
            var usize j = 0; while (j < e.edge_count) { let usize id = *v::at<usize>(edges,e.edge_start + j); if (id == 0 || id > i || h::type_code(v::at<h::Expression>(expressions,id - 1).type) != shapes::array_element(nominal,type)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; } j += 1; }
        }
        if (kind == 16 && !refs::valid(type) && ((!views::valid(type)) || e.boolean != views::mutable(type) || e.left == 0 || (h::expression_code(v::at<h::Expression>(expressions,e.left - 1).kind) != 3 && h::expression_code(v::at<h::Expression>(expressions,e.left - 1).kind) != 10) || !arrays::valid(h::type_code(v::at<h::Expression>(expressions,e.left - 1).type)) || shapes::array_element(nominal,h::type_code(v::at<h::Expression>(expressions,e.left - 1).type)) != views::element(type) || e.right != 0 || e.alternative != 0 || e.edge_count != 0 || e.binding != 0)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if(kind==16 && refs::valid(type)){if(e.left==0 || (h::expression_code(v::at<h::Expression>(expressions,e.left-1).kind)!=3 && h::expression_code(v::at<h::Expression>(expressions,e.left-1).kind)!=10 && h::expression_code(v::at<h::Expression>(expressions,e.left-1).kind)!=13 && !(h::expression_code(v::at<h::Expression>(expressions,e.left-1).kind)==4 && h::operator_code(v::at<h::Expression>(expressions,e.left-1).operator)==5)) || h::type_code(v::at<h::Expression>(expressions,e.left-1).type)!=refs::inner(type) || e.boolean!=refs::mutable(type) || e.right!=0 || e.alternative!=0 || e.binding!=0 || e.edge_count!=0){return h::Problem{.code=4,.start=e.start,.end=e.end};}}
        if(kind==16 && refs::mutable(type)){var usize root=e.left;var usize steps=0;while(steps<64 && (h::expression_code(v::at<h::Expression>(expressions,root-1).kind)==10 || h::expression_code(v::at<h::Expression>(expressions,root-1).kind)==13 || h::expression_code(v::at<h::Expression>(expressions,root-1).kind)==4 && h::operator_code(v::at<h::Expression>(expressions,root-1).operator)==5)){root=v::at<h::Expression>(expressions,root-1).left;steps+=1;}let base=*v::at<h::Expression>(expressions,root-1);if(h::expression_code(base.kind)!=3 || h::binding_code(v::at<h::Binding>(bindings,base.binding-1).kind)!=4 && h::binding_code(v::at<h::Binding>(bindings,base.binding-1).kind)!=10 && !refs::mutable(h::type_code(base.type))){return h::Problem{.code=4,.start=e.start,.end=e.end};}}
        if(kind==16 && (views::valid(type))){var usize root=e.left;var usize steps=0;
            while(steps<64 && h::expression_code(v::at<h::Expression>(expressions,root-1).kind)==10){root=v::at<h::Expression>(expressions,root-1).left;steps+=1;}
            let base=*v::at<h::Expression>(expressions,root-1);
            if(h::expression_code(base.kind)!=3 || views::mutable(type) && (h::binding_code(v::at<h::Binding>(bindings,base.binding-1).kind)!=4 || resources::valid(h::type_code(base.type)))){return h::Problem{.code=4,.start=e.start,.end=e.end};}
        }
        if (kind == 17 && ((!views::valid(type)) || e.left == 0 || e.right == 0 || e.alternative == 0 || h::type_code(v::at<h::Expression>(expressions,e.left - 1).type) != type || h::type_code(v::at<h::Expression>(expressions,e.right - 1).type) != 5 || h::type_code(v::at<h::Expression>(expressions,e.alternative - 1).type) != 5 || e.binding != 0 || e.edge_count != 0)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if (kind == 15 && (!literals::valid(type) || type == 9 && !literals::text(bytes::view(payload)[e.payload_start..e.payload_end]) || type==12 && !literals::cstring(bytes::view(payload)[e.payload_start..e.payload_end]) || e.left != 0 || e.right != 0 || e.binding != 0 || e.edge_count != 0 || e.block != 0 || e.alternative != 0 || e.boolean || h::operator_code(e.operator) != 0)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        if (kind == 13 || kind == 14) {
            if (e.left == 0 || !(arrays::valid(h::type_code(v::at<h::Expression>(expressions,e.left - 1).type)) || literals::valid(h::type_code(v::at<h::Expression>(expressions,e.left - 1).type)) || views::valid(h::type_code(v::at<h::Expression>(expressions,e.left - 1).type))) || e.binding != 0 || e.edge_count != 0) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
            let u32 owner = h::type_code(v::at<h::Expression>(expressions,e.left - 1).type);if(owner==12){return h::Problem{.code=4,.start=e.start,.end=e.end};}
            if (kind == 13 && (e.right == 0 || h::type_code(v::at<h::Expression>(expressions,e.right - 1).type) != 5 || type != if (views::valid(owner)) { views::element(owner) } else { shapes::array_element(nominal,owner) } || owner == 9 || e.boolean) || kind == 14 && (e.right != 0 || type != if (e.boolean) { 6 } else { 5 } || e.boolean && !literals::valid(owner) && !views::valid(owner))) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        }
        if (kind == 10 || kind == 11) {
            if (e.binding == 0 || e.binding > shapes::field_count(nominal) || e.left == 0) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
            let shapes::Field f = shapes::field_at(nominal,e.binding);
            let u32 left_type = h::type_code(v::at<h::Expression>(expressions,e.left - 1).type);
            if (arrays::valid(f.owner) || type != f.type || (kind == 10 && (if(refs::valid(left_type)){refs::inner(left_type)}else{left_type}) != f.owner) || (kind == 11 && left_type != f.type)) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
        }
        if (kind == 9) {
            let usize shape = shapes::find(nominal,type);
            if (shape == 0 || arrays::valid(type) || e.binding == 0) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
            let shapes::Shape s = shapes::at(nominal,shape - 1);
            if (e.binding != s.binding || h::binding_code(v::at<h::Binding>(bindings,e.binding - 1).kind) != if(resources::valid(type)){8}else{6} || e.edge_count != s.field_count) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
            var u8[1024] seen = [0;1024]; var usize j = 0;
            while (j < e.edge_count) {
                let usize id = *v::at<usize>(edges,e.edge_start + j);
                if (id == 0 || id > i) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
                let h::Expression f = *v::at<h::Expression>(expressions,id - 1);
                if (h::expression_code(f.kind) != 11 || f.binding <= s.field_start || f.binding > s.field_start + s.field_count || seen[f.binding - 1] != 0) { return h::Problem { .code = 4, .start = e.start, .end = e.end }; }
                seen[f.binding - 1] = 1; j += 1;
            }
        }
        if(!enum_check::enum_expression(nominal,expressions,blocks,statements,bindings,edges,e,i)){return h::Problem{.code=4,.start=e.start,.end=e.end};}
    return h::Problem{.code=0,.start=0,.end=0};
}
