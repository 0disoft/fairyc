module module_unit_c_storage;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_storage as proof;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import c_text as w;
import std::bytes as bytes;

pub fn bool expression(ref ir::Program p,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){let e=ir::node_at(b,id-1).value;let code=e.builtin.kind;let args=proof::arguments(b,e,work);if(!args.valid || code<101 || code>107 || e.builtin.element==0){s.code=1001;return false;}
    if(code<=102){return scalar::value(out,s,id) && w::text(out,s,if(code==101){b"=sizeof("}else{b"=_Alignof("}) && aggregate::type(p,e.builtin.element,out,s) && w::text(out,s,b");\n");}
    if(code==103 || code==104){if(!w::text(out,s,b"if(!") || !scalar::value(out,s,args.first) || !w::text(out,s,b")fy_runtime_trap(7);\nfy_unit_copy(")){return false;}return (if(code==103){w::text(out,s,b"&") && scalar::value(out,s,id) && w::text(out,s,b",") && scalar::value(out,s,args.first) && w::text(out,s,b",sizeof ") && scalar::value(out,s,id)}else{scalar::value(out,s,args.first) && w::text(out,s,b",&") && scalar::value(out,s,args.second) && w::text(out,s,b",sizeof ") && scalar::value(out,s,args.second)}) && w::text(out,s,b");\n");}
    if(code==105){return w::text(out,s,b"if(") && scalar::value(out,s,args.second) && w::text(out,s,b">SIZE_MAX/sizeof(") && aggregate::type(p,e.builtin.element,out,s) && w::text(out,s,b"))fy_runtime_trap(1);\n") && scalar::value(out,s,id) && w::text(out,s,b"=(void *)(") && scalar::value(out,s,args.first) && w::text(out,s,b"?(uint8_t *)") && scalar::value(out,s,args.first) && w::text(out,s,b"+") && scalar::value(out,s,args.second) && w::text(out,s,b"*sizeof(") && aggregate::type(p,e.builtin.element,out,s) && w::text(out,s,b"):NULL);\n");}
    if(!scalar::value(out,s,id) || !w::text(out,s,b"=(") || !aggregate::type(p,e.type,out,s) || !w::text(out,s,if(e.builtin.slice){b"){.fy_data=("}else{b")"})){return false;}
    if(e.builtin.slice){if(!aggregate::type(p,e.builtin.element,out,s) || !w::text(out,s,if(code==107){b" *)"}else{b" const *)"})){return false;}}
    return scalar::value(out,s,args.second) && ( !e.builtin.slice || w::text(out,s,b",.fy_length=") && scalar::value(out,s,args.third) && w::text(out,s,b"}")) && w::text(out,s,b";\n");
}
