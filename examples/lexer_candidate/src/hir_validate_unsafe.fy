module hir_validate_unsafe;
import hir_model as h;
import std::bytes as bytes;

struct UnsafeBudget { usize work; h::Problem problem; }
fn bool unsafe_fail(ref mut UnsafeBudget c,u32 code,usize start,usize end){
    if(c.problem.code==0){c.problem=h::Problem{.code,.start,.end};}return false;
}
fn bool unsafe_expression(ref h::Program p,usize id,bool allowed,usize depth,ref mut UnsafeBudget c,ref mut u8[] calls){
    if(id==0){return true;}let e=h::expression_at(p,id);let u32 kind=h::expression_code(e.kind);
    if(depth>=64 || c.work>=1048576){return unsafe_fail(c,2,e.start,e.end);}c.work+=1;
    if(kind==28 || kind==29 || kind==30){if(!allowed || !e.boolean){return unsafe_fail(c,5101,e.start,e.end);}calls[id-1]=1;}
    if(kind==6){let b=h::binding_at(p,e.binding);if(e.boolean!=b.unsafe_call || b.unsafe_call && !allowed){return unsafe_fail(c,5101,e.start,e.end);}calls[id-1]=1;}
    if(!unsafe_expression(p,e.left,allowed,depth+1,c,calls) || !unsafe_expression(p,e.right,allowed,depth+1,c,calls) || !unsafe_expression(p,e.alternative,allowed,depth+1,c,calls)){return false;}
    if(e.block!=0 && !unsafe_block(p,e.block,allowed,depth+1,c,calls)){return false;}
    // Capture edges name bindings, all other expression edges name expressions.
    var usize i=0;while(kind!=21 && i<e.edge_count){if(!unsafe_expression(p,h::edge_at(p,e.edge_start+i),allowed,depth+1,c,calls)){return false;}i+=1;}return true;
}
fn bool unsafe_statement(ref h::Program p,usize id,bool allowed,usize depth,ref mut UnsafeBudget c,ref mut u8[] calls){
    if(id==0){return true;}let s=h::statement_at(p,id);
    if(depth>=64 || c.work>=1048576){return unsafe_fail(c,2,s.start,s.end);}c.work+=1;
    return unsafe_expression(p,s.expression,allowed,depth+1,c,calls) && unsafe_expression(p,s.target,allowed,depth+1,c,calls)
        && (s.body==0 || unsafe_block(p,s.body,allowed,depth+1,c,calls))
        && unsafe_statement(p,s.initial,allowed,depth+1,c,calls) && unsafe_statement(p,s.post,allowed,depth+1,c,calls);
}
fn bool unsafe_block(ref h::Program p,usize id,bool inherited,usize depth,ref mut UnsafeBudget c,ref mut u8[] calls){
    let b=h::block_at(p,id);if(depth>=64 || c.work>=1048576){return unsafe_fail(c,2,b.start,b.end);}c.work+=1;
    let bool allowed=inherited || b.unsafe_body;var usize i=0;
    while(i<b.edge_count){if(!unsafe_statement(p,h::edge_at(p,b.edge_start+i),allowed,depth+1,c,calls)){return false;}i+=1;}
    return unsafe_expression(p,b.tail,allowed,depth+1,c,calls);
}
pub fn h::Problem unsafe_check(ref h::Program p){
    var UnsafeBudget c=UnsafeBudget{.work=0,.problem=h::Problem{.code=0,.start=0,.end=0}};var usize i=1;var bool needed=false;
    while(i<=h::binding_count(p)){let b=h::binding_at(p,i);if(b.unsafe_call && h::binding_code(b.kind)!=1 && h::binding_code(b.kind)!=5 && h::binding_code(b.kind)!=11){discard unsafe_fail(ref mut c,4,b.start,b.end);return c.problem;}i+=1;}
    i=1;while(i<=h::binding_count(p)){needed=needed || h::binding_at(p,i).unsafe_call;i+=1;}
    i=1;while(i<=h::expression_count(p)){let e=h::expression_at(p,i);needed=needed || h::expression_code(e.kind)==28 || h::expression_code(e.kind)==29 || h::expression_code(e.kind)==30 || h::expression_code(e.kind)==6 && e.boolean;i+=1;}
    if(!needed){return c.problem;}
    var bytes::Buffer visited=bytes::empty();
    if(!bytes::resize_zeroed(ref mut visited,h::expression_count(p)).is_ok){return h::Problem{.code=3,.start=0,.end=0};}
    i=0;while(i<h::function_count(p)){let f=h::function_at(p,i);if(!unsafe_block(p,f.body,h::binding_at(p,f.binding).unsafe_call,0,ref mut c,bytes::view_mut(ref mut visited))){return c.problem;}i+=1;}
    i=1;while(i<=h::expression_count(p)){let e=h::expression_at(p,i);if((h::expression_code(e.kind)==6 || (h::expression_code(e.kind)==28 || h::expression_code(e.kind)==29 || h::expression_code(e.kind)==30)) && bytes::view(ref visited)[i-1]==0){discard unsafe_fail(ref mut c,4,e.start,e.end);return c.problem;}i+=1;}return c.problem;
}
