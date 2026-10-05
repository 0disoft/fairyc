module tests::hir;
import hir_model as h;
import hir_lower as lower;
import name_resolution as names;
import type_check as typing;
import module_registry as modules;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] source) {
    var bytes::Buffer input = bytes::empty(); assert(bytes::append(ref mut input,source).is_ok);
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) { case .ok(value): { value } case .err(_): { assert(false); return parsed(b"module main;"); } };
    let syntax::Parsed result = switch(syntax::parse(stream,512)) { case .ok(value): { value } case .err(_): { assert(false); return parsed(b"module main;"); } };
    assert(syntax::problem_count(ref result) == 0); return result;
}
fn h::Program retained(ref u8[] source, usize module_index) {
    let typing::Typed typed = typing::check(names::resolve(parsed(source)));
    assert(typing::problem(ref typed).code == 0);
    return switch(lower::lower(typed,module_index)) { case .ok(value): { value } case .err(_): { assert(false); return retained(b"module main;",0); } };
}
fn h::Statement ordered(ref h::Program p, usize block, usize index) { let h::Block b = h::block_at(p,block); assert(index < b.edge_count); return h::statement_at(p,h::edge_at(p,b.edge_start + index)); }
fn usize ordered_id(ref h::Program p, usize block, usize index) { let h::Block b = h::block_at(p,block); assert(index < b.edge_count); return h::edge_at(p,b.edge_start + index); }
fn h::Expression operand(ref h::Program p, h::Statement s) { return h::expression_at(p,s.expression); }
fn u32 rejection(ref u8[] source) { return switch(lower::lower(typing::check(names::resolve(parsed(source))),0)) { case .ok(_): { 0 } case .err(problem): { problem.code } }; }

@test
fn void owned_literals_groups_and_local_calls() {
    let ref u8[] source = b"module main; pub fn i64 id(i64 x){return x;} fn i64 f(i64 x){var i64 y=(0x2a); {let i64 x=y; discard id(x);} y+=id(x); return y;}";
    let h::Program p = retained(source,7);
    assert(bytes::equal(h::source_view(ref p),source)); assert(h::module_index(ref p) == 7);
    assert(h::function_count(ref p) == 2);
    let h::Function identity = h::function_at(ref p,0); let h::Function function = h::function_at(ref p,1);
    assert(identity.public && !function.public); assert(identity.parameter_count == 1 && function.parameter_count == 1);
    let usize x = h::edge_at(ref p,function.parameter_start);
    let h::Binding parameter = h::binding_at(ref p,x);
    assert(h::binding_code(parameter.kind) == 2 && h::type_code(parameter.type) == 2);
    assert(parameter.owner == function.binding && parameter.scope == function.scope);
    let h::Block body = h::block_at(ref p,function.body); assert(body.edge_count == 4 && body.tail == 0);
    assert(body.owner == function.binding && body.parent_scope == function.scope);
    let h::Statement declaration = ordered(ref p,function.body,0); assert(h::statement_code(declaration.kind) == 1);
    let h::Expression literal = operand(ref p,declaration);
    assert(h::expression_code(literal.kind) == 1 && h::type_code(literal.type) == 2);
    assert(bytes::equal(h::integer_text(ref p,declaration.expression),b"42")); assert(literal.left == 0 && literal.right == 0);
    let h::Binding y = h::binding_at(ref p,declaration.binding); assert(y.owner == function.binding && y.scope == body.scope);
    let h::Expression nested = operand(ref p,ordered(ref p,function.body,1)); assert(h::expression_code(nested.kind) == 7);
    let h::Block inner = h::block_at(ref p,nested.block); assert(inner.parent_scope == body.scope && inner.edge_count == 2);
    let h::Statement shadow = ordered(ref p,nested.block,0); assert(shadow.binding != x);
    let h::Binding local = h::binding_at(ref p,shadow.binding); assert(local.owner == function.binding && local.scope == inner.scope);
    assert(bytes::equal(h::binding_text(ref p,shadow.binding),b"x"));
    let h::Expression inner_call = operand(ref p,ordered(ref p,nested.block,1)); assert(h::expression_code(inner_call.kind) == 6 && inner_call.binding == identity.binding);
    let h::Expression inner_argument = h::expression_at(ref p,h::edge_at(ref p,inner_call.edge_start)); assert(inner_argument.binding == shadow.binding);
    let h::Statement assignment = ordered(ref p,function.body,2); assert(h::statement_code(assignment.kind) == 3 && h::operator_code(assignment.operator) == 1);
    assert(h::expression_at(ref p,assignment.target).binding == declaration.binding);
    let h::Expression call = operand(ref p,assignment); assert(call.binding == identity.binding && call.edge_count == 1);
    assert(h::expression_at(ref p,h::edge_at(ref p,call.edge_start)).binding == x);
    let h::Binding target = h::binding_at(ref p,call.binding); assert(target.module_index == 7 && target.declaration == 2 && target.signature_count == 1);
    assert(h::type_code(h::signature_at(ref p,target.signature_start)) == 2);
    assert(h::statement_code(ordered(ref p,function.body,3).kind) == 2);
    // Integer + three local refs + two calls + block + assignment refs + return
    // contain no callable-path or parenthesis-only nodes, and every retained
    // reference targets an owned local/parameter binding after Typed has gone.
    var usize id = 1;
    while (id <= h::expression_count(ref p)) { let h::Expression e = h::expression_at(ref p,id); if (h::expression_code(e.kind) == 3) { assert(h::binding_at(ref p,e.binding).owner == function.binding || h::binding_at(ref p,e.binding).owner == identity.binding); } id += 1; }
}

