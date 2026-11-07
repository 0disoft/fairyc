module main;
import module_unit_body_ast as ast;
import copied_closure_reach as reach;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

resource Expected{v::Vec<usize> expressions;v::Vec<usize> statements;v::Vec<usize> entries;v::Vec<usize> types;usize reach_work;cleanup{}}
@effects(console) fn void check(usize id,bool ok){if(!ok){discard io::write_stderr("AST fixture check ");discard io::write_stderr_usize(id);discard io::write_stderr("\n");assert(false);}}
@effects(console) fn void add(ref mut bytes::Buffer raw,ref u8[] value){check(1,bytes::append(raw,value).is_ok);}
@effects(console) fn syntax::Parsed parsed(bytes::Buffer raw){
 check(2,bytes::len(ref raw)<65536);let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{check(3,false);return parsed(bytes::empty());}};
 let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{check(4,false);return parsed(bytes::empty());}};check(5,syntax::problem_count(ref p)==0);return p;
}
@effects(console) fn syntax::Parsed text(ref u8[] raw){let data=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{check(6,false);return text(raw);}};return parsed(data);}
fn usize arena(ref syntax::Parsed p){return syntax::expression_count(p)+syntax::statement_count(p)+syntax::entry_count(p)+syntax::type_count(p);}
fn usize retained(ref ast::Ast p){return ast::expression_count(p)+ast::statement_count(p)+ast::entry_count(p)+ast::type_count(p);}
@effects(console) fn ast::Ast built(ref syntax::Parsed file,usize declaration,ref mut usize work){return switch(ast::build(file,declaration,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);check(7,false);return built(file,declaration,work);}};}
@effects(console) fn v::Vec<usize> selected(ref syntax::Parsed p,syntax::Declaration d,u32 kind,ref reach::Mask mask){
 let count=if(kind==1){syntax::expression_count(p)}else if(kind==2){syntax::statement_count(p)}else if(kind==3){syntax::entry_count(p)}else{syntax::type_count(p)};
 var v::Vec<usize> ids=v::empty<usize>();var usize i=1;while(i<=count){var usize start=0;var usize end=0;
  if(kind==1){let n=syntax::expression_at(p,i);start=n.start;end=n.end;}else if(kind==2){let n=syntax::statement_at(p,i);start=n.start;end=n.end;}else if(kind==3){let n=syntax::entry_at(p,i-1);start=n.start;end=n.end;}else{let n=syntax::type_at(p,i);start=n.start;end=n.end;}
  if(start>=d.start && end<=d.end && reach::selected(mask,kind,i)){check(8,v::push<usize>(ref mut ids,i).is_ok);}i+=1;
 }return ids;
}
@effects(console) fn Expected expected(ref syntax::Parsed p,syntax::Declaration d){var usize work=0;let mask=switch(reach::build(p,d,ref mut work)){case .ok(m):{m}case .err(_):{check(9,false);return expected(p,d);}};let expressions=selected(p,d,1,ref mask);let statements=selected(p,d,2,ref mask);let entries=selected(p,d,3,ref mask);let types=selected(p,d,4,ref mask);return Expected{.expressions,.statements,.entries,.types,.reach_work=work};}
fn usize origin(ref Expected e,u32 kind,usize id){if(id==0){return 0;}let ids=if(kind==1){ref e.expressions}else if(kind==2){ref e.statements}else if(kind==3){ref e.entries}else{ref e.types};assert(id<=v::len<usize>(ids));return *v::at<usize>(ids,id-1);}
// Full-scan oracle checks source order and every rebased edge independently of
// the optimized map/copy implementation. Its work is not charged to ast::build.
@effects(console) fn usize verify(ref syntax::Parsed file,syntax::Declaration original,ref ast::Ast p,bool cleaning){
 var selection=original;if(cleaning){selection.start=original.cleanup_start;selection.end=original.cleanup_end;}let e=expected(file,selection);
 check(10,ast::expression_count(p)==v::len<usize>(ref e.expressions) && ast::statement_count(p)==v::len<usize>(ref e.statements) && ast::entry_count(p)==v::len<usize>(ref e.entries) && ast::type_count(p)==v::len<usize>(ref e.types));
 var usize i=1;while(i<=ast::expression_count(p)){let id=origin(ref e,1,i);let old=syntax::expression_at(file,id);let n=ast::expression_at(p,i);check(11,ast::expression_origin(p,i)==id && n.kind==old.kind && n.start==old.start && n.end==old.end && n.token==old.token);check(12,origin(ref e,if(n.kind==12 || n.kind==14 || n.kind==31){2}else{1},n.left)==old.left && origin(ref e,if(n.kind==18){2}else if(n.kind==20 || n.kind==23){4}else{1},n.right)==old.right);i+=1;}
 i=1;while(i<=ast::statement_count(p)){let old=syntax::statement_at(file,origin(ref e,2,i));let n=ast::statement_at(p,i);check(13,n.kind==old.kind && n.start==old.start && n.end==old.end && n.operator==old.operator && origin(ref e,1,n.expression)==old.expression && origin(ref e,1,n.target)==old.target && origin(ref e,2,n.first)==old.first && origin(ref e,3,n.entry)==old.entry);if(n.kind==7 || n.kind==14 || n.kind==15){check(14,origin(ref e,2,n.second)==old.second);}else if(n.kind==18){check(15,origin(ref e,4,n.second)==old.second);}else{check(16,n.second==old.second);}i+=1;}
 i=1;while(i<=ast::entry_count(p)){let id=origin(ref e,3,i);let old=syntax::entry_at(file,id-1);let n=ast::entry_at(p,i-1);check(17,ast::entry_origin(p,i)==id && n.id==i && n.kind==old.kind && n.start==old.start && n.end==old.end && n.name_start==old.name_start && n.name_end==old.name_end && n.type_start==old.type_start && n.type_end==old.type_end && origin(ref e,4,n.type_root)==old.type_root && origin(ref e,1,n.expression)==old.expression);i+=1;}
 i=1;while(i<=ast::type_count(p)){let id=origin(ref e,4,i);let old=syntax::type_at(file,id);let n=ast::type_at(p,i);check(18,ast::type_origin(p,i)==id && n.kind==old.kind && n.start==old.start && n.end==old.end && origin(ref e,4,n.left)==old.left && origin(ref e,4,n.right)==old.right);i+=1;}
 let d=ast::declaration(p);check(19,d.kind==original.kind && d.start==original.start && d.end==original.end && d.name_start==original.name_start && d.name_end==original.name_end && origin(ref e,1,d.expression)==original.expression && origin(ref e,2,d.body)==original.body && origin(ref e,2,d.cleanup_body)==original.cleanup_body && origin(ref e,4,d.result_type)==original.result_type && d.generic_parameters==0 && d.field_count==0);
 if(!syntax::closure_prepared(file) && original.entry_count!=0){check(20,d.entry_count==original.entry_count && origin(ref e,3,d.entry_start+1)==original.entry_start+1);}return e.reach_work;
}
@effects(console) fn syntax::Parsed large(){
 var bytes::Buffer raw=bytes::empty();add(ref mut raw,b"module root;resource Empty{i32 n;cleanup{}}");var usize i=0;
 while(i<80){if(i==40){add(ref mut raw,b"fn void filler(){");var usize j=0;while(j<1800){add(ref mut raw,b"discard(1+1);");j+=1;}add(ref mut raw,b"}");}add(ref mut raw,b"fn i32 f");check(21,bytes::append_usize(ref mut raw,i).is_ok);add(ref mut raw,b"(i32 value){let i32 local=value+1;return local;}");i+=1;}
 return parsed(raw);
}
@effects(console) fn void dense(){
 let file=text(b"module dense;fn i32 only(i32 value){return value+1;}");var usize work=0;let p=built(ref file,2,ref mut work);let n=arena(ref file);check(22,retained(ref p)==n && work==2*n);discard verify(ref file,syntax::declaration_at(ref file,1),ref p,false);
 work=1048576-2*n;let exact=built(ref file,2,ref mut work);check(23,work==1048576 && retained(ref exact)==n);work=1048576-2*n+1;check(24,switch(ast::build(ref file,2,ref mut work)){case .ok(_):{false}case .err(code):{code==10}} && work<=1048576);
}
@effects(console) fn void check_cleanup(ref syntax::Parsed file){
 var usize work=0;let p=switch(ast::build_cleanup(file,2,ref mut work)){case .ok(p):{p}case .err(_):{check(25,false);return;}};let d=ast::declaration(ref p);let block=ast::statement_at(ref p,1);let old=syntax::statement_at(file,syntax::declaration_at(file,1).cleanup_body);check(26,work==1 && ast::statement_count(ref p)==1 && retained(ref p)==1 && ast::attribute_count(ref p)==0 && d.cleanup_body==1 && d.body==0 && d.field_count==0 && block.start==old.start && block.end==old.end && block.first==0 && block.second==0);
 work=1048575;check(27,ast::build_cleanup(file,2,ref mut work).is_ok && work==1048576);check(28,switch(ast::build_cleanup(file,2,ref mut work)){case .ok(_):{false}case .err(code):{code==10}} && work==1048576);
 let active=text(b"module active;resource R{i32 n;cleanup{discard this.n;}}");work=0;let q=switch(ast::build_cleanup(ref active,2,ref mut work)){case .ok(p):{p}case .err(_):{check(29,false);return;}};let measured=verify(ref active,syntax::declaration_at(ref active,1),ref q,true);check(30,work==arena(ref active)+retained(ref q)+measured);
}
@effects(console) fn void sparse(){
 let file=text(b"module sparse;fn i32 before(){return 0;}fn i32 main(){let i32 offset=40;let add=fn [i32 offset] i32(i32 x){return offset+x;};return add(2);}fn i32 after(){return 9;}");check(31,syntax::closure_prepared(ref file));var usize i=0;var usize hidden=0;
 while(i<syntax::declaration_count(ref file)){let d=syntax::declaration_at(ref file,i);if(d.kind==3){var usize work=0;let p=built(ref file,i+1,ref mut work);let reach_work=verify(ref file,d,ref p,false);check(32,work==arena(ref file)+retained(ref p)+reach_work && work<=2*arena(ref file)+reach_work);if(bytes::equal(syntax::source_view(ref file)[d.name_start..d.name_end],b"main")){var usize inside=0;var usize j=1;while(j<=syntax::expression_count(ref file)){let n=syntax::expression_at(ref file,j);if(n.start>=d.start && n.end<=d.end){inside+=1;}j+=1;}check(33,inside>ast::expression_count(ref p));hidden+=1;}}i+=1;}check(34,hidden==1);
}
@effects(console) fn i32 main(){
 let file=large();check(35,syntax::declaration_count(ref file)==83 && !syntax::closure_prepared(ref file));let n=arena(ref file);check(36,n>=8000 && 160*n>1048576);var usize work=0;var usize selected_count=0;var usize i=0;
 while(i<80){let declaration=if(i<40){i+3}else{i+4};let before=work;let p=built(ref file,declaration,ref mut work);let d=syntax::declaration_at(ref file,declaration-1);let extra=verify(ref file,d,ref p,false);check(37,extra==0 && work-before==n+retained(ref p) && ast::entry_count(ref p)>=2 && ast::expression_count(ref p)>=4);selected_count+=retained(ref p);if(i==79){check(38,ast::expression_origin(ref p,1)>syntax::expression_count(ref file)/2);}i+=1;}
 check(39,work==80*n+selected_count && work<1048576);var usize invalid=0;check(40,switch(ast::build(ref file,0,ref mut invalid)){case .ok(_):{false}case .err(code):{code==9}} && invalid==0);invalid=1048577;check(41,switch(ast::build(ref file,3,ref mut invalid)){case .ok(_):{false}case .err(code):{code==10}} && invalid==1048577);
 dense();check_cleanup(ref file);sparse();discard io::write("functions=80 arena=");discard io::write_usize(n);discard io::write(" selected=");discard io::write_usize(selected_count);discard io::write(" work=");discard io::write_usize(work);discard io::write(" legacy_lower=");discard io::write_usize(160*n);discard io::write("\ndense=ok cleanup=ok reach=ok\n");return 0;
}
