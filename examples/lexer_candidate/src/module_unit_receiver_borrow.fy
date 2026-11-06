module module_unit_receiver_borrow;
import module_unit_value_model as m;
import module_types as types;
import syntax as syntax;

// The original expression stays a value. Its checked first-argument pair keeps
// the written owner type and the receiver's expected borrow type separately.
pub fn bool conversion(types::Type expected,types::Type actual,usize actual_id){
    return (expected.kind==7 || expected.kind==8) && expected.first==actual_id && (actual.kind==2 || actual.kind==3);
}
pub fn bool place(m::Expression e,bool mutable){
    return e.kind==2 && e.binding!=0 && e.place.kind==1 && e.place.root!=0 && e.place.borrowable && !e.place.indirect && !e.place.resource_field && (!mutable || e.place.mutable);
}
pub fn bool selected(ref syntax::Parsed source,m::Invocation call,m::Argument argument){
    if(argument.position!=0 || call.kind!=1){return false;}let e=syntax::expression_at(source,call.source_index);
    let callee=syntax::expression_at(source,e.left);return e.kind==6 && callee.kind==8 && callee.left==argument.source_index && syntax::expression_at(source,callee.left).kind==2;
}
pub fn Result<usize,m::Problem> first(ref m::Values body,usize original,ref mut usize work){
    var usize low=0;var usize high=m::call_count(body);
    while(low<high){if(!m::charge(work,1)){return .err(m::fail(m::frame(body),10,0,0,0,0));}let mid=low+(high-low)/2;if(m::call_at(body,mid).source_index<original){low=mid+1;}else{high=mid;}}
    if(low>=m::call_count(body)){return .ok(0);}let call=m::call_at(body,low);return .ok(if(call.source_index==original && call.argument_count!=0){call.argument_start+1}else{0});
}
