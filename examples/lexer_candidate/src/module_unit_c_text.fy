module module_unit_c_text;
import module_unit_body_ir_model as ir;
import module_unit_c_aggregate as aggregate;
import c_text as w;
import std::bytes as bytes;

// The authenticated storage entry points already copy via void*, so the
// private String/CString carrier does not alias a runtime C struct lvalue.
pub fn bool definitions(ref ir::Program p,usize type,usize layout,ref mut bytes::Buffer out,ref mut w::State s){if(type==0){return true;}let shape=ir::layout_at(p,layout-1).value;
    if(!w::text(out,s,b"_Static_assert(sizeof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b")==32&&_Alignof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b")==8,\"SDK owned text storage layout\");\n")){return false;}
    var usize i=0;while(i<4){if(!w::text(out,s,b"_Static_assert(offsetof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b",") || !aggregate::field(out,s,shape.field_start+i+1) || !w::text(out,s,b")==") || !w::number(out,s,i*8) || !w::text(out,s,b",\"SDK owned text storage offset\");\n")){return false;}i+=1;}return true;
}
