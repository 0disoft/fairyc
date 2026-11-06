module module_unit_body_ir_receiver;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as m;
import module_unit_receiver_borrow as receiver;

// Recheck the exact call/argument/local relationship after linking or decoding.
// A mere owner/ref type pair cannot authorize ordinary arguments or FFI casts.
pub fn bool implicit(ref clean::Program cleaned,ref ir::Body b,m::Argument a){
    let types=clean::type_count(cleaned);
    if(!a.checked || a.shared || a.position!=0 || a.invocation==0 || a.invocation>ir::call_count(b) || a.expected==0 || a.expected>types || a.actual==0 || a.actual>types || a.expression==0 || a.expression>ir::node_count(b)){return false;}
    let expected=clean::type_at(cleaned,a.expected);let actual=clean::type_at(cleaned,a.actual);
    if(!receiver::conversion(expected,actual,a.actual)){return false;}let call=ir::call_at(b,a.invocation-1);
    if(call.kind!=1 || call.expression==0 || call.expression>ir::node_count(b)){return false;}
    let e=ir::node_at(b,call.expression-1).value;let callee=ir::node(b,e.left);if(e.kind!=6 || e.builtin.kind!=0 || callee==0){return false;}
    let method=ir::node_at(b,callee-1).value;let owner=ir::node_at(b,a.expression-1).value;
    return method.kind==8 && method.left==a.source_index && owner.source_index==a.source_index && owner.type==a.actual && owner.state==1 && receiver::place(owner,expected.kind==8);
}
