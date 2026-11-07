module main;
import module_unit_interfaces as interfaces;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn syntax::Parsed parsed(bytes::Buffer raw){
 let s=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{ensure(12,false);return parsed(bytes::empty());}};
 let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{ensure(13,false);return parsed(bytes::empty());}};ensure(13,syntax::problem_count(ref p)==0);return p;
}
@effects(console) fn void append(ref mut bytes::Buffer raw,ref u8[] text){ensure(15,bytes::append(raw,text).is_ok);}
// Build actual imported metadata. All declarations and graph records stay below
// the existing 512/4096 limits; no synthetic Interfaces or private index is used.
@effects(console) fn interfaces::Interfaces built(){
 var bytes::Buffer root=bytes::empty();append(ref mut root,b"module main;import left as l;import right as r;fn i32 main(){return 0;}");
 var bytes::Buffer left=bytes::empty();append(ref mut left,b"module left;");var bytes::Buffer right=bytes::empty();append(ref mut right,b"module right;");var usize i=0;
 while(i<80){
  append(ref mut left,b"pub struct S");ensure(22,bytes::append_usize(ref mut left,i).is_ok);append(ref mut left,b"{i32 first;i64 last;}");
  append(ref mut right,b"pub struct S");ensure(23,bytes::append_usize(ref mut right,i+80).is_ok);append(ref mut right,b"{i32 first;i64 last;}");
  append(ref mut root,b"fn void inspect");ensure(24,bytes::append_usize(ref mut root,i).is_ok);append(ref mut root,b"(ref l::S");ensure(24,bytes::append_usize(ref mut root,i).is_ok);append(ref mut root,b" a,ref mut l::S");ensure(24,bytes::append_usize(ref mut root,i).is_ok);append(ref mut root,b" b,ref r::S");ensure(24,bytes::append_usize(ref mut root,i+80).is_ok);append(ref mut root,b" c,ref mut r::S");ensure(24,bytes::append_usize(ref mut root,i+80).is_ok);append(ref mut root,b" d){}");i+=1;
 }
 append(ref mut left,b"pub resource Owner{i32 first;i64 last;cleanup{}}");append(ref mut right,b"pub enum Flag{Off;On;}");
 append(ref mut root,b"fn void owner(ref l::Owner a,ref mut l::Owner b){}fn void flag(r::Flag a){}");
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();ensure(28,v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);ensure(28,v::push<syntax::Parsed>(ref mut sources,parsed(left)).is_ok);ensure(28,v::push<syntax::Parsed>(ref mut sources,parsed(right)).is_ok);
 let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{ensure(28,false);return built();}};
 let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{ensure(29,false);return built();}};
 return switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);ensure(30,false);return built();}};
}
// The pre-index lookup, expressed only with public retained records. This is an
// independent result/order oracle, not a replacement for the production lookup.
fn Result<usize,u32> linear(ref interfaces::Interfaces p,usize type,usize caller,ref u8[] name,ref mut usize probes){
 if(type==0 || type>interfaces::type_count(p)){return .err(18);}var usize owner=type;let t=interfaces::type_at(p,owner);if(t.kind==7 || t.kind==8){owner=t.first;}
 var usize i=0;while(i<interfaces::shape_count(p)){*probes=*probes+1;let s=interfaces::shape_at(p,i);if(s.type==owner){let inner=interfaces::type_at(p,owner);if(inner.kind!=2 && inner.kind!=3 && inner.kind!=13 && inner.kind!=20 && inner.kind!=21){return .err(18);}if((inner.kind==3 || inner.kind==13 || inner.kind==21) && caller!=s.module_index){return .err(6201);}var usize f=0;while(f<s.field_count){*probes=*probes+name.len+1;if(bytes::equal(interfaces::field_name(p,s.field_start+f),name)){return .ok(s.field_start+f+1);}f+=1;}return .err(18);}i+=1;}return .err(18);
}
@effects(console) fn usize query(ref interfaces::Interfaces p,usize type,usize caller,ref u8[] name,usize expected,u32 code,usize cost,ref mut usize queries){
 var usize work=17;let actual=interfaces::field_measured(p,type,caller,name,ref mut work);ensure(39,work==17+cost);var usize old_work=0;let old=linear(p,type,caller,name,ref mut old_work);
 ensure(40,switch(actual){case .ok(id):{code==0 && id==expected}case .err(e):{e==code}});
 ensure(41,switch(old){case .ok(id):{code==0 && id==expected}case .err(e):{e==code}});*queries=*queries+1;return old_work;
}
@effects(console) fn i32 main(){
 let p=built();ensure(44,interfaces::shape_count(ref p)==162 && interfaces::type_count(ref p)<=4096 && interfaces::work(ref p)<=1048576);
 var usize queries=0;var usize maximum=0;var usize records=0;var usize resources=0;var usize enums=0;var usize i=0;
 while(i<interfaces::shape_count(ref p)){
  let s=interfaces::shape_at(ref p,i);let t=interfaces::type_at(ref p,s.type);ensure(47,s.module_index==1 || s.module_index==2);
  if(t.kind==2 || t.kind==3){
   ensure(49,s.field_count==2 && bytes::equal(interfaces::field_name(ref p,s.field_start),b"first") && bytes::equal(interfaces::field_name(ref p,s.field_start+1),b"last"));
   discard query(ref p,s.type,s.module_index,b"first",s.field_start+1,0,7,ref mut queries);let old=query(ref p,s.type,s.module_index,b"last",s.field_start+2,0,11,ref mut queries);if(old>maximum){maximum=old;}
   discard query(ref p,s.type,s.module_index,b"missing",0,18,17,ref mut queries);
   if(t.kind==3){resources+=1;discard query(ref p,s.type,0,b"first",0,6201,1,ref mut queries);discard query(ref p,s.type,0,b"missing",0,6201,1,ref mut queries);}else{records+=1;discard query(ref p,s.type,0,b"last",s.field_start+2,0,11,ref mut queries);}
   var usize shared=0;var usize exclusive=0;var usize j=1;while(j<=interfaces::type_count(ref p)){let view=interfaces::type_at(ref p,j);if((view.kind==7 || view.kind==8) && view.first==s.type){discard query(ref p,j,s.module_index,b"last",s.field_start+2,0,11,ref mut queries);if(view.kind==7){shared+=1;}else{exclusive+=1;}if(t.kind==3){discard query(ref p,j,0,b"last",0,6201,1,ref mut queries);}}j+=1;}ensure(53,shared==1 && exclusive==1);
  }else{ensure(54,t.kind==4);enums+=1;discard query(ref p,s.type,s.module_index,b"Off",0,18,1,ref mut queries);}i+=1;
 }
 ensure(56,records==160 && resources==1 && enums==1 && maximum>=170);
 discard query(ref p,0,0,b"missing",0,18,0,ref mut queries);discard query(ref p,interfaces::type_count(ref p)+1,0,b"missing",0,18,0,ref mut queries);
 var usize scalars=0;i=1;while(i<=interfaces::type_count(ref p)){let t=interfaces::type_at(ref p,i);if(t.kind==1){discard query(ref p,i,0,b"missing",0,18,1,ref mut queries);scalars+=1;}i+=1;}ensure(58,scalars>0);
 discard io::write("shapes=");discard io::write_usize(interfaces::shape_count(ref p));discard io::write(" types=");discard io::write_usize(interfaces::type_count(ref p));discard io::write(" work=");discard io::write_usize(interfaces::work(ref p));discard io::write(" indexed=11 linear=");discard io::write_usize(maximum);discard io::write(" queries=");discard io::write_usize(queries);discard io::write("\nok\n");return 0;
}
@effects(console) fn void ensure(usize line,bool condition){if(!condition){discard io::write_stderr("fixture line ");discard io::write_stderr_usize(line);discard io::write_stderr("\n");assert(false);}}
