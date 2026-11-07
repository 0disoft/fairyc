module main;
import module_project_compile as compile;
import module_signatures as signatures;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import syntax as syntax;
import scanner as scanner;
import sha256 as sha;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

struct Stamp {usize bytes;u32[8] digest;}
@effects(console) fn void check(usize line,bool good){if(!good){discard io::write_stderr("cleanup fields check ");discard io::write_stderr_usize(line);discard io::write_stderr("\n");assert(false);}}
@effects(console) fn void text(ref mut bytes::Buffer out,ref u8[] raw){check(1,bytes::append(out,raw).is_ok);}
@effects(console) fn void number(ref mut bytes::Buffer out,usize n){check(2,bytes::append_usize(out,n).is_ok);}
@effects(console) fn syntax::Parsed parsed(bytes::Buffer raw){
 let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{check(3,false);return parsed(bytes::empty());}};
 return switch(syntax::parse(stream,512)){case .ok(p):{check(4,syntax::problem_count(ref p)==0 && syntax::field_count(ref p)<=512 && syntax::source_view(ref p).len<=65536);p}case .err(_):{check(5,false);parsed(bytes::empty())}};
}
@effects(console) fn void push(bytes::Buffer raw,ref mut v::Vec<syntax::Parsed> sources,ref mut v::Vec<Stamp> stamps){
 let stamp=Stamp{.bytes=bytes::len(ref raw),.digest=sha::digest(bytes::view(ref raw))};check(6,v::push<syntax::Parsed>(sources,parsed(raw)).is_ok && v::push<Stamp>(stamps,stamp).is_ok);
}
// At most 16 resources * 32 fields per provider: raw source field admission is
// unchanged. Root functions seed owned parameter types in batches of eight.
@effects(console) fn void inputs(usize fields,ref mut v::Vec<syntax::Parsed> sources,ref mut v::Vec<Stamp> stamps){
 let resources=(fields+31)/32;let providers=(resources+15)/16;var bytes::Buffer root=bytes::empty();text(ref mut root,b"module root;");var usize i=0;
 while(i<providers){text(ref mut root,b"import provider");number(ref mut root,i);text(ref mut root,b" as p");number(ref mut root,i);text(ref mut root,b";");i+=1;}
 text(ref mut root,b"fn void main(){}");i=0;
 while(i<resources){let stop=if(resources-i>8){i+8}else{resources};text(ref mut root,b"fn void consume");number(ref mut root,i/8);text(ref mut root,b"(");var usize j=i;
  while(j<stop){if(j!=i){text(ref mut root,b",");}text(ref mut root,b"p");number(ref mut root,j/16);text(ref mut root,b"::R");number(ref mut root,j%16);text(ref mut root,b" value");number(ref mut root,j);j+=1;}
  text(ref mut root,b"){}");i=stop;
 }push(root,sources,stamps);i=0;
 while(i<providers){var bytes::Buffer provider=bytes::empty();text(ref mut provider,b"module provider");number(ref mut provider,i);text(ref mut provider,b";");var usize j=0;
  while(j<16 && i*16+j<resources){text(ref mut provider,b"pub resource R");number(ref mut provider,j);text(ref mut provider,b"{");let first=(i*16+j)*32;let usize count=if(fields-first>32){32}else{fields-first};var usize k=0;
   while(k<count){text(ref mut provider,b"i32 f");number(ref mut provider,k);text(ref mut provider,b";");k+=1;}text(ref mut provider,b"cleanup{}}");j+=1;
  }push(provider,sources,stamps);i+=1;
 }
}
@effects(console) fn void unchanged(ref v::Vec<syntax::Parsed> sources,ref v::Vec<Stamp> stamps){
 check(7,v::len<syntax::Parsed>(sources)==v::len<Stamp>(stamps));var usize i=0;while(i<v::len<Stamp>(stamps)){let before=*v::at<Stamp>(stamps,i);let raw=syntax::source_view(v::at<syntax::Parsed>(sources,i));check(8,raw.len==before.bytes);let digest=sha::digest(raw);var usize j=0;while(j<8){check(9,digest[j]==before.digest[j]);j+=1;}i+=1;}
}
@effects(console) fn void recipes(ref ir::Program p,ref v::Vec<syntax::Parsed> sources,usize expected){
 let q=ir::cleanup_ir(p);check(10,clean::field_count(q)==expected && ir::field_count(p)==expected && clean::glue_count(q)==clean::type_count(q) && !clean::project(q) && clean::work(q)<=1048576 && ir::work(p)<=1048576);
 var usize i=1;var usize covered=0;var usize owners=0;while(i<=clean::type_count(q)){let t=clean::type_at(q,i);let g=clean::glue_at(q,i);
  if(t.kind==3){owners+=1;check(11,t.module_index>0 && t.module_index<v::len<syntax::Parsed>(sources));let source=v::at<syntax::Parsed>(sources,t.module_index);let d=syntax::declaration_at(source,t.declaration-1);
   check(12,d.kind==6 && d.field_count>0 && d.field_count<=32 && g.kind==1 && !g.nonempty && g.frame==0 && g.effects==0 && g.field_start==covered && g.field_count==d.field_count && g.field_start<=expected && g.field_count<=expected-g.field_start);
   var usize j=0;while(j<g.field_count){let field=clean::field_at(q,g.field_start+j);let original=syntax::field_at(source,d.field_start+j);let element=clean::type_at(q,field.type);
    check(13,field.owner==i && field.ordinal==j && element.kind==1 && element.scalar==1 && bytes::equal(syntax::type_text(source,original.type_root),b"i32"));j+=1;
   }covered+=g.field_count;
  }else{check(14,g.kind==0 && g.field_count==0);}i+=1;
 }check(15,covered==expected && owners==(expected+31)/32 && ir::layout_count(p)==owners);
}
@effects(console) fn void run(usize count){
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<Stamp> stamps=v::empty<Stamp>();inputs(count,ref mut sources,ref mut stamps);var usize raw_fields=0;var usize i=0;
 while(i<v::len<syntax::Parsed>(ref sources)){raw_fields+=syntax::field_count(v::at<syntax::Parsed>(ref sources,i));i+=1;}check(16,raw_fields==count);unchanged(ref sources,ref stamps);
 let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_stderr("signatures ");discard io::write_stderr_u32(e.code);check(17,false);return;}};
 switch(compile::range(ref sources,ref metadata,0,1)){
  case .ok(p):{check(18,count<=2048);recipes(ref p,ref sources,count);discard io::write("fields=");discard io::write_usize(count);discard io::write(" accepted work=");discard io::write_usize(ir::work(ref p));}
  case .err(e):{if(count<=2048){discard io::write_stderr("fields ");discard io::write_stderr_usize(count);discard io::write_stderr(" stage/code ");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);}check(19,count==2049 && (e.stage==3 && e.code==10 || e.stage==6 && e.code==1002));discard io::write("fields=2049 rejected stage=");discard io::write_u32(e.stage);discard io::write(" code=");discard io::write_u32(e.code);}
 };unchanged(ref sources,ref stamps);discard io::write(" input=unchanged\n");
}
@effects(console) fn i32 main(){run(1023);run(1024);run(1025);run(2047);run(2048);run(2049);return 0;}
