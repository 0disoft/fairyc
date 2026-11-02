module static_operation_syntax;
import static_operation_declarations as operations;
import syntax as syntax;
import body_state as nodes;
import scanner as scanner;
import std::vec as v;

// Private virtual path tokens preserve the operation's source span while naming
// its local ordinary provider, as in the Go-authored specialization wrapper.
pub fn usize token_flag(){return 1048576;}
fn usize token(ref syntax::Parsed p,usize start){var usize i=0;while(i<syntax::source_token_count(p)){if(scanner::token_info(syntax::source_stream(p),i).start==start){return i;}i+=1;}return token_flag();}
fn Result<usize,u32> expression(ref mut v::Vec<nodes::Node> table,nodes::Node n){if(v::len<nodes::Node>(table)>=16384){return .err(4503);}if(!v::push<nodes::Node>(table,n).is_ok){return .err(11);}return .ok(v::len<nodes::Node>(table));}
fn Result<usize,u32> statement(ref mut v::Vec<nodes::Statement> table,nodes::Statement n){if(v::len<nodes::Statement>(table)>=8192){return .err(4503);}if(!v::push<nodes::Statement>(table,n).is_ok){return .err(11);}return .ok(v::len<nodes::Statement>(table));}
pub fn bool needed(ref syntax::Parsed p){var usize i=1;while(i<syntax::declaration_count(p)){if(operations::operation(p,i+1).kind==1){return true;}i+=1;}return false;}
fn Result<usize,u32> inject(ref syntax::Parsed p,ref mut v::Vec<syntax::Declaration> declarations,ref mut v::Vec<nodes::Node> expressions,ref mut v::Vec<nodes::Statement> statements,usize declaration,usize provider){
    let d=syntax::declaration_at(p,declaration-1);let target=syntax::declaration_at(p,provider-1);let callee_token=token(p,target.name_start);if(callee_token==token_flag()){return .err(4501);}
    let callee=switch(expression(expressions,nodes::node_value(2,d.name_start,d.name_end,callee_token+token_flag(),0,0))){case .ok(n):{n}case .err(c):{return .err(c);}};var usize args=0;var usize i=d.entry_start;
    while(i<d.entry_start+d.entry_count){let entry=syntax::entry_at(p,i);if(entry.kind==1){let t=token(p,entry.name_start);if(t==token_flag()){return .err(4501);}let value=switch(expression(expressions,nodes::node_value(2,entry.name_start,entry.name_end,t,0,0))){case .ok(n):{n}case .err(c):{return .err(c);}};args=switch(expression(expressions,nodes::node_value(7,entry.name_start,entry.name_end,t,value,args))){case .ok(n):{n}case .err(c):{return .err(c);}};}i+=1;}
    let call=switch(expression(expressions,nodes::node_value(6,d.start,d.end,0,callee,args))){case .ok(n):{n}case .err(c):{return .err(c);}};
    let ret=switch(statement(statements,nodes::Statement{.kind=3,.start=d.start,.end=d.end,.expression=call,.target=0,.first=0,.second=0,.entry=0,.operator=0})){case .ok(n):{n}case .err(c):{return .err(c);}};
    let link=switch(statement(statements,nodes::Statement{.kind=7,.start=d.start,.end=d.end,.expression=0,.target=0,.first=ret,.second=0,.entry=0,.operator=0})){case .ok(n):{n}case .err(c):{return .err(c);}};
    let old=syntax::statement_at(p,d.body);let body=switch(statement(statements,nodes::Statement{.kind=1,.start=old.start,.end=old.end,.expression=0,.target=0,.first=link,.second=1,.entry=0,.operator=0})){case .ok(n):{n}case .err(c):{return .err(c);}};{let changed=v::at_mut<syntax::Declaration>(declarations,declaration-1);changed.body=body;}return .ok(callee);
}
pub fn u32 prepare(ref syntax::Parsed p,ref mut v::Vec<syntax::Declaration> declarations,ref mut v::Vec<nodes::Node> expressions,ref mut v::Vec<nodes::Statement> statements){
    if(syntax::problem_count(p)!=0){return 0;}var usize work=0;if(operations::validate(p,ref mut work)!=0){return 0;}var usize i=1;
    while(i<syntax::declaration_count(p)){let op=operations::operation(p,i+1);if(op.kind==1){var usize j=1;var usize selected=0;while(j<syntax::declaration_count(p) && selected==0){let provider=operations::provider(p,j+1);if(provider.kind==2 && provider.operation==i+1){selected=j+1;}j+=1;}if(selected!=0){let result=inject(p,declarations,expressions,statements,i+1,selected);switch(result){case .ok(_):{}case .err(code):{return code;}}}}i+=1;}return 0;
}
pub fn usize provider_token(ref syntax::Parsed p,usize provider){let d=syntax::declaration_at(p,provider-1);return token(p,d.name_start)+token_flag();}
