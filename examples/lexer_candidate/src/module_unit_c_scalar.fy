module module_unit_c_scalar;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import c_text as w;
import c_abi_types as casts;
import std::bytes as bytes;

pub fn u32 scalar(ref ir::Program p,usize type){if(type==0 || type>clean::type_count(ir::cleanup_ir(p))){return 0;}let t=clean::type_at(ir::cleanup_ir(p),type);return if(t.kind==1){t.scalar}else{0};}
pub fn bool value(ref mut bytes::Buffer out,ref mut w::State s,usize id){return w::text(out,s,b"fy_v_") && w::number(out,s,id);}
pub fn bool capture(ref mut bytes::Buffer out,ref mut w::State s,usize id){return w::text(out,s,b"fy_capture_") && w::number(out,s,id);}
pub fn u32 operator(ref u8[] op){return if(bytes::equal(op,b"+")){1}else if(bytes::equal(op,b"-")){2}else if(bytes::equal(op,b"!")){3}else if(bytes::equal(op,b"~")){4}else if(bytes::equal(op,b"*")){5}else if(bytes::equal(op,b"/")){6}else if(bytes::equal(op,b"%")){7}else if(bytes::equal(op,b"&")){8}else if(bytes::equal(op,b"|")){9}else if(bytes::equal(op,b"^")){10}else if(bytes::equal(op,b"<<")){11}else if(bytes::equal(op,b">>")){12}else if(bytes::equal(op,b"==")){13}else if(bytes::equal(op,b"!=")){14}else if(bytes::equal(op,b"<")){15}else if(bytes::equal(op,b"<=")){16}else if(bytes::equal(op,b">")){17}else if(bytes::equal(op,b">=")){18}else{0};}
// Checked arithmetic uses widened, bounded helpers. Unsigned promotions never
// turn u8 multiplication or high-bit shifts into signed C arithmetic.
pub fn bool helpers(ref mut bytes::Buffer out,ref mut w::State s){
    return w::text(out,s,b"static int64_t fy_unit_bits(uint64_t x,unsigned width){if(width==32){uint32_t b=(uint32_t)x;int32_t r;memcpy(&r,&b,4);return r;}int64_t r;memcpy(&r,&x,8);return r;}\nstatic uint64_t fy_unit_unsigned(unsigned op,uint64_t a,uint64_t b,uint64_t max,unsigned width){switch(op){case 1:if(a>max-b)fy_runtime_trap(1);return a+b;case 2:if(a<b)fy_runtime_trap(1);return a-b;case 5:if(b&&a>max/b)fy_runtime_trap(1);return a*b;case 6:case 7:if(!b)fy_runtime_trap(2);return op==6?a/b:a%b;case 8:return a&b;case 9:return a|b;case 10:return a^b;case 11:case 12:if(b>=width)fy_runtime_trap(4);if(op==11){if(a>(max>>b))fy_runtime_trap(1);return a<<b;}return a>>b;default:fy_runtime_trap(7);}}\nstatic int64_t fy_unit_signed(unsigned op,int64_t a,int64_t b,int64_t min,int64_t max,unsigned width){switch(op){case 1:if((b>0&&a>max-b)||(b<0&&a<min-b))fy_runtime_trap(1);return a+b;case 2:if((b<0&&a>max+b)||(b>0&&a<min+b))fy_runtime_trap(1);return a-b;case 5:if(a>0?(b>0?a>max/b:b<min/a):(a<0?(b>0?a<min/b:b<0&&a<max/b):0))fy_runtime_trap(1);return a*b;case 6:case 7:if(!b)fy_runtime_trap(2);if(a==min&&b==-1)fy_runtime_trap(3);return op==6?a/b:a%b;case 8:return fy_unit_bits((uint64_t)a&(uint64_t)b,width);case 9:return fy_unit_bits((uint64_t)a|(uint64_t)b,width);case 10:return fy_unit_bits((uint64_t)a^(uint64_t)b,width);case 11:case 12:if(b<0||(uint64_t)b>=width)fy_runtime_trap(4);if(op==11){if(a<0||(uint64_t)a>((uint64_t)max>>b))fy_runtime_trap(1);return (int64_t)((uint64_t)a<<b);}if(!b||a>=0)return a>>b;return fy_unit_bits(((uint64_t)a>>b)|(UINT64_MAX<<(64-(unsigned)b)),width);default:fy_runtime_trap(7);}}\n");
}
pub fn bool math_start(ref mut bytes::Buffer out,ref mut w::State s,u32 type,u32 op){return w::text(out,s,if(type==1 || type==2){b"fy_unit_signed("}else{b"fy_unit_unsigned("}) && w::number(out,s,op as usize) && w::text(out,s,b",");}
pub fn bool math_end(ref mut bytes::Buffer out,ref mut w::State s,u32 type){return w::text(out,s,b",") && (type!=1 && type!=2 || w::minimum(out,s,type) && w::text(out,s,b",")) && w::maximum(out,s,type) && w::text(out,s,b",") && w::number(out,s,if(type==3){8}else if(type==1 || type==4){32}else{64}) && w::text(out,s,b")");}
fn bool literal(ref ir::Body b,usize id,u32 type,ref mut bytes::Buffer out,ref mut w::State s){
    if(type==6){let text=ir::node_text(b,id-1);if(!bytes::equal(text,b"true") && !bytes::equal(text,b"false")){s.code=1001;return false;}return w::text(out,s,if(bytes::equal(text,b"true")){b"1"}else{b"0"});}
    let text=ir::node_cooked(b,id-1);if(text.len==0 || text.len>20){s.code=1001;return false;}var usize i=0;while(i<text.len){if(text[i]<48 || text[i]>57){s.code=1001;return false;}i+=1;}
    return (type!=1 && type!=2 || w::text(out,s,b"fy_unit_bits(")) && w::text(out,s,b"UINT64_C(") && w::text(out,s,text) && w::text(out,s,b")") && (type!=1 && type!=2 || w::text(out,s,if(type==1){b",32)"}else{b",64)"}));
}
pub fn bool expression(ref ir::Program p,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s){
    let e=ir::node_at(b,id-1).value;let type=scalar(p,e.type);let left=ir::node(b,e.left);let right=ir::node(b,e.right);let op=operator(ir::node_text(b,id-1));
    if(type==0 || type==7 || type>6 && e.kind!=23){s.code=1004;return false;}
    if(e.kind==23){
        if(left==0 || e.target!=e.type || !bytes::equal(ir::node_text(b,id-1),b"as")){s.code=1001;return false;}
        let source=scalar(p,ir::node_at(b,left-1).value.type);
        if(!casts::allowed(source,type)){s.code=1004;return false;}
    }
    if(!value(out,s,id) || !w::text(out,s,b"=")){return false;}
    if(e.kind==1){if(!literal(b,id,type,out,s)){return false;}}
    else if(e.kind==2){if(e.binding==0 || e.binding>ir::local_count(b) || ir::local_at(b,e.binding).type!=e.type){s.code=1001;return false;}if(!w::local(out,s,e.binding)){return false;}}
    else if(e.kind==3){if(!value(out,s,left)){return false;}}
    else if(e.kind==4){
        if(left==0 || op==0 || op>4){s.code=1001;return false;}
        if(op==1){if(!value(out,s,left)){return false;}}
        else if(op==3){if(!w::text(out,s,b"!") || !value(out,s,left)){return false;}}
        else if(op==2){if(e.minimum){if(!w::minimum(out,s,type)){return false;}}else if(!math_start(out,s,type,2) || !w::text(out,s,b"0,") || !value(out,s,left) || !math_end(out,s,type)){return false;}}
        else if(type==1 || type==2){if(!w::text(out,s,b"fy_unit_bits(~(uint64_t)") || !value(out,s,left) || !w::text(out,s,if(type==1){b",32)"}else{b",64)"})){return false;}}
        else if(!w::text(out,s,b"(") || !w::type(out,s,type) || !w::text(out,s,b")~(uint64_t)") || !value(out,s,left)){return false;}
    }
    else if(e.kind==5){
        if(left==0 || right==0){s.code=1001;return false;}
        let text=ir::node_text(b,id-1);let operand=scalar(p,ir::node_at(b,left-1).value.type);
        if(bytes::equal(text,b"&&") || bytes::equal(text,b"||")){if(!value(out,s,left) || !w::text(out,s,text) || !value(out,s,right)){return false;}}
        else if(op>=13){if(!value(out,s,left) || !w::operator(out,s,op) || !value(out,s,right)){return false;}}
        else if(op==0 || op==3 || op==4 || operand==0 || operand==6 || operand==7){s.code=1004;return false;}
        else if(!math_start(out,s,operand,op) || !value(out,s,left) || !w::text(out,s,b",") || !value(out,s,right) || !math_end(out,s,operand)){return false;}
    }else if(e.kind==23){if(!w::text(out,s,b"(") || !w::type(out,s,type) || !w::text(out,s,b")") || !value(out,s,left)){return false;}}
    else{s.code=1004;return false;}
    return w::text(out,s,b";\n");
}
