module hir_model;
import hir_validate_symbols as symbols;
import hir_validate_control as control;
import hir_validate_drops as drops;
import hir_validate_expr_basic as expr_basic;
import hir_validate_expr_unsafe as expr_unsafe;
import hir_validate_expr_values as expr_values;
import hir_validate_unsafe as unsafe_proof;
import scalar_views as views;
import candidate_limits as limits;
import c_abi_types as c_types;
import pointer_types as pointers;
import borrow_shapes as borrows;
import reference_types as refs;
import array_types as arrays;
import resource_identity as resources;
import enum_types as enums;
import tagged_types as tagged;
import literal_types as literals;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;

// Private owned HIR. No syntax/type input table is retained. All entity IDs are
// one-based; zero is absent. Query table indexes and edge offsets are zero-based.
// Module indexes remain zero-based and declaration IDs retain their source index
// plus one (including module/import declarations). Binding IDs preserve resolved
// symbol IDs, but their owner/target/type facts below are independently owned.
pub enum Type { ScalarView{u32 id;}; CScalar{u32 id;}; Pointer {u32 id;}; Borrowed { u32 id; }; Invalid; I32; I64; U8; U32; Usize; Bool; Void; Callable; Resource { u32 id; }; Nominal { u32 id; }; Array { u32 id; }; Text; Bytes; MutableBytes; CStr; Enumeration { u32 id; }; Option { u32 id; }; Result { u32 id; }; }
pub enum BindingKind { Function; Parameter; Let; Var; External; Foreign; Struct; Enum; Resource; DropFunction; DropReceiver; }
pub enum ExpressionKind { Integer; Boolean; Reference; Unary; Binary; Call; Block; If; Construct; Field; Initializer; Array; Index; Length; Literal; Borrow; Slice; Variant; Switch; Arm; Capture; Propagate; IsOk; Move; Reborrow; Cast; Null; PointerCast; RawView; Storage {u32 code;u32 element;}; }
pub enum StatementKind { Binding; Return; Assignment; Expression; While; For; Break; Continue; Assert; Discard; Defer; Burn; }
pub enum Operator { None; Plus; Minus; Not; Complement; Multiply; Divide; Remainder; BitAnd; BitOr; BitXor; ShiftLeft; ShiftRight; Equal; NotEqual; Less; LessEqual; Greater; GreaterEqual; And; Or; Assign; }
// The fixed SDK/compiler admits enum switching, not enum equality or enum casts.
// These exhaustive private-format tags also permit exact scalar comparisons.
pub fn u32 type_code(Type value) { return switch(value) { case .ScalarView{.id=id}:{id} case .CScalar{.id=id}:{id} case .Pointer{.id=id}:{id} case .Borrowed { .id=id }: { id } case .Invalid: { 0 } case .I32: { 1 } case .I64: { 2 } case .U8: { 3 } case .U32: { 4 } case .Usize: { 5 } case .Bool: { 6 } case .Void: { 7 } case .Callable: { 8 } case .Text: { 9 } case .Bytes: { 10 } case .MutableBytes: { 11 } case .CStr:{12} case .Option { .id = id }: { id } case .Result { .id = id }: { id } case .Enumeration { .id = id }: { id } case .Resource { .id = id }: { id } case .Nominal { .id = id }: { id } case .Array { .id = id }: { id } }; }
pub fn bool represented_type(Type value) { return switch(value) { case .ScalarView{.id=id}:{views::extended(id)} case .CScalar{.id=id}:{c_types::valid(id)} case .Pointer{.id=id}:{pointers::valid(id)} case .Borrowed { .id=id }: { refs::valid(id) } case .Invalid: { false } case .Callable: { false } case .Option { .id = id }: { tagged::option(id) } case .Result { .id = id }: { tagged::result(id) } case .Enumeration { .id = id }: { enums::valid(id) && !tagged::builtin(id) } case .Resource { .id = id }: { resources::valid(id) } case .Nominal { .id = id }: { id >= 256 && id <= 767 } case .Array { .id = id }: { arrays::valid(id) } case .I32: { true } case .I64: { true } case .U8: { true } case .U32: { true } case .Usize: { true } case .Bool: { true } case .Void: { true } case .Text: { true } case .Bytes: { true } case .MutableBytes: { true } case .CStr:{true} }; }
pub fn u32 binding_code(BindingKind value) { return switch(value) { case .Function: { 1 } case .Parameter: { 2 } case .Let: { 3 } case .Var: { 4 } case .Foreign:{11} case .External: { 5 } case .Struct: { 6 } case .Enum: { 7 } case .Resource: { 8 } case .DropFunction: { 9 } case .DropReceiver: { 10 } }; }
pub fn u32 expression_code(ExpressionKind value) { return switch(value) { case .Integer: { 1 } case .Boolean: { 2 } case .Reference: { 3 } case .Unary: { 4 } case .Binary: { 5 } case .Call: { 6 } case .Block: { 7 } case .If: { 8 } case .Construct: { 9 } case .Field: { 10 } case .Initializer: { 11 } case .Array: { 12 } case .Index: { 13 } case .Length: { 14 } case .Literal: { 15 } case .Borrow: { 16 } case .Slice: { 17 } case .Variant: { 18 } case .Switch: { 19 } case .Arm: { 20 } case .Capture: { 21 } case .Propagate: { 22 } case .IsOk: { 23 } case .Move: { 24 } case .Reborrow: { 25 } case .Cast: { 26 } case .Null:{27} case .PointerCast:{28} case .RawView:{29} case .Storage{.code=code,.element=element}:{30} }; }
pub fn u32 storage_code(ExpressionKind k){return switch(k){case .Storage{.code=code,.element=_}:{code}case .Integer:{0} case .Boolean:{0} case .Reference:{0} case .Unary:{0} case .Binary:{0} case .Call:{0} case .Block:{0} case .If:{0} case .Construct:{0} case .Field:{0} case .Initializer:{0} case .Array:{0} case .Index:{0} case .Length:{0} case .Literal:{0} case .Borrow:{0} case .Slice:{0} case .Variant:{0} case .Switch:{0} case .Arm:{0} case .Capture:{0} case .Propagate:{0} case .IsOk:{0} case .Move:{0} case .Reborrow:{0} case .Cast:{0} case .Null:{0} case .PointerCast:{0} case .RawView:{0}};}
pub fn u32 storage_element(ExpressionKind k){return switch(k){case .Storage{.code=_,.element=element}:{element}case .Integer:{0} case .Boolean:{0} case .Reference:{0} case .Unary:{0} case .Binary:{0} case .Call:{0} case .Block:{0} case .If:{0} case .Construct:{0} case .Field:{0} case .Initializer:{0} case .Array:{0} case .Index:{0} case .Length:{0} case .Literal:{0} case .Borrow:{0} case .Slice:{0} case .Variant:{0} case .Switch:{0} case .Arm:{0} case .Capture:{0} case .Propagate:{0} case .IsOk:{0} case .Move:{0} case .Reborrow:{0} case .Cast:{0} case .Null:{0} case .PointerCast:{0} case .RawView:{0}};}
pub fn u32 statement_code(StatementKind value) { return switch(value) { case .Binding: { 1 } case .Return: { 2 } case .Assignment: { 3 } case .Expression: { 4 } case .While: { 5 } case .For: { 6 } case .Break: { 7 } case .Continue: { 8 } case .Assert: { 9 } case .Discard: { 10 } case .Defer: { 11 } case .Burn: { 12 } }; }
pub fn u32 operator_code(Operator value) { return switch(value) { case .None: { 0 } case .Plus: { 1 } case .Minus: { 2 } case .Not: { 3 } case .Complement: { 4 } case .Multiply: { 5 } case .Divide: { 6 } case .Remainder: { 7 } case .BitAnd: { 8 } case .BitOr: { 9 } case .BitXor: { 10 } case .ShiftLeft: { 11 } case .ShiftRight: { 12 } case .Equal: { 13 } case .NotEqual: { 14 } case .Less: { 15 } case .LessEqual: { 16 } case .Greater: { 17 } case .GreaterEqual: { 18 } case .And: { 19 } case .Or: { 20 } case .Assign: { 21 } }; }
pub struct Problem { u32 code; usize start; usize end; }
pub struct Signature { Type type; }
// Binding owner is the enclosing Function Binding ID; scope is its lexical scope
// ID. Function/External use module_index+declaration as authoritative call target.
// Callable result is type; signature_start/count index the Type signature arena.
pub struct Binding { usize link_start; usize link_end; bool unsafe_call; bool must_use; u32 effects; BindingKind kind; Type type; usize start; usize end; usize owner; usize scope; usize module_index; usize declaration; usize signature_start; usize signature_count; }
pub struct Function { usize binding; usize start; usize end; usize scope; bool public; usize body; usize parameter_start; usize parameter_count; }
// An empty body has no callable/receiver. Every resource owns exactly one
// classified drop definition; a nonempty body links one private typed function.
pub struct Drop { u32 abilities; u32 type; usize function; usize receiver; usize start; usize end; bool nonempty; }
// left/right/alternative are Expression IDs. If uses left=condition,
// block=consequent Block ID, alternative=else Expression ID (zero if absent),
// right=0. Binary uses left/right; Unary uses left.
// block is Block ID (Block expression or If consequent). Call binding is target;
// edge_start/count index ordered argument Expression IDs. Reference binding is
// authoritative; callee syntax and grouping nodes are eliminated. Integer stores
// canonical decimal ASCII in payload[start..end]; signed-minimum has no child.
// Construct binding is a Struct or Resource Binding ID; its edges retain Initializer
// Expression IDs in written order. Initializer/Field binding is a one-based
// nominal Field ID. Initializer left is the value; Field left is the owner.
pub struct Expression { ExpressionKind kind; Type type; Operator operator; usize start; usize end; usize left; usize right; usize alternative; usize block; usize binding; usize edge_start; usize edge_count; usize payload_start; usize payload_end; bool boolean; }
// expression and target are Expression IDs; binding is defined local Binding ID.
// body is Block ID; initial/post are Statement IDs for explicit For clauses.
// Omitted for condition is expression=0 (true), initial/post=0 mean absent.
// Assignment operator is Assign or its arithmetic/bitwise operation; RHS is
// evaluated once. Defer captures an unevaluated void Call expression for scope
// exit. Break/Continue target the nearest enclosing While/For Statement ID.
pub struct Statement { StatementKind kind; usize start; usize end; usize scope; usize expression; usize target; usize binding; Operator operator; usize body; usize initial; usize post; usize loop_target; }
// Ordered statement IDs in edge arena; tail is optional Expression ID.
// owner is Function Binding ID; scope is the resolver lexical scope ID.
pub struct Block { bool unsafe_body; usize start; usize end; usize owner; usize scope; usize parent_scope; usize edge_start; usize edge_count; usize tail; }
pub resource Program {
    shapes::Table nominal;
    v::Vec<Drop> drops;
    bytes::Buffer source; bytes::Buffer payload;
    v::Vec<Function> functions; v::Vec<Binding> bindings;
    v::Vec<Expression> expressions; v::Vec<Statement> statements;
    v::Vec<Block> blocks; v::Vec<usize> edges; v::Vec<Signature> signatures;
    usize module_index; bool success;
    cleanup { }
}
// Hard caps: source 1048576 bytes, 512 functions/declarations, 1152 bindings,
// 16384 expressions, 8192 statements, 8192 blocks, 32768 edges, 32768 signature
// entries, 344064 canonical payload bytes (16384*21), recursion depth 64.
// Errors: 1 unsuccessful input, 2 budget, 3 allocation, 4 unsupported/malformed
// semantic graph. Failure returns .err only, never a partially usable Program.
pub fn usize function_count(ref Program p) { assert(p.success); return v::len<Function>(ref p.functions); }
pub fn usize binding_count(ref Program p) { assert(p.success); return v::len<Binding>(ref p.bindings); }
pub fn usize expression_count(ref Program p) { assert(p.success); return v::len<Expression>(ref p.expressions); }
pub fn usize statement_count(ref Program p) { assert(p.success); return v::len<Statement>(ref p.statements); }
pub fn usize block_count(ref Program p) { assert(p.success); return v::len<Block>(ref p.blocks); }
pub fn usize edge_count(ref Program p) { assert(p.success); return v::len<usize>(ref p.edges); }
pub fn usize signature_count(ref Program p) { assert(p.success); return v::len<Signature>(ref p.signatures); }
pub fn bool may_drop(ref Program p,u32 type,usize depth){
    if(depth>=64){return false;}if(!shapes::needs_drop(type)){return true;}
    if(tagged::option(type)){return may_drop(p,tagged::inner(type),depth+1);}if(tagged::result(type)){return may_drop(p,tagged::success(type),depth+1) && may_drop(p,tagged::failure(type),depth+1);}
    var usize at=0;while(at<drop_count(p) && drop_at(p,at).type!=type){at+=1;}if(at==drop_count(p) || (drop_at(p,at).abilities&1)==0){return false;}
    let sh=shapes::at(nominal(p),shapes::find(nominal(p),type)-1);var usize i=0;while(i<sh.field_count){if(!may_drop(p,shapes::field_at(nominal(p),sh.field_start+i+1).type,depth+1)){return false;}i+=1;}return true;
}
struct EffectBudget {usize work;}
fn u32 drop_effects_walk(ref Program p,u32 type,usize depth,ref mut EffectBudget work){
    if(depth>=64 || work.work>=16384){return 4096;}work.work+=1;if(!shapes::needs_drop(type)){return 0;}
    if(tagged::option(type)){return drop_effects_walk(p,tagged::inner(type),depth+1,work);}
    if(tagged::result(type)){let u32 first=drop_effects_walk(p,tagged::success(type),depth+1,work);return first|drop_effects_walk(p,tagged::failure(type),depth+1,work);}
    let shape=shapes::at(nominal(p),shapes::find(nominal(p),type)-1);var u32 bits=binding_at(p,shape.binding).effects;var usize i=0;
    while(i<shape.field_count){bits|=drop_effects_walk(p,shapes::field_at(nominal(p),shape.field_start+i+1).type,depth+1,work);i+=1;}return bits;
}
pub fn u32 required_drop_effects(ref Program p,u32 type){var EffectBudget work=EffectBudget{.work=0};return drop_effects_walk(p,type,0,ref mut work);}
pub fn usize drop_count(ref Program p){assert(p.success);return v::len<Drop>(ref p.drops);}
pub fn Drop drop_at(ref Program p,usize index){assert(index<drop_count(p));return *v::at<Drop>(ref p.drops,index);}
pub fn bool function_kind(BindingKind kind){let u32 code=binding_code(kind);return code==1 || code==9;}
pub fn ref u8[] literal_bytes(ref Program p, usize id) { let Expression e = expression_at(p,id); assert(expression_code(e.kind) == 15); return bytes::view(ref p.payload)[e.payload_start..e.payload_end]; }
pub fn bool has_views(ref Program p) { var usize i=1; while(i<=expression_count(p)){let u32 kind=expression_code(expression_at(p,i).kind);if(kind==16 || kind==17 || borrows::count(type_code(expression_at(p,i).type))!=0 || type_code(expression_at(p,i).type)==11){return true;}i+=1;}return false; }
pub fn bool has_literals(ref Program p) { var usize i = 1; while (i <= binding_count(p)) { if (literals::valid(type_code(binding_at(p,i).type))) { return true; } i += 1; } i = 1; while (i <= expression_count(p)) { if (literals::valid(type_code(expression_at(p,i).type))) { return true; } i += 1; } return false; }
pub fn ref shapes::Table nominal(ref Program p) { assert(p.success); return ref p.nominal; }
pub fn usize module_index(ref Program p) { assert(p.success); return p.module_index; }
pub fn Function function_at(ref Program p, usize index) { assert(index < function_count(p)); return *v::at<Function>(ref p.functions,index); }
pub fn Binding binding_at(ref Program p, usize id) { assert(id > 0 && id <= binding_count(p)); return *v::at<Binding>(ref p.bindings,id - 1); }
pub fn Expression expression_at(ref Program p, usize id) { assert(id > 0 && id <= expression_count(p)); return *v::at<Expression>(ref p.expressions,id - 1); }
pub fn Statement statement_at(ref Program p, usize id) { assert(id > 0 && id <= statement_count(p)); return *v::at<Statement>(ref p.statements,id - 1); }
pub fn Block block_at(ref Program p, usize id) { assert(id > 0 && id <= block_count(p)); return *v::at<Block>(ref p.blocks,id - 1); }
pub fn usize edge_at(ref Program p, usize index) { assert(index < edge_count(p)); return *v::at<usize>(ref p.edges,index); }
pub fn Type signature_at(ref Program p, usize index) { assert(index < signature_count(p)); return v::at<Signature>(ref p.signatures,index).type; }
pub fn ref u8[] source_view(ref Program p) { assert(p.success); return bytes::view(ref p.source); }
pub fn ref u8[] integer_text(ref Program p, usize id) { let Expression e = expression_at(p,id); assert(expression_code(e.kind) == expression_code(.Integer)); return bytes::view(ref p.payload)[e.payload_start..e.payload_end]; }
pub fn ref u8[] binding_text(ref Program p, usize id) { let Binding b = binding_at(p,id); return source_view(p)[b.start..b.end]; }

