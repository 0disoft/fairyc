module generic_expand;
import syntax as syntax;
import scanner as scanner;
import records as records;
import attributes as attributes;
import types as types;
import bodies as bodies;
import body_state as ast;
import type_state as scalar;
import std::bytes as bytes;
import std::vec as v;

// Build-local declaration identity, explicit ordered scalar arguments, bounded AST
// cloning. Original source and templates remain owned; no textual substitution.
pub struct Problem {u32 code;usize start;usize end;}
struct Instance {usize declaration;usize instance;usize count;u32[8] types;u8 active;}
struct Plan {Problem problem;usize work;usize expressions;usize statements;usize entries;usize types;}
fn bool fail(ref mut Plan plan,u32 code,usize start,usize end){if(plan.problem.code==0){plan.problem=Problem{.code,.start,.end};}return false;}
fn v::Vec<usize> zero(usize count){var v::Vec<usize> result=v::empty<usize>();while(v::len<usize>(ref result)<=count){if(!v::push<usize>(ref mut result,0).is_ok){return v::empty<usize>();}}return result;}
fn usize get(ref v::Vec<usize> map,usize id){return *v::at<usize>(map,id);}
fn usize mapped(ref v::Vec<usize> map,usize id){if(id==0 || id>=v::len<usize>(map)){return id;}let usize value=get(map,id);return if(value==0){id}else{value};}
fn void set(ref mut v::Vec<usize> map,usize id,usize value){let ref mut usize slot=v::at_mut<usize>(map,id);*slot=value;}
fn ast::Node expression_at(ref v::Vec<ast::Node> nodes,usize id){return *v::at<ast::Node>(nodes,id-1);}
fn types::Node type_at(ref v::Vec<types::Node> nodes,usize id){return *v::at<types::Node>(nodes,id-1);}
fn usize argument_count(ref v::Vec<types::Node> nodes,usize root){var usize at=root;var usize count=0;while(at!=0){count+=1;at=type_at(nodes,at).right;}return count;}
fn usize argument(ref v::Vec<types::Node> nodes,usize root,usize index){var usize at=root;var usize remaining=argument_count(nodes,root)-1;while(remaining>index){at=type_at(nodes,at).right;remaining-=1;}return type_at(nodes,at).left;}
fn u32 scalar_type(ref u8[] name){return if(bytes::equal(name,b"i32")){1}else if(bytes::equal(name,b"i64")){2}else if(bytes::equal(name,b"u8")){3}else if(bytes::equal(name,b"u32")){4}else if(bytes::equal(name,b"usize")){5}else if(bytes::equal(name,b"bool")){6}else{7};}
fn bool inside(usize start,usize end,syntax::Declaration d){return start>=d.start && end<=d.end;}
fn usize parameter(ref u8[] source,ref v::Vec<types::Node> types,syntax::Declaration d,ref u8[] name){var usize i=0;while(i<argument_count(types,d.generic_parameters)){let n=type_at(types,argument(types,d.generic_parameters,i));if(bytes::equal(source[n.start..n.end],name)){return i+1;}i+=1;}return 0;}
fn usize instance(ref u8[] source,ref mut v::Vec<syntax::Declaration> declarations,ref mut v::Vec<ast::Node> expressions,ref mut v::Vec<ast::Statement> statements,ref mut v::Vec<bodies::Entry> entries,ref mut v::Vec<types::Node> types,ref mut v::Vec<Instance> instances,ref mut Plan plan,usize id,usize depth){
    let call=expression_at(expressions,id);let base=expression_at(expressions,call.left);
    if(depth>=32){discard fail(plan,4503,call.start,call.end);return 0;}
    var usize declaration=0;var usize i=0;
    while(i<v::len<syntax::Declaration>(declarations)){let d=*v::at<syntax::Declaration>(declarations,i);if(d.generic_parameters!=0 && bytes::equal(source[d.name_start..d.name_end],source[base.start..base.end])){declaration=i+1;break;}i+=1;}
    if(declaration==0){discard fail(plan,4501,call.start,call.end);return 0;}
    let d=*v::at<syntax::Declaration>(declarations,declaration-1);let usize count=argument_count(types,call.right);
    if(count==0 || count>8 || count!=argument_count(types,d.generic_parameters)){discard fail(plan,4501,call.start,call.end);return 0;}
    var u32[8] args=[0;8];var usize[8] roots=[0;8];i=0;
    while(i<count){let usize root=argument(types,call.right,i);let u32 type=scalar_type(source[type_at(types,root).start..type_at(types,root).end]);if(type_at(types,root).kind!=1 || type==7){discard fail(plan,4502,call.start,call.end);return 0;}args[i]=type;roots[i]=root;i+=1;}
    i=0;var usize per=0;
    while(i<v::len<Instance>(instances)){let old=*v::at<Instance>(instances,i);if(old.declaration==declaration){per+=1;var bool same=old.count==count;var usize n=0;while(n<count){same=same && old.types[n]==args[n];n+=1;}if(same){if(old.active==1){discard fail(plan,4503,call.start,call.end);return 0;}return old.instance;}}i+=1;}
    if(v::len<Instance>(instances)>=64 || per>=16 || v::len<syntax::Declaration>(declarations)>=512){discard fail(plan,4503,call.start,call.end);return 0;}
    let usize slot=v::len<Instance>(instances);let usize target=v::len<syntax::Declaration>(declarations)+1;
    if(!v::push<Instance>(instances,Instance{.declaration,.instance=target,.count,.types=args,.active=1}).is_ok || !v::push<syntax::Declaration>(declarations,d).is_ok){discard fail(plan,5,call.start,call.end);return 0;}
    let usize ne=plan.expressions;let usize ns=plan.statements;let usize nt=plan.types;let usize nb=plan.entries;
    plan.work+=ne+ns+nt+nb;if(plan.work>1048576){discard fail(plan,4503,call.start,call.end);return 0;}
    var v::Vec<usize> expression_map=zero(ne);var v::Vec<usize> statement_map=zero(ns);var v::Vec<usize> entry_map=zero(nb);var v::Vec<usize> type_map=zero(nt);
    if(v::len<usize>(ref expression_map)==0 || v::len<usize>(ref statement_map)==0 || v::len<usize>(ref entry_map)==0 || v::len<usize>(ref type_map)==0){discard fail(plan,5,call.start,call.end);return 0;}
    i=1;while(i<=nt){let node=type_at(types,i);if(inside(node.start,node.end,d)){
        let usize param=if(node.kind==1 || node.kind==13){parameter(source,types,d,source[node.start..node.end])}else{0};
        if(param!=0){set(ref mut type_map,i,roots[param-1]);}
        else{var types::Node copy=node;copy.left=mapped(ref type_map,node.left);copy.right=mapped(ref type_map,node.right);if(v::len<types::Node>(types)>=4096 || !v::push<types::Node>(types,copy).is_ok){discard fail(plan,4503,call.start,call.end);return 0;}set(ref mut type_map,i,v::len<types::Node>(types));}
    }i+=1;}
    i=1;while(i<=ne){let node=*v::at<ast::Node>(expressions,i-1);if(inside(node.start,node.end,d)){if(v::len<ast::Node>(expressions)>=4096 || !v::push<ast::Node>(expressions,node).is_ok){discard fail(plan,4503,call.start,call.end);return 0;}set(ref mut expression_map,i,v::len<ast::Node>(expressions));}i+=1;}
    i=1;while(i<=ns){let node=*v::at<ast::Statement>(statements,i-1);if(inside(node.start,node.end,d)){if(v::len<ast::Statement>(statements)>=8192 || !v::push<ast::Statement>(statements,node).is_ok){discard fail(plan,4503,call.start,call.end);return 0;}set(ref mut statement_map,i,v::len<ast::Statement>(statements));}i+=1;}
    let usize entry_start=v::len<bodies::Entry>(entries);i=d.entry_start+1;
    while(i<=d.entry_start+d.entry_count){var usize at=0;while(at<nb && v::at<bodies::Entry>(entries,at).id!=i){at+=1;}if(at==nb){discard fail(plan,4501,call.start,call.end);return 0;}var bodies::Entry node=*v::at<bodies::Entry>(entries,at);node.id=v::len<bodies::Entry>(entries)+1;node.type_root=mapped(ref type_map,node.type_root);node.expression=mapped(ref expression_map,node.expression);if(v::len<bodies::Entry>(entries)>=1024 || !v::push<bodies::Entry>(entries,node).is_ok){discard fail(plan,4503,call.start,call.end);return 0;}set(ref mut entry_map,i,node.id);i+=1;}
    i=1;while(i<=ne){let usize target_node=get(ref expression_map,i);if(target_node!=0){var ast::Node node=*v::at<ast::Node>(expressions,target_node-1);
        if(node.kind==12 || node.kind==14){node.left=mapped(ref statement_map,node.left);}else{node.left=mapped(ref expression_map,node.left);}
        if(node.kind==18){node.right=mapped(ref statement_map,node.right);}else if(node.kind==20 || node.kind==23){node.right=mapped(ref type_map,node.right);}else{node.right=mapped(ref expression_map,node.right);}
        let ref mut ast::Node cell=v::at_mut<ast::Node>(expressions,target_node-1);*cell=node;
    }i+=1;}
    i=1;while(i<=ns){let usize target_node=get(ref statement_map,i);if(target_node!=0){var ast::Statement node=*v::at<ast::Statement>(statements,target_node-1);node.expression=mapped(ref expression_map,node.expression);node.target=mapped(ref expression_map,node.target);node.entry=mapped(ref entry_map,node.entry);node.first=mapped(ref statement_map,node.first);if(node.kind!=1){node.second=mapped(ref statement_map,node.second);}let ref mut ast::Statement cell=v::at_mut<ast::Statement>(statements,target_node-1);*cell=node;}i+=1;}
    var syntax::Declaration concrete=d;concrete.generic_parameters=0;concrete.generic_start=declaration;concrete.generic_end=slot+1;concrete.result_type=mapped(ref type_map,d.result_type);concrete.entry_start=entry_start;concrete.body=mapped(ref statement_map,d.body);
    {let ref mut syntax::Declaration cell=v::at_mut<syntax::Declaration>(declarations,target-1);*cell=concrete;}
    i=1;while(i<=ne){let usize target_node=get(ref expression_map,i);if(target_node!=0 && expression_at(expressions,target_node).kind==20){if(!request(source,declarations,expressions,statements,entries,types,instances,plan,target_node,depth+1)){return 0;}}i+=1;}
    {let ref mut Instance cell=v::at_mut<Instance>(instances,slot);cell.active=2;}return target;
}
fn bool request(ref u8[] source,ref mut v::Vec<syntax::Declaration> declarations,ref mut v::Vec<ast::Node> expressions,ref mut v::Vec<ast::Statement> statements,ref mut v::Vec<bodies::Entry> entries,ref mut v::Vec<types::Node> types,ref mut v::Vec<Instance> instances,ref mut Plan plan,usize id,usize depth){let usize target=instance(source,declarations,expressions,statements,entries,types,instances,plan,id,depth);if(target==0){return false;}let node=expression_at(expressions,id);let base=expression_at(expressions,node.left);let ref mut ast::Node cell=v::at_mut<ast::Node>(expressions,id-1);*cell=ast::Node{.kind=2,.start=base.start,.end=base.end,.token=base.token,.left=0,.right=target};return true;}
fn Problem prepare(ref u8[] source,ref mut v::Vec<syntax::Declaration> declarations,ref mut v::Vec<ast::Node> expressions,ref mut v::Vec<ast::Statement> statements,ref mut v::Vec<bodies::Entry> entries,ref mut v::Vec<types::Node> types,ref mut v::Vec<Instance> instances,ref mut Plan plan){
let usize originals=v::len<syntax::Declaration>(declarations);let usize expression_limit=v::len<ast::Node>(expressions);var usize i=0;
    while(i<originals){let d=*v::at<syntax::Declaration>(declarations,i);if(d.generic_parameters!=0){
        if(d.kind!=3 || d.owner_end!=0 || d.attribute_count!=0 || argument_count(types,d.generic_parameters)>8){discard fail(plan,4501,d.start,d.end);return plan.problem;}
        var usize j=0;while(j<originals){let other=*v::at<syntax::Declaration>(declarations,j);if(j!=i && bytes::equal(source[d.name_start..d.name_end],source[other.name_start..other.name_end])){discard fail(plan,2,d.name_start,d.name_end);return plan.problem;}j+=1;}
        j=0;while(j<v::len<bodies::Entry>(entries)){let entry=*v::at<bodies::Entry>(entries,j);if(entry.name_end>entry.name_start && bytes::equal(source[d.name_start..d.name_end],source[entry.name_start..entry.name_end])){discard fail(plan,4501,entry.name_start,entry.name_end);return plan.problem;}j+=1;}
        j=0;while(j<argument_count(types,d.generic_parameters)){let a=type_at(types,argument(types,d.generic_parameters,j));var usize n=0;while(n<j){let b=type_at(types,argument(types,d.generic_parameters,n));if(bytes::equal(source[a.start..a.end],source[b.start..b.end])){discard fail(plan,4501,a.start,a.end);return plan.problem;}n+=1;}j+=1;}
    }i+=1;}
    i=1;while(i<=expression_limit){let node=expression_at(expressions,i);if(node.kind==20){var bool template=false;var usize n=0;while(n<originals){let d=*v::at<syntax::Declaration>(declarations,n);template=template || d.generic_parameters!=0 && inside(node.start,node.end,d);n+=1;}if(!template && !request(source,declarations,expressions,statements,entries,types,instances,plan,i,0)){return plan.problem;}}i+=1;}
    i=0;while(i<originals){let d=*v::at<syntax::Declaration>(declarations,i);if(d.generic_parameters!=0){let ref mut syntax::Declaration cell=v::at_mut<syntax::Declaration>(declarations,i);cell.kind=12;}i+=1;}return plan.problem;
}

