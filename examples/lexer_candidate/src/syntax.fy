module syntax;
import syntax_parse_driver as driver;
import scanner as scanner;
import syntax_arena_copy as arenas;
import std::vec as v;
import types as types;
import records as records;
import bodies as bodies;
import body_state as state;
import attributes as attributes;
import std::bytes as bytes;

// Arena IDs: docs/architecture/candidate-source-join-v0.md.
pub struct Declaration { usize cleanup_attribute_start; usize cleanup_attribute_count; usize unsafe_start; usize unsafe_end; usize owner_start; usize owner_end; u32 kind; usize start; usize end; usize name_start; usize name_end; usize expression; usize entry_start; usize entry_count; usize body; usize result_type; usize field_start; usize field_count; bool public; usize public_start; usize public_end; usize attribute_start; usize attribute_count; usize cleanup_start; usize cleanup_end; usize cleanup_body; usize generic_parameters; usize generic_start; usize generic_end; }
pub struct Entry { u32 kind; usize start; usize end; usize name_start; usize name_end; usize type_start; usize type_end; usize type_root; usize expression; }
pub struct Expression { u32 kind; usize start; usize end; usize token; usize left; usize right; }
pub struct Problem { u32 code; usize start; usize end; u32 expected; usize token; }
pub struct Statement { u32 kind; usize start; usize end; usize expression; usize target; usize first; usize second; usize entry; usize operator; }
pub resource Parsed {
    scanner::Stream source;
    v::Vec<Declaration> declarations;
    v::Vec<Problem> problems;
    v::Vec<state::Node> expressions;
    v::Vec<bodies::Entry> entries;
    v::Vec<state::Statement> statements;
    v::Vec<types::Node> types;
    v::Vec<records::Field> fields;
    v::Vec<attributes::Attribute> attributes;
    v::Vec<attributes::Argument> attribute_arguments;
    cleanup { }
}

