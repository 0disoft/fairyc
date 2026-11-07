module main;
import module_unit_body_ast as ast;
import module_unit_ordinary_names as ordinary;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_value_model as model;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

struct Metrics{usize arena;usize selected;usize projection;usize names;usize values;usize expressions;usize locals;usize returns;}
@effects(console) fn void check(usize id,bool ok){if(!ok){discard io::write_stderr("ordinary AST reuse check ");discard io::write_stderr_usize(id);discard io::write_stderr("\n");assert(false);}}
@effects(console) fn void add(ref mut bytes::Buffer raw,ref u8[] text){check(1,bytes::append(raw,text).is_ok);}
// The generic filler is never instantiated. Both inputs select the same 80
// ordinary functions; only the unrelated source arena changes.
@effects(console) fn syntax::Parsed source(usize filler){
 var bytes::Buffer raw=bytes::empty();add(ref mut raw,b"module root;");var usize i=0;
 while(i<80){if(i==40){add(ref mut raw,b"fn void filler<T>(){");var usize j=0;while(j<filler){add(ref mut raw,b"discard(1+1);");j+=1;}add(ref mut raw,b"}");}add(ref mut raw,b"fn i32 f");check(2,bytes::append_usize(ref mut raw,i).is_ok);add(ref mut raw,b"(i32 value){let i32 local=value;return local;}");i+=1;}
 check(3,bytes::len(ref raw)<65536);let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{check(4,false);return source(filler);}};
 let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{check(5,false);return source(filler);}};check(6,syntax::problem_count(ref p)==0 && syntax::declaration_count(ref p)==82 && !syntax::closure_prepared(ref p));return p;
}
fn usize arena(ref syntax::Parsed p){return syntax::expression_count(p)+syntax::statement_count(p)+syntax::entry_count(p)+syntax::type_count(p);}
fn usize selected(ref ast::Ast a){return ast::expression_count(a)+ast::statement_count(a)+ast::entry_count(a)+ast::type_count(a);}
// Compare all projected links with a separately executed public AST builder,
// and compare original IDs/spans/tokens directly with the parsed source.
@effects(console) fn void same(ref syntax::Parsed file,ref ordinary::Function function,ref ast::Ast expected){
 let a=ordinary::body_ast(function);let d=ast::declaration(a);let e=ast::declaration(expected);check(10,d.kind==e.kind && d.start==e.start && d.end==e.end && d.name_start==e.name_start && d.name_end==e.name_end && d.expression==e.expression && d.entry_start==e.entry_start && d.entry_count==e.entry_count && d.body==e.body && d.result_type==e.result_type && d.cleanup_body==e.cleanup_body && d.generic_parameters==e.generic_parameters);
 check(11,ast::expression_count(a)==ast::expression_count(expected) && ast::statement_count(a)==ast::statement_count(expected) && ast::entry_count(a)==ast::entry_count(expected) && ast::type_count(a)==ast::type_count(expected));var usize i=1;
 while(i<=ast::expression_count(a)){let n=ast::expression_at(a,i);let x=ast::expression_at(expected,i);let id=ast::expression_origin(a,i);let original=syntax::expression_at(file,id);check(12,id==ordinary::expression_origin(function,i) && id==ast::expression_origin(expected,i) && n.kind==x.kind && n.start==x.start && n.end==x.end && n.token==x.token && n.left==x.left && n.right==x.right);check(13,n.kind==original.kind && n.start==original.start && n.end==original.end && n.token==original.token);if(i>1){check(14,ast::expression_origin(a,i-1)<id);}i+=1;}
 i=1;while(i<=ast::statement_count(a)){let n=ast::statement_at(a,i);let x=ast::statement_at(expected,i);check(15,n.kind==x.kind && n.start==x.start && n.end==x.end && n.expression==x.expression && n.target==x.target && n.first==x.first && n.second==x.second && n.entry==x.entry && n.operator==x.operator);i+=1;}
 i=1;while(i<=ast::entry_count(a)){let n=ast::entry_at(a,i-1);let x=ast::entry_at(expected,i-1);let id=ast::entry_origin(a,i);let original=syntax::entry_at(file,id-1);check(16,id==ordinary::entry_origin(function,i) && id==ast::entry_origin(expected,i) && n.kind==x.kind && n.start==x.start && n.end==x.end && n.name_start==x.name_start && n.name_end==x.name_end && n.type_start==x.type_start && n.type_end==x.type_end && n.type_root==x.type_root && n.expression==x.expression && n.id==x.id);check(17,n.name_start==original.name_start && n.name_end==original.name_end);i+=1;}
 i=1;while(i<=ast::type_count(a)){let n=ast::type_at(a,i);let x=ast::type_at(expected,i);let id=ast::type_origin(a,i);let original=syntax::type_at(file,id);check(18,id==ast::type_origin(expected,i) && n.kind==x.kind && n.start==x.start && n.end==x.end && n.left==x.left && n.right==x.right && n.kind==original.kind && n.start==original.start && n.end==original.end);i+=1;}
}
@effects(console) fn Metrics exercise(usize filler){
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();check(20,v::push<syntax::Parsed>(ref mut sources,source(filler)).is_ok);
 let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(p):{discard io::write_stderr_u32(p.code);check(21,false);return exercise(filler);}};
 let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(p):{discard io::write_stderr_u32(p.code);check(22,false);return exercise(filler);}};
 let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{discard io::write_stderr_u32(p.code);check(23,false);return exercise(filler);}};
 let direct=switch(ordinary::build(ref sources,ref metadata,ref bound,ref imported)){case .ok(p):{p}case .err(p):{discard io::write_stderr_u32(p.code);check(24,false);return exercise(filler);}};
 let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(p):{discard io::write_stderr_u32(p.code);check(25,false);return exercise(filler);}};
 let plan=closure::original_names(ref ready);check(26,ordinary::count(plan)==80 && ordinary::count(ref direct)==80 && ordinary::work(plan)==ordinary::work(ref direct) && ordinary::symbols(plan)==160 && ordinary::references(plan)==160);
 let file=v::at<syntax::Parsed>(ref sources,0);let n=arena(file);var usize projection=0;var usize retained=0;var usize i=1;
 while(i<=80){let f=ordinary::at(plan,i);let other=ordinary::at(ref direct,i);let declaration=if(i<=40){i+1}else{i+2};check(27,ordinary::module_index(f)==0 && ordinary::declaration(f)==declaration && ordinary::declaration(other)==declaration);
  let before=projection;let fresh=switch(ast::build(file,declaration,ref mut projection)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);check(28,false);return exercise(filler);}};same(file,f,ref fresh);same(file,other,ref fresh);let count=selected(ref fresh);check(29,projection-before==n+count);retained+=count;
  if(i==80){check(30,ast::expression_origin(ordinary::body_ast(f),1)>syntax::expression_count(file)/2);}i+=1;
 }
 check(31,projection==80*n+retained && projection<1048576 && ordinary::work(plan)<1048576);
 let last=ordinary::body_ast(ordinary::at(plan,80));let last_entries=ast::entry_count(last);let last_origin=ast::expression_origin(last,ast::expression_count(last));
 let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(p):{discard io::write_stderr_u32(p.code);check(32,false);return exercise(filler);}};
 check(33,values::count(ref typed)==80 && values::locals(ref typed)==160 && values::returns(ref typed)==80 && values::checked_returns(ref typed)==80 && values::unresolved(ref typed)==0 && values::work(ref typed)<1048576);i=0;
 while(i<values::count(ref typed)){let body=values::at(ref typed,i);let frame=model::frame(body);let f=ordinary::at(plan,i+1);let a=ordinary::body_ast(f);check(34,frame.kind==3 && frame.context==i+1 && frame.module_index==0 && frame.declaration==ordinary::declaration(f) && model::expression_count(body)==ast::expression_count(a) && model::local_count(body)==2 && model::return_count(body)==1 && model::return_at(body,0).checked);var usize j=1;
  while(j<=model::expression_count(body)){let e=model::expression_at(body,j);let id=ast::expression_origin(a,j);let old=syntax::expression_at(file,id);check(35,e.source_index==id && e.kind==old.kind && e.start==old.start && e.end==old.end && e.left==old.left && e.right==old.right && e.type!=0 && values::type_at(ref typed,e.type).scalar==1);j+=1;}
  j=1;while(j<=model::local_count(body)){let local=model::local_at(body,j);let entry=syntax::entry_at(file,local.source_index-1);check(36,local.source_index==ast::entry_origin(a,j) && local.start==entry.name_start && local.end==entry.name_end && local.initializer==entry.expression && local.type!=0 && values::type_at(ref typed,local.type).scalar==1);j+=1;}
  let result=model::return_at(body,0);check(37,result.expression==ast::expression_origin(a,ast::expression_count(a)) && result.expected==frame.result && result.actual==frame.result);i+=1;
 }
 // Read the retained owner again after Values finishes; it was borrowed, not
 // consumed or replaced by another projection.
 let after=ordinary::body_ast(ordinary::at(plan,80));check(38,ordinary::count(plan)==80 && ast::entry_count(after)==last_entries && ast::expression_origin(after,ast::expression_count(after))==last_origin);
 return Metrics{.arena=n,.selected=retained,.projection,.names=ordinary::work(plan),.values=values::work(ref typed),.expressions=values::expressions(ref typed),.locals=values::locals(ref typed),.returns=values::returns(ref typed)};
}
@effects(console) fn void report(ref str name,Metrics m){discard io::write(name);discard io::write(" functions=80 arena=");discard io::write_usize(m.arena);discard io::write(" selected=");discard io::write_usize(m.selected);discard io::write(" projection=");discard io::write_usize(m.projection);discard io::write(" names=");discard io::write_usize(m.names);discard io::write(" values=");discard io::write_usize(m.values);discard io::write(" expressions=");discard io::write_usize(m.expressions);discard io::write(" locals=");discard io::write_usize(m.locals);discard io::write(" returns=");discard io::write_usize(m.returns);discard io::write("\n");}
@effects(console) fn i32 main(){let small=exercise(0);let large=exercise(1800);check(40,large.arena>8000 && large.arena>small.arena && large.selected==small.selected && large.expressions==small.expressions && large.locals==small.locals && large.returns==small.returns);check(41,large.names-small.names==80*(large.arena-small.arena) && large.projection-small.projection==large.names-small.names);check(42,large.values==small.values && large.values<large.projection && large.projection>800000);report("small",small);report("large",large);discard io::write("reuse=ok origins=ok links=ok values=ok\n");return 0;}
