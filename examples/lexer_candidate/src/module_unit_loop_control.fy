module module_unit_loop_control;
import module_unit_body_ast as ast;
import module_unit_value_model as m;

// The copied AST retains exact source spans. A control statement is in a loop
// only when its span is inside that loop's body, never merely in its header.
// Each function/cleanup has its own arena, so another function cannot confer
// loop scope. Expression blocks and switch arms retain the same lexical rule.
pub fn m::Problem check(ref ast::Ast body,m::Frame frame,ref mut usize work){
    var usize at=1;while(at<=ast::statement_count(body)){let s=ast::statement_at(body,at);if(!m::charge(work,1)){return m::fail(frame,10,s.start,s.end,0,0);}if(s.kind==9 || s.kind==10){var bool enclosed=false;var usize i=1;
        while(i<=ast::statement_count(body)){let loop=ast::statement_at(body,i);if(!m::charge(work,1)){return m::fail(frame,10,s.start,s.end,0,0);}if(loop.kind==8 || loop.kind==14 || loop.kind==18){if(loop.first==0 || loop.first>ast::statement_count(body)){return m::fail(frame,12,loop.start,loop.end,0,0);}let block=ast::statement_at(body,loop.first);if(block.kind!=1 || block.start<loop.start || block.end>loop.end){return m::fail(frame,12,loop.start,loop.end,0,0);}if(block.start<=s.start && s.end<=block.end){enclosed=true;break;}}i+=1;}
        if(!enclosed){return m::fail(frame,1012,s.start,s.end,0,0);}
    }at+=1;}return m::fail(frame,0,0,0,0,0);
}