// Keep enum shape/capture checks outside the large general assembly verifier.
// Earlier expressions have already passed their local shape checks.
// Resource fields are module-private. This checked factory is the lowering
// handoff, after all local SDK containers have been populated. No container
// mutation through a resource field is required by either construction or query.
pub fn Result<Program,Problem> assemble(bytes::Buffer source, bytes::Buffer payload, v::Vec<Function> functions, v::Vec<Binding> bindings, v::Vec<Expression> expressions, v::Vec<Statement> statements, v::Vec<Block> blocks, v::Vec<usize> edges, v::Vec<Signature> signatures, usize module_index) {
    return assemble_nominal(source,payload,functions,bindings,expressions,statements,blocks,edges,signatures,module_index,shapes::empty());
}
pub fn Result<Program,Problem> assemble_nominal(bytes::Buffer source, bytes::Buffer payload, v::Vec<Function> functions, v::Vec<Binding> bindings, v::Vec<Expression> expressions, v::Vec<Statement> statements, v::Vec<Block> blocks, v::Vec<usize> edges, v::Vec<Signature> signatures, usize module_index, shapes::Table nominal) {
    return assemble_resources(source,payload,functions,bindings,expressions,statements,blocks,edges,signatures,module_index,nominal,v::empty<Drop>());
}
pub fn Result<Program,Problem> assemble_resources(bytes::Buffer source, bytes::Buffer payload, v::Vec<Function> functions, v::Vec<Binding> bindings, v::Vec<Expression> expressions, v::Vec<Statement> statements, v::Vec<Block> blocks, v::Vec<usize> edges, v::Vec<Signature> signatures, usize module_index, shapes::Table nominal,v::Vec<Drop> drops) {
    let usize ns = bytes::len(ref source); let usize np = bytes::len(ref payload);
    let usize nf = v::len<Function>(ref functions); let usize nb = v::len<Binding>(ref bindings);
    let usize ne = v::len<Expression>(ref expressions); let usize nt = v::len<Statement>(ref statements);
    let usize nk = v::len<Block>(ref blocks); let usize na = v::len<usize>(ref edges); let usize ntypes = v::len<Signature>(ref signatures);
    if (ns > limits::program_bytes() || np > limits::program_expressions()*21 || nf > 512 || nb > 1152 || ne > limits::program_expressions() || nt > 8192 || nk > 8192 || na > 32768 || ntypes > 32768 || v::len<Drop>(ref drops)>512) { return .err(Problem { .code = 2, .start = 0, .end = ns }); }
    let symbols_problem=symbols::check(ref source, ref functions, ref bindings, ref blocks, ref edges, ref signatures, ref nominal, module_index);if(symbols_problem.code!=0){return .err(symbols_problem);}
    var usize i=0;while(i<ne){
        let Expression e=*v::at<Expression>(ref expressions,i);
    let expr_basic_problem=expr_basic::check(ref source, ref payload, ref functions, ref bindings, ref expressions, ref blocks, ref edges, ref signatures, ref nominal, e, i);if(expr_basic_problem.code!=0){return .err(expr_basic_problem);}
    let expr_unsafe_problem=expr_unsafe::check(ref source, ref bindings, ref expressions, ref nominal, e);if(expr_unsafe_problem.code!=0){return .err(expr_unsafe_problem);}
    let expr_values_problem=expr_values::check(ref payload, ref bindings, ref expressions, ref statements, ref blocks, ref edges, ref nominal, e, i);if(expr_values_problem.code!=0){return .err(expr_values_problem);}
        i+=1;
    }
    let control_problem=control::check(ref source, ref bindings, ref expressions, ref statements, ref blocks, ref edges);if(control_problem.code!=0){return .err(control_problem);}
    let drops_problem=drops::check(ref source, ref functions, ref bindings, ref edges, ref nominal, ref drops);if(drops_problem.code!=0){return .err(drops_problem);}
    let Program result=Program { .nominal, .drops, .source, .payload, .functions, .bindings, .expressions, .statements, .blocks, .edges, .signatures, .module_index, .success = true };
    let Problem unsafe_proof=unsafe_check(ref result);if(unsafe_proof.code!=0){return .err(unsafe_proof);}return .ok(result);
}

// Walk owned lexical structure rather than trusting the source-side permission
// flag. Unsafe never grants effect authority or alters borrow/cleanup policy.
pub fn Problem unsafe_check(ref Program p){return unsafe_proof::unsafe_check(p);}
