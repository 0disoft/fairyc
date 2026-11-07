module tests::closure_value_source_prepare;
import scanner as scanner;
import syntax as syntax;
import copied_closure_tables as private;
import closure_value_type_source as closure_types;
import generic_source_function_json as functions;
import name_resolution as names;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] raw){
  let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};
  let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};
  return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};
}
pub fn void refused(ref u8[] raw){
  let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)!=0);}case .err(_):{}};
}
fn void inspect(ref syntax::Parsed p,ref syntax::Parsed written,usize values,usize direct){
  assert(bytes::equal(syntax::source_view(p),syntax::source_view(written)));
  assert(syntax::declaration_count(p)==syntax::declaration_count(written)+values+direct);
  var usize i=0;while(i<syntax::entry_count(written)){let a=syntax::entry_at(p,i);let b=syntax::entry_at(written,i);assert(a.kind==b.kind && a.start==b.start && a.end==b.end && a.name_start==b.name_start && a.name_end==b.name_end && a.type_root==b.type_root && a.expression==b.expression);i+=1;}
  var usize creates=0;i=1;while(i<=syntax::expression_count(p)){
    let n=syntax::expression_at(p,i);if(n.kind==36){
      creates+=1;let target=syntax::expression_at(p,n.left);assert(target.kind==2 && target.token>private::helper_flag() && target.token<private::snapshot_flag());
      let declaration=target.token-private::helper_flag();let d=syntax::declaration_at(p,declaration-1);let count=syntax::call_argument_count(p,i);
      assert(count<=4 && syntax::closure_value_captures(p,declaration)==count+1 && !d.public && d.kind==3 && d.entry_count==count+1);
      assert(bytes::equal(syntax::expression_text(p,n.left),syntax::declaration_name(p,declaration)));
      var usize original=0;var usize j=1;while(j<=syntax::expression_count(written)){let old=syntax::expression_at(written,j);if(old.kind==34 && old.start==n.start && old.end==n.end){assert(original==0);original=j;}j+=1;}
      assert(original!=0);let literal=syntax::expression_at(written,original);let header=syntax::expression_at(written,literal.right);assert(header.right/129==count && header.right%129==1);
      var usize owner=0;j=1;while(j<syntax::declaration_count(written)){let candidate=syntax::declaration_at(written,j);if(candidate.kind==3 && n.start>=candidate.start && n.end<=candidate.end){owner=j+1;}j+=1;}
      assert(owner!=0);let body=syntax::declaration_at(p,owner-1).body;var usize position=0;var bool found=false;
      j=0;while(j<syntax::block_statement_count(p,body)){let s=syntax::statement_at(p,syntax::block_statement(p,body,j));if(s.kind==2 && s.expression==i){assert(!found);found=true;position=j;let e=syntax::entry_at(p,s.entry-1);assert(e.kind==2 && e.expression==i && closure_types::signature(p,e.type_root));}j+=1;}assert(found && position>=count);
      j=0;while(j<count){let entry=syntax::call_argument_entry(p,i,j);assert(!syntax::call_argument_labeled(p,entry));let captured=syntax::expression_at(p,syntax::call_argument(p,i,j));let original_capture=syntax::entry_at(written,header.left+j-1);
        assert(captured.kind==2 && captured.token>private::snapshot_flag());let snapshot=captured.token-private::snapshot_flag();let e=syntax::entry_at(p,snapshot-1);assert(e.kind==102 && bytes::equal(syntax::expression_text(p,syntax::call_argument(p,i,j)),syntax::source_view(written)[original_capture.name_start..original_capture.name_end]));
        let statement=syntax::statement_at(p,syntax::block_statement(p,body,position-count+j));assert(statement.kind==2 && statement.entry==snapshot && statement.expression==e.expression);
        let parameter=syntax::entry_at(p,d.entry_start+j);assert(parameter.kind==1 && parameter.name_start==original_capture.name_start && parameter.type_root==original_capture.type_root);j+=1;
      }
    }i+=1;
  }assert(creates==values);
  var usize encoded=0;i=syntax::declaration_count(written);while(i<syntax::declaration_count(p)){let captures=syntax::closure_value_captures(p,i+1);assert(captures<=5);if(captures!=0){encoded+=1;}i+=1;}assert(encoded==values);
  // Module composition uses this same append operation with a declaration base.
  var bytes::Buffer data=bytes::empty();var v::Vec<private::Row> rows=v::empty<private::Row>();assert(private::append(ref mut data,ref mut rows,syntax::private_names(p),7)==0);let moved=private::assemble(data,rows);
  i=1;while(i<=syntax::declaration_count(p)){assert(private::value_captures(ref moved,i+7)==syntax::closure_value_captures(p,i));assert(bytes::equal(private::name(ref moved,i+7),private::name(syntax::private_names(p),i)));i+=1;}
  assert(names::check(p).code==0);
}
@effects(console) pub fn void prepared(ref u8[] raw,usize values,usize direct){
  let p=parsed(raw);var usize work=0;let written=switch(syntax::written_copy(ref p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  inspect(ref p,ref written,values,direct);work=0;let copied=switch(syntax::copy(ref p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};inspect(ref copied,ref written,values,direct);
  var usize i=1;while(i<syntax::declaration_count(ref written)){if(syntax::declaration_at(ref written,i).kind==3){work=0;let a=switch(functions::encode(ref p,i,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let b=switch(functions::encode(ref copied,i,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref a),bytes::view(ref b)));work=0;let digest=switch(functions::definition(ref copied,i,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(io::write_bytes(bytes::view(ref digest)).is_ok && io::write("\n").is_ok);}i+=1;}
}
