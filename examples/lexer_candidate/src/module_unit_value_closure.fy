module module_unit_value_closure;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_value_builtin as builtins;
import module_unit_interfaces as interfaces;
import module_types as types;
import copied_closure_tables as private;
import copied_closure_admission as admission;
import closure_value_type_source as closure_types;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn m::Problem ok(m::Frame frame){return m::fail(frame,0,0,0,0,0);}
fn bool i32_type(ref syntax::Parsed source,usize root){if(root==0 || root>syntax::type_count(source)){return false;}let t=syntax::type_at(source,root);return t.kind==1 && t.left==0 && t.right==0 && bytes::equal(syntax::type_text(source,root),b"i32");}
// Preparation preserves the original literal, header and binding entry. The
// private token alone cannot grant authority to an arbitrary helper or span.
fn Result<usize,m::Problem> original(ref syntax::Parsed source,m::Frame frame,m::Expression e,ref mut usize work){
    var usize found=0;var usize i=1;
    while(i<=syntax::expression_count(source)){if(!m::charge(work,1)){return .err(m::fail(frame,10,e.start,e.end,0,0));}let n=syntax::expression_at(source,i);if(n.kind==34 && n.start==e.start && n.end==e.end){if(found!=0){return .err(m::fail(frame,4502,e.start,e.end,0,0));}found=i;}i+=1;}
    // Admission is bounded to 64 expression nodes and 16 body locals; account
    // for its repeated bounded name checks without changing the unit limit.
    if(!m::charge(work,2048)){return .err(m::fail(frame,10,e.start,e.end,0,0));}if(found==0 || admission::validate(source,found)!=0){return .err(m::fail(frame,4502,e.start,e.end,0,0));}
    i=0;while(i<syntax::entry_count(source)){if(!m::charge(work,1)){return .err(m::fail(frame,10,e.start,e.end,0,0));}let entry=syntax::entry_at(source,i);if(entry.kind==2 && entry.expression==found && closure_types::signature(source,entry.type_root) && closure_types::root(source,entry.type_root)){return .ok(found);}i+=1;}
    return .err(m::fail(frame,4502,e.start,e.end,0,0));
}
// A written tail expression is normalized to a final return by preparation.
// All preceding source statement identities must still agree exactly.
fn bool body_matches(ref syntax::Parsed source,syntax::Expression literal,syntax::Declaration helper){
    let body=syntax::statement_at(source,literal.left);if(body.expression==0){return helper.body==literal.left;}
    if(helper.body==0 || helper.body>syntax::statement_count(source)){return false;}let lifted=syntax::statement_at(source,helper.body);
    if(lifted.kind!=1 || lifted.start!=body.start || lifted.end!=body.end || lifted.expression!=0 || lifted.second!=body.second+1){return false;}
    var usize i=0;while(i<body.second){if(syntax::block_statement(source,literal.left,i)!=syntax::block_statement(source,helper.body,i)){return false;}i+=1;}
    let last=syntax::statement_at(source,syntax::block_statement(source,helper.body,body.second));return last.kind==3 && last.start==body.start && last.end==body.end && last.expression==body.expression && last.target==0 && last.first==0 && last.second==0 && last.entry==0 && last.operator==0;
}
fn m::Problem signature(ref syntax::Parsed source,m::Frame frame,m::Expression e,usize declaration,syntax::Expression literal,usize captures,ref interfaces::Interfaces imported,usize i32,ref mut usize work){
    let helper=syntax::declaration_at(source,declaration-1);let header=syntax::expression_at(source,literal.right);
    if(helper.kind!=3 || helper.public || helper.unsafe_end!=0 || helper.owner_end!=0 || helper.generic_parameters!=0 || helper.start!=literal.start || helper.end!=literal.end || helper.result_type!=header.token || !i32_type(source,helper.result_type) || helper.entry_count!=captures+1 || helper.entry_start>syntax::entry_count(source) || helper.entry_count>syntax::entry_count(source)-helper.entry_start || !body_matches(source,literal,helper)){return m::fail(frame,4502,e.start,e.end,0,0);}
    var usize i=0;while(i<captures+1){if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}let written=syntax::entry_at(source,header.left+i-1);let lifted=syntax::entry_at(source,helper.entry_start+i);if(lifted.kind!=1 || lifted.type_root!=written.type_root || !i32_type(source,lifted.type_root) || lifted.name_start!=written.name_start || lifted.name_end!=written.name_end || lifted.expression!=0){return m::fail(frame,4502,lifted.start,lifted.end,0,0);}i+=1;}
    i=0;while(i<interfaces::callable_count(imported)){if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}let c=interfaces::callable_at(imported,i);
        if(c.module_index==frame.module_index && c.declaration==declaration){if(c.template || c.foreign || c.unsafe_call || c.effects!=0 || c.result!=i32 || c.parameter_count!=captures+1){return m::fail(frame,4502,e.start,e.end,0,0);}var usize j=0;while(j<c.parameter_count){if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}if(interfaces::parameter_at(imported,c.parameter_start+j)!=i32){return m::fail(frame,4502,e.start,e.end,0,0);}j+=1;}return ok(frame);}i+=1;
    }return m::fail(frame,12,e.start,e.end,0,0);
}
pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);if(e.source_index==0 || e.source_index>syntax::expression_count(source)){return m::fail(frame,12,e.start,e.end,0,0);}let node=syntax::expression_at(source,e.source_index);
    if(e.kind!=36 || e.state==2 || node.kind!=36 || node.start!=e.start || node.end!=e.end || node.left!=e.left || node.right!=e.right || node.left==0 || node.left>syntax::expression_count(source)){return m::fail(frame,4502,e.start,e.end,0,0);}let target=syntax::expression_at(source,node.left);
    if(target.kind!=2 || target.token<=private::helper_flag() || target.token>=private::snapshot_flag()){return m::fail(frame,4502,e.start,e.end,0,0);}let declaration=target.token-private::helper_flag();
    let callee=m::index(expressions,e.left);if(callee==0){return m::fail(frame,12,target.start,target.end,0,0);}let symbolic=*v::at<m::Expression>(expressions,callee-1);
    if(symbolic.kind!=2 || symbolic.state!=2 || symbolic.type!=0 || symbolic.binding!=0 || symbolic.start!=target.start || symbolic.end!=target.end || symbolic.left!=target.left || symbolic.right!=target.right){return m::fail(frame,4502,target.start,target.end,0,0);}
    if(declaration>syntax::declaration_count(source) || private::name(syntax::private_names(source),declaration).len==0 || e.target!=0 && e.target!=declaration){return m::fail(frame,4502,e.start,e.end,0,0);}
    let encoded=private::value_captures(syntax::private_names(source),declaration);if(encoded==0 || encoded>5){return m::fail(frame,4502,e.start,e.end,0,0);}let captures=encoded-1;
    let receipt=switch(original(source,frame,e,work)){case .ok(r):{r}case .err(p):{return p;}};let literal=syntax::expression_at(source,receipt);let header=syntax::expression_at(source,literal.right);
    if(node.token!=literal.token || header.right/129!=captures || header.right%129!=1){return m::fail(frame,4502,e.start,e.end,0,0);}let usize i32=m::atom(atoms,1);if(i32==0 || i32>types::count(graph)){return m::fail(frame,12,e.start,e.end,0,0);}let scalar_type=types::at(graph,i32);
    if(scalar_type.kind!=1 || scalar_type.scalar!=1 || scalar_type.module_index!=0 || scalar_type.declaration!=0 || scalar_type.first!=0 || scalar_type.second!=0){return m::fail(frame,12,e.start,e.end,0,i32);}let checked=signature(source,frame,e,declaration,literal,captures,imported,i32,work);if(checked.code!=0){return checked;}
    var usize[4] operands=[0;4];var usize count=0;var usize link=node.right;
    while(link!=0){if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}if(link>syntax::expression_count(source) || count>=captures){return m::fail(frame,4502,e.start,e.end,0,0);}let row=syntax::expression_at(source,link);
        if(row.kind!=7 || row.left==0 || row.left>syntax::expression_count(source) || row.right>=link){return m::fail(frame,4502,row.start,row.end,0,0);}operands[count]=row.left;count+=1;link=row.right;
    }if(count!=captures){return m::fail(frame,4502,e.start,e.end,0,0);}
    var bool complete=true;var usize previous=0;var usize i=0;
    while(i<captures){if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}let operand=syntax::expression_at(source,operands[captures-i-1]);let written=syntax::entry_at(source,header.left+i-1);
        if(operand.kind!=2 || operand.token<=private::snapshot_flag() || operand.token-private::snapshot_flag()>syntax::entry_count(source)){return m::fail(frame,4502,operand.start,operand.end,0,0);}let snapshot=operand.token-private::snapshot_flag();let local=syntax::entry_at(source,snapshot-1);
        if(snapshot<=previous || local.kind!=102 || local.name_start!=written.name_start || local.name_end!=written.name_end || local.type_root!=written.type_root || operand.start!=written.name_start || operand.end!=written.name_end || local.expression==0 || local.expression>syntax::expression_count(source)){return m::fail(frame,4502,operand.start,operand.end,0,0);}previous=snapshot;
        let initializer=syntax::expression_at(source,local.expression);if(initializer.kind!=2 || initializer.token>=private::helper_flag() || initializer.start!=written.name_start || initializer.end!=written.name_end){return m::fail(frame,4502,operand.start,operand.end,0,0);}
        let child=m::index(expressions,operands[captures-i-1]);if(child==0){return m::fail(frame,12,operand.start,operand.end,0,0);}let value=*v::at<m::Expression>(expressions,child-1);if(value.kind!=2 || value.binding==0 || value.start!=operand.start || value.end!=operand.end || value.left!=operand.left || value.right!=operand.right){return m::fail(frame,4502,operand.start,operand.end,0,0);}
        let p=scalar::evaluate(source,frame,child,i32,imported,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let actual=*v::at<m::Expression>(expressions,child-1);
        if(actual.state==2 || actual.type!=0 && actual.type!=i32){return m::fail(frame,1006,actual.start,actual.end,i32,actual.type);}if(actual.state!=1 || actual.type==0){complete=false;}i+=1;
    }
    // Hidden snapshot initializers are ordinary locals. Do not manufacture a
    // type for a pending name; the value walk forwards it in a later round.
    {let slot=v::at_mut<m::Expression>(expressions,id-1);slot.target=declaration;if(!complete){slot.state=0;slot.type=0;}}if(!complete){return ok(frame);}
    let type=switch(builtins::canonical(frame,e,types::Type{.kind=23,.scalar=0,.module_index=0,.declaration=0,.first=0,.second=0},graph,work)){case .ok(t):{t}case .err(p):{return p;}};
    if(e.type!=0 && e.type!=type || e.state==2){return m::fail(frame,4502,e.start,e.end,type,e.type);}builtins::record(expressions,id,expected,type,m::no_place());return ok(frame);
}
