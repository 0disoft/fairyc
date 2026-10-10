module module_unit_c_buffer;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_aggregate as aggregate;
import c_text as w;
import std::bytes as bytes;

pub fn bool definitions(ref ir::Program p,usize type,usize layout,ref mut bytes::Buffer out,ref mut w::State s){if(type==0){return true;}let shape=ir::layout_at(p,layout-1).value;
    if(!w::text(out,s,b"struct fy_std_system_heap_block_v0{void *data;size_t size;size_t alignment;};\nstruct fy_std_buffer_v0{struct fy_std_system_heap_block_v0 block;size_t length;};\n_Static_assert(sizeof(struct fy_std_buffer_v0)==32&&_Alignof(struct fy_std_buffer_v0)==8,\"runtime Buffer wire layout\");\n_Static_assert(sizeof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b")==sizeof(struct fy_std_buffer_v0)&&_Alignof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b")==_Alignof(struct fy_std_buffer_v0),\"SDK Buffer storage layout\");\n")){return false;}var usize i=0;while(i<4){if(!w::text(out,s,b"_Static_assert(offsetof(") || !aggregate::type(p,type,out,s) || !w::text(out,s,b",") || !aggregate::field(out,s,shape.field_start+i+1) || !w::text(out,s,b")==") || !w::number(out,s,i*8) || !w::text(out,s,b",\"SDK Buffer storage offset\");\n")){return false;}i+=1;}return true;
}
pub fn bool signature(usize code,ref u8[] symbol,usize id,ref mut bytes::Buffer out,ref mut w::State s){
    let init=code==9;let append=code==11;let unary=code==14 || code==15;
    if(!w::text(out,s,b"extern uint32_t ") || !w::text(out,s,symbol) || !w::text(out,s,if(init){b"(size_t,struct fy_std_buffer_v0 *);\n"}else if(append){b"(struct fy_std_buffer_v0 *,uint8_t const *,size_t);\n"}else if(unary){b"(struct fy_std_buffer_v0 *);\n"}else{b"(struct fy_std_buffer_v0 *,size_t);\n"}) || !w::text(out,s,b"static uint32_t fy_unit_sdk_call_") || !w::number(out,s,id) || !w::text(out,s,if(init){b"(uintptr_t a0,void *a1){"}else if(append){b"(void *a0,uint8_t const *a1,uintptr_t a2){"}else if(unary){b"(void *a0){"}else{b"(void *a0,uintptr_t a1){"})){return false;}
    let owner=if(init){b"a1"}else{b"a0"};if(!w::text(out,s,b"if(!") || !w::text(out,s,owner) || !w::text(out,s,b")return ") || !w::text(out,s,symbol) || !w::text(out,s,if(init){b"(a0,NULL);"}else if(append){b"(NULL,a1,a2);"}else if(unary){b"(NULL);"}else{b"(NULL,a1);"}) || !w::text(out,s,b"struct fy_std_buffer_v0 wire;")){return false;}
    if(!init && (!w::text(out,s,b"fy_unit_copy(&wire,a0,sizeof wire);"))){return false;}
    return w::text(out,s,b"uint32_t status=") && w::text(out,s,symbol) && w::text(out,s,if(init){b"(a0,&wire);if(!status)"}else if(append){b"(&wire,a1,a2);"}else if(unary){b"(&wire);"}else{b"(&wire,a1);"}) && w::text(out,s,b"fy_unit_copy(") && w::text(out,s,owner) && w::text(out,s,b",&wire,sizeof wire);return status;}\n");
}
