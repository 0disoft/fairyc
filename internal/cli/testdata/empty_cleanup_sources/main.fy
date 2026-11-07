module main;
import module_unit_bodies as bodies;
import module_unit_body_ast as ast;
import module_unit_names as names;
import module_unit_interfaces as interfaces;
import module_unit_bind as binding;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

resource Held {bodies::Plan owned;names::Plan resolved;scanner::Stream expected;cleanup{}}
@effects(console) fn void check(usize step,bool condition){if(!condition){discard io::write_stderr("empty cleanup step ");discard io::write_stderr_usize(step);discard io::write_stderr("\n");assert(false);}}
@effects(console) fn void append(ref mut bytes::Buffer raw,ref u8[] text){check(1,bytes::append(raw,text).is_ok);}
@effects(console) fn syntax::Parsed parsed(bytes::Buffer raw){
 let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{check(2,false);return parsed(bytes::empty());}};
 let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{check(3,false);return parsed(bytes::empty());}};check(4,syntax::problem_count(ref p)==0);return p;
}
@effects(console) fn bytes::Buffer provider(usize mode){
 var raw=bytes::empty();append(ref mut raw,b"module provider;");
 if(mode==4){append(ref mut raw,b"@abilities(drop)");}
 append(ref mut raw,b"pub resource Empty{cleanup{}}");
 if(mode==2){append(ref mut raw,b"pub resource Active{i32 value;cleanup{discard this.value;}}");}
 if(mode==3){append(ref mut raw,b"pub fn T id<T>(T value){return value;}");}
 // The uncalled function contributes over 1000 real tokens and nonempty,
 // escaped cooked strings. It must not become a retained imported body.
 append(ref mut raw,b"fn void noise(){");var usize i=0;
 while(i<384){append(ref mut raw,b"discard \"unrelated-");check(5,bytes::append_usize(ref mut raw,i).is_ok);append(ref mut raw,b"\\n\";\n");i+=1;}
 append(ref mut raw,b"}\n");return raw;
}
fn ref u8[] root(usize mode){
 if(mode==2){return b"module root;import provider as p;fn void consume(p::Empty a,p::Active b){}fn i32 main(){return 0;}";}
 if(mode==3){return b"module root;import provider as p;fn void consume(p::Empty value){}fn i32 main(){return p::id<i32>(7);}";}
 return b"module root;import provider as p;fn void consume(p::Empty value){}fn i32 main(){return 0;}";
}
@effects(console) fn Held retained(usize mode){
 let first=switch(bytes::copy(root(mode))){case .ok(p):{p}case .err(_):{check(6,false);return retained(mode);}};
 var sources=v::empty<syntax::Parsed>();check(7,v::push<syntax::Parsed>(ref mut sources,parsed(first)).is_ok);check(8,v::push<syntax::Parsed>(ref mut sources,parsed(provider(mode))).is_ok);
 let original=v::at<syntax::Parsed>(ref sources,1);check(9,syntax::source_token_count(original)>1000 && scanner::storage(syntax::source_stream(original)).cooked_bytes>0);
 var usize work=0;let expected=switch(syntax::copy_source(original,ref mut work)){case .ok(p):{p}case .err(_):{check(10,false);return retained(mode);}};
 let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);check(11,false);return retained(mode);}};
 let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);check(12,false);return retained(mode);}};
 let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);check(13,false);return retained(mode);}};
 let owned=switch(bodies::build(ref sources,ref metadata,ref imported)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);check(14,false);return retained(mode);}};
 let resolved=switch(names::build(ref sources,ref metadata,ref imported,ref owned)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.code);check(15,false);return retained(mode);}};
 // Explicitly destroy all original Parsed/Stream owners before returning the
 // Plans. The separately copied oracle was never passed to either builder.
 while(v::len<syntax::Parsed>(ref sources)>0){discard v::pop<syntax::Parsed>(ref mut sources);}
 return Held{.owned,.resolved,.expected};
}
@effects(console) fn void stream_equal(ref scanner::Stream actual,ref scanner::Stream expected,bool full){
 let a=scanner::storage(actual);let e=scanner::storage(expected);check(16,bytes::equal(scanner::source_view(actual),scanner::source_view(expected)) && a.source_bytes==e.source_bytes && a.diagnostics==0);
 if(!full){check(17,scanner::token_count(actual)==0 && a.tokens==0 && a.cooked_bytes==0);return;}
 check(18,a.tokens==e.tokens && a.tokens>1000 && a.cooked_bytes==e.cooked_bytes && a.cooked_bytes>0);var usize i=0;
 while(i<a.tokens){let x=scanner::token_info(actual,i);let y=scanner::token_info(expected,i);check(19,x.kind==y.kind && x.trivia==y.trivia && x.start==y.start && x.end==y.end && bytes::equal(scanner::token_raw(actual,i),scanner::token_raw(expected,i)) && bytes::equal(scanner::token_cooked(actual,i),scanner::token_cooked(expected,i)));i+=1;}
}
@effects(console) fn void verify(usize mode){
 let kept=retained(mode);let p=ref kept.owned;let n=ref kept.resolved;let expected=ref kept.expected;let text=scanner::source_view(expected);
 check(20,bodies::source_count(p)==1 && bodies::count(p)==if(mode==3){1}else{0} && bodies::cleanup_count(p)==if(mode==2){2}else{1});
 check(21,names::function_count(n)==bodies::count(p) && names::cleanup_count(n)==bodies::cleanup_count(p) && bodies::usage(p).bytes==text.len && bodies::cooked_bytes(p)==if(mode==1){0}else{scanner::storage(expected).cooked_bytes});
 var usize empty=0;var usize active=0;var usize i=1;
 while(i<=bodies::cleanup_count(p)){
  let c=bodies::cleanup_at(p,i);let info=bodies::cleanup_info(c);let a=bodies::cleanup_ast(c);let source=bodies::cleanup_source_at(p,i);let resolved=names::cleanup_at(n,i);stream_equal(source,expected,mode!=1);
  check(22,info.module_index==1 && info.receiver_kind==10 && names::local_count(resolved)==1 && bodies::cleanup_edge_count(c)==0);
  let receiver=names::symbol_at(resolved,1);check(23,receiver.kind==10 && receiver.type==info.type && receiver.entry==0 && receiver.pattern==0 && bytes::equal(scanner::source_view(source)[receiver.start..receiver.end],b"cleanup"));
  if(ast::expression_count(a)==0){empty+=1;check(24,names::references(resolved)==0 && names::use_count(resolved)==0 && ast::statement_count(a)==1 && ast::entry_count(a)==0 && ast::type_count(a)==0 && ast::declaration(a).cleanup_body==1);if(mode==4){check(25,ast::attribute_count(a)==1);}}
  else{
   active+=1;check(26,mode==2 && names::references(resolved)==1);let r=names::reference_at(resolved,1);check(27,r.kind==1 && r.symbol==1 && bytes::equal(scanner::source_view(source)[r.start..r.end],b"this") && r.source_index==ast::expression_origin(a,r.expression));
   var usize fields=0;var usize e=1;while(e<=ast::expression_count(a)){let node=ast::expression_at(a,e);check(28,node.token<scanner::token_count(source));if(node.kind==8){check(29,node.left==r.expression && bytes::equal(scanner::token_raw(source,node.token),b"value"));fields+=1;}e+=1;}check(30,fields==1);
  }i+=1;
 }
 check(31,empty==1 && active==if(mode==2){1}else{0});
 if(mode==3){let a=bodies::body_ast(bodies::at(p,1));let source=bodies::source_at(p,1);let resolved=names::function_at(n,1);stream_equal(source,expected,true);check(32,names::references(resolved)==1 && names::local_count(resolved)==1);let r=names::reference_at(resolved,1);check(33,r.kind==1 && r.symbol==1 && bytes::equal(scanner::source_view(source)[r.start..r.end],b"value") && r.source_index==ast::expression_origin(a,r.expression));}
 let source=bodies::cleanup_source_at(p,1);discard io::write("case=");discard io::write_usize(mode);discard io::write(" original=");discard io::write_usize(scanner::token_count(expected));discard io::write(" cloned=");discard io::write_usize(scanner::token_count(source));discard io::write(" bytes=");discard io::write_usize(text.len);discard io::write(" cooked=");discard io::write_usize(bodies::cooked_bytes(p));discard io::write("\n");
}
@effects(console) fn i32 main(){var usize mode=1;while(mode<=4){verify(mode);mode+=1;}discard io::write("ok\n");return 0;}