@test
fn void contextual_canonical_integer_boundaries_and_prefixes() {
    let h::Program p = retained(b"module main; fn i32 a(){return -2147483648;} fn i64 b(){return -0x8000000000000000;} fn usize c(){return 0xffff_ffff_ffff_ffff;} fn u8 d(){return 0b1111_1111;} fn u32 e(){return 4_294_967_295;} fn i64 f(){return -(1);}",0);
    let h::Statement a = ordered(ref p,h::function_at(ref p,0).body,0); assert(bytes::equal(h::integer_text(ref p,a.expression),b"-2147483648"));
    let h::Statement b = ordered(ref p,h::function_at(ref p,1).body,0); assert(bytes::equal(h::integer_text(ref p,b.expression),b"-9223372036854775808"));
    let h::Expression minimum = operand(ref p,b); assert(h::type_code(minimum.type) == 2 && minimum.left == 0 && minimum.right == 0);
    let h::Statement c = ordered(ref p,h::function_at(ref p,2).body,0); assert(bytes::equal(h::integer_text(ref p,c.expression),b"18446744073709551615")); assert(h::type_code(operand(ref p,c).type) == 5);
    let h::Statement d = ordered(ref p,h::function_at(ref p,3).body,0); assert(bytes::equal(h::integer_text(ref p,d.expression),b"255")); assert(h::type_code(operand(ref p,d).type) == 3);
    let h::Statement e = ordered(ref p,h::function_at(ref p,4).body,0); assert(bytes::equal(h::integer_text(ref p,e.expression),b"4294967295")); assert(h::type_code(operand(ref p,e).type) == 4);
    let h::Expression prefix = operand(ref p,ordered(ref p,h::function_at(ref p,5).body,0));
    assert(h::expression_code(prefix.kind) == 4 && h::operator_code(prefix.operator) == 2);
    assert(bytes::equal(h::integer_text(ref p,prefix.left),b"1"));
    assert(h::expression_count(ref p) == 7);
}

