module sdk_ast_compact;
import sdk_reachability as reach;
import module_signatures as signatures;
import module_catalog as catalog;
import syntax as syntax;
import body_state as ast;
import bodies as bodies;
import types as types;
import records as records;
import attributes as attributes;
import std::vec as v;

resource Maps {v::Vec<usize> expressions;v::Vec<usize> statements;v::Vec<usize> entries;v::Vec<usize> types;cleanup{}}
fn catalog::Problem fail(u32 code,usize module_index){return catalog::Problem{.code,.module_index,.start=0,.end=0};}
fn usize linked(ref v::Vec<usize> map,usize id,ref mut u32 code){
    if(id==0){return 0;}if(id>=v::len<usize>(map)){*code=9;return 0;}let usize result=*v::at<usize>(map,id);if(result==0){*code=9;}return result;
}
fn usize indexed(ref v::Vec<usize> map,usize start,usize count,ref mut u32 code){
    if(count==0){return 0;}let usize first=linked(map,start+1,code);if(first==0){return 0;}
    var usize i=1;while(i<count){if(linked(map,start+i+1,code)!=first+i){*code=9;return 0;}i+=1;}return first-1;
}
fn Result<v::Vec<usize>,u32> mapping(ref syntax::Parsed file,ref reach::Plan plan,usize module_index,u32 kind,ref mut usize work){
    var v::Vec<usize> map=v::empty<usize>();if(!v::push<usize>(ref mut map,0).is_ok){return .err(11);}var usize next=0;
    let usize count=if(kind==1){syntax::expression_count(file)}else if(kind==2){syntax::statement_count(file)}else if(kind==3){syntax::entry_count(file)}else{syntax::type_count(file)};
    var usize previous=0;var usize i=1;while(i<=count){if(!reach::charge(work,1)){return .err(10);}var usize start=0;var usize end=0;
        if(kind==1){let n=syntax::expression_at(file,i);start=n.start;end=n.end;}else if(kind==2){let n=syntax::statement_at(file,i);start=n.start;end=n.end;}
        else if(kind==3){let n=syntax::entry_at(file,i-1);start=n.start;end=n.end;}else{let n=syntax::type_at(file,i);start=n.start;end=n.end;}
        let bool keep=reach::keep(plan,module_index,start,end,ref mut previous,work);if(*work>=1048576){return .err(10);}if(keep){next+=1;}
        if(!v::push<usize>(ref mut map,if(keep){next}else{0}).is_ok){return .err(11);}i+=1;
    }return .ok(map);
}
fn Result<Maps,u32> maps(ref syntax::Parsed file,ref reach::Plan plan,usize module_index,ref mut usize work){
    let expressions=switch(mapping(file,plan,module_index,1,work)){case .ok(m):{m}case .err(c):{return .err(c);}};
    let statements=switch(mapping(file,plan,module_index,2,work)){case .ok(m):{m}case .err(c):{return .err(c);}};
    let entries=switch(mapping(file,plan,module_index,3,work)){case .ok(m):{m}case .err(c):{return .err(c);}};
    let types=switch(mapping(file,plan,module_index,4,work)){case .ok(m):{m}case .err(c):{return .err(c);}};
    return .ok(Maps{.expressions,.statements,.entries,.types});
}
fn syntax::Declaration declaration(syntax::Declaration d,ref Maps m,ref mut u32 code){
    return syntax::Declaration{.cleanup_attribute_start=d.cleanup_attribute_start,.cleanup_attribute_count=d.cleanup_attribute_count,
        .unsafe_start=d.unsafe_start,.unsafe_end=d.unsafe_end,.owner_start=d.owner_start,.owner_end=d.owner_end,.kind=d.kind,.start=d.start,.end=d.end,.name_start=d.name_start,.name_end=d.name_end,
        .expression=linked(ref m.expressions,d.expression,code),.entry_start=indexed(ref m.entries,d.entry_start,d.entry_count,code),.entry_count=d.entry_count,.body=linked(ref m.statements,d.body,code),.result_type=linked(ref m.types,d.result_type,code),
        .field_start=d.field_start,.field_count=d.field_count,.public=d.public,.public_start=d.public_start,.public_end=d.public_end,.attribute_start=d.attribute_start,.attribute_count=d.attribute_count,
        .cleanup_start=d.cleanup_start,.cleanup_end=d.cleanup_end,.cleanup_body=linked(ref m.statements,d.cleanup_body,code),.generic_parameters=linked(ref m.types,d.generic_parameters,code),.generic_start=d.generic_start,.generic_end=d.generic_end};
}
fn ast::Node expression(syntax::Expression n,ref Maps m,ref mut u32 code){
    let usize left=if(n.kind==12 || n.kind==14 || n.kind==31){linked(ref m.statements,n.left,code)}else{linked(ref m.expressions,n.left,code)};
    let usize right=if(n.kind==18){linked(ref m.statements,n.right,code)}else if(n.kind==20 || n.kind==23){linked(ref m.types,n.right,code)}else{linked(ref m.expressions,n.right,code)};
    return ast::Node{.kind=n.kind,.start=n.start,.end=n.end,.token=n.token,.left,.right};
}
fn ast::Statement statement(syntax::Statement n,ref Maps m,ref mut u32 code){
    let usize second=if(n.kind==7 || n.kind==14 || n.kind==15){linked(ref m.statements,n.second,code)}else if(n.kind==18){linked(ref m.types,n.second,code)}else{n.second};
    return ast::Statement{.kind=n.kind,.start=n.start,.end=n.end,.expression=linked(ref m.expressions,n.expression,code),.target=linked(ref m.expressions,n.target,code),.first=linked(ref m.statements,n.first,code),.second,.entry=linked(ref m.entries,n.entry,code),.operator=n.operator};
}
fn bodies::Entry entry(syntax::Entry n,usize id,ref Maps m,ref mut u32 code){return bodies::Entry{.kind=n.kind,.start=n.start,.end=n.end,.name_start=n.name_start,.name_end=n.name_end,.type_start=n.type_start,.type_end=n.type_end,.type_root=linked(ref m.types,n.type_root,code),.expression=linked(ref m.expressions,n.expression,code),.id};}
fn types::Node type(types::Node n,ref Maps m,ref mut u32 code){return types::Node{.kind=n.kind,.start=n.start,.end=n.end,.left=linked(ref m.types,n.left,code),.right=linked(ref m.types,n.right,code)};}
fn records::Field field(records::Field n,ref Maps m,ref mut u32 code){return records::Field{.kind=n.kind,.start=n.start,.end=n.end,.name_start=n.name_start,.name_end=n.name_end,.type_root=linked(ref m.types,n.type_root,code),.key_type=linked(ref m.types,n.key_type,code),.public=n.public,.public_start=n.public_start,.public_end=n.public_end,.payload_start=n.payload_start,.payload_count=n.payload_count,.payload_open=n.payload_open,.payload_end=n.payload_end};}
fn Result<syntax::Parsed,u32> copy_file(ref syntax::Parsed file,ref reach::Plan plan,usize module_index,ref mut usize work){
    let m=switch(maps(file,plan,module_index,work)){case .ok(m):{m}case .err(c):{return .err(c);}};
    let source=switch(syntax::copy_source(file,work)){case .ok(s):{s}case .err(c):{return .err(c);}};
    var v::Vec<syntax::Declaration> declarations=v::empty<syntax::Declaration>();var v::Vec<syntax::Problem> problems=v::empty<syntax::Problem>();var v::Vec<ast::Node> expressions=v::empty<ast::Node>();var v::Vec<ast::Statement> statements=v::empty<ast::Statement>();var v::Vec<bodies::Entry> entries=v::empty<bodies::Entry>();var v::Vec<types::Node> types=v::empty<types::Node>();var v::Vec<records::Field> fields=v::empty<records::Field>();var v::Vec<attributes::Attribute> attributes=v::empty<attributes::Attribute>();var v::Vec<attributes::Argument> arguments=v::empty<attributes::Argument>();var u32 code=0;
    var usize i=0;while(i<syntax::declaration_count(file)){if(!reach::charge(work,1)){return .err(10);}if(reach::needed(plan,module_index,i) && !v::push<syntax::Declaration>(ref mut declarations,declaration(syntax::declaration_at(file,i),ref m,ref mut code)).is_ok){return .err(11);}i+=1;}
    i=1;while(i<=syntax::expression_count(file)){if(!reach::charge(work,1)){return .err(10);}if(*v::at<usize>(ref m.expressions,i)!=0 && !v::push<ast::Node>(ref mut expressions,expression(syntax::expression_at(file,i),ref m,ref mut code)).is_ok){return .err(11);}i+=1;}
    i=1;while(i<=syntax::statement_count(file)){if(!reach::charge(work,1)){return .err(10);}if(*v::at<usize>(ref m.statements,i)!=0 && !v::push<ast::Statement>(ref mut statements,statement(syntax::statement_at(file,i),ref m,ref mut code)).is_ok){return .err(11);}i+=1;}
    i=0;while(i<syntax::entry_count(file)){if(!reach::charge(work,1)){return .err(10);}let usize id=*v::at<usize>(ref m.entries,i+1);if(id!=0 && !v::push<bodies::Entry>(ref mut entries,entry(syntax::entry_at(file,i),id,ref m,ref mut code)).is_ok){return .err(11);}i+=1;}
    i=1;while(i<=syntax::type_count(file)){if(!reach::charge(work,1)){return .err(10);}if(*v::at<usize>(ref m.types,i)!=0 && !v::push<types::Node>(ref mut types,type(syntax::type_at(file,i),ref m,ref mut code)).is_ok){return .err(11);}i+=1;}
    i=0;while(i<syntax::field_count(file)){if(!reach::charge(work,1)){return .err(10);}if(!v::push<records::Field>(ref mut fields,field(syntax::field_at(file,i),ref m,ref mut code)).is_ok){return .err(11);}i+=1;}
    i=0;while(i<syntax::attribute_count(file)){if(!reach::charge(work,1)){return .err(10);}if(!v::push<attributes::Attribute>(ref mut attributes,syntax::attribute_at(file,i)).is_ok){return .err(11);}i+=1;}
    i=0;while(i<syntax::attribute_argument_count(file)){if(!reach::charge(work,1)){return .err(10);}if(!v::push<attributes::Argument>(ref mut arguments,syntax::attribute_argument_at(file,i)).is_ok){return .err(11);}i+=1;}
    if(code!=0){return .err(code);}return .ok(syntax::assemble(source,declarations,problems,expressions,entries,statements,types,fields,attributes,arguments));
}
// Authenticate first; retain source owners and compact SDK function arenas.
pub fn Result<v::Vec<syntax::Parsed>,catalog::Problem> compact(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){
    var usize work=0;let plan=switch(reach::build(sources,metadata,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var v::Vec<syntax::Parsed> output=v::empty<syntax::Parsed>();var usize i=0;while(i<v::len<syntax::Parsed>(sources)){
        let p=switch(copy_file(v::at<syntax::Parsed>(sources,i),ref plan,i,ref mut work)){case .ok(p):{p}case .err(c):{return .err(fail(c,i));}};
        if(!v::push<syntax::Parsed>(ref mut output,p).is_ok){return .err(fail(11,i));}i+=1;
    }return .ok(output);
}
