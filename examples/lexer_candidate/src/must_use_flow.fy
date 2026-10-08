module must_use_flow;
import syntax as syntax;
import name_resolution as names;
import type_state as state;
import effect_declarations as attributes;
import borrow_shapes as borrows;
import tagged_types as tagged;
import resource_properties as resources;
import std::vec as v;
struct Context {usize work;state::Problem problem;}
fn void fail(ref mut Context c,u32 code,usize start,usize end){if(c.problem.code==0){c.problem=state::fail(code,start,end,0,0);}}
pub fn bool eligible(u32 type){return type!=0 && type!=7 && type!=8 && !tagged::result(type) && borrows::count(type)==0 && !resources::needs_drop(type);}
pub fn state::Problem declarations(ref names::Resolved resolved,ref v::Vec<state::Slot> types){let p=names::parsed(resolved);var usize id=1;
    while(id<=names::symbol_count(resolved)){let s=names::symbol_at(resolved,id);if(s.kind==1 || s.kind==11){let info=attributes::read(p,s.declaration-1);if(info.must_use && !eligible(state::symbols(types,id-1))){return state::fail(4201,info.must_start,info.must_end,0,0);}}id+=1;}return state::ok();
}
fn void expression(ref names::Resolved resolved,ref v::Vec<state::Slot> types,ref mut Context c,usize id,bool handled,usize depth){
    if(id==0 || c.problem.code!=0){return;}let p=names::parsed(resolved);let e=syntax::expression_at(p,id);c.work+=1;
    if(depth>=64 || c.work>1048576){fail(c,10,e.start,e.end);return;}
    if(e.kind==6){let usize target=state::uses(types,e.left-1);if(target!=0){let symbol=names::symbol_at(resolved,target);
        if(symbol.kind==1 || symbol.kind==11){let info=attributes::read(p,symbol.declaration-1);if(info.must_use && !handled){fail(c,4202,e.start,e.end);c.problem.move_start=info.must_start;c.problem.move_end=info.must_end;return;}}
        else if(symbol.kind==5 && names::external_signature(resolved,target).must_use && !handled){fail(c,4202,e.start,e.end);return;}}
        var usize i=0;while(i<syntax::call_argument_count(p,id)){expression(resolved,types,c,syntax::call_argument(p,id,i),true,depth+1);i+=1;}return;
    }
    if(e.kind==3){expression(resolved,types,c,e.left,handled,depth+1);return;}
    if((e.kind==12 || e.kind==31)){block(resolved,types,c,e.left,handled,depth+1);return;}
    if(e.kind==13){expression(resolved,types,c,e.left,true,depth+1);let branches=syntax::expression_at(p,e.right);block(resolved,types,c,branches.left,handled,depth+1);expression(resolved,types,c,branches.right,handled,depth+1);return;}
    if(e.kind==15){expression(resolved,types,c,e.left,true,depth+1);var usize link=e.right;while(link!=0 && c.problem.code==0){let pair=syntax::expression_at(p,link);let arm=syntax::expression_at(p,pair.left);block(resolved,types,c,arm.right,handled,depth+1);link=pair.right;}return;}
    if(e.kind==10){var usize i=0;while(i<syntax::record_field_count(p,id)){let f=syntax::expression_at(p,syntax::record_field(p,id,i));expression(resolved,types,c,f.left,true,depth+1);i+=1;}return;}
    if(e.kind==24){var usize i=0;while(i<syntax::array_element_count(p,id)){expression(resolved,types,c,syntax::array_element(p,id,i),true,depth+1);i+=1;}return;}
    if(e.kind==29){expression(resolved,types,c,e.left,true,depth+1);return;}
    if(e.kind==26){expression(resolved,types,c,e.left,true,depth+1);let bounds=syntax::expression_at(p,e.right);expression(resolved,types,c,bounds.left,true,depth+1);expression(resolved,types,c,bounds.right,true,depth+1);return;}
    if(e.kind==23){expression(resolved,types,c,e.left,true,depth+1);return;}
    if(e.kind!=1 && e.kind!=2 && e.kind!=28){expression(resolved,types,c,e.left,true,depth+1);expression(resolved,types,c,e.right,true,depth+1);}
}
fn void statement(ref names::Resolved resolved,ref v::Vec<state::Slot> types,ref mut Context c,usize id,usize depth){
    if(id==0 || c.problem.code!=0){return;}let p=names::parsed(resolved);let s=syntax::statement_at(p,id);
    if(depth>=64){fail(c,10,s.start,s.end);return;}
    if(s.kind==8 || s.kind==14 || s.kind==18){if(s.kind==14){let header=syntax::statement_at(p,s.second);statement(resolved,types,c,header.first,depth+1);statement(resolved,types,c,header.second,depth+1);}expression(resolved,types,c,s.expression,true,depth+1);block(resolved,types,c,s.first,false,depth+1);return;}
    if(s.kind==4){expression(resolved,types,c,s.target,true,depth+1);}
    expression(resolved,types,c,s.expression,s.kind!=6,depth+1);
}
fn void block(ref names::Resolved resolved,ref v::Vec<state::Slot> types,ref mut Context c,usize id,bool handled,usize depth){
    if(id==0 || c.problem.code!=0){return;}let p=names::parsed(resolved);let b=syntax::statement_at(p,id);if(depth>=64){fail(c,10,b.start,b.end);return;}
    var usize i=0;while(i<syntax::block_statement_count(p,id)){statement(resolved,types,c,syntax::block_statement(p,id,i),depth+1);i+=1;}expression(resolved,types,c,b.expression,handled,depth+1);
}
pub fn state::Problem check(ref names::Resolved resolved,ref v::Vec<state::Slot> types){let p=names::parsed(resolved);var Context c=Context{.work=0,.problem=state::ok()};var usize i=0;
    while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==3){block(resolved,types,ref mut c,d.body,true,0);}else if(d.kind==6){block(resolved,types,ref mut c,d.cleanup_body,false,0);}i+=1;}return c.problem;
}
