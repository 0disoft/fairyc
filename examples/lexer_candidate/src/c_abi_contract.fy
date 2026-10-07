module c_abi_contract;
import c_abi_types as c_types;
import type_state as ids;
import mir_model as m;
import hir_shapes as shapes;
import c_text as w;
import std::bytes as bytes;
fn bool check(u32 type,usize size,ref mut bytes::Buffer out,ref mut w::State s){
    return w::text(out,s,b"_Static_assert(sizeof(") && w::text(out,s,c_types::spelling(type)) && w::text(out,s,b") == ") && w::number(out,s,size) && w::text(out,s,b" && _Alignof(") && w::text(out,s,c_types::spelling(type)) && w::text(out,s,b") == ") && w::number(out,s,size) && w::text(out,s,b",\"candidate target C scalar size/alignment\");\n");
}
pub fn bool emit(ref m::Program p,ref mut bytes::Buffer out,ref mut w::State s){
    var bool[16] used=[false;16];var usize i=1;var bool needed=false;
    while(i<=m::local_count(p)){let u32 t=m::local_at(p,i).type;if(c_types::valid(t)){used[(t-128) as usize]=true;needed=true;}i+=1;}
    i=1;while(i<=m::target_count(p)){let u32 t=m::target_at(p,i).result;if(c_types::valid(t)){used[(t-128) as usize]=true;needed=true;}i+=1;}
    i=0;while(i<m::signature_count(p)){let u32 t=m::signature_at(p,i);if(c_types::valid(t)){used[(t-128) as usize]=true;needed=true;}i+=1;}
    i=1;while(i<=shapes::field_count(m::nominal(p))){let u32 t=shapes::field_at(m::nominal(p),i).type;if(c_types::valid(t)){used[(t-128) as usize]=true;needed=true;}i+=1;}
    if(!needed){return true;}if(!w::text(out,s,b"#include <stddef.h>\n")){return false;}
    i=0;while(i<16){let u32 t=128+ids::small_id(i);if(used[i]){
        let usize size=if(t==128 || t==129 || t==135 || t==139){1}else if(t==136 || t==143){2}else if(t==130 || t==134 || t==137 || t==142){8}else{4};
        if(t==133 || t==141){if(!w::text(out,s,b"#if defined(_WIN32)\n") || !check(t,4,out,s) || !w::text(out,s,b"#elif defined(__linux__)\n") || !check(t,8,out,s) || !w::text(out,s,b"#else\n#error unsupported candidate C scalar target\n#endif\n")){return false;}}
        else if(!check(t,size,out,s)){return false;}
        if(t!=128 && t!=130 && t!=131){let bool signed=t==129 || t==132 || t==133 || t==134 || t==135 || t==136;
            if(!w::text(out,s,b"_Static_assert(((") || !w::text(out,s,c_types::spelling(t)) || !w::text(out,s,b")-1) ") || !w::text(out,s,if(signed){b"<"}else{b">"}) || !w::text(out,s,b" 0,\"candidate target C scalar signedness\");\n")){return false;}}
        if(t==132 && !w::text(out,s,b"_Static_assert(INT_MIN==INT32_MIN && INT_MAX==INT32_MAX,\"candidate c.int signed 32-bit conversion\");\n")){return false;}
    }i+=1;}return true;
}
