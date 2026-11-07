module hir_closure_values;
import hir_model as h;
import std::vec as v;

fn h::Problem fail(h::Expression e){return h::Problem{.code=4,.start=e.start,.end=e.end};}
pub fn h::Problem check(ref v::Vec<h::Binding> bindings,ref v::Vec<h::Expression> expressions,ref v::Vec<usize> edges,ref v::Vec<h::Signature> signatures,h::Expression e,usize at){
    let kind=h::expression_code(e.kind);
    if(e.alternative!=0 || e.block!=0 || h::operator_code(e.operator)!=0 || e.boolean || e.payload_start!=e.payload_end){return fail(e);}
    if(kind==31){
        if(h::type_code(e.type)!=13 || e.binding==0 || e.binding>v::len<h::Binding>(bindings) || e.left!=0 || e.right!=0 || e.edge_count>4){return fail(e);}
        let b=*v::at<h::Binding>(bindings,e.binding-1);
        if(h::binding_code(b.kind)!=1 || b.owner!=0 || b.scope!=0 || b.unsafe_call || b.effects!=0 || h::type_code(b.type)!=1 || b.signature_count!=e.edge_count+1 || b.signature_start>v::len<h::Signature>(signatures) || b.signature_count>v::len<h::Signature>(signatures)-b.signature_start){return fail(e);}
        var usize i=0;while(i<b.signature_count){if(h::type_code(v::at<h::Signature>(signatures,b.signature_start+i).type)!=1){return fail(e);}i+=1;}
        i=0;while(i<e.edge_count){let child=*v::at<usize>(edges,e.edge_start+i);if(child==0 || child>at || h::type_code(v::at<h::Expression>(expressions,child-1).type)!=1){return fail(e);}i+=1;}
    }else if(kind==32){
        if(h::type_code(e.type)!=1 || e.binding!=0 || e.left==0 || e.right==0 || e.edge_count!=0){return fail(e);}
        let callable=*v::at<h::Expression>(expressions,e.left-1);let argument=*v::at<h::Expression>(expressions,e.right-1);
        if(h::expression_code(callable.kind)!=3 || h::type_code(callable.type)!=13 || h::type_code(argument.type)!=1 || callable.binding==0 || callable.binding>v::len<h::Binding>(bindings)){return fail(e);}
        let b=*v::at<h::Binding>(bindings,callable.binding-1);if(h::binding_code(b.kind)!=2 && h::binding_code(b.kind)!=3){return fail(e);}
    }else{return fail(e);}
    return h::Problem{.code=0,.start=0,.end=0};
}
