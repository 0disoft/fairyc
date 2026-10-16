module hir_validate_expr_unsafe;
import hir_model as h;
import pointer_types as pointers;
import raw_view_intrinsics as raw;
import storage_intrinsics as storage;
import c_abi_types as casts;
import reference_types as refs;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;

pub fn h::Problem check(ref bytes::Buffer source, ref v::Vec<h::Binding> bindings, ref v::Vec<h::Expression> expressions, ref shapes::Table nominal, h::Expression e){
        let u32 kind=h::expression_code(e.kind);let u32 type=h::type_code(e.type);
        if(kind==30){let u32 code=h::storage_code(e.kind);let u32 element=h::storage_element(e.kind);let usize args=storage::arguments(code);
            if(!shapes::valid_type(nominal,element,false) || storage::result(code,element)!=type || !e.boolean || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.edge_start!=0 || (if(args<3){e.alternative!=0}else{e.alternative==0 || h::type_code(v::at<h::Expression>(expressions,e.alternative-1).type)!=5}) || e.payload_start!=0 || e.payload_end!=0 || h::operator_code(e.operator)!=0
                || !shapes::storage_rule(nominal,bytes::view(source),e.start,e.end,code)
                || (if(args==0){e.left!=0}else{e.left==0 || !(if(code>=6){storage::owner(code,h::type_code(v::at<h::Expression>(expressions,e.left-1).type)) && shapes::storage_owner(nominal,h::type_code(v::at<h::Expression>(expressions,e.left-1).type))}else{h::type_code(v::at<h::Expression>(expressions,e.left-1).type)==storage::argument_type(code,element,0)})})
                || (if(args<2){e.right!=0}else{e.right==0 || h::type_code(v::at<h::Expression>(expressions,e.right-1).type)!=storage::argument_type(code,element,1)})){return h::Problem{.code=5102,.start=e.start,.end=e.end};}
        }
        if(kind==29){
            if((type!=9 && type!=10 && type!=11 && type!=12) || !e.boolean || e.left==0 || e.right==0 || e.alternative==0 || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.payload_start!=0 || e.payload_end!=0 || h::operator_code(e.operator)!=0
                || !raw::owner(h::type_code(v::at<h::Expression>(expressions,e.left-1).type),type) || !raw::data(h::type_code(v::at<h::Expression>(expressions,e.right-1).type),type) || h::type_code(v::at<h::Expression>(expressions,e.alternative-1).type)!=5){return h::Problem{.code=4,.start=e.start,.end=e.end};}
        }
        if(kind==28){
            if(e.left==0 || !pointers::cast_allowed(h::type_code(v::at<h::Expression>(expressions,e.left-1).type),type) || !e.boolean || e.right!=0 || e.alternative!=0 || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.edge_start!=0 || e.payload_start!=0 || e.payload_end!=0 || h::operator_code(e.operator)!=0){return h::Problem{.code=4,.start=e.start,.end=e.end};}
            if(pointers::inner(type)==7 || type==1048601){let child=*v::at<h::Expression>(expressions,e.left-1);if(h::expression_code(child.kind)!=3 && !(h::expression_code(child.kind)==16 && h::expression_code(v::at<h::Expression>(expressions,child.left-1).kind)==3)){return h::Problem{.code=4,.start=e.start,.end=e.end};}}
        }
        if(kind==27 && (!pointers::valid(type) || e.left!=0 || e.right!=0 || e.alternative!=0 || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.edge_start!=0 || e.payload_start!=0 || e.payload_end!=0 || e.boolean || h::operator_code(e.operator)!=0)){return h::Problem{.code=4,.start=e.start,.end=e.end};}
        if(kind==26 && (e.left==0 || !casts::allowed(h::type_code(v::at<h::Expression>(expressions,e.left-1).type),type) || e.right!=0 || e.alternative!=0 || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.payload_start!=0 || e.payload_end!=0 || e.boolean || h::operator_code(e.operator)!=0)){return h::Problem{.code=4,.start=e.start,.end=e.end};}
        if(kind==25 && (e.left==0 || !refs::valid(type) || refs::mutable(type) || !refs::mutable(h::type_code(v::at<h::Expression>(expressions,e.left-1).type)) || refs::inner(h::type_code(v::at<h::Expression>(expressions,e.left-1).type))!=refs::inner(type) || e.right!=0 || e.alternative!=0 || e.binding!=0 || e.edge_count!=0)){return h::Problem{.code=4,.start=e.start,.end=e.end};}
        if(kind==24 && (!shapes::needs_drop(type) || h::binding_code(v::at<h::Binding>(bindings,e.binding-1).kind)==10 || e.left!=0 || e.right!=0 || e.alternative!=0 || e.block!=0 || e.edge_count!=0 || e.payload_start!=0 || e.payload_end!=0 || e.boolean)){return h::Problem{.code=4,.start=e.start,.end=e.end};}
    return h::Problem{.code=0,.start=0,.end=0};
}