pub fn Result<Parsed, i32> parse(scanner::Stream input, usize max_items) {
    if (scanner::diagnostic_count(ref input) != 0) { return .err(5); }
    return parse_diagnostics(input,max_items);
}
// Diagnostic recovery retains lexical failures in source_stream. Consumers
// must check both diagnostic owners; the ordinary executable parse rejects
// lexical failures before reaching this recovery-only entry.
pub fn Result<Parsed,i32> parse_diagnostics(scanner::Stream input,usize max_items){return driver::parse_diagnostics(input,max_items);}
pub fn usize source_token_count(ref Parsed parsed) { return scanner::token_count(ref parsed.source); }
pub fn usize declaration_count(ref Parsed parsed) { return v::len<Declaration>(ref parsed.declarations); }
pub fn usize problem_count(ref Parsed parsed) { return v::len<Problem>(ref parsed.problems); }
pub fn Declaration declaration_at(ref Parsed parsed, usize index) { return *v::at<Declaration>(ref parsed.declarations, index); }
pub fn Problem problem_at(ref Parsed parsed, usize index) { return *v::at<Problem>(ref parsed.problems, index); }
pub fn ref u8[] problem_actual(ref Parsed parsed, usize index) {
    return scanner::token_raw(ref parsed.source, v::at<Problem>(ref parsed.problems, index).token);
}
pub fn ref u8[] expected_text(u32 code) {
    if (code == 1) { return b"identifier"; }
    if (code == 2) { return b"type"; }
    if (code == 3) { return b"expression"; }
    if (code == 4) { return b"("; }
    if (code == 5) { return b")"; }
    if (code == 6) { return b"{"; }
    if (code == 7) { return b"}"; }
    if (code == 8) { return b";"; }
    if (code == 9) { return b"="; }
    if (code == 10) { return b"as"; }
    if (code == 11) { return b"]"; }
    if (code == 12) { return b"module"; }
    if (code == 13) { return b">"; }
    if (code == 14) { return b"."; }
    if (code == 15) { return b"attribute argument"; }
    if (code == 16) { return b"array length"; }
    if (code == 17) { return b":"; }
    if (code == 18) { return b"case"; }
    if (code == 19) { return b"extern"; }
    if (code == 20) { return b"C"; }
    if (code == 21) { return b"fn"; }
    if (code == 22) { return b"cell or map"; }
    if (code == 23) { return b"in"; }
    if (code == 24) { return b"cleanup"; }
    if (code == 25) { return b"}"; }
    return b"supported syntax";
}
pub fn ref u8[] source_view(ref Parsed parsed) { return scanner::source_view(ref parsed.source); }
pub fn ref scanner::Stream source_stream(ref Parsed parsed) { return ref parsed.source; }
// Preserve scanner ownership, token indices and cooked bytes without rescanning.
pub fn Result<scanner::Stream,u32> copy_source(ref Parsed parsed,ref mut usize work) { return scanner::copy(ref parsed.source,work); }
pub fn usize expression_count(ref Parsed parsed) { return v::len<state::Node>(ref parsed.expressions); }
pub fn usize entry_count(ref Parsed parsed) { return v::len<bodies::Entry>(ref parsed.entries); }
pub fn usize statement_count(ref Parsed parsed) { return v::len<state::Statement>(ref parsed.statements); }
pub fn Statement statement_at(ref Parsed parsed, usize id) {
    let state::Statement s = *v::at<state::Statement>(ref parsed.statements,id - 1);
    return Statement { .kind = s.kind, .start = s.start, .end = s.end, .expression = s.expression, .target = s.target, .first = s.first, .second = s.second, .entry = s.entry, .operator = s.operator };
}
pub fn ref u8[] assignment_operator(ref Parsed parsed, usize id) {
    let Statement value = statement_at(parsed, id); assert(value.kind == 4);
    return scanner::token_raw(ref parsed.source, value.operator);
}
pub fn usize block_statement_count(ref Parsed parsed, usize id) {
    let Statement value = statement_at(parsed, id); assert(value.kind == 1); return value.second;
}
pub fn usize block_statement(ref Parsed parsed, usize id, usize index) {
    let Statement value = statement_at(parsed, id); assert(value.kind == 1 && index < value.second);
    var usize link = value.first; var usize remaining = value.second - 1;
    while (remaining > index) { link = statement_at(parsed, link).second; remaining -= 1; }
    return statement_at(parsed, link).first;
}
pub fn Entry entry_at(ref Parsed parsed, usize index) {
    let usize count = v::len<bodies::Entry>(ref parsed.entries);
    assert(index < count);
    var usize slot = 0;
    while (slot < count && v::at<bodies::Entry>(ref parsed.entries,slot).id != index + 1) { slot += 1; }
    assert(slot < count);
    let bodies::Entry e = *v::at<bodies::Entry>(ref parsed.entries,slot);
    return Entry { .kind = e.kind, .start = e.start, .end = e.end, .name_start = e.name_start, .name_end = e.name_end, .type_start = e.type_start, .type_end = e.type_end, .type_root = e.type_root, .expression = e.expression };
}
pub fn Expression expression_at(ref Parsed parsed, usize id) {
    let state::Node node = *v::at<state::Node>(ref parsed.expressions, id - 1);
    return Expression { .kind = node.kind, .start = node.start, .end = node.end, .token = node.token, .left = node.left, .right = node.right };
}
pub fn ref u8[] expression_text(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed, id);
    if (node.kind == 2) { return scanner::source_view(ref parsed.source)[node.start..node.end]; }
    return scanner::token_raw(ref parsed.source, node.token);
}
pub fn usize call_argument_count(ref Parsed parsed, usize call) {
    let Expression node = expression_at(parsed, call); assert(node.kind == 6);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = expression_at(parsed, link).right; }
    return count;
}
pub fn u32 literal_kind(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed, id); assert(node.kind == 1);
    return scanner::token_info(ref parsed.source, node.token).kind;
}
pub fn ref u8[] literal_cooked(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed, id); assert(node.kind == 1);
    return scanner::token_cooked(ref parsed.source, node.token);
}
pub fn usize call_argument(ref Parsed parsed, usize call, usize index) {
    return expression_at(parsed,call_argument_entry(parsed,call,index)).left;
}
// A labeled entry retains the identifier token after its opening dot; ordinary
// entries retain the first value token at entry.start. Value children stay intact.
pub fn bool call_argument_labeled(ref Parsed parsed, usize entry) {
    let Expression node = expression_at(parsed,entry);
    return scanner::token_info(ref parsed.source,node.token).start != node.start;
}
pub fn usize call_argument_entry(ref Parsed parsed, usize call, usize index) {
    let usize count = call_argument_count(parsed, call); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed, call).right;
    while (remaining > index) { link = expression_at(parsed, link).right; remaining -= 1; }
    return link;
}


