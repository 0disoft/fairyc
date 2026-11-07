module module_unit_c_closure;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_body_ir_closures as proof;
import module_unit_c_names as names;
import module_unit_c_scalar as scalar;
import module_unit_value_model as budget;
import c_text as w;
import std::bytes as bytes;

pub fn bool leaf(ref ir::Program p,usize id){return proof::leaf(ir::cleanup_ir(p),id);}
fn bool adapter(ref mut bytes::Buffer out,ref mut w::State s,usize frame,usize id){return w::text(out,s,b"fy_unit_closure_") && w::number(out,s,frame) && w::text(out,s,b"_") && w::number(out,s,id);}
// Return an actual selected frame, never a source declaration cast to a frame.
pub fn usize helper(ref ir::Program p,usize frame,usize id,ref mut usize work,ref mut u32 code){
    let b=ir::at(p,frame);let e=ir::node_at(b,id-1).value;if(e.kind!=36 || !leaf(p,e.type)){*code=1001;return 0;}
    var usize link=e.right;var usize captures=0;while(link!=0){if(!budget::charge(work,1)){*code=1002;return 0;}let at=ir::node(b,link);if(at==0 || captures>=4){*code=1001;return 0;}let pair=ir::node_at(b,at-1).value;link=pair.right;captures+=1;}
    let found=proof::helper(p,frame,e.target,captures,work);if(found==0){*code=if(*work>=1048576){1002}else{1001};}return found;
}
pub fn bool adapters(ref ir::Program p,ref names::Table named,usize frame,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    if(!clean::closures(ir::cleanup_ir(p))){return true;}let b=ir::at(p,frame);var usize id=1;while(id<=ir::node_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let e=ir::node_at(b,id-1).value;if(e.kind==36){
        let target=helper(p,frame,id,work,ref mut s.code);if(target==0){return false;}let body=ir::at(p,target);var usize count=0;var usize i=1;
        while(i<=ir::local_count(body)){if(!budget::charge(work,1)){s.code=1002;return false;}let l=ir::local_at(body,i);if(l.kind==1){if(scalar::scalar(p,l.type)!=1){s.code=1001;return false;}count+=1;}i+=1;}if(count==0 || count>5){s.code=1001;return false;}
        if(!w::text(out,s,b"static int32_t ") || !adapter(out,s,frame,id) || !w::text(out,s,b"(int32_t const *fy_env,int32_t fy_arg){(void)fy_env;return ") || !names::function(named,target,out,s) || !w::text(out,s,b"(")){return false;}
        i=0;while(i+1<count){if(i!=0 && !w::text(out,s,b",")){return false;}if(!w::text(out,s,b"fy_env[") || !w::number(out,s,i) || !w::text(out,s,b"]")){return false;}i+=1;}
        if(i!=0 && !w::text(out,s,b",")){return false;}if(!w::text(out,s,b"fy_arg);}\n")){return false;}
    }id+=1;}return true;
}
pub fn bool construct(ref ir::Program p,ref ir::Body b,usize frame,usize id,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work){
    let e=ir::node_at(b,id-1).value;if(!leaf(p,e.type)){s.code=1001;return false;}
    if(!scalar::value(out,s,id) || !w::text(out,s,b".fy_invoke=") || !adapter(out,s,frame,id) || !w::text(out,s,b";\n")){return false;}
    var usize link=e.right;var usize count=0;while(link!=0){if(!budget::charge(work,1)){s.code=1002;return false;}let at=ir::node(b,link);if(at==0 || count>=4){s.code=1001;return false;}count+=1;link=ir::node_at(b,at-1).value.right;}
    var usize i=0;while(i<4){if(!scalar::value(out,s,id) || !w::text(out,s,b".fy_captures[") || !w::number(out,s,i) || !w::text(out,s,b"]=0;\n")){return false;}i+=1;}
    // Argument chains are stored newest-first. Slots retain written order.
    link=e.right;while(link!=0){if(!budget::charge(work,1)){s.code=1002;return false;}let pair=ir::node_at(b,ir::node(b,link)-1).value;let operand=ir::node(b,pair.left);if(pair.kind!=7 || operand==0 || scalar::scalar(p,ir::node_at(b,operand-1).value.type)!=1){s.code=1001;return false;}count-=1;
        if(!scalar::value(out,s,id) || !w::text(out,s,b".fy_captures[") || !w::number(out,s,count) || !w::text(out,s,b"]=") || !scalar::value(out,s,operand) || !w::text(out,s,b";\n")){return false;}link=pair.right;
    }return true;
}
pub fn bool invoke(ref ir::Program p,ref ir::Body b,usize expression,usize index,ref mut bytes::Buffer out,ref mut w::State s){
    let c=ir::call_at(b,index-1);let e=ir::node_at(b,expression-1).value;let left=ir::node(b,e.left);
    if(!c.checked || c.expression!=expression || c.kind!=4 || c.target_module!=0 || c.target_declaration!=0 || c.instance!=0 || c.argument_count!=1 || left==0 || !leaf(p,ir::node_at(b,left-1).value.type) || scalar::scalar(p,e.type)!=1){s.code=1001;return false;}
    let arg=ir::argument_at(b,c.argument_start);if(!arg.checked || arg.actual!=arg.expected || scalar::scalar(p,arg.actual)!=1 || arg.expression==0 || scalar::scalar(p,ir::node_at(b,arg.expression-1).value.type)!=1){s.code=1001;return false;}
    return scalar::value(out,s,expression) && w::text(out,s,b"=") && scalar::value(out,s,left) && w::text(out,s,b".fy_invoke(") && scalar::value(out,s,left) && w::text(out,s,b".fy_captures,") && scalar::value(out,s,arg.expression) && w::text(out,s,b");\n");
}