// Clone only for a source set containing templates or explicit requests.
pub fn syntax::Parsed expand(syntax::Parsed p){
    var bool needed=false;var usize i=0;while(i<syntax::declaration_count(ref p)){needed=needed || syntax::declaration_at(ref p,i).generic_parameters!=0;i+=1;}i=1;while(i<=syntax::expression_count(ref p)){needed=needed || syntax::expression_at(ref p,i).kind==20;i+=1;}if(!needed){return p;}
    var bytes::Buffer input=bytes::empty();assert(bytes::append(ref mut input,syntax::source_view(ref p)).is_ok);
    let scanner::Stream stream=switch(scanner::collect(input,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{return p;}};
    var v::Vec<syntax::Declaration> declarations=v::empty<syntax::Declaration>();var v::Vec<ast::Node> expressions=v::empty<ast::Node>();var v::Vec<ast::Statement> statements=v::empty<ast::Statement>();var v::Vec<bodies::Entry> entries=v::empty<bodies::Entry>();var v::Vec<types::Node> types=v::empty<types::Node>();var v::Vec<syntax::Problem> problems=v::empty<syntax::Problem>();var v::Vec<records::Field> fields=v::empty<records::Field>();var v::Vec<attributes::Attribute> attributes=v::empty<attributes::Attribute>();var v::Vec<attributes::Argument> attribute_arguments=v::empty<attributes::Argument>();
    i=0;while(i<syntax::declaration_count(ref p)){assert(v::push<syntax::Declaration>(ref mut declarations,syntax::declaration_at(ref p,i)).is_ok);i+=1;}
    i=1;while(i<=syntax::expression_count(ref p)){let n=syntax::expression_at(ref p,i);assert(v::push<ast::Node>(ref mut expressions,ast::Node{.kind=n.kind,.start=n.start,.end=n.end,.token=n.token,.left=n.left,.right=n.right}).is_ok);i+=1;}
    i=1;while(i<=syntax::statement_count(ref p)){let n=syntax::statement_at(ref p,i);assert(v::push<ast::Statement>(ref mut statements,ast::Statement{.kind=n.kind,.start=n.start,.end=n.end,.expression=n.expression,.target=n.target,.first=n.first,.second=n.second,.entry=n.entry,.operator=n.operator}).is_ok);i+=1;}
    i=0;while(i<syntax::entry_count(ref p)){let n=syntax::entry_at(ref p,i);assert(v::push<bodies::Entry>(ref mut entries,bodies::Entry{.kind=n.kind,.start=n.start,.end=n.end,.name_start=n.name_start,.name_end=n.name_end,.type_start=n.type_start,.type_end=n.type_end,.type_root=n.type_root,.expression=n.expression,.id=i+1}).is_ok);i+=1;}
    i=1;while(i<=syntax::type_count(ref p)){assert(v::push<types::Node>(ref mut types,syntax::type_at(ref p,i)).is_ok);i+=1;}
    i=0;while(i<syntax::field_count(ref p)){assert(v::push<records::Field>(ref mut fields,syntax::field_at(ref p,i)).is_ok);i+=1;}
    i=0;while(i<syntax::attribute_count(ref p)){assert(v::push<attributes::Attribute>(ref mut attributes,syntax::attribute_at(ref p,i)).is_ok);i+=1;}
    i=0;while(i<syntax::attribute_argument_count(ref p)){assert(v::push<attributes::Argument>(ref mut attribute_arguments,syntax::attribute_argument_at(ref p,i)).is_ok);i+=1;}
    var v::Vec<Instance> instances=v::empty<Instance>();var Plan plan=Plan{.problem=Problem{.code=0,.start=0,.end=0},.work=0,.expressions=v::len<ast::Node>(ref expressions),.statements=v::len<ast::Statement>(ref statements),.entries=v::len<bodies::Entry>(ref entries),.types=v::len<types::Node>(ref types)};let Problem result=prepare(syntax::source_view(ref p),ref mut declarations,ref mut expressions,ref mut statements,ref mut entries,ref mut types,ref mut instances,ref mut plan);
    if(result.code!=0){assert(v::push<syntax::Problem>(ref mut problems,syntax::Problem{.code=result.code,.start=result.start,.end=result.end,.expected=0,.token=0}).is_ok);}
    return syntax::assemble(stream,declarations,problems,expressions,entries,statements,types,fields,attributes,attribute_arguments);
}
