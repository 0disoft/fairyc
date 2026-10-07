module resource_use;
import syntax as syntax;
import name_resolution as names;
import type_state as state;
import resource_properties as properties;
import std::vec as v;

// Classify consumption after exact typing, without sharing the field/member
// flag arena. This unit records transfers; liveness is checked by later flow.
struct Context { usize work; usize receiver; state::Problem problem; }
fn void fail(ref mut Context c,u32 code,usize start,usize end){if(c.problem.code==0){c.problem=state::fail(code,start,end,0,0);}}
fn void expression(ref syntax::Parsed p,ref mut v::Vec<state::Slot> tables,ref mut Context c,usize id,bool consume,usize depth){
    if(id==0 || c.problem.code!=0){return;}let e=syntax::expression_at(p,id);c.work+=1;
    if(depth>=64 || c.work>1048576){fail(c,10,e.start,e.end);return;}
    if(e.kind==2){let bool moving=consume && properties::needs_drop(state::expressions(tables,id-1));
        if(moving && c.receiver!=0 && state::uses(tables,id-1)==c.receiver){fail(c,48,e.start,e.end);return;}
        state::set_moving(tables,id-1,moving);return;}
    if(e.kind==3){expression(p,tables,c,e.left,consume,depth+1);return;}
    if(e.kind==8){
        let bool is_ok=state::flags(tables,id-1)==2048;
        if(!is_ok && consume && properties::needs_drop(state::expressions(tables,id-1))){fail(c,44,e.start,e.end);return;}
        expression(p,tables,c,e.left,is_ok,depth+1);return;
    }
    if(e.kind==6){var usize i=0;while(i<syntax::call_argument_count(p,id)){expression(p,tables,c,syntax::call_argument(p,id,i),true,depth+1);i+=1;}return;}
    if(e.kind==10){var usize i=0;while(i<syntax::record_field_count(p,id)){let f=syntax::expression_at(p,syntax::record_field(p,id,i));expression(p,tables,c,f.left,true,depth+1);i+=1;}return;}
    if(e.kind==12){block(p,tables,c,e.left,consume,depth+1);return;}
    if(e.kind==13){expression(p,tables,c,e.left,false,depth+1);let branches=syntax::expression_at(p,e.right);block(p,tables,c,branches.left,consume,depth+1);expression(p,tables,c,branches.right,consume,depth+1);return;}
    if(e.kind==15){expression(p,tables,c,e.left,true,depth+1);var usize link=e.right;while(link!=0){let pair=syntax::expression_at(p,link);let arm=syntax::expression_at(p,pair.left);block(p,tables,c,arm.right,consume,depth+1);link=pair.right;}return;}
    if(e.kind==30){expression(p,tables,c,e.left,true,depth+1);return;}
    if(e.kind==24){var usize i=0;while(i<syntax::array_element_count(p,id)){expression(p,tables,c,syntax::array_element(p,id,i),true,depth+1);i+=1;}return;}
    if(e.kind==29){expression(p,tables,c,e.left,true,depth+1);return;}
    if(e.kind==26){expression(p,tables,c,e.left,false,depth+1);let bounds=syntax::expression_at(p,e.right);expression(p,tables,c,bounds.left,false,depth+1);expression(p,tables,c,bounds.right,false,depth+1);return;}
    if(e.kind!=1 && e.kind!=28){expression(p,tables,c,e.left,false,depth+1);expression(p,tables,c,e.right,false,depth+1);}
}
fn void statement(ref syntax::Parsed p,ref mut v::Vec<state::Slot> tables,ref mut Context c,usize id,usize depth){
    if(id==0 || c.problem.code!=0){return;}let s=syntax::statement_at(p,id);
    if(depth>=64){fail(c,10,s.start,s.end);return;}
    if(s.kind==2 || s.kind==3 || s.kind==12 || s.kind==16){expression(p,tables,c,s.expression,true,depth+1);return;}
    if(s.kind==4){
        // A resource root replacement consumes the RHS before installing the
        // target; its target is a place, not a second consuming expression.
        expression(p,tables,c,s.expression,true,depth+1);expression(p,tables,c,s.target,false,depth+1);return;
    }
    if(s.kind==8 || s.kind==14){
        if(s.kind==14){let header=syntax::statement_at(p,s.second);statement(p,tables,c,header.first,depth+1);statement(p,tables,c,header.second,depth+1);}
        expression(p,tables,c,s.expression,false,depth+1);block(p,tables,c,s.first,false,depth+1);return;
    }
    expression(p,tables,c,s.expression,false,depth+1);
}
fn void block(ref syntax::Parsed p,ref mut v::Vec<state::Slot> tables,ref mut Context c,usize id,bool consume,usize depth){
    if(id==0 || c.problem.code!=0){return;}if(depth>=64){fail(c,10,0,0);return;}
    var usize i=0;while(i<syntax::block_statement_count(p,id)){statement(p,tables,c,syntax::block_statement(p,id,i),depth+1);i+=1;}
    expression(p,tables,c,syntax::statement_at(p,id).expression,consume,depth+1);
}
pub fn state::Problem classify(ref names::Resolved resolved,ref mut v::Vec<state::Slot> tables){
    let ref syntax::Parsed p=names::parsed(resolved);var Context c=Context{.work=0,.receiver=0,.problem=state::ok()};var usize i=0;
    while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);
        if(d.kind==3){c.receiver=0;block(p,tables,ref mut c,d.body,true,0);}
        else if(d.kind==6){c.receiver=0;var usize n=1;while(n<=names::symbol_count(resolved)){let s=names::symbol_at(resolved,n);if(s.kind==10 && s.declaration==i+1){c.receiver=n;}n+=1;}block(p,tables,ref mut c,d.cleanup_body,false,0);}
        i+=1;}
    return c.problem;
}
