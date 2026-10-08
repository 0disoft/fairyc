module tests::module_ast_join;
import module_ast_join as join;
import syntax as syntax;
import scanner as scanner;
import types as types;
import records as records;
import attributes as attributes;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] input){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,input).is_ok);let s=switch(scanner::collect(b,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module a;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn syntax::Parsed owner(ref u8[] first,ref u8[] second){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(first)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(second)).is_ok);return switch(join::join(ref files)){case .ok(value):{value}case .err(_):{assert(false);return owner(b"module a;",b"module b;");}};}
fn join::Problem rejection(ref u8[] first,ref u8[] second){var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut files,parsed(first)).is_ok && v::push<syntax::Parsed>(ref mut files,parsed(second)).is_ok);return switch(join::join(ref files)){case .ok(_):{join::Problem{.code=0,.module_index=0,.start=0,.end=0}}case .err(p):{p}};}
fn bool span(ref u8[] source,usize start,usize end,ref u8[] expected){assert(start<=end && end<=source.len);return bytes::equal(source[start..end],expected);}
fn usize expression_kind(ref syntax::Parsed p,u32 kind,usize ordinal){var usize found=0;var usize i=1;while(i<=syntax::expression_count(p)){if(syntax::expression_at(p,i).kind==kind){if(found==ordinal){return i;}found+=1;}i+=1;}assert(false);return 0;}
fn usize statement_kind(ref syntax::Parsed p,u32 kind,usize ordinal){var usize found=0;var usize i=1;while(i<=syntax::statement_count(p)){if(syntax::statement_at(p,i).kind==kind){if(found==ordinal){return i;}found+=1;}i+=1;}assert(false);return 0;}
fn usize declaration_kind(ref syntax::Parsed p,u32 kind,usize ordinal){var usize found=0;var usize i=0;while(i<syntax::declaration_count(p)){if(syntax::declaration_at(p,i).kind==kind){if(found==ordinal){return i;}found+=1;}i+=1;}assert(false);return 0;}

// Every cloned node must still describe the same original bytes at its shifted
// span, and field names must keep their own spans. Bases are the counts of the
// preceding inputs, so the first input passes zeroes.
fn void require_spans(ref syntax::Parsed owner,ref syntax::Parsed original,usize declarations,usize expressions,usize statements,usize entries,usize types_base,usize fields,usize attributes_base,usize arguments){
    let ref u8[] joined=syntax::source_view(owner);
    let ref u8[] origin=syntax::source_view(original);
    var usize i=0;
    while(i<syntax::declaration_count(original)){let syntax::Declaration d=syntax::declaration_at(original,i);let syntax::Declaration q=syntax::declaration_at(owner,i+declarations);assert(bytes::equal(joined[q.start..q.end],origin[d.start..d.end]));i+=1;}
    i=1;
    while(i<=syntax::expression_count(original)){let syntax::Expression d=syntax::expression_at(original,i);let syntax::Expression q=syntax::expression_at(owner,i+expressions);assert(bytes::equal(joined[q.start..q.end],origin[d.start..d.end]));i+=1;}
    i=1;
    while(i<=syntax::statement_count(original)){let syntax::Statement d=syntax::statement_at(original,i);let syntax::Statement q=syntax::statement_at(owner,i+statements);assert(bytes::equal(joined[q.start..q.end],origin[d.start..d.end]));i+=1;}
    i=0;
    while(i<syntax::entry_count(original)){let syntax::Entry d=syntax::entry_at(original,i);let syntax::Entry q=syntax::entry_at(owner,i+entries);assert(bytes::equal(joined[q.start..q.end],origin[d.start..d.end]));i+=1;}
    i=1;
    while(i<=syntax::type_count(original)){let types::Node d=syntax::type_at(original,i);let types::Node q=syntax::type_at(owner,i+types_base);assert(bytes::equal(joined[q.start..q.end],origin[d.start..d.end]));i+=1;}
    i=0;
    while(i<syntax::field_count(original)){let records::Field d=syntax::field_at(original,i);let records::Field q=syntax::field_at(owner,i+fields);assert(bytes::equal(joined[q.start..q.end],origin[d.start..d.end]) && bytes::equal(joined[q.name_start..q.name_end],origin[d.name_start..d.name_end]));i+=1;}
    i=0;
    while(i<syntax::attribute_count(original)){let attributes::Attribute d=syntax::attribute_at(original,i);let attributes::Attribute q=syntax::attribute_at(owner,i+attributes_base);assert(bytes::equal(joined[q.start..q.end],origin[d.start..d.end]) && bytes::equal(joined[q.name_start..q.name_end],origin[d.name_start..d.name_end]));i+=1;}
    i=0;
    while(i<syntax::attribute_argument_count(original)){let attributes::Argument d=syntax::attribute_argument_at(original,i);let attributes::Argument q=syntax::attribute_argument_at(owner,i+arguments);assert(bytes::equal(joined[q.start..q.end],origin[d.start..d.end]));i+=1;}
}

