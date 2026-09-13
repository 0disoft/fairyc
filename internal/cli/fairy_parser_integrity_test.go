package cli

import "testing"

func TestWindowsFairyParserIntegrity(t *testing.T) {
	runParserProbe(t, `fn i32 payload(){
 return switch(make(b"module main; enum E { None; Some { i32 x; u8 y; }; Empty {}; Next { u8 z; }; }")) {
 case .ok(parsed): {
  assert(syntax::problem_count(ref parsed) == 0 && syntax::field_count(ref parsed) == 7 && integrity::valid(ref parsed));
  var usize i = 0; while(i < 7) { assert(integrity::valid_field(ref parsed,syntax::field_at(ref parsed,i),i)); i += 1; }
  let member = syntax::field_at(ref parsed,3);
  assert(!integrity::valid_field(ref parsed,member,7));
  var f = member; f.kind=999; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.type_root=1; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.key_type=1; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.public=true; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.public_start=1; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.name_end=f.name_start; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.end=999999; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_start=4; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_count=999999; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_start=0; f.payload_count=3; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_open=0; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_open=f.name_start; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_end=f.payload_open; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_end=f.end; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_open=syntax::field_at(ref parsed,1).end; assert(!integrity::valid_field(ref parsed,f,3));
  f=member; f.payload_end=syntax::field_at(ref parsed,2).start; assert(!integrity::valid_field(ref parsed,f,3));
  let child=syntax::field_at(ref parsed,1);
  f=child; f.type_root=0; assert(!integrity::valid_field(ref parsed,f,1));
  f=child; f.type_root=999999; assert(!integrity::valid_field(ref parsed,f,1));
  f=child; f.key_type=1; assert(!integrity::valid_field(ref parsed,f,1));
  f=child; f.public=true; assert(!integrity::valid_field(ref parsed,f,1));
  f=child; f.public_start=1; assert(!integrity::valid_field(ref parsed,f,1));
  f=child; f.public_end=1; assert(!integrity::valid_field(ref parsed,f,1));
  f=child; f.payload_count=1; assert(!integrity::valid_field(ref parsed,f,1));
  f=syntax::field_at(ref parsed,0); f.payload_start=1; assert(!integrity::valid_field(ref parsed,f,0));
  f=syntax::field_at(ref parsed,4); f.payload_count=1; assert(!integrity::valid_field(ref parsed,f,4));
  let owner=syntax::declaration_at(ref parsed,1);
  assert(integrity::valid_declaration_fields(ref parsed,owner));
  var d=owner; d.field_start=8; assert(!integrity::valid_declaration_fields(ref parsed,d));
  d=owner; d.field_count=999999; assert(!integrity::valid_declaration_fields(ref parsed,d));
  d=owner; d.kind=4; assert(!integrity::valid_declaration_fields(ref parsed,d));
  d=owner; d.field_count=3; assert(!integrity::valid_declaration_fields(ref parsed,d));
  d=owner; d.field_start=2; d.field_count=5; assert(!integrity::valid_declaration_fields(ref parsed,d));
  d=owner; d.end=member.end; assert(!integrity::valid_declaration_fields(ref parsed,d));
  assert(integrity::valid(ref parsed));
  0
 } case .err(_): {1}
 };
}
fn i32 main(){
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
 assert(result==0 && payload()==0);return 0;
}`)
}