@test
fn void ordered_control_for_clauses_defer_and_nearest_loop() {
    let h::Program p = retained(b"module main; fn void done(i32 x){} fn void work(){var i32 n=0; defer done(n); assert(true); discard done(1); if (true && !false){n+=1;}else{n=2;} while(n<3){for(var i32 i=0;i<2;i+=1){if(i==1){continue;} break;} n+=1; continue;} for(;;){break;} return;}",0);
    let h::Function work = h::function_at(ref p,1); let h::Block body = h::block_at(ref p,work.body); assert(body.edge_count == 8);
    assert(h::statement_code(ordered(ref p,work.body,0).kind) == 1);
    let h::Statement deferred = ordered(ref p,work.body,1); assert(h::statement_code(deferred.kind) == 11 && h::expression_code(operand(ref p,deferred).kind) == 6);
    assert(h::type_code(operand(ref p,deferred).type) == 7);
    let h::Statement assertion = ordered(ref p,work.body,2); assert(h::statement_code(assertion.kind) == 9 && operand(ref p,assertion).boolean);
    assert(h::statement_code(ordered(ref p,work.body,3).kind) == 10);
    let h::Expression branch = operand(ref p,ordered(ref p,work.body,4)); assert(h::expression_code(branch.kind) == 8 && branch.right == 0 && branch.alternative != 0);
    let h::Expression condition = h::expression_at(ref p,branch.left); assert(h::operator_code(condition.operator) == 19);
    let h::Expression not_false = h::expression_at(ref p,condition.right); assert(h::operator_code(not_false.operator) == 3 && !h::expression_at(ref p,not_false.left).boolean);
    assert(h::statement_code(ordered(ref p,branch.block,0).kind) == 3);
    let h::Expression alternative = h::expression_at(ref p,branch.alternative); assert(h::expression_code(alternative.kind) == 7 && h::block_at(ref p,alternative.block).edge_count == 1);
    let usize outer_id = ordered_id(ref p,work.body,5); let h::Statement outer = h::statement_at(ref p,outer_id); assert(h::statement_code(outer.kind) == 5);
    let usize for_id = ordered_id(ref p,outer.body,0); let h::Statement loop = h::statement_at(ref p,for_id); assert(h::statement_code(loop.kind) == 6);
    assert(loop.initial != 0 && loop.expression != 0 && loop.post != 0 && loop.body != 0);
    let h::Statement initial = h::statement_at(ref p,loop.initial); let h::Statement post = h::statement_at(ref p,loop.post);
    assert(h::statement_code(initial.kind) == 1 && h::statement_code(post.kind) == 3 && h::operator_code(post.operator) == 1);
    assert(h::expression_at(ref p,post.target).binding == initial.binding);
    let h::Expression loop_condition = h::expression_at(ref p,loop.expression); assert(h::expression_at(ref p,loop_condition.left).binding == initial.binding);
    let h::Binding counter = h::binding_at(ref p,initial.binding); assert(counter.owner == work.binding && counter.scope == loop.scope);
    assert(h::block_at(ref p,loop.body).parent_scope == loop.scope);
    let h::Expression inner_if = operand(ref p,ordered(ref p,loop.body,0)); let h::Statement inner_continue = ordered(ref p,inner_if.block,0);
    assert(h::statement_code(inner_continue.kind) == 8 && inner_continue.loop_target == for_id);
    let h::Statement inner_break = ordered(ref p,loop.body,1); assert(h::statement_code(inner_break.kind) == 7 && inner_break.loop_target == for_id);
    let h::Statement outer_continue = ordered(ref p,outer.body,2); assert(outer_continue.loop_target == outer_id);
    let usize empty_id = ordered_id(ref p,work.body,6); let h::Statement empty = h::statement_at(ref p,empty_id);
    assert(empty.initial == 0 && empty.expression == 0 && empty.post == 0 && empty.body != 0);
    assert(ordered(ref p,empty.body,0).loop_target == empty_id);
    let h::Statement returned = ordered(ref p,work.body,7); assert(h::statement_code(returned.kind) == 2 && returned.expression == 0);
}