@test fn void joined_arenas_remap_ids_spans_and_tokens(){
    let ref u8[] first=b"module alpha;resource Handle{i32 id;cleanup{discard this.id;}}@effects(console)fn i32 token(i32 n){return n;}";
    let ref u8[] second=b"module beta;enum Kind{Empty;Point{i32 x;i32 y;};}@effects(console)fn i32 main(){var i32 total=0;let i32 step=3;total+=step;let i32 value=token(total)?+0x2A;let Kind kind=.Point{.x=value,.y=1};if(value>0){total=value;}else{total=0;}for(var i32 i=0;i<2;i+=1){total+=i;}return switch(kind){case .Point{.x=first,.y=second}:{first+second}case .Empty:{0}};}";
    let syntax::Parsed a=parsed(first);
    let syntax::Parsed b=parsed(second);
    let syntax::Parsed joined=owner(first,second);
    let usize declarations_base=syntax::declaration_count(ref a);
    let usize expressions_base=syntax::expression_count(ref a);
    let usize statements_base=syntax::statement_count(ref a);
    let usize entries_base=syntax::entry_count(ref a);
    let usize types_base=syntax::type_count(ref a);
    let usize fields_base=syntax::field_count(ref a);
    let usize attributes_base=syntax::attribute_count(ref a);
    let usize arguments_base=syntax::attribute_argument_count(ref a);
    // Original bytes stay byte-exact with one separator, and both modules survive.
    var bytes::Buffer expected=bytes::empty();
    assert(bytes::append(ref mut expected,first).is_ok && bytes::append(ref mut expected,b"\n").is_ok && bytes::append(ref mut expected,second).is_ok);
    assert(bytes::equal(syntax::source_view(ref joined),bytes::view(ref expected)) && syntax::module_count(ref joined)==2);
    assert(syntax::declaration_count(ref joined)==syntax::declaration_count(ref a)+syntax::declaration_count(ref b));
    assert(syntax::expression_count(ref joined)==syntax::expression_count(ref a)+syntax::expression_count(ref b));
    assert(syntax::statement_count(ref joined)==syntax::statement_count(ref a)+syntax::statement_count(ref b));
    assert(syntax::entry_count(ref joined)==syntax::entry_count(ref a)+syntax::entry_count(ref b));
    assert(syntax::type_count(ref joined)==syntax::type_count(ref a)+syntax::type_count(ref b));
    assert(syntax::field_count(ref joined)==syntax::field_count(ref a)+syntax::field_count(ref b));
    assert(syntax::attribute_count(ref joined)==syntax::attribute_count(ref a)+syntax::attribute_count(ref b));
    assert(syntax::attribute_argument_count(ref joined)==syntax::attribute_argument_count(ref a)+syntax::attribute_argument_count(ref b));
    require_spans(ref joined,ref a,0,0,0,0,0,0,0,0);
    require_spans(ref joined,ref b,declarations_base,expressions_base,statements_base,entries_base,types_base,fields_base,attributes_base,arguments_base);
    // A joined byte position recovers its original module through the module arena.
    var usize i=0; var usize second_module=0;
    while(i<syntax::declaration_count(ref joined)){let syntax::Declaration d=syntax::declaration_at(ref joined,i);if(d.kind==1 && syntax::module_index(ref joined,d.start)==1){second_module=d.start;}i+=1;}
    assert(second_module!=0 && syntax::module_index(ref joined,0)==0 && syntax::module_index(ref joined,second_module)==1);
    // Name expressions keep their original span, so text is read from the joined bytes.
    let usize call=expression_kind(ref b,6,0);
    let usize joined_call=call+expressions_base;
    assert(bytes::equal(syntax::expression_text(ref joined,syntax::expression_at(ref joined,joined_call).left),b"token"));
    assert(syntax::call_argument_count(ref joined,joined_call)==1 && syntax::call_argument_count(ref joined,joined_call)==syntax::call_argument_count(ref b,call));
    let syntax::Expression argument=syntax::expression_at(ref joined,syntax::call_argument(ref joined,joined_call,0));
    assert(span(syntax::source_view(ref joined),argument.start,argument.end,b"total"));
    // Cooked literals and their token kind and span survive the offset shift.
    var usize literal=0; var usize found=0;
    i=1;
    while(i<=syntax::expression_count(ref b)){let syntax::Expression n=syntax::expression_at(ref b,i);if(n.kind==1 && bytes::equal(syntax::source_view(ref b)[n.start..n.end],b"0x2A")){if(found==0){literal=i;}found+=1;}i+=1;}
    assert(found==1 && literal!=0);
    let usize joined_literal=literal+expressions_base;
    let syntax::Expression literal_node=syntax::expression_at(ref joined,joined_literal);
    let scanner::TokenInfo literal_token=syntax::token_info(ref joined,literal_node.token);
    assert(bytes::equal(syntax::literal_cooked(ref joined,joined_literal),syntax::literal_cooked(ref b,literal)));
    assert(syntax::literal_kind(ref joined,joined_literal)==syntax::literal_kind(ref b,literal) && literal_token.kind==syntax::literal_kind(ref joined,joined_literal));
    assert(literal_token.start==literal_node.start && literal_token.end==literal_node.end);
    // Assignment operator tokens remap for every assignment statement.
    var usize assignments=0; var usize plus=0;
    i=1;
    while(i<=syntax::statement_count(ref b)){let syntax::Statement s=syntax::statement_at(ref b,i);if(s.kind==4){let usize shifted=i+statements_base;assert(bytes::equal(syntax::assignment_operator(ref joined,shifted),syntax::assignment_operator(ref b,i)));if(bytes::equal(syntax::assignment_operator(ref joined,shifted),b"+=")){plus+=1;}assignments+=1;}i+=1;}
    assert(assignments==5 && plus==3);
    // The query operator keeps its own token and the whole shifted span.
    let usize query=expression_kind(ref b,30,0);
    let usize joined_query=query+expressions_base;
    let syntax::Expression query_node=syntax::expression_at(ref joined,joined_query);
    assert(query_node.kind==30 && bytes::equal(syntax::expression_text(ref joined,joined_query),b"?"));
    assert(span(syntax::source_view(ref joined),query_node.start,query_node.end,b"token(total)?"));
    // Tagged switch: arm bodies are statements, named patterns keep capture spans.
    let usize switch_expression=expression_kind(ref b,15,0);
    let usize joined_switch=switch_expression+expressions_base;
    assert(syntax::switch_case_count(ref joined,joined_switch)==2);
    let syntax::Expression arm=syntax::expression_at(ref joined,syntax::switch_case(ref joined,joined_switch,0));
    assert(arm.kind==18);
    let syntax::Expression pattern=syntax::expression_at(ref joined,arm.left);
    assert(pattern.kind==32 && syntax::pattern_field_count(ref joined,arm.left)==2);
    let syntax::Expression capture=syntax::expression_at(ref joined,syntax::expression_at(ref joined,syntax::pattern_field(ref joined,arm.left,0)).left);
    assert(capture.kind==17 && span(syntax::source_view(ref joined),capture.start,capture.end,b"first"));
    let syntax::Statement arm_body=syntax::statement_at(ref joined,arm.right);
    assert(arm_body.kind==1 && syntax::block_statement_count(ref joined,arm.right)==0 && arm_body.expression!=0);
    let syntax::Expression tail=syntax::expression_at(ref joined,arm_body.expression);
    assert(span(syntax::source_view(ref joined),tail.start,tail.end,b"first+second"));
    assert(syntax::expression_at(ref joined,syntax::expression_at(ref joined,syntax::switch_case(ref joined,joined_switch,1)).left).kind==16);
    // If and else branches keep block statements, and the block count is not shifted.
    let usize conditional=expression_kind(ref b,13,0);
    let syntax::Expression branches=syntax::expression_at(ref joined,syntax::expression_at(ref joined,conditional+expressions_base).right);
    assert(branches.kind==14 && syntax::statement_at(ref joined,branches.left).kind==1 && syntax::block_statement_count(ref joined,branches.left)==1);
    let syntax::Expression alternative=syntax::expression_at(ref joined,branches.right);
    assert(alternative.kind==12 && syntax::block_statement_count(ref joined,alternative.left)==1);
    assert(syntax::statement_at(ref joined,syntax::block_statement(ref joined,branches.left,0)).kind==4);
    // For headers keep their statement links, and entries keep their slots and types.
    let usize loop_statement=statement_kind(ref b,14,0);
    let syntax::Statement iteration=syntax::statement_at(ref joined,loop_statement+statements_base);
    let syntax::Statement header=syntax::statement_at(ref joined,iteration.second);
    assert(header.kind==15 && syntax::statement_at(ref joined,iteration.first).kind==1 && syntax::block_statement_count(ref joined,iteration.first)==1);
    let syntax::Statement initial=syntax::statement_at(ref joined,header.first);
    assert(initial.kind==2 && syntax::statement_at(ref joined,header.second).kind==4 && bytes::equal(syntax::assignment_operator(ref joined,header.second),b"+="));
    let syntax::Entry variable=syntax::entry_at(ref joined,initial.entry-1);
    assert(span(syntax::source_view(ref joined),variable.name_start,variable.name_end,b"i") && bytes::equal(syntax::type_text(ref joined,variable.type_root),b"i32"));
    let syntax::Expression initializer=syntax::expression_at(ref joined,initial.expression);
    assert(span(syntax::source_view(ref joined),initializer.start,initializer.end,b"0"));
    // Cleanup bodies stay reachable and keep their member token and receiver name.
    let usize resource_slot=declaration_kind(ref a,6,0);
    let syntax::Declaration handle=syntax::declaration_at(ref joined,resource_slot);
    assert(handle.kind==6 && handle.cleanup_body!=0 && handle.field_count==1);
    assert(syntax::statement_at(ref joined,handle.cleanup_body).kind==1 && syntax::block_statement_count(ref joined,handle.cleanup_body)==1);
    let syntax::Statement dropped=syntax::statement_at(ref joined,syntax::block_statement(ref joined,handle.cleanup_body,0));
    let syntax::Expression member=syntax::expression_at(ref joined,dropped.expression);
    assert(dropped.kind==12 && member.kind==8 && bytes::equal(syntax::expression_text(ref joined,dropped.expression),b"id") && bytes::equal(syntax::expression_text(ref joined,member.left),b"this"));
    // Enum payload fields precede their member and shift together with the field base.
    let syntax::Declaration enumeration=syntax::declaration_at(ref joined,declaration_kind(ref b,11,0)+declarations_base);
    assert(enumeration.kind==11 && enumeration.field_count==4 && enumeration.field_start==fields_base);
    assert(syntax::field_at(ref joined,enumeration.field_start).kind==3 && span(syntax::source_view(ref joined),syntax::field_at(ref joined,enumeration.field_start).name_start,syntax::field_at(ref joined,enumeration.field_start).name_end,b"Empty"));
    let records::Field payload_x=syntax::field_at(ref joined,enumeration.field_start+1);
    let records::Field payload_y=syntax::field_at(ref joined,enumeration.field_start+2);
    assert(payload_x.kind==4 && span(syntax::source_view(ref joined),payload_x.name_start,payload_x.name_end,b"x") && bytes::equal(syntax::type_text(ref joined,payload_x.type_root),b"i32"));
    assert(payload_y.kind==4 && span(syntax::source_view(ref joined),payload_y.name_start,payload_y.name_end,b"y") && bytes::equal(syntax::type_text(ref joined,payload_y.type_root),b"i32"));
    let records::Field point=syntax::field_at(ref joined,enumeration.field_start+3);
    assert(point.kind==3 && point.payload_count==2 && point.payload_start==enumeration.field_start+1 && span(syntax::source_view(ref joined),point.name_start,point.name_end,b"Point"));
    // Attributes and arguments shift by their own arena bases and keep their token.
    let attributes::Attribute effects=syntax::attribute_at(ref joined,attributes_base);
    assert(span(syntax::source_view(ref joined),effects.name_start,effects.name_end,b"effects") && effects.argument_count==1 && attributes_base>0);
    let attributes::Argument console=syntax::attribute_argument_at(ref joined,arguments_base);
    let scanner::TokenInfo console_token=syntax::token_info(ref joined,console.token);
    assert(span(syntax::source_view(ref joined),console.start,console.end,b"console") && console_token.kind==3 && console_token.start==console.start && console_token.end==console.end);
    // Empty, oversized and generic inputs fail closed with module-local positions.
    var v::Vec<syntax::Parsed> none=v::empty<syntax::Parsed>();
    assert(switch(join::join(ref none)){case .ok(_):{let u32 zero=0;zero}case .err(p):{p.code}}==10);
    let join::Problem generic_declaration=rejection(b"module g;",b"module h;fn i32 f<U>(U x){return 0;}");
    assert(generic_declaration.code==9 && generic_declaration.module_index==1 && generic_declaration.end>generic_declaration.start);
    assert(rejection(b"module g;fn i32 f<U>(U x){return 0;}",b"module h;").code==9);
    assert(rejection(b"module g;fn i32 f(){return 0;}",b"module h;fn i32 g(){return f<i32>(0);}").code==9);
    var v::Vec<syntax::Parsed> many=v::empty<syntax::Parsed>();
    i=0;
    while(i<257){assert(v::push<syntax::Parsed>(ref mut many,parsed(b"module m;")).is_ok);i+=1;}
    assert(switch(join::join(ref many)){case .ok(_):{let u32 zero=0;zero}case .err(p):{p.code}}==10);
}
