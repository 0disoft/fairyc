module tests::closure_value_source_policy;
import syntax as syntax;
import scanner as scanner;
import name_resolution as names;
import type_check as typing;
import hir_lower as hir;
import mir_lower as mir;
import mir_verify as proof;
import std::bytes as bytes;
import std::io as io;

struct Report{u32 phase;u32 code;usize start;usize end;}
fn Report report(u32 phase,u32 code,usize start,usize end){return Report{.phase,.code,.start,.end};}
fn u32 positive(i32 value){assert(value>=0 && value<=65536);var u32 n=0;var i32 left=value;while(left!=0){n+=1;left-=1;}return n;}
// Only original bytes enter the candidate. Names, typing and both IR lowerers
// consume the candidate's own parsed/expanded owners; Go supplies no AST facts.
fn Report pipeline(ref u8[] raw){
  let input=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return report(1,11,0,raw.len);}};
  let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(code):{return report(2,positive(code),0,raw.len);}};
  let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(code):{return report(3,positive(code),0,raw.len);}};
  if(syntax::problem_count(ref parsed)!=0){let p=syntax::problem_at(ref parsed,0);return report(3,p.code,p.start,p.end);}
  let resolved=names::resolve(parsed);let n=names::problem(ref resolved);if(n.code!=0){return report(4,n.code,n.start,n.end);}
  let typed=typing::check(resolved);let t=typing::problem(ref typed);if(t.code!=0){return report(5,t.code,t.start,t.end);}
  let h=switch(hir::lower(typed,0)){case .ok(p):{p}case .err(e):{return report(6,e.code,e.start,e.end);}};
  let m=switch(mir::lower(h)){case .ok(p):{p}case .err(e):{return report(7,e.code,e.start,e.end);}};
  let checked=proof::check(ref m);if(checked.code!=0){return report(8,checked.code,checked.start,checked.end);}return report(9,0,0,0);
}
@effects(console) pub fn void probe(ref u8[] name,ref u8[] raw){
  assert(name.len!=0);let r=pipeline(raw);assert(io::write_u32(r.phase).is_ok && io::write(" ").is_ok && io::write_u32(r.code).is_ok && io::write(" ").is_ok && io::write_usize(r.start).is_ok && io::write(" ").is_ok && io::write_usize(r.end).is_ok && io::write("\n").is_ok);
}