@test
fn void tail_if_else_if_and_scalar_operators() {
    let h::Program p = retained(b"module main; fn i32 f(i32 x){var i32 a=+x; a=-a; a=~a; a*=2; a/=2; a%=2; a&=2; a|=2; a^=2; a<<=1; a>>=1; let bool b=(a==1 || a!=2) && (a<=3 && a>=0); return if(b){a+1}else if(a>2){a-1}else{a*2};}",0);
    let h::Function function = h::function_at(ref p,0); let h::Block body = h::block_at(ref p,function.body); assert(body.edge_count == 13 && body.tail == 0);
    let h::Expression positive = operand(ref p,ordered(ref p,function.body,0)); assert(h::operator_code(positive.operator) == 1);
    assert(h::operator_code(operand(ref p,ordered(ref p,function.body,1)).operator) == 2);
    assert(h::operator_code(operand(ref p,ordered(ref p,function.body,2)).operator) == 4);
    assert(h::operator_code(ordered(ref p,function.body,3).operator) == 5);
    assert(h::operator_code(ordered(ref p,function.body,4).operator) == 6);
    assert(h::operator_code(ordered(ref p,function.body,5).operator) == 7);
    assert(h::operator_code(ordered(ref p,function.body,6).operator) == 8);
    assert(h::operator_code(ordered(ref p,function.body,7).operator) == 9);
    assert(h::operator_code(ordered(ref p,function.body,8).operator) == 10);
    assert(h::operator_code(ordered(ref p,function.body,9).operator) == 11);
    assert(h::operator_code(ordered(ref p,function.body,10).operator) == 12);
    let h::Statement returned = ordered(ref p,function.body,12); assert(h::statement_code(returned.kind) == 2);
    let h::Expression branch = operand(ref p,returned); assert(h::expression_code(branch.kind) == 8 && h::type_code(branch.type) == 1);
    assert(h::block_at(ref p,branch.block).edge_count == 0 && h::block_at(ref p,branch.block).tail != 0);
    let h::Expression elseif = h::expression_at(ref p,branch.alternative); assert(h::expression_code(elseif.kind) == 8);
    assert(h::expression_code(h::expression_at(ref p,elseif.alternative).kind) == 7);
}

fn h::Program imported() {
    let ref u8[] main_source = b"module main; import lib as m; fn i64 f(i64 x){return m::id(x);}";
    var v::Vec<syntax::Parsed> files = v::empty<syntax::Parsed>();
    assert(v::push<syntax::Parsed>(ref mut files,parsed(main_source)).is_ok);
    assert(v::push<syntax::Parsed>(ref mut files,parsed(b"module lib; fn void hidden(){} pub fn i64 id(i64 x){return x;}")).is_ok);
    let modules::Registry registry = switch(modules::build(ref files)) { case .ok(value): { value } case .err(_): { assert(false); return imported(); } };
    let typing::Typed typed = typing::check(names::resolve_modules(parsed(main_source),ref registry,0)); assert(typing::problem(ref typed).code == 0);
    return switch(lower::lower(typed,0)) { case .ok(value): { value } case .err(_): { assert(false); return imported(); } };
}
@test
fn void imported_target_survives_all_input_owners() {
    let h::Program p = imported(); let h::Function f = h::function_at(ref p,0);
    let h::Expression call = operand(ref p,ordered(ref p,f.body,0)); assert(h::expression_code(call.kind) == 6);
    let h::Binding target = h::binding_at(ref p,call.binding);
    assert(h::binding_code(target.kind) == 5 && target.module_index == 1 && target.declaration == 3);
    assert(h::type_code(target.type) == 2 && target.signature_count == 1 && h::type_code(h::signature_at(ref p,target.signature_start)) == 2);
    assert(call.edge_count == 1); let h::Expression arg = h::expression_at(ref p,h::edge_at(ref p,call.edge_start));
    assert(arg.binding == h::edge_at(ref p,f.parameter_start)); assert(h::binding_at(ref p,arg.binding).owner == f.binding);
    assert(bytes::equal(h::binding_text(ref p,call.binding),b"m::id"));
}

@test
fn void invalid_typed_input_returns_only_error() {
    assert(rejection(b"module main; fn i64 f(){return true;}") == 1);
    assert(rejection(b"module main; fn void f(){break;}") == 1);
    assert(rejection(b"module main; fn i64 f(){return 9223372036854775808;}") == 1);
    assert(rejection(b"module main; fn i32 f(i32 x){return x;} fn i32 g(){return f();}") == 1);
    let h::Program empty = retained(b"module main;",0); assert(h::function_count(ref empty) == 0 && h::expression_count(ref empty) == 0);
}
