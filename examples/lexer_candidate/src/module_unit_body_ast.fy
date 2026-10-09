module module_unit_body_ast;
import syntax as syntax;
import types as types;
import body_state as ast;
import bodies as entries;
import attributes as attrs;
import std::vec as v;

// Body-local arena IDs; spans and token IDs stay in the owned provider stream.
pub resource Ast {syntax::Declaration declaration;v::Vec<ast::Node> expressions;v::Vec<ast::Statement> statements;v::Vec<entries::Entry> entries;v::Vec<types::Node> types;v::Vec<usize> original_types;v::Vec<usize> original_expressions;v::Vec<attrs::Attribute> attributes;v::Vec<attrs::Argument> arguments;cleanup{}}
resource Maps {v::Vec<usize> expressions;v::Vec<usize> statements;v::Vec<usize> entries;v::Vec<usize> types;cleanup{}}
pub fn bool charge(ref mut usize work,usize amount){if(*work>1048576 || amount>1048576-*work){return false;};*work=*work+amount;return true;}
fn bool inside(usize start,usize end,syntax::Declaration d){return start>=d.start && end<=d.end;}
fn usize linked(ref v::Vec<usize> map,usize id,ref mut u32 code){if(id==0){return 0;}if(id>=v::len<usize>(map)){*code=9;return 0;}let usize result=*v::at<usize>(map,id);if(result==0){*code=9;}return result;}
fn usize indexed(ref v::Vec<usize> map,usize start,usize count,ref mut u32 code){if(count==0){return 0;}let usize first=linked(map,start+1,code);if(first==0){return 0;}var usize i=1;while(i<count){if(linked(map,start+i+1,code)!=first+i){*code=9;return 0;}i+=1;}return first-1;}
fn Result<v::Vec<usize>,u32> mapping(ref syntax::Parsed file,syntax::Declaration d,u32 kind,ref mut usize work){
    var v::Vec<usize> map=v::empty<usize>();if(!v::push<usize>(ref mut map,0).is_ok){return .err(11);}var usize next=0;
    let usize count=if(kind==1){syntax::expression_count(file)}else if(kind==2){syntax::statement_count(file)}else if(kind==3){syntax::entry_count(file)}else{syntax::type_count(file)};
    var usize i=1;while(i<=count){if(!charge(work,1)){return .err(10);}var usize start=0;var usize end=0;
        if(kind==1){let n=syntax::expression_at(file,i);start=n.start;end=n.end;}else if(kind==2){let n=syntax::statement_at(file,i);start=n.start;end=n.end;}
        else if(kind==3){let n=syntax::entry_at(file,i-1);start=n.start;end=n.end;}else{let n=syntax::type_at(file,i);start=n.start;end=n.end;}
        if(inside(start,end,d)){next+=1;}if(!v::push<usize>(ref mut map,if(inside(start,end,d)){next}else{0}).is_ok){return .err(11);}i+=1;
    }return .ok(map);
}
fn Result<Maps,u32> maps(ref syntax::Parsed file,syntax::Declaration d,ref mut usize work){
    let expressions=switch(mapping(file,d,1,work)){case .ok(m):{m}case .err(c):{return .err(c);}};
    let statements=switch(mapping(file,d,2,work)){case .ok(m):{m}case .err(c):{return .err(c);}};
    let entries=switch(mapping(file,d,3,work)){case .ok(m):{m}case .err(c):{return .err(c);}};
    let types=switch(mapping(file,d,4,work)){case .ok(m):{m}case .err(c):{return .err(c);}};return .ok(Maps{.expressions,.statements,.entries,.types});
}
fn ast::Node expression(syntax::Expression n,ref Maps m,ref mut u32 code){
    let usize left=if(n.kind==12 || n.kind==14 || n.kind==31){linked(ref m.statements,n.left,code)}else{linked(ref m.expressions,n.left,code)};
    let usize right=if(n.kind==18){linked(ref m.statements,n.right,code)}else if(n.kind==20 || n.kind==23){linked(ref m.types,n.right,code)}else{linked(ref m.expressions,n.right,code)};
    return ast::Node{.kind=n.kind,.start=n.start,.end=n.end,.token=n.token,.left,.right};
}
fn ast::Statement statement(syntax::Statement n,ref Maps m,ref mut u32 code){let usize second=if(n.kind==7 || n.kind==14 || n.kind==15){linked(ref m.statements,n.second,code)}else if(n.kind==18){linked(ref m.types,n.second,code)}else{n.second};
    return ast::Statement{.kind=n.kind,.start=n.start,.end=n.end,.expression=linked(ref m.expressions,n.expression,code),.target=linked(ref m.expressions,n.target,code),.first=linked(ref m.statements,n.first,code),.second,.entry=linked(ref m.entries,n.entry,code),.operator=n.operator};
}
pub fn Result<Ast,u32> build(ref syntax::Parsed file,usize declaration,ref mut usize work){
    if(syntax::problem_count(file)!=0 || declaration==0 || declaration>syntax::declaration_count(file)){return .err(9);}var syntax::Declaration d=syntax::declaration_at(file,declaration-1);
    if(d.kind!=3 || d.body==0 || d.field_count!=0 || d.cleanup_body!=0 || d.cleanup_attribute_count!=0){return .err(9);}
    let m=switch(maps(file,d,work)){case .ok(m):{m}case .err(c):{return .err(c);}};var u32 code=0;
    var v::Vec<ast::Node> expressions=v::empty<ast::Node>();var v::Vec<ast::Statement> statements=v::empty<ast::Statement>();var v::Vec<entries::Entry> entries=v::empty<entries::Entry>();var v::Vec<types::Node> types=v::empty<types::Node>();var v::Vec<usize> original_types=v::empty<usize>();var v::Vec<usize> original_expressions=v::empty<usize>();var v::Vec<attrs::Attribute> attributes=v::empty<attrs::Attribute>();var v::Vec<attrs::Argument> arguments=v::empty<attrs::Argument>();
    var usize i=1;while(i<=syntax::expression_count(file)){if(!charge(work,1)){return .err(10);}if(*v::at<usize>(ref m.expressions,i)!=0){if(!v::push<ast::Node>(ref mut expressions,expression(syntax::expression_at(file,i),ref m,ref mut code)).is_ok || !v::push<usize>(ref mut original_expressions,i).is_ok){return .err(11);}}i+=1;}
    i=1;while(i<=syntax::statement_count(file)){if(!charge(work,1)){return .err(10);}if(*v::at<usize>(ref m.statements,i)!=0 && !v::push<ast::Statement>(ref mut statements,statement(syntax::statement_at(file,i),ref m,ref mut code)).is_ok){return .err(11);}i+=1;}
    i=0;while(i<syntax::entry_count(file)){if(!charge(work,1)){return .err(10);}let usize id=*v::at<usize>(ref m.entries,i+1);if(id!=0){let n=syntax::entry_at(file,i);if(!v::push<entries::Entry>(ref mut entries,entries::Entry{.kind=n.kind,.start=n.start,.end=n.end,.name_start=n.name_start,.name_end=n.name_end,.type_start=n.type_start,.type_end=n.type_end,.type_root=linked(ref m.types,n.type_root,ref mut code),.expression=linked(ref m.expressions,n.expression,ref mut code),.id}).is_ok){return .err(11);}}i+=1;}
    i=1;while(i<=syntax::type_count(file)){if(!charge(work,1)){return .err(10);}if(*v::at<usize>(ref m.types,i)!=0){let n=syntax::type_at(file,i);if(!v::push<types::Node>(ref mut types,types::Node{.kind=n.kind,.start=n.start,.end=n.end,.left=linked(ref m.types,n.left,ref mut code),.right=linked(ref m.types,n.right,ref mut code)}).is_ok || !v::push<usize>(ref mut original_types,i).is_ok){return .err(11);}}i+=1;}
    i=d.attribute_start;while(i<d.attribute_start+d.attribute_count){if(!charge(work,1)){return .err(10);}var a=syntax::attribute_at(file,i);let usize first=v::len<attrs::Argument>(ref arguments);var usize j=a.argument_start;while(j<a.argument_start+a.argument_count){if(!charge(work,1)){return .err(10);}if(!v::push<attrs::Argument>(ref mut arguments,syntax::attribute_argument_at(file,j)).is_ok){return .err(11);}j+=1;}a.argument_start=first;if(!v::push<attrs::Attribute>(ref mut attributes,a).is_ok){return .err(11);}i+=1;}
    d.expression=linked(ref m.expressions,d.expression,ref mut code);d.entry_start=indexed(ref m.entries,d.entry_start,d.entry_count,ref mut code);d.body=linked(ref m.statements,d.body,ref mut code);d.result_type=linked(ref m.types,d.result_type,ref mut code);d.generic_parameters=0;d.attribute_start=0;
    if(code!=0){return .err(code);}return .ok(Ast{.declaration=d,.expressions,.statements,.entries,.types,.original_types,.original_expressions,.attributes,.arguments});
}
pub fn syntax::Declaration declaration(ref Ast p){return p.declaration;}
pub fn usize expression_count(ref Ast p){return v::len<ast::Node>(ref p.expressions);}
pub fn ast::Node expression_at(ref Ast p,usize id){return *v::at<ast::Node>(ref p.expressions,id-1);}
pub fn usize expression_origin(ref Ast p,usize id){return *v::at<usize>(ref p.original_expressions,id-1);}
pub fn usize statement_count(ref Ast p){return v::len<ast::Statement>(ref p.statements);}
pub fn ast::Statement statement_at(ref Ast p,usize id){return *v::at<ast::Statement>(ref p.statements,id-1);}
pub fn usize entry_count(ref Ast p){return v::len<entries::Entry>(ref p.entries);}
pub fn entries::Entry entry_at(ref Ast p,usize index){return *v::at<entries::Entry>(ref p.entries,index);}
pub fn usize type_count(ref Ast p){return v::len<types::Node>(ref p.types);}
pub fn types::Node type_at(ref Ast p,usize id){return *v::at<types::Node>(ref p.types,id-1);}
pub fn usize type_origin(ref Ast p,usize id){return *v::at<usize>(ref p.original_types,id-1);}
pub fn usize attribute_count(ref Ast p){return v::len<attrs::Attribute>(ref p.attributes);}
pub fn attrs::Attribute attribute_at(ref Ast p,usize index){return *v::at<attrs::Attribute>(ref p.attributes,index);}
pub fn usize argument_count(ref Ast p){return v::len<attrs::Argument>(ref p.arguments);}
pub fn attrs::Argument argument_at(ref Ast p,usize index){return *v::at<attrs::Argument>(ref p.arguments,index);}
