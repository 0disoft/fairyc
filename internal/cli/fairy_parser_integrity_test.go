package cli

import "testing"

func TestWindowsFairyParserIntegrity(t *testing.T) {
	runParserProbe(t, `fn i32 main(){
 let i32 result=switch(make(b"module main;resource R {u8 x;cleanup {assert(true);}}fn void f(){return [x;N];}")) {
 case .ok(parsed): {
  assert(integrity::valid(ref parsed));
  let usize id=syntax::expression_count(ref parsed);
  var syntax::Expression e=syntax::expression_at(ref parsed,id);
  assert(integrity::valid_expression(ref parsed,e,id));
  e.left=0;assert(!integrity::valid_expression(ref parsed,e,id));
  e=syntax::expression_at(ref parsed,id);e.right=id;assert(!integrity::valid_expression(ref parsed,e,id));
  e=syntax::expression_at(ref parsed,id);e.kind=999;assert(!integrity::valid_expression(ref parsed,e,id));
  e=syntax::expression_at(ref parsed,id);e.end=999999;assert(!integrity::valid_expression(ref parsed,e,id));
  e=syntax::expression_at(ref parsed,id);e.token=999999;assert(!integrity::valid_expression(ref parsed,e,id));
  e=syntax::expression_at(ref parsed,id);e.kind=23;e.right=999999;assert(!integrity::valid_expression(ref parsed,e,id));
  var node=syntax::type_at(ref parsed,1);node.kind=4;node.left=1;assert(!integrity::valid_type(ref parsed,node,1));
  let usize block=syntax::declaration_at(ref parsed,2).body;
  var syntax::Statement s=syntax::statement_at(ref parsed,block);
  s.second=999;assert(!integrity::valid_statement(ref parsed,s,block));
  s=syntax::statement_at(ref parsed,block);s.first=block;assert(!integrity::valid_statement(ref parsed,s,block));
  0
 }case .err(_):{1}};
 assert(result==0);return 0;
}`)
}