pub fn usize array_element_count(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed,id); assert(node.kind == 24);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = expression_at(parsed,link).right; }
    return count;
}
pub fn usize array_element(ref Parsed parsed, usize id, usize index) {
    let usize count = array_element_count(parsed,id); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed,id).right;
    while (remaining > index) { link = expression_at(parsed,link).right; remaining -= 1; }
    return expression_at(parsed,link).left;
}
pub fn usize type_count(ref Parsed parsed) { return v::len<types::Node>(ref parsed.types); }
pub fn usize record_field_count(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed, id); assert(node.kind == 10 || node.kind == 32);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = expression_at(parsed, link).right; }
    return count;
}
pub fn usize record_field(ref Parsed parsed, usize id, usize index) {
    let usize count = record_field_count(parsed, id); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed, id).right;
    while (remaining > index) { link = expression_at(parsed, link).right; remaining -= 1; }
    return link;
}
pub fn types::Node type_at(ref Parsed parsed, usize id) { return *v::at<types::Node>(ref parsed.types,id - 1); }
// Owned arena composition remaps token IDs by exact original source offsets.
pub fn scanner::TokenInfo token_info(ref Parsed parsed,usize id){return scanner::token_info(ref parsed.source,id);}
pub fn usize token_count(ref Parsed parsed){return scanner::token_count(ref parsed.source);}
// Single-file parser results contain one module. Joined private arenas retain
// each original module declaration and therefore recover its source domain.
pub fn usize module_index(ref Parsed parsed,usize position){var usize module_index=0;var bool seen=false;var usize i=0;while(i<declaration_count(parsed)){let d=declaration_at(parsed,i);if(d.kind==1 && d.start<=position){if(seen){module_index+=1;}seen=true;}i+=1;}return module_index;}
pub fn usize module_count(ref Parsed parsed){var usize count=0;var usize i=0;while(i<declaration_count(parsed)){if(declaration_at(parsed,i).kind==1){count+=1;}i+=1;}return count;}
pub fn usize pattern_field_count(ref Parsed parsed, usize id) {
    assert(expression_at(parsed,id).kind == 32); return record_field_count(parsed,id);
}
pub fn usize pattern_field(ref Parsed parsed, usize id, usize index) {
    assert(expression_at(parsed,id).kind == 32); return record_field(parsed,id,index);
}
pub fn ref u8[] type_text(ref Parsed parsed, usize id) {
    let types::Node node = type_at(parsed,id);
    return scanner::source_view(ref parsed.source)[node.start..node.end];
}


pub fn usize field_count(ref Parsed parsed) { return v::len<records::Field>(ref parsed.fields); }
pub fn records::Field field_at(ref Parsed parsed, usize index) { return *v::at<records::Field>(ref parsed.fields,index); }
pub fn usize type_argument_count(ref Parsed parsed, usize id) {
    let types::Node node = type_at(parsed,id); assert(node.kind == 6);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = type_at(parsed,link).right; }
    return count;
}
pub fn usize type_argument(ref Parsed parsed, usize id, usize index) {
    let usize count = type_argument_count(parsed,id); assert(index < count);
    var usize remaining = count - 1; var usize link = type_at(parsed,id).right;
    while (remaining > index) { link = type_at(parsed,link).right; remaining -= 1; }
    return type_at(parsed,link).left;
}
pub fn usize attribute_count(ref Parsed parsed) { return v::len<attributes::Attribute>(ref parsed.attributes); }
pub fn usize attribute_argument_count(ref Parsed parsed) { return v::len<attributes::Argument>(ref parsed.attribute_arguments); }
pub fn usize specialization_argument_count(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed,id); assert(node.kind == 20);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = type_at(parsed,link).right; }
    return count;
}
pub fn usize specialization_argument(ref Parsed parsed, usize id, usize index) {
    let usize count = specialization_argument_count(parsed,id); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed,id).right;
    while (remaining > index) { link = type_at(parsed,link).right; remaining -= 1; }
    return type_at(parsed,link).left;
}
pub fn usize switch_case_count(ref Parsed parsed, usize id) {
    let Expression node = expression_at(parsed,id); assert(node.kind == 15);
    var usize link = node.right; var usize count = 0;
    while (link != 0) { count += 1; link = expression_at(parsed,link).right; }
    return count;
}
pub fn usize switch_case(ref Parsed parsed, usize id, usize index) {
    let usize count = switch_case_count(parsed,id); assert(index < count);
    var usize remaining = count - 1; var usize link = expression_at(parsed,id).right;
    while (remaining > index) { link = expression_at(parsed,link).right; remaining -= 1; }
    return expression_at(parsed,link).left;
}
pub fn attributes::Attribute attribute_at(ref Parsed parsed, usize index) { return *v::at<attributes::Attribute>(ref parsed.attributes,index); }
pub fn attributes::Argument attribute_argument_at(ref Parsed parsed, usize index) { return *v::at<attributes::Argument>(ref parsed.attribute_arguments,index); }
pub fn ref u8[] attribute_argument_text(ref Parsed parsed, usize index) {
    let attributes::Argument argument = attribute_argument_at(parsed,index);
    return scanner::source_view(ref parsed.source)[argument.start..argument.end];
}
pub fn usize generic_parameter_count(ref Parsed parsed, usize declaration) {
    var usize link = declaration_at(parsed,declaration).generic_parameters; var usize count = 0;
    while (link != 0) { count += 1; link = type_at(parsed,link).right; }
    return count;
}
pub fn types::Node generic_parameter_at(ref Parsed parsed, usize declaration, usize index) {
    let usize count = generic_parameter_count(parsed,declaration); assert(index < count);
    var usize link = declaration_at(parsed,declaration).generic_parameters; var usize remaining = count - 1;
    while (remaining > index) { link = type_at(parsed,link).right; remaining -= 1; }
    return type_at(parsed,type_at(parsed,link).left);
}

