module module_unit_call_contracts;
import module_signatures as signatures;
import module_ast_join as join;
import effect_declarations as effects;
import syntax as syntax;
import std::vec as v;

struct Context {usize module_index;u32 authority;bool unsafe_function;usize work;join::Problem problem;}
fn void fail(ref mut Context c,u32 code,usize start,usize end){if(c.problem.code==0){c.problem=join::Problem{.code,.module_index=c.module_index,.start,.end};}}
fn bool charge(ref mut Context c,usize amount,usize start,usize end){if(c.work>1048576 || amount>1048576-c.work){fail(c,10,start,end);return false;}c.work+=amount;return true;}
fn void expression(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref syntax::Parsed source,ref mut Context c,usize id,bool handled,bool unsafe_context,usize depth){
    if(id==0 || c.problem.code!=0){return;}let e=syntax::expression_at(source,id);
    if(depth>=64 || !charge(c,1,e.start,e.end)){fail(c,10,e.start,e.end);return;}
    if(e.kind==6){
        let callee=syntax::expression_at(source,e.left);var target=callee;if(target.kind==20){target=syntax::expression_at(source,target.left);}
        if(target.kind==2){var usize probes=0;let found=signatures::declaration_lookup_measured(metadata,c.module_index,syntax::source_view(source)[target.start..target.end],ref mut probes);
            if(!charge(c,probes,e.start,e.end)){return;}
            // Name resolution already validates calls. Intrinsics have no source
            // declaration; their unsafe/type rules remain with value checking.
            if(found.code==0){let d=signatures::declaration_at(metadata,found.item);if(d.kind==3 || d.kind==9){
                let provider=v::at<syntax::Parsed>(sources,d.module_index);let info=effects::read(provider,d.declaration-1);
                if(info.code!=0){fail(c,info.code,e.start,e.end);return;}
                if(!effects::includes(c.authority,info.bits)){fail(c,7102,callee.start,callee.end);return;}
                let declaration=syntax::declaration_at(provider,d.declaration-1);
                if(declaration.unsafe_end!=0 && !c.unsafe_function && !unsafe_context){fail(c,5101,e.start,e.end);return;}
                if(info.must_use && !handled){fail(c,4202,e.start,e.end);return;}
            }}
        }
        var usize i=0;while(i<syntax::call_argument_count(source,id)){expression(sources,metadata,source,c,syntax::call_argument(source,id,i),true,unsafe_context,depth+1);i+=1;}return;
    }
    if(e.kind==3){expression(sources,metadata,source,c,e.left,handled,unsafe_context,depth+1);return;}
    if(e.kind==12 || e.kind==31){block(sources,metadata,source,c,e.left,handled,unsafe_context || e.kind==31,depth+1);return;}
    if(e.kind==13){expression(sources,metadata,source,c,e.left,true,unsafe_context,depth+1);let branches=syntax::expression_at(source,e.right);block(sources,metadata,source,c,branches.left,handled,unsafe_context,depth+1);expression(sources,metadata,source,c,branches.right,handled,unsafe_context,depth+1);return;}
    if(e.kind==15){expression(sources,metadata,source,c,e.left,true,unsafe_context,depth+1);var usize link=e.right;while(link!=0 && c.problem.code==0){let pair=syntax::expression_at(source,link);let arm=syntax::expression_at(source,pair.left);block(sources,metadata,source,c,arm.right,handled,unsafe_context,depth+1);link=pair.right;}return;}
    if(e.kind==10){var usize i=0;while(i<syntax::record_field_count(source,id)){let f=syntax::expression_at(source,syntax::record_field(source,id,i));expression(sources,metadata,source,c,f.left,true,unsafe_context,depth+1);i+=1;}return;}
    if(e.kind==24){var usize i=0;while(i<syntax::array_element_count(source,id)){expression(sources,metadata,source,c,syntax::array_element(source,id,i),true,unsafe_context,depth+1);i+=1;}return;}
    if(e.kind==29){expression(sources,metadata,source,c,e.left,true,unsafe_context,depth+1);return;}
    if(e.kind==26){expression(sources,metadata,source,c,e.left,true,unsafe_context,depth+1);let bounds=syntax::expression_at(source,e.right);expression(sources,metadata,source,c,bounds.left,true,unsafe_context,depth+1);expression(sources,metadata,source,c,bounds.right,true,unsafe_context,depth+1);return;}
    if(e.kind==23){expression(sources,metadata,source,c,e.left,true,unsafe_context,depth+1);return;}
    if(e.kind!=1 && e.kind!=2 && e.kind!=28){expression(sources,metadata,source,c,e.left,true,unsafe_context,depth+1);expression(sources,metadata,source,c,e.right,true,unsafe_context,depth+1);}
}
fn void statement(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref syntax::Parsed source,ref mut Context c,usize id,bool unsafe_context,usize depth){
    if(id==0 || c.problem.code!=0){return;}let s=syntax::statement_at(source,id);
    if(depth>=64 || !charge(c,1,s.start,s.end)){fail(c,10,s.start,s.end);return;}
    if(s.kind==8 || s.kind==14 || s.kind==17 || s.kind==18){if(s.kind==14){let h=syntax::statement_at(source,s.second);statement(sources,metadata,source,c,h.first,unsafe_context,depth+1);statement(sources,metadata,source,c,h.second,unsafe_context,depth+1);}expression(sources,metadata,source,c,s.expression,true,unsafe_context,depth+1);block(sources,metadata,source,c,s.first,false,unsafe_context,depth+1);return;}
    if(s.kind==4){expression(sources,metadata,source,c,s.target,true,unsafe_context,depth+1);}
    expression(sources,metadata,source,c,s.expression,s.kind!=6,unsafe_context,depth+1);
}
fn void block(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref syntax::Parsed source,ref mut Context c,usize id,bool handled,bool unsafe_context,usize depth){
    if(id==0 || c.problem.code!=0){return;}let b=syntax::statement_at(source,id);
    if(depth>=64 || !charge(c,1,b.start,b.end)){fail(c,10,b.start,b.end);return;}
    var usize i=0;while(i<syntax::block_statement_count(source,id)){statement(sources,metadata,source,c,syntax::block_statement(source,id,i),unsafe_context,depth+1);i+=1;}
    expression(sources,metadata,source,c,b.expression,handled,unsafe_context,depth+1);
}
pub fn join::Problem check(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize module_index,usize declaration,bool is_cleanup,ref mut usize work){
    let source=v::at<syntax::Parsed>(sources,module_index);let d=syntax::declaration_at(source,declaration-1);let authority=effects::read(source,declaration-1);
    if(authority.code!=0){return join::Problem{.code=authority.code,.module_index,.start=authority.start,.end=authority.end};}
    var Context c=Context{.module_index,.authority=authority.bits,.unsafe_function=!is_cleanup && d.unsafe_end!=0,.work=*work,.problem=join::Problem{.code=0,.module_index,.start=0,.end=0}};
    block(sources,metadata,source,ref mut c,if(is_cleanup){d.cleanup_body}else{d.body},!is_cleanup,false,0);*work=c.work;return c.problem;
}
