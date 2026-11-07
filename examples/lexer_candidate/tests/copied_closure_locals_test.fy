module tests::copied_closure_locals_test;
import scanner as scanner;
import syntax as syntax;
import module_signatures as signatures;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import generic_source_function_json as functions;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){
  let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};
  let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};
  return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};
}

fn bool accepted(ref u8[] raw){
  let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return false;}};
  let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return false;}};
  let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{return false;}};
  if(syntax::problem_count(ref p)!=0){return false;}
  var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,p).is_ok);
  let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{return false;}};
  let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{return false;}};
  let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{return false;}};
  return switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{values::unresolved(ref p)==0}case .err(_):{false}};
}

@effects(console) pub fn void admission(ref u8[] label,ref u8[] raw,bool expected){
  if(accepted(raw)!=expected){discard io::write_stderr_bytes(label);assert(false);}
}

// The parser owns the written arena IDs. Lifting may append rows; it must not
// replace an original entry, alter its source range, or cross-wire two bodies.
fn void origins(ref syntax::Parsed p,ref syntax::Parsed written){
  assert(bytes::equal(syntax::source_view(p),syntax::source_view(written)));
  assert(syntax::entry_count(p)>=syntax::entry_count(written));var usize i=0;
  while(i<syntax::entry_count(written)){
    let a=syntax::entry_at(written,i);let b=syntax::entry_at(p,i);
    assert(a.kind==b.kind && a.start==b.start && a.end==b.end && a.name_start==b.name_start && a.name_end==b.name_end && a.type_start==b.type_start && a.type_end==b.type_end && a.type_root==b.type_root && a.expression==b.expression);i+=1;
  }
  var usize helpers=0;i=syntax::declaration_count(written);
  while(i<syntax::declaration_count(p)){
    let d=syntax::declaration_at(p,i);let name=syntax::declaration_name(p,i+1);
    assert(name.len>12 && bytes::equal(name[..12],b"_fy_closure_"));var usize j=syntax::declaration_count(written);
    while(j<i){assert(!bytes::equal(name,syntax::declaration_name(p,j+1)));j+=1;}
    var usize original=0;j=1;
    while(j<=syntax::expression_count(written)){
      let n=syntax::expression_at(written,j);if(n.kind==34 && n.start==d.start && n.end==d.end){assert(original==0);original=n.left;}j+=1;
    }
    assert(original!=0);let old=syntax::statement_at(written,original);let body=syntax::statement_at(p,d.body);
    let count=old.second+if(old.expression!=0){1}else{0};assert(body.kind==1 && body.expression==0 && body.second==count);
    j=0;while(j<old.second){
      let a=syntax::statement_at(written,syntax::block_statement(written,original,j));let b=syntax::statement_at(p,syntax::block_statement(p,d.body,j));
      assert(a.kind==b.kind && a.start==b.start && a.end==b.end && a.expression==b.expression);
      if(a.kind==2){let x=syntax::entry_at(written,a.entry-1);let y=syntax::entry_at(p,b.entry-1);assert(x.name_start==y.name_start && x.name_end==y.name_end && x.expression==y.expression && y.expression==b.expression && y.start>=d.start && y.end<=d.end);}
      j+=1;
    }
    if(old.expression!=0){let ret=syntax::statement_at(p,syntax::block_statement(p,d.body,count-1));assert(ret.kind==3 && ret.expression==old.expression);}
    helpers+=1;i+=1;
  }
  var usize expected=0;i=1;while(i<=syntax::expression_count(written)){if(syntax::expression_at(written,i).kind==34){expected+=1;}i+=1;}assert(helpers==expected && helpers!=0);
}

// Compare the original closure/local AST and its definition digest with Go,
// before and after copying the prepared arena. Synthetic helpers stay private.
@effects(console) pub fn void source_case(ref u8[] raw){
  let p=parsed(raw);var usize work=0;
  let written=switch(syntax::written_copy(ref p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  origins(ref p,ref written);work=0;
  let copied=switch(syntax::copy(ref p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  origins(ref copied,ref written);var usize i=1;
  while(i<syntax::declaration_count(ref written)){
    if(syntax::declaration_at(ref written,i).kind==3){
      work=0;let a=switch(functions::encode(ref p,i,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
      work=0;let b=switch(functions::encode(ref copied,i,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
      assert(bytes::equal(bytes::view(ref a),bytes::view(ref b)));work=0;
      let digest=switch(functions::definition(ref copied,i,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
      assert(io::write_bytes(bytes::view(ref digest)).is_ok && io::write("\n").is_ok);
    }i+=1;
  }
}