// Compiler-private owner publication; source/arenas are transferred as whole values.
pub fn Parsed assemble(scanner::Stream source,v::Vec<Declaration> declarations,v::Vec<Problem> problems,v::Vec<state::Node> expressions,v::Vec<bodies::Entry> entries,v::Vec<state::Statement> statements,v::Vec<types::Node> types,v::Vec<records::Field> fields,v::Vec<attributes::Attribute> attributes,v::Vec<attributes::Argument> attribute_arguments){return Parsed{.source,.declarations,.problems,.expressions,.entries,.statements,.types,.fields,.attributes,.attribute_arguments};}
// Independently own the exact parser/scanner arenas; preserve physical entry
// order, lexical IDs, cooked-token slots and original byte spans. No rescan or
// reparse. The caller can close the original source set immediately afterward.
pub fn Result<Parsed,u32> copy(ref Parsed original,ref mut usize work){return copy_mode(original,false,work);}
pub fn Result<Parsed,u32> copy_tests(ref Parsed original,ref mut usize work){return copy_mode(original,true,work);}
fn Result<Parsed,u32> copy_mode(ref Parsed original,bool tests,ref mut usize work){
    let source=switch(scanner::copy(ref original.source,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var declarations=switch(arenas::copy_declarations(ref original.declarations,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let problems=switch(arenas::copy_problems(ref original.problems,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let expressions=switch(arenas::copy_expressions(ref original.expressions,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let entries=switch(arenas::copy_entries(ref original.entries,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let statements=switch(arenas::copy_statements(ref original.statements,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let types=switch(arenas::copy_types(ref original.types,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let fields=switch(arenas::copy_fields(ref original.fields,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var attributes=switch(arenas::copy_attributes(ref original.attributes,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let attribute_arguments=switch(arenas::copy_attribute_arguments(ref original.attribute_arguments,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    if(tests){consume_test_metadata(source_view(original),ref mut declarations,ref mut attributes);}
    return .ok(Parsed{.source,.declarations,.problems,.expressions,.entries,.statements,.types,.fields,.attributes,.attribute_arguments});
}



fn void consume_test_metadata(ref u8[] source,ref mut v::Vec<Declaration> declarations,ref mut v::Vec<attributes::Attribute> attrs){var usize i=0;while(i<v::len<Declaration>(declarations)){var Declaration d=*v::at<Declaration>(declarations,i);var usize kept=0;var usize j=0;while(j<d.attribute_count){let a=*v::at<attributes::Attribute>(attrs,d.attribute_start+j);if(!bytes::equal(source[a.name_start..a.name_end],b"test")){let slot=v::at_mut<attributes::Attribute>(attrs,d.attribute_start+kept);*slot=a;kept+=1;}j+=1;}d.attribute_count=kept;let slot=v::at_mut<Declaration>(declarations,i);*slot=d;i+=1;}}
