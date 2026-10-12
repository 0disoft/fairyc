module generic_source_dependency_walk;
import generic_source_dependency_model as model;
import generic_source_catalog as catalog;
import generic_source_arguments as arguments;
import generic_source_view_loop_json as loops;
import generic_source_json as json;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Iterative traversal follows Go transform's executable AST projection. Parser
// list/header nodes and inert written ByteFor view/type facts are not expressions.
struct Task {u32 kind;usize node;}
struct Context {usize template_index;usize module_index;usize first;bool vector;bool root;}
fn u32 push(ref mut v::Vec<Task> pending,u32 kind,usize node,ref mut usize work){if(node==0){return 0;}if(v::len<Task>(pending)>=65536 || !budget::charge(work,1)){return 10;}return if(v::push<Task>(pending,Task{.kind,.node}).is_ok){0}else{11};}
fn u32 list(ref syntax::Parsed p,usize tail,u32 kind,ref mut v::Vec<Task> pending,ref mut usize work){var usize link=tail;while(link!=0){if(!budget::charge(work,1)){return 10;}if(link>syntax::type_count(p)){return 9;}let n=syntax::type_at(p,link);if(n.kind!=kind || n.left==0 || n.left>=link || n.right>=link){return 9;}let code=push(pending,1,n.left,work);if(code!=0){return code;}link=n.right;}return 0;}
fn u32 edge(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,ref syntax::Parsed p,Context context,ref u8[] name,usize tail,u32 kind,ref mut bytes::Buffer names,ref mut v::Vec<model::Argument> args,ref mut v::Vec<model::Edge> edges,ref mut v::Vec<model::Range> ranges,ref mut usize work){
    let target=switch(catalog::lookup(c,context.module_index,name,kind,work)){case .ok(value):{value}case .err(code):{return code;}};if(target==0){return if(kind==0){0}else{4501};}let template=catalog::template_at(c,target-1);var usize count=0;var usize link=tail;var usize[8] roots=[0;8];
    while(link!=0){if(!budget::charge(work,1)){return 10;}if(link>syntax::type_count(p) || count>=8){return 4501;}let n=syntax::type_at(p,link);if(n.kind!=7 || n.left==0 || n.left>=link || n.right>=link){return 9;}roots[count]=n.left;count+=1;link=n.right;}if(count!=template.count){return 4501;}
    let first=v::len<model::Argument>(args);let names_first=bytes::len(names);var usize i=count;while(i>0){i-=1;let n=syntax::type_at(p,roots[i]);if(n.kind!=1){return 4501;}let text=switch(arguments::path(p,n.start,n.end,work)){case .ok(value):{value}case .err(code):{return code;}};var usize parameter=0;if(!context.root){parameter=switch(arguments::parameter(c,context.template_index,bytes::view(ref text),work)){case .ok(value):{value}case .err(code):{return code;}};}
        if(parameter!=0){let code=model::argument(names,args,bytes::view(ref text),true,work);if(code!=0){return code;}}
        else{let concrete=switch(arguments::concrete(c,sources,context.module_index,bytes::view(ref text),work)){case .ok(value):{value}case .err(code):{return code;}};let code=model::argument(names,args,bytes::view(ref concrete),false,work);if(code!=0){return code;}}
    }return model::edge_bounded(names,args,edges,target,catalog::text(c,template.id),context.first,first,names_first,if(context.root){4096}else{32},work);
}
fn bool storage(ref u8[] name){return bytes::equal(name,b"storage_size") || bytes::equal(name,b"storage_alignment") || bytes::equal(name,b"storage_take") || bytes::equal(name,b"storage_put") || bytes::equal(name,b"storage_borrow") || bytes::equal(name,b"storage_borrow_mut") || bytes::equal(name,b"storage_element_address");}
fn u32 binding(ref catalog::Catalog c,Context context,ref u8[] name,ref mut usize work){if(!budget::charge(work,name.len+1)){return 10;}if(name.len>=12 && bytes::equal(name[..12],b"_fy_generic_") || context.vector && storage(name)){return 4501;}let function=switch(catalog::lookup(c,context.module_index,name,3,work)){case .ok(value):{value}case .err(code):{return code;}};let aggregate=switch(catalog::lookup(c,context.module_index,name,0,work)){case .ok(value):{value}case .err(code):{return code;}};return if(function!=0 || aggregate!=0){4501}else{0};}
fn u32 typ(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,ref syntax::Parsed p,Context context,usize id,ref mut bytes::Buffer names,ref mut v::Vec<model::Argument> args,ref mut v::Vec<model::Edge> edges,ref mut v::Vec<model::Range> ranges,ref mut v::Vec<Task> pending,ref mut usize work){
    if(id>syntax::type_count(p)){return 9;}let n=syntax::type_at(p,id);if(n.kind==1 || n.kind==6){let base=if(n.kind==1){id}else{n.left};if(base==0 || base>syntax::type_count(p) || syntax::type_at(p,base).kind!=1){return 9;}let b=syntax::type_at(p,base);let name=switch(arguments::path(p,b.start,b.end,work)){case .ok(value):{value}case .err(code):{return code;}};
        if(switch(arguments::generated(bytes::view(ref name),work)){case .ok(value):{value}case .err(code):{return code;}}){return 4501;}
        // Builtin named types have no generic aggregate declaration to select.
        if(!arguments::scalar(bytes::view(ref name)) && !bytes::equal(bytes::view(ref name),b"void") && !bytes::equal(bytes::view(ref name),b"never") && !bytes::equal(bytes::view(ref name),b"str")){let code=edge(c,sources,p,context,bytes::view(ref name),if(n.kind==6){n.right}else{0},0,names,args,edges,ranges,work);if(code!=0){return code;}}
        return if(n.kind==6){list(p,n.right,7,pending,work)}else{0};
    }
    if(n.kind==2 || n.kind==3 || n.kind==4 || n.kind==5 || n.kind==8 || n.kind==11 || n.kind==12){return push(pending,1,n.left,work);}
    if(n.kind==15){let code=push(pending,1,n.left,work);return if(code!=0){code}else{list(p,n.right,16,pending,work)};}return 9;
}
fn u32 expr_list(ref syntax::Parsed p,usize tail,u32 kind,bool entries,ref mut v::Vec<Task> pending,ref mut usize work){var usize link=tail;while(link!=0){if(!budget::charge(work,1)){return 10;}if(link>syntax::expression_count(p)){return 9;}let n=syntax::expression_at(p,link);if(n.kind!=kind || n.left==0 || n.left>=link || n.right>=link){return 9;}let code=push(pending,2,if(entries){link}else{n.left},work);if(code!=0){return code;}link=n.right;}return 0;}
fn u32 expression(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,ref syntax::Parsed p,Context context,usize id,ref mut bytes::Buffer names,ref mut v::Vec<model::Argument> args,ref mut v::Vec<model::Edge> edges,ref mut v::Vec<model::Range> ranges,ref mut v::Vec<Task> pending,ref mut usize work){
    if(id>syntax::expression_count(p)){return 9;}let n=syntax::expression_at(p,id);
    if(n.kind==1 || n.kind==17 || n.kind==28 || n.kind==33){return 0;}if(n.kind==2){let name=switch(arguments::path(p,n.start,n.end,work)){case .ok(value):{value}case .err(code):{return code;}};return switch(arguments::generated(bytes::view(ref name),work)){case .ok(value):{if(value){4501}else{0}}case .err(code):{code}};}
    if(n.kind==12 || n.kind==31){return push(pending,3,n.left,work);}
    if(n.kind==13){if(n.right==0 || n.right>=id){return 9;}let branches=syntax::expression_at(p,n.right);if(branches.kind!=14){return 9;}let a=push(pending,2,n.left,work);if(a!=0){return a;}let b=push(pending,3,branches.left,work);return if(b!=0){b}else{push(pending,2,branches.right,work)};}
    if(n.kind==15){let code=push(pending,2,n.left,work);return if(code!=0){code}else{expr_list(p,n.right,19,false,pending,work)};}
    if(n.kind==18){let code=push(pending,2,n.left,work);return if(code!=0){code}else{push(pending,3,n.right,work)};}
    if(n.kind==16){return push(pending,2,n.left,work);}if(n.kind==32){return expr_list(p,n.right,33,true,pending,work);}
    if(n.kind==24){return expr_list(p,n.right,25,false,pending,work);}
    if(n.kind==20){if(n.left==0 || n.left>=id || syntax::expression_at(p,n.left).kind!=2){return 4501;}let base=syntax::expression_at(p,n.left);let name=switch(arguments::path(p,base.start,base.end,work)){case .ok(value):{value}case .err(code):{return code;}};if(!(context.vector && storage(bytes::view(ref name)))){let code=edge(c,sources,p,context,bytes::view(ref name),n.right,3,names,args,edges,ranges,work);if(code!=0){return code;}}
        let code=list(p,n.right,7,pending,work);return if(code!=0){code}else{push(pending,2,n.left,work)};
    }
    if(n.kind==10){if(n.left==0 || n.left>=id){return 9;}var usize base=n.left;let g=syntax::expression_at(p,base);if(g.kind==20){base=g.left;if(base==0 || base>=n.left || syntax::expression_at(p,base).kind!=2){return 4501;}let path=syntax::expression_at(p,base);let name=switch(arguments::path(p,path.start,path.end,work)){case .ok(value):{value}case .err(code):{return code;}};let target=switch(catalog::lookup(c,context.module_index,bytes::view(ref name),0,work)){case .ok(value):{value}case .err(code):{return code;}};if(target==0){return 4501;}let code=edge(c,sources,p,context,bytes::view(ref name),g.right,0,names,args,edges,ranges,work);if(code!=0){return code;}let arg_code=list(p,g.right,7,pending,work);if(arg_code!=0){return arg_code;}}
        let code=push(pending,2,base,work);return if(code!=0){code}else{expr_list(p,n.right,11,true,pending,work)};
    }
    if(n.kind==6){if(n.left==0 || n.left>=id){return 9;}let base=syntax::expression_at(p,n.left);if(base.kind==2){let name=switch(arguments::path(p,base.start,base.end,work)){case .ok(value):{value}case .err(code):{return code;}};let found=switch(catalog::lookup(c,context.module_index,bytes::view(ref name),3,work)){case .ok(value):{value}case .err(code):{return code;}};if(found!=0){return 4501;}}let code=push(pending,2,n.left,work);return if(code!=0){code}else{expr_list(p,n.right,7,false,pending,work)};}
    if(n.kind==26){if(n.right==0 || n.right>=id){return 9;}let bounds=syntax::expression_at(p,n.right);if(bounds.kind!=27){return 9;}let a=push(pending,2,n.left,work);if(a!=0){return a;}let b=push(pending,2,bounds.left,work);return if(b!=0){b}else{push(pending,2,bounds.right,work)};}
    if(n.kind==23){let code=push(pending,2,n.left,work);return if(code!=0){code}else{push(pending,1,n.right,work)};}
    if(n.kind==3 || n.kind==4 || n.kind==8 || n.kind==11 || n.kind==21 || n.kind==22 || n.kind==30){return push(pending,2,n.left,work);}
    if(n.kind==5 || n.kind==9 || n.kind==29){let code=push(pending,2,n.left,work);return if(code!=0){code}else{push(pending,2,n.right,work)};}return 9;
}
fn u32 statement(ref catalog::Catalog c,ref syntax::Parsed p,Context context,usize id,ref mut v::Vec<Task> pending,ref mut usize work){
    if(id>syntax::statement_count(p)){return 9;}let s=syntax::statement_at(p,id);
    if(s.kind==1){let code=push(pending,2,s.expression,work);if(code!=0){return code;}var usize link=s.first;while(link!=0){if(!budget::charge(work,1)){return 10;}if(link>syntax::statement_count(p)){return 9;}let n=syntax::statement_at(p,link);if(n.kind!=7 || n.first==0 || n.first>=link || n.second>=link){return 9;}let next=push(pending,3,n.first,work);if(next!=0){return next;}link=n.second;}return 0;}
    if(s.kind==2){if(s.entry==0 || s.entry>syntax::entry_count(p)){return 9;}let e=syntax::entry_at(p,s.entry-1);let code=binding(c,context,syntax::source_view(p)[e.name_start..e.name_end],work);if(code!=0){return code;}let value=push(pending,2,e.expression,work);return if(value!=0){value}else{push(pending,1,e.type_root,work)};}
    if(s.kind==9 || s.kind==10){return 0;}
    if(s.kind==14){if(s.second==0 || s.second>syntax::statement_count(p)){return 9;}let header=syntax::statement_at(p,s.second);if(header.kind!=15){return 9;}let a=push(pending,3,header.first,work);if(a!=0){return a;}let b=push(pending,3,header.second,work);if(b!=0){return b;}let block=push(pending,3,s.first,work);return if(block!=0){block}else{push(pending,2,s.expression,work)};}
    if(s.kind==8 || s.kind==17){let code=push(pending,3,s.first,work);return if(code!=0){code}else{push(pending,2,s.expression,work)};}
    if(s.kind==18){var json::Ledger ledger=json::empty();let admitted=loops::admitted(p,s,ref mut ledger);if(!budget::charge(work,ledger.used)){return 10;}if(ledger.code!=0){return ledger.code;}let code=push(pending,3,s.first,work);if(code!=0){return code;}if(!admitted){return 0;}let name=syntax::expression_at(p,s.target);let checked=binding(c,context,syntax::source_view(p)[name.start..name.end],work);if(checked!=0){return checked;}let value=push(pending,2,s.expression,work);return if(value!=0){value}else{push(pending,1,s.second,work)};}
    if(s.kind==3 || s.kind==4 || s.kind==6 || s.kind==11 || s.kind==12 || s.kind==13 || s.kind==16){let code=push(pending,2,s.expression,work);return if(code!=0 || s.kind!=4){code}else{push(pending,2,s.target,work)};}return 9;
}
fn u32 scan(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,usize declaration,Context context,ref mut bytes::Buffer names,ref mut v::Vec<model::Argument> args,ref mut v::Vec<model::Edge> edges,ref mut v::Vec<model::Range> ranges,ref mut usize work){
    let p=v::at<syntax::Parsed>(sources,context.module_index);let d=syntax::declaration_at(p,declaration);var v::Vec<Task> pending=v::empty<Task>();var u32 code=0;
    if(d.kind==3){code=push(ref mut pending,1,d.result_type,work);var usize i=d.entry_start;while(i<d.entry_start+d.entry_count && code==0){if(!budget::charge(work,1)){code=10;break;}let e=syntax::entry_at(p,i);if(e.kind==1){code=binding(c,context,syntax::source_view(p)[e.name_start..e.name_end],work);if(code==0){code=push(ref mut pending,1,e.type_root,work);}}i+=1;}if(code==0){code=push(ref mut pending,3,d.body,work);}}
    else{var usize i=d.field_start;while(i<d.field_start+d.field_count && code==0){if(!budget::charge(work,1)){code=10;break;}code=push(ref mut pending,1,syntax::field_at(p,i).type_root,work);i+=1;}if(code==0){code=push(ref mut pending,3,d.cleanup_body,work);}}
    while(v::len<Task>(ref pending)!=0 && code==0){if(!budget::charge(work,1)){code=10;break;}let task=*v::at<Task>(ref pending,v::len<Task>(ref pending)-1);discard v::pop<Task>(ref mut pending);
        code=if(task.kind==1){typ(c,sources,p,context,task.node,names,args,edges,ranges,ref mut pending,work)}else if(task.kind==2){expression(c,sources,p,context,task.node,names,args,edges,ranges,ref mut pending,work)}else{statement(c,p,context,task.node,ref mut pending,work)};
    }return code;
}
pub fn model::Problem collect(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,usize template_index,ref mut bytes::Buffer names,ref mut v::Vec<model::Argument> args,ref mut v::Vec<model::Edge> edges,ref mut v::Vec<model::Range> ranges,ref mut usize work){
    let row=catalog::template_at(c,template_index);let context=Context{.template_index,.module_index=row.module_index,.first=v::len<model::Edge>(edges),.vector=row.kind!=3 || !catalog::parameter_at(c,row.first).scalar,.root=false};var u32 code=scan(c,sources,row.declaration,context,names,args,edges,ranges,work);if(code==0){code=model::finish(edges,names,ranges,context.first,work);}return model::Problem{.code,.module_index=row.module_index,.declaration=row.declaration,.start=row.start,.end=row.end};
}
// Direct source roots reuse the executable projection, never a flat expression scan.
// Global deduplication preserves one exact declaration/ordered-concrete tuple.
pub fn model::Problem collect_root(ref catalog::Catalog c,ref v::Vec<syntax::Parsed> sources,usize module_index,usize declaration,bool vector,ref mut bytes::Buffer names,ref mut v::Vec<model::Argument> args,ref mut v::Vec<model::Edge> edges,ref mut v::Vec<model::Range> ranges,ref mut usize work){
    if(module_index>=v::len<syntax::Parsed>(sources) || declaration==0 || declaration>=syntax::declaration_count(v::at<syntax::Parsed>(sources,module_index))){return model::Problem{.code=9,.module_index,.declaration,.start=0,.end=0};}let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,module_index),declaration);if(d.generic_parameters!=0 || (d.kind!=3 && d.kind!=4 && d.kind!=6)){return model::Problem{.code=9,.module_index,.declaration,.start=d.start,.end=d.end};}
    let context=Context{.template_index=0,.module_index,.first=0,.vector,.root=true};let code=scan(c,sources,declaration,context,names,args,edges,ranges,work);return model::Problem{.code,.module_index,.declaration,.start=d.start,.end=d.end};
}
