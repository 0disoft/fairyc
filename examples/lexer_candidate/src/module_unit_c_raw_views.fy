module module_unit_c_raw_views;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import c_text as w;
import std::bytes as bytes;

pub fn bool expression(ref ir::Program p,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){let e=ir::node_at(b,id-1).value;let args=proof::raw_arguments(b,e,work);if(e.kind!=6 || e.builtin.kind<9 || e.builtin.kind>12 || !args.valid){s.code=1001;return false;}
    // The owner argument was evaluated in the CFG and remains the view origin;
    // only the already evaluated raw pointer and initialized extent travel in C.
    return scalar::value(out,s,id) && w::text(out,s,b"=(") && aggregate::type(p,e.type,out,s) && w::text(out,s,b"){.fy_data=(uint8_t ") && w::text(out,s,if(e.builtin.kind==11){b"*"}else{b"const *"}) && w::text(out,s,b")") && scalar::value(out,s,args.data) && w::text(out,s,if(e.builtin.kind==12){b",.fy_storage_length="}else{b",.fy_length="}) && scalar::value(out,s,args.length) && w::text(out,s,b"};\n");
}
