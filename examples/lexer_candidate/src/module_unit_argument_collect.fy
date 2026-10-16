module module_unit_argument_collect;
import module_unit_value_model as m;
import module_signatures as signatures;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_types as types;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

pub fn Result<bool,m::Problem> labeled(ref syntax::Parsed source,m::Frame frame,usize id,ref mut usize work){
    let e=syntax::expression_at(source,id);var usize link=e.right;var bool found=false;while(link!=0){let row=syntax::expression_at(source,link);if(!m::charge(work,1)){return .err(m::fail(frame,10,e.start,e.end,0,0));}if(row.kind!=7){return .err(m::fail(frame,12,row.start,row.end,0,0));}if(syntax::call_argument_labeled(source,link)){found=true;}link=row.right;}return .ok(found);
}
fn m::Problem label(ref syntax::Parsed caller,ref syntax::Parsed target,m::Frame frame,usize declaration,usize row,usize position,ref mut usize work){
    let e=syntax::expression_at(caller,row);let token=scanner::token_info(syntax::source_stream(caller),e.token);let d=syntax::declaration_at(target,declaration-1);var usize ordinal=0;var usize i=d.entry_start;
    while(i<d.entry_start+d.entry_count){if(!m::charge(work,1)){return m::fail(frame,10,token.start,token.end,0,0);}let p=syntax::entry_at(target,i);if(p.kind==1){if(ordinal==position){return if(bytes::equal(syntax::expression_text(caller,row),syntax::source_view(target)[p.name_start..p.name_end])){m::fail(frame,0,0,0,0,0)}else{m::fail(frame,1014,e.start,e.end,0,0)};}ordinal+=1;}i+=1;}return m::fail(frame,12,token.start,token.end,0,0);
}
pub fn m::Problem record(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref interfaces::Interfaces imported,m::Frame frame,usize expression,usize source_index,u32 kind,usize target_module,usize target_declaration,usize instance,usize parameter_start,usize parameter_count,ref v::Vec<types::Type> graph,ref v::Vec<m::Expression> expressions,ref mut v::Vec<m::Invocation> calls,ref mut v::Vec<m::Argument> arguments,ref mut usize work){
    let source=v::at<syntax::Parsed>(sources,frame.module_index);let e=syntax::expression_at(source,source_index);let target=v::at<syntax::Parsed>(sources,target_module);let d=syntax::declaration_at(target,target_declaration-1);let has_labels=switch(labeled(source,frame,source_index,work)){case .ok(b):{b}case .err(p):{return p;}};
    if(has_labels && (kind!=1 || target_module!=frame.module_index || d.kind!=3 || d.public || d.generic_parameters!=0 || d.owner_end!=0)){return m::fail(frame,1014,e.start,e.end,0,0);}
    var v::Vec<usize> order=v::empty<usize>();var usize link=e.right;while(link!=0){if(!m::charge(work,1) || v::len<usize>(ref order)>=16384){return m::fail(frame,10,e.start,e.end,0,0);}if(!v::push<usize>(ref mut order,link).is_ok){return m::fail(frame,11,e.start,e.end,0,0);}link=syntax::expression_at(source,link).right;}
    let callee=syntax::expression_at(source,e.left);let receiver=d.owner_end!=0 && callee.kind==8;let usize offset=if(receiver){1}else{0};
    if(v::len<usize>(ref order)+offset!=parameter_count){return m::fail(frame,1010,e.start,e.end,parameter_count,v::len<usize>(ref order));}if(v::len<m::Invocation>(calls)>=16384 || parameter_count>16384-v::len<m::Argument>(arguments)){return m::fail(frame,10,e.start,e.end,0,0);}
    let usize invocation=v::len<m::Invocation>(calls)+1;let usize argument_start=v::len<m::Argument>(arguments);var usize position=0;
    while(position<parameter_count){let implicit=receiver && position==0;let usize row=if(implicit){0}else{*v::at<usize>(ref order,parameter_count-position-1)};let arg=syntax::expression_at(source,if(implicit){callee.left}else{row});let selected=if(implicit){callee.left}else{arg.left};let node=syntax::expression_at(source,selected);let usize local=m::index(expressions,selected);if(local==0){return m::fail(frame,12,node.start,node.end,0,0);}let usize expected=if(kind==1){signatures::parameter_at(metadata,parameter_start+position)}else{generics::parameter_at(interfaces::specializations(imported),parameter_start+position)};
        if(!m::charge(work,1)){return m::fail(frame,10,node.start,node.end,0,0);}if(expected==0 || expected>types::count(graph)){return m::fail(frame,12,node.start,node.end,0,expected);}var usize label_start=0;var usize label_end=0;
        if(!implicit && syntax::call_argument_labeled(source,row)){let p=label(source,target,frame,target_declaration,row,position,work);if(p.code!=0){return p;}let token=scanner::token_info(syntax::source_stream(source),arg.token);label_start=token.start;label_end=token.end;}
        if(!v::push<m::Argument>(arguments,m::Argument{.invocation,.position,.expression=local,.source_index=selected,.expected,.actual=0,.start=node.start,.end=node.end,.label_start,.label_end,.checked=false,.shared=false}).is_ok){return m::fail(frame,11,node.start,node.end,0,0);}position+=1;
    }
    if(!v::push<m::Invocation>(calls,m::Invocation{.kind,.expression,.source_index,.target_module,.target_declaration,.instance,.argument_start,.argument_count=parameter_count,.start=e.start,.end=e.end,.checked=false}).is_ok){return m::fail(frame,11,e.start,e.end,0,0);}return m::fail(frame,0,0,0,0,0);
}
