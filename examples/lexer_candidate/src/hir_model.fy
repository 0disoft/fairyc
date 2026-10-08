module hir_model;
import candidate_limits as limits;
import c_abi_types as c_types;
import pointer_types as pointers;
import raw_view_intrinsics as raw;
import borrow_shapes as borrows;
import scalar_casts as casts;
import must_use_flow as must_use;
import ffi_symbols as ffi;
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
pub enum Type { CScalar{u32 id;}; Pointer {u32 id;}; Borrowed { u32 id; }; Invalid; I32; I64; U8; U32; Usize; Bool; Void; Callable; Resource { u32 id; }; Nominal { u32 id; }; Array { u32 id; }; Text; Bytes; MutableBytes; CStr; Enumeration { u32 id; }; Option { u32 id; }; Result { u32 id; }; }
pub enum BindingKind { Function; Parameter; Let; Var; External; Foreign; Struct; Enum; Resource; DropFunction; DropReceiver; }
pub enum ExpressionKind { Integer; Boolean; Reference; Unary; Binary; Call; Block; If; Construct; Field; Initializer; Array; Index; Length; Literal; Borrow; Slice; Variant; Switch; Arm; Capture; Propagate; IsOk; Move; Reborrow; Cast; Null; PointerCast; RawView; }
pub enum StatementKind { Binding; Return; Assignment; Expression; While; For; Break; Continue; Assert; Discard; Defer; Burn; }
pub enum Operator { None; Plus; Minus; Not; Complement; Multiply; Divide; Remainder; BitAnd; BitOr; BitXor; ShiftLeft; ShiftRight; Equal; NotEqual; Less; LessEqual; Greater; GreaterEqual; And; Or; Assign; }
// The fixed SDK/compiler admits enum switching, not enum equality or enum casts.
// These exhaustive private-format tags also permit exact scalar comparisons.
pub fn u32 type_code(Type value) { return switch(value) { case .CScalar{.id=id}:{id} case .Pointer{.id=id}:{id} case .Borrowed { .id=id }: { id } case .Invalid: { 0 } case .I32: { 1 } case .I64: { 2 } case .U8: { 3 } case .U32: { 4 } case .Usize: { 5 } case .Bool: { 6 } case .Void: { 7 } case .Callable: { 8 } case .Text: { 9 } case .Bytes: { 10 } case .MutableBytes: { 11 } case .CStr:{12} case .Option { .id = id }: { id } case .Result { .id = id }: { id } case .Enumeration { .id = id }: { id } case .Resource { .id = id }: { id } case .Nominal { .id = id }: { id } case .Array { .id = id }: { id } }; }
pub fn bool represented_type(Type value) { return switch(value) { case .CScalar{.id=id}:{c_types::valid(id)} case .Pointer{.id=id}:{pointers::valid(id)} case .Borrowed { .id=id }: { refs::valid(id) } case .Invalid: { false } case .Callable: { false } case .Option { .id = id }: { tagged::option(id) } case .Result { .id = id }: { tagged::result(id) } case .Enumeration { .id = id }: { enums::valid(id) && !tagged::builtin(id) } case .Resource { .id = id }: { resources::valid(id) } case .Nominal { .id = id }: { id >= 256 && id <= 767 } case .Array { .id = id }: { arrays::valid(id) } case .I32: { true } case .I64: { true } case .U8: { true } case .U32: { true } case .Usize: { true } case .Bool: { true } case .Void: { true } case .Text: { true } case .Bytes: { true } case .MutableBytes: { true } case .CStr:{true} }; }
pub fn u32 binding_code(BindingKind value) { return switch(value) { case .Function: { 1 } case .Parameter: { 2 } case .Let: { 3 } case .Var: { 4 } case .Foreign:{11} case .External: { 5 } case .Struct: { 6 } case .Enum: { 7 } case .Resource: { 8 } case .DropFunction: { 9 } case .DropReceiver: { 10 } }; }
pub fn u32 expression_code(ExpressionKind value) { return switch(value) { case .Integer: { 1 } case .Boolean: { 2 } case .Reference: { 3 } case .Unary: { 4 } case .Binary: { 5 } case .Call: { 6 } case .Block: { 7 } case .If: { 8 } case .Construct: { 9 } case .Field: { 10 } case .Initializer: { 11 } case .Array: { 12 } case .Index: { 13 } case .Length: { 14 } case .Literal: { 15 } case .Borrow: { 16 } case .Slice: { 17 } case .Variant: { 18 } case .Switch: { 19 } case .Arm: { 20 } case .Capture: { 21 } case .Propagate: { 22 } case .IsOk: { 23 } case .Move: { 24 } case .Reborrow: { 25 } case .Cast: { 26 } case .Null:{27} case .PointerCast:{28} case .RawView:{29} }; }
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
// 4096 expressions, 8192 statements, 8192 blocks, 32768 edges, 32768 signature
// entries, 86016 canonical payload bytes (4096*21), recursion depth 64.
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
fn bool terminal_arm(ref v::Vec<Block> blocks,ref v::Vec<Statement> statements,ref v::Vec<Expression> expressions,ref v::Vec<usize> edges,usize id,bool allow_return) {
    if(id==0 || id>v::len<Block>(blocks)){return false;}let b=*v::at<Block>(blocks,id-1);
    if(b.tail!=0 || b.edge_count==0 || b.edge_start>v::len<usize>(edges) || b.edge_count>v::len<usize>(edges)-b.edge_start){return false;}
    let usize sid=*v::at<usize>(edges,b.edge_start+b.edge_count-1);if(sid==0 || sid>v::len<Statement>(statements)){return false;}
    let s=*v::at<Statement>(statements,sid-1);if(allow_return && statement_code(s.kind)==2){return true;}if(statement_code(s.kind)!=9 || s.expression==0 || s.expression>v::len<Expression>(expressions)){return false;}
    let e=*v::at<Expression>(expressions,s.expression-1);return expression_code(e.kind)==2 && type_code(e.type)==6 && !e.boolean;
}
fn bool enum_expression(ref shapes::Table nominal, ref v::Vec<Expression> expressions, ref v::Vec<Block> blocks, ref v::Vec<Statement> statements, ref v::Vec<Binding> bindings, ref v::Vec<usize> edges, Expression e, usize i) {
    let u32 kind=expression_code(e.kind);let u32 type=type_code(e.type);let usize nb=v::len<Binding>(bindings);
        if (kind == 23) {
            return e.left != 0 && type == 6 && tagged::result(type_code(v::at<Expression>(expressions,e.left-1).type))
                && e.binding == 0 && e.right == 0 && e.alternative == 0 && e.block == 0 && e.edge_count == 0 && e.edge_start == 0
                && e.payload_start == 0 && e.payload_end == 0 && !e.boolean && operator_code(e.operator) == 0;
        }
        if (kind == 22) {
            if (e.left == 0 || e.binding == 0 || e.binding > nb || e.right != 0 || e.alternative != 0 || e.block != 0
                || e.edge_count != 0 || e.edge_start != 0 || e.payload_start != 0 || e.payload_end != 0 || e.boolean || operator_code(e.operator) != 0) { return false; }
            let u32 operand = type_code(v::at<Expression>(expressions,e.left-1).type);
            let Binding owner = *v::at<Binding>(bindings,e.binding-1); let u32 returned = type_code(owner.type);
            return tagged::result(operand) && tagged::result(returned) && binding_code(owner.kind) == 1
                && tagged::failure(operand) == tagged::failure(returned) && type == tagged::success(operand);
        }
        if (kind == 18 || kind == 20) {
            if (e.binding == 0 || e.binding > shapes::field_count(nominal) || e.left != 0 || e.right != 0 || e.alternative != 0 || e.boolean || e.payload_start != 0 || e.payload_end != 0 || operator_code(e.operator) != 0) { return false; }
            let shapes::Field member = shapes::field_at(nominal,e.binding);
            if (!enums::valid(member.owner) || member.type != 0 || kind == 18 && (type != member.owner || e.block != 0) || kind == 20 && e.block == 0) { return false; }
            if(e.edge_count!=member.payload_count){return false;}
            var u8[1024] payload_seen=[0;1024];var usize n=0;
            while(n<e.edge_count){let usize id=*v::at<usize>(edges,e.edge_start+n);if(id==0 || id>i){return false;}
                let Expression f=*v::at<Expression>(expressions,id-1);if(expression_code(f.kind)!=if(kind==18){11}else{21} || f.binding<=member.payload_start || f.binding>member.payload_start+member.payload_count || payload_seen[f.binding-1]!=0){return false;}
                payload_seen[f.binding-1]=1;
                if(kind==20 && f.edge_count==1){let usize binding=*v::at<usize>(edges,f.edge_start);let Binding b=*v::at<Binding>(bindings,binding-1);let Block arm=*v::at<Block>(blocks,e.block-1);if(b.scope!=arm.scope || b.owner!=arm.owner){return false;}}
                n+=1;
            }
            if (kind == 20) { let Block arm = *v::at<Block>(blocks,e.block-1); if (arm.tail > i) { return false; } if (type != if(arm.tail==0){7}else{type_code(v::at<Expression>(expressions,arm.tail-1).type)} && !(tagged::builtin(member.owner) && terminal_arm(blocks,statements,expressions,edges,e.block,tagged::result(member.owner)))) { return false; } }
        }
        if(kind==21){
            if(e.binding==0 || e.binding>shapes::field_count(nominal) || e.edge_count>1 || e.left!=0 || e.right!=0 || e.alternative!=0 || e.block!=0 || e.boolean || e.payload_start!=0 || e.payload_end!=0 || operator_code(e.operator)!=0){return false;}
            let shapes::Field field=shapes::field_at(nominal,e.binding);if(!enums::valid(field.owner) || field.type==0 || type!=field.type){return false;}
            if(e.edge_count==1){let usize binding=*v::at<usize>(edges,e.edge_start);if(binding==0 || binding>nb){return false;}let Binding b=*v::at<Binding>(bindings,binding-1);if(binding_code(b.kind)!=3 || type_code(b.type)!=type){return false;}}
        }
        if (kind == 19) {
            if(e.left==0 || e.right!=0 || e.alternative!=0 || e.block!=0 || e.binding!=0 || e.boolean || e.payload_start!=0 || e.payload_end!=0 || operator_code(e.operator)!=0){return false;}
            let u32 subject = type_code(v::at<Expression>(expressions,e.left-1).type); let usize shape = shapes::find(nominal,subject);
            if(!enums::valid(subject) || shape==0){return false;}
            let shapes::Shape layout = shapes::at(nominal,shape-1);if(e.edge_count!=shapes::member_count(nominal,subject)){return false;}
            var u8[1024] seen=[0;1024];var usize j=0;
            while(j<e.edge_count){let usize id=*v::at<usize>(edges,e.edge_start+j);if(id==0 || id>i){return false;}
                let Expression arm=*v::at<Expression>(expressions,id-1);if(expression_code(arm.kind)!=20 || type_code(arm.type)!=type || arm.binding<=layout.field_start || arm.binding>layout.field_start+layout.field_count || seen[arm.binding-1]!=0){return false;}
                seen[arm.binding-1]=1;j+=1;
            }
        }
    return true;
}

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
    if (ns > limits::program_bytes() || np > 86016 || nf > 512 || nb > 1152 || ne > 4096 || nt > 8192 || nk > 8192 || na > 32768 || ntypes > 32768 || v::len<Drop>(ref drops)>512) { return .err(Problem { .code = 2, .start = 0, .end = ns }); }
    if (!shapes::valid(ref nominal,ns,nb)) { return .err(Problem { .code = 4, .start = 0, .end = ns }); }
    if(!shapes::abi_valid(ref nominal,bytes::view(ref source))){return .err(Problem{.code=5102,.start=0,.end=ns});}
    var usize shape_index = 0;
    while (shape_index < shapes::count(ref nominal)) {
        let shapes::Shape shape = shapes::at(ref nominal,shape_index);
        if(tagged::builtin(shape.type) || arrays::owned(shape.type)){shape_index+=1;continue;}
        let Binding b = *v::at<Binding>(ref bindings,shape.binding - 1);
        let usize abi_slot=shapes::abi_find(ref nominal,shape.type);
        if(abi_slot!=0){let a=shapes::abi_at(ref nominal,abi_slot-1);if(a.name_start!=b.start || a.name_end!=b.end || a.declaration!=b.declaration){return .err(Problem{.code=5102,.start=b.start,.end=b.end});}}
        if (binding_code(b.kind) != if(resources::valid(shape.type)){8}else if(enums::valid(shape.type)){7}else{6} || type_code(b.type) != shape.type || b.declaration != (shape.type - if(resources::valid(shape.type)){98303}else if(enums::valid(shape.type)){131071}else{255}) as usize || b.signature_count != 0 || b.module_index != module_index) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        shape_index += 1;
    }
    var usize i = 0;
    while (i < nb) {
        let Binding b = *v::at<Binding>(ref bindings,i);
        if(binding_code(b.kind)==11){if(!b.unsafe_call || b.link_start>=b.link_end || b.link_end>ns || !ffi::valid(bytes::view(ref source)[b.link_start..b.link_end]) || !shapes::foreign_result(ref nominal,type_code(b.type))){return .err(Problem{.code=5106,.start=b.start,.end=b.end});}
            var usize n=0;while(n<b.signature_count){if(b.signature_start+n>=ntypes || !ffi::scalar(type_code(v::at<Signature>(ref signatures,b.signature_start+n).type),false)){return .err(Problem{.code=5102,.start=b.start,.end=b.end});}n+=1;}
            var usize other=0;while(other<i){let old=*v::at<Binding>(ref bindings,other);if(binding_code(old.kind)==11 && old.link_end<=ns && bytes::equal(bytes::view(ref source)[old.link_start..old.link_end],bytes::view(ref source)[b.link_start..b.link_end])){if(type_code(old.type)!=type_code(b.type) || old.signature_count!=b.signature_count){return .err(Problem{.code=5106,.start=b.start,.end=b.end});}n=0;while(n<b.signature_count){if(type_code(v::at<Signature>(ref signatures,old.signature_start+n).type)!=type_code(v::at<Signature>(ref signatures,b.signature_start+n).type)){return .err(Problem{.code=5106,.start=b.start,.end=b.end});}n+=1;}}other+=1;}
        }else if(b.link_start!=0 || b.link_end!=0){return .err(Problem{.code=4,.start=b.start,.end=b.end});}
        if (b.must_use && (binding_code(b.kind)!=1 && binding_code(b.kind)!=5 && binding_code(b.kind)!=11 || !must_use::eligible(type_code(b.type))) || b.effects>4095 || (b.effects!=0 && !function_kind(b.kind) && binding_code(b.kind)!=5 && binding_code(b.kind)!=8 && binding_code(b.kind)!=11) || b.start > b.end || b.end > ns || b.owner > nb || type_code(b.type) == type_code(.Invalid) || type_code(b.type) == type_code(.Callable) || b.signature_start > ntypes || b.signature_count > ntypes - b.signature_start) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if (!represented_type(b.type) || !shapes::valid_type(ref nominal,type_code(b.type),true) || ((binding_code(b.kind) >= 6 && binding_code(b.kind) <= 8) && shapes::find(ref nominal,type_code(b.type)) == 0)) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if ((binding_code(b.kind) >= 6 && binding_code(b.kind) <= 8) && shapes::at(ref nominal,shapes::find(ref nominal,type_code(b.type)) - 1).binding != i + 1) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if (function_kind(b.kind) || (binding_code(b.kind) == binding_code(.External) || binding_code(b.kind)==11) || binding_code(b.kind) == binding_code(.Struct) || binding_code(b.kind) == 7 || binding_code(b.kind)==8) { if (b.owner != 0 || b.scope != 0 || b.declaration == 0) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); } }
        else if (b.owner == 0 || b.scope == 0 || type_code(b.type) == type_code(.Void) || b.signature_count != 0 || !function_kind(v::at<Binding>(ref bindings,b.owner - 1).kind)) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        i += 1;
    }
    i = 0; while (i < ntypes) { let Type type = v::at<Signature>(ref signatures,i).type; if (!represented_type(type) || !shapes::valid_type(ref nominal,type_code(type),false)) { return .err(Problem { .code = 4, .start = 0, .end = ns }); } i += 1; }
    i = 0;
    while (i < nf) {
        let Function f = *v::at<Function>(ref functions,i);
        if (f.start > f.end || f.end > ns || f.binding == 0 || f.binding > nb || f.body == 0 || f.body > nk || f.parameter_start > na || f.parameter_count > na - f.parameter_start || f.scope == 0) { return .err(Problem { .code = 4, .start = f.start, .end = f.end }); }
        let Binding b = *v::at<Binding>(ref bindings,f.binding - 1); let Block body = *v::at<Block>(ref blocks,f.body - 1);
        if (!function_kind(b.kind) || b.module_index != module_index || b.signature_count != f.parameter_count || body.owner != f.binding || body.parent_scope != f.scope) { return .err(Problem { .code = 4, .start = f.start, .end = f.end }); }
        var usize j = 0;
        while (j < f.parameter_count) {
            let usize id = *v::at<usize>(ref edges,f.parameter_start + j);
            if (id == 0 || id > nb) { return .err(Problem { .code = 4, .start = f.start, .end = f.end }); }
            let Binding parameter = *v::at<Binding>(ref bindings,id - 1);
            if (binding_code(parameter.kind) != if(binding_code(b.kind)==9){10}else{2} || parameter.owner != f.binding || parameter.scope != f.scope || type_code(parameter.type) != type_code(v::at<Signature>(ref signatures,b.signature_start + j).type)) { return .err(Problem { .code = 4, .start = parameter.start, .end = parameter.end }); }
            j += 1;
        }
        i += 1;
    }
    i = 0;
    while (i < ne) {
        let Expression e = *v::at<Expression>(ref expressions,i);
        let u32 kind = expression_code(e.kind); let u32 type = type_code(e.type);
        let bool is_field = kind == 10 || kind == 11 || kind == 18 || kind == 20 || kind == 21;
        if (e.start > e.end || e.end > ns || !represented_type(e.type) || !shapes::valid_type(ref nominal,type,true) || e.left > ne || e.right > ne || e.alternative > ne || e.block > nk || (!is_field && e.binding > nb) || e.edge_start > na || e.edge_count > na - e.edge_start || e.payload_start > e.payload_end || e.payload_end > np) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        // Expression links are postorder and cannot create cycles.
        if (e.left > i || e.right > i || e.alternative > i) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if (expression_code(e.kind) == expression_code(.Integer) && (e.payload_start == e.payload_end || type < 1 || type > 5)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if (expression_code(e.kind) == expression_code(.Boolean) && type_code(e.type) != type_code(.Bool)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if ((kind==3 || kind==24 || kind==6) && e.binding == 0) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if ((expression_code(e.kind) == expression_code(.Unary) || expression_code(e.kind) == expression_code(.Binary)) && (e.left == 0 || operator_code(e.operator) == operator_code(.None))) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if (expression_code(e.kind) == expression_code(.Binary) && e.right == 0) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if ((expression_code(e.kind) == expression_code(.Block) || expression_code(e.kind) == expression_code(.If)) && e.block == 0) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if (expression_code(e.kind) == expression_code(.If) && (e.left == 0 || type_code(v::at<Expression>(ref expressions,e.left - 1).type) != type_code(.Bool))) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if (kind == 3 || kind == 24 || kind == 6) {
            let Binding b = *v::at<Binding>(ref bindings,e.binding - 1);
            if (type_code(e.type) != type_code(b.type) || ((kind == 3 || kind == 24) && (function_kind(b.kind) || (binding_code(b.kind) == binding_code(.External) || binding_code(b.kind)==11) || binding_code(b.kind) == binding_code(.Struct) || binding_code(b.kind) == 7 || binding_code(b.kind)==8)) || (expression_code(e.kind) == expression_code(.Call) && binding_code(b.kind) != binding_code(.Function) && binding_code(b.kind) != binding_code(.External) && binding_code(b.kind)!=11)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
            if (expression_code(e.kind) == expression_code(.Call)) {
                var bool attached=false;var usize owner_index=0;
                while(owner_index<nf){let f=*v::at<Function>(ref functions,owner_index);if(f.start<=e.start && e.end<=f.end){attached=true;if((v::at<Binding>(ref bindings,f.binding-1).effects&b.effects)!=b.effects){return .err(Problem{.code=7102,.start=e.start,.end=e.end});}}owner_index+=1;}
                if(!attached){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
                if (e.edge_count != b.signature_count) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
                var usize j = 0;
                while (j < e.edge_count) {
                    let usize id = *v::at<usize>(ref edges,e.edge_start + j);
                    if (id == 0 || id > i || type_code(v::at<Expression>(ref expressions,id - 1).type) != type_code(v::at<Signature>(ref signatures,b.signature_start + j).type)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); } j += 1;
                }
            }
        }
        if(kind==29){
            if((type!=9 && type!=10 && type!=11 && type!=12) || !e.boolean || e.left==0 || e.right==0 || e.alternative==0 || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.payload_start!=0 || e.payload_end!=0 || operator_code(e.operator)!=0
                || !raw::owner(type_code(v::at<Expression>(ref expressions,e.left-1).type),type) || !raw::data(type_code(v::at<Expression>(ref expressions,e.right-1).type),type) || type_code(v::at<Expression>(ref expressions,e.alternative-1).type)!=5){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
        }
        if(kind==28){
            if(e.left==0 || !pointers::cast_allowed(type_code(v::at<Expression>(ref expressions,e.left-1).type),type) || !e.boolean || e.right!=0 || e.alternative!=0 || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.edge_start!=0 || e.payload_start!=0 || e.payload_end!=0 || operator_code(e.operator)!=0){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
            if(pointers::inner(type)==7 || type==1048601){let child=*v::at<Expression>(ref expressions,e.left-1);if(expression_code(child.kind)!=3 && !(expression_code(child.kind)==16 && expression_code(v::at<Expression>(ref expressions,child.left-1).kind)==3)){return .err(Problem{.code=4,.start=e.start,.end=e.end});}}
        }
        if(kind==27 && (!pointers::valid(type) || e.left!=0 || e.right!=0 || e.alternative!=0 || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.edge_start!=0 || e.payload_start!=0 || e.payload_end!=0 || e.boolean || operator_code(e.operator)!=0)){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
        if(kind==26 && (e.left==0 || !casts::allowed(type_code(v::at<Expression>(ref expressions,e.left-1).type),type) || e.right!=0 || e.alternative!=0 || e.block!=0 || e.binding!=0 || e.edge_count!=0 || e.payload_start!=0 || e.payload_end!=0 || e.boolean || operator_code(e.operator)!=0)){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
        if(kind==25 && (e.left==0 || !refs::valid(type) || refs::mutable(type) || !refs::mutable(type_code(v::at<Expression>(ref expressions,e.left-1).type)) || refs::inner(type_code(v::at<Expression>(ref expressions,e.left-1).type))!=refs::inner(type) || e.right!=0 || e.alternative!=0 || e.binding!=0 || e.edge_count!=0)){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
        if(kind==24 && (!shapes::needs_drop(type) || binding_code(v::at<Binding>(ref bindings,e.binding-1).kind)==10 || e.left!=0 || e.right!=0 || e.alternative!=0 || e.block!=0 || e.edge_count!=0 || e.payload_start!=0 || e.payload_end!=0 || e.boolean)){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
        if (kind == 12) {
            if (!arrays::valid(type) || e.edge_count != if (e.boolean) { 1 } else { arrays::length(type) } || e.left != 0 || e.right != 0 || e.binding != 0) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
            var usize j = 0; while (j < e.edge_count) { let usize id = *v::at<usize>(ref edges,e.edge_start + j); if (id == 0 || id > i || type_code(v::at<Expression>(ref expressions,id - 1).type) != shapes::array_element(ref nominal,type)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); } j += 1; }
        }
        if (kind == 16 && !refs::valid(type) && ((type != 10 && type != 11) || e.boolean != (type == 11) || e.left == 0 || (expression_code(v::at<Expression>(ref expressions,e.left - 1).kind) != 3 && expression_code(v::at<Expression>(ref expressions,e.left - 1).kind) != 10) || !arrays::valid(type_code(v::at<Expression>(ref expressions,e.left - 1).type)) || shapes::array_element(ref nominal,type_code(v::at<Expression>(ref expressions,e.left - 1).type)) != 3 || e.right != 0 || e.alternative != 0 || e.edge_count != 0 || e.binding != 0)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if(kind==16 && refs::valid(type)){if(e.left==0 || (expression_code(v::at<Expression>(ref expressions,e.left-1).kind)!=3 && expression_code(v::at<Expression>(ref expressions,e.left-1).kind)!=10 && expression_code(v::at<Expression>(ref expressions,e.left-1).kind)!=13 && !(expression_code(v::at<Expression>(ref expressions,e.left-1).kind)==4 && operator_code(v::at<Expression>(ref expressions,e.left-1).operator)==5)) || type_code(v::at<Expression>(ref expressions,e.left-1).type)!=refs::inner(type) || e.boolean!=refs::mutable(type) || e.right!=0 || e.alternative!=0 || e.binding!=0 || e.edge_count!=0){return .err(Problem{.code=4,.start=e.start,.end=e.end});}}
        if(kind==16 && refs::mutable(type)){var usize root=e.left;var usize steps=0;while(steps<64 && (expression_code(v::at<Expression>(ref expressions,root-1).kind)==10 || expression_code(v::at<Expression>(ref expressions,root-1).kind)==13 || expression_code(v::at<Expression>(ref expressions,root-1).kind)==4 && operator_code(v::at<Expression>(ref expressions,root-1).operator)==5)){root=v::at<Expression>(ref expressions,root-1).left;steps+=1;}let base=*v::at<Expression>(ref expressions,root-1);if(expression_code(base.kind)!=3 || binding_code(v::at<Binding>(ref bindings,base.binding-1).kind)!=4 && !refs::mutable(type_code(base.type))){return .err(Problem{.code=4,.start=e.start,.end=e.end});}}
        if(kind==16 && (type==10 || type==11)){var usize root=e.left;var usize steps=0;
            while(steps<64 && expression_code(v::at<Expression>(ref expressions,root-1).kind)==10){root=v::at<Expression>(ref expressions,root-1).left;steps+=1;}
            let base=*v::at<Expression>(ref expressions,root-1);
            if(expression_code(base.kind)!=3 || type==11 && (binding_code(v::at<Binding>(ref bindings,base.binding-1).kind)!=4 || resources::valid(type_code(base.type)))){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
        }
        if (kind == 17 && ((type != 10 && type != 11) || e.left == 0 || e.right == 0 || e.alternative == 0 || type_code(v::at<Expression>(ref expressions,e.left - 1).type) != type || type_code(v::at<Expression>(ref expressions,e.right - 1).type) != 5 || type_code(v::at<Expression>(ref expressions,e.alternative - 1).type) != 5 || e.binding != 0 || e.edge_count != 0)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if (kind == 15 && (!literals::valid(type) || type == 9 && !literals::text(bytes::view(ref payload)[e.payload_start..e.payload_end]) || type==12 && !literals::cstring(bytes::view(ref payload)[e.payload_start..e.payload_end]) || e.left != 0 || e.right != 0 || e.binding != 0 || e.edge_count != 0 || e.block != 0 || e.alternative != 0 || e.boolean || operator_code(e.operator) != 0)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        if (kind == 13 || kind == 14) {
            if (e.left == 0 || !(arrays::valid(type_code(v::at<Expression>(ref expressions,e.left - 1).type)) || literals::valid(type_code(v::at<Expression>(ref expressions,e.left - 1).type)) || type_code(v::at<Expression>(ref expressions,e.left - 1).type) == 11) || e.binding != 0 || e.edge_count != 0) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
            let u32 owner = type_code(v::at<Expression>(ref expressions,e.left - 1).type);if(owner==12){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
            if (kind == 13 && (e.right == 0 || type_code(v::at<Expression>(ref expressions,e.right - 1).type) != 5 || type != if (owner == 10 || owner == 11) { 3 } else { shapes::array_element(ref nominal,owner) } || owner == 9 || e.boolean) || kind == 14 && (e.right != 0 || type != if (e.boolean) { 6 } else { 5 } || e.boolean && !literals::valid(owner) && owner != 11)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        }
        if (kind == 10 || kind == 11) {
            if (e.binding == 0 || e.binding > shapes::field_count(ref nominal) || e.left == 0) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
            let shapes::Field f = shapes::field_at(ref nominal,e.binding);
            let u32 left_type = type_code(v::at<Expression>(ref expressions,e.left - 1).type);
            if (arrays::valid(f.owner) || type != f.type || (kind == 10 && (if(refs::valid(left_type)){refs::inner(left_type)}else{left_type}) != f.owner) || (kind == 11 && left_type != f.type)) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
        }
        if (kind == 9) {
            let usize shape = shapes::find(ref nominal,type);
            if (shape == 0 || arrays::valid(type) || e.binding == 0) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
            let shapes::Shape s = shapes::at(ref nominal,shape - 1);
            if (e.binding != s.binding || binding_code(v::at<Binding>(ref bindings,e.binding - 1).kind) != if(resources::valid(type)){8}else{6} || e.edge_count != s.field_count) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
            var u8[1024] seen = [0;1024]; var usize j = 0;
            while (j < e.edge_count) {
                let usize id = *v::at<usize>(ref edges,e.edge_start + j);
                if (id == 0 || id > i) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
                let Expression f = *v::at<Expression>(ref expressions,id - 1);
                if (expression_code(f.kind) != 11 || f.binding <= s.field_start || f.binding > s.field_start + s.field_count || seen[f.binding - 1] != 0) { return .err(Problem { .code = 4, .start = e.start, .end = e.end }); }
                seen[f.binding - 1] = 1; j += 1;
            }
        }
        if(!enum_expression(ref nominal,ref expressions,ref blocks,ref statements,ref bindings,ref edges,e,i)){return .err(Problem{.code=4,.start=e.start,.end=e.end});}
        i += 1;
    }
    i = 0;
    while (i < nt) {
        let Statement s = *v::at<Statement>(ref statements,i);
        if (s.start > s.end || s.end > ns || s.expression > ne || s.target > ne || s.binding > nb || s.body > nk || s.initial > nt || s.post > nt || s.loop_target > nt || s.scope == 0) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
        if (statement_code(s.kind) == statement_code(.Break) || statement_code(s.kind) == statement_code(.Continue)) {
            if (s.loop_target == 0 || s.loop_target > i) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
            let Statement target = *v::at<Statement>(ref statements,s.loop_target - 1);
            if (statement_code(target.kind) != statement_code(.While) && statement_code(target.kind) != statement_code(.For)) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
        }
        if (statement_code(s.kind) == statement_code(.Binding) && (s.binding == 0 || s.expression == 0)) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
        if (statement_code(s.kind) == statement_code(.Assignment) && (s.target == 0 || s.expression == 0 || operator_code(s.operator) == operator_code(.None))) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
        if ((statement_code(s.kind) == statement_code(.While) || statement_code(s.kind) == statement_code(.For)) && s.body == 0) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
        if ((statement_code(s.kind) == statement_code(.Assert) || statement_code(s.kind) == statement_code(.While)) && (s.expression == 0 || type_code(v::at<Expression>(ref expressions,s.expression - 1).type) != type_code(.Bool))) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
        if (statement_code(s.kind) == statement_code(.For) && s.expression != 0 && type_code(v::at<Expression>(ref expressions,s.expression - 1).type) != type_code(.Bool)) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
        if (statement_code(s.kind) == statement_code(.Defer) && (s.expression == 0 || expression_code(v::at<Expression>(ref expressions,s.expression - 1).kind) != expression_code(.Call) || type_code(v::at<Expression>(ref expressions,s.expression - 1).type) != type_code(.Void))) { return .err(Problem { .code = 4, .start = s.start, .end = s.end }); }
        i += 1;
    }
    i = 0;
    while (i < nk) {
        let Block k = *v::at<Block>(ref blocks,i);
        if (k.start > k.end || k.end > ns || k.owner == 0 || k.owner > nb || k.scope == 0 || k.parent_scope == 0 || k.edge_start > na || k.edge_count > na - k.edge_start || k.tail > ne || !function_kind(v::at<Binding>(ref bindings,k.owner - 1).kind)) { return .err(Problem { .code = 4, .start = k.start, .end = k.end }); }
        var usize j = 0; while (j < k.edge_count) { let usize id = *v::at<usize>(ref edges,k.edge_start + j); if (id == 0 || id > nt) { return .err(Problem { .code = 4, .start = k.start, .end = k.end }); } j += 1; } i += 1;
    }
    // Classification is owned data; never reconstruct a nonempty body from
    // fields or silently turn a missing private callable into an empty drop.
    var usize resources_count=0;i=0;
    while(i<shapes::count(ref nominal)){if(resources::valid(shapes::at(ref nominal,i).type)){resources_count+=1;}i+=1;}
    if(resources_count!=v::len<Drop>(ref drops)){return .err(Problem{.code=4,.start=0,.end=ns});}
    var u8[1153] linked=[0;1153];i=0;
    while(i<v::len<Drop>(ref drops)){
        let Drop d=*v::at<Drop>(ref drops,i);
        if(!resources::valid(d.type) || shapes::find(ref nominal,d.type)==0 || d.start>=d.end || d.end>ns || d.function>nb || d.receiver>nb){return .err(Problem{.code=4,.start=d.start,.end=d.end});}
        var usize j=0;while(j<i){if(v::at<Drop>(ref drops,j).type==d.type){return .err(Problem{.code=4,.start=d.start,.end=d.end});}j+=1;}
        if(!d.nonempty){if(v::at<Binding>(ref bindings,shapes::at(ref nominal,shapes::find(ref nominal,d.type)-1).binding-1).effects!=0){return .err(Problem{.code=4,.start=d.start,.end=d.end});}if(d.function!=0 || d.receiver!=0){return .err(Problem{.code=4,.start=d.start,.end=d.end});}}
        else{
            if(d.function==0 || d.receiver==0 || d.function==d.receiver || linked[d.function]!=0 || linked[d.receiver]!=0){return .err(Problem{.code=4,.start=d.start,.end=d.end});}
            let Binding b=*v::at<Binding>(ref bindings,d.function-1);let Binding receiver=*v::at<Binding>(ref bindings,d.receiver-1);
            let u32 authority=v::at<Binding>(ref bindings,shapes::at(ref nominal,shapes::find(ref nominal,d.type)-1).binding-1).effects;
            if(b.effects!=authority || binding_code(b.kind)!=9 || type_code(b.type)!=7 || b.declaration!=(d.type-98303) as usize || b.signature_count!=1 || b.start!=d.start || b.end!=d.end || binding_code(receiver.kind)!=10 || receiver.owner!=d.function || type_code(receiver.type)!=d.type || receiver.declaration!=0){return .err(Problem{.code=4,.start=d.start,.end=d.end});}
            var usize matches=0;j=0;while(j<nf){let Function f=*v::at<Function>(ref functions,j);if(f.binding==d.function){
                if(f.public || f.parameter_count!=1 || *v::at<usize>(ref edges,f.parameter_start)!=d.receiver || f.start!=d.start || f.end!=d.end){return .err(Problem{.code=4,.start=d.start,.end=d.end});}matches+=1;
            }j+=1;}if(matches!=1){return .err(Problem{.code=4,.start=d.start,.end=d.end});}
            linked[d.function]=1;linked[d.receiver]=1;
        }i+=1;
    }
    i=1;while(i<=nb){let u32 kind=binding_code(v::at<Binding>(ref bindings,i-1).kind);if((kind==9 || kind==10) && linked[i]!=1){return .err(Problem{.code=4,.start=0,.end=ns});}i+=1;}
    let Program result=Program { .nominal, .drops, .source, .payload, .functions, .bindings, .expressions, .statements, .blocks, .edges, .signatures, .module_index, .success = true };
    let Problem unsafe_proof=unsafe_check(ref result);if(unsafe_proof.code!=0){return .err(unsafe_proof);}return .ok(result);
}

// Walk owned lexical structure rather than trusting the source-side permission
// flag. Unsafe never grants effect authority or alters borrow/cleanup policy.
struct UnsafeBudget { u8[4096] calls; usize work; Problem problem; }
fn bool unsafe_fail(ref mut UnsafeBudget c,u32 code,usize start,usize end){
    if(c.problem.code==0){c.problem=Problem{.code,.start,.end};}return false;
}
fn bool unsafe_expression(ref Program p,usize id,bool allowed,usize depth,ref mut UnsafeBudget c){
    if(id==0){return true;}let e=expression_at(p,id);let u32 kind=expression_code(e.kind);
    if(depth>=64 || c.work>=1048576){return unsafe_fail(c,2,e.start,e.end);}c.work+=1;
    if(kind==28 || kind==29){if(!allowed || !e.boolean){return unsafe_fail(c,5101,e.start,e.end);}c.calls[id-1]=1;}
    if(kind==6){let b=binding_at(p,e.binding);if(e.boolean!=b.unsafe_call || b.unsafe_call && !allowed){return unsafe_fail(c,5101,e.start,e.end);}c.calls[id-1]=1;}
    if(!unsafe_expression(p,e.left,allowed,depth+1,c) || !unsafe_expression(p,e.right,allowed,depth+1,c) || !unsafe_expression(p,e.alternative,allowed,depth+1,c)){return false;}
    if(e.block!=0 && !unsafe_block(p,e.block,allowed,depth+1,c)){return false;}
    // Capture edges name bindings, all other expression edges name expressions.
    var usize i=0;while(kind!=21 && i<e.edge_count){if(!unsafe_expression(p,edge_at(p,e.edge_start+i),allowed,depth+1,c)){return false;}i+=1;}return true;
}
fn bool unsafe_statement(ref Program p,usize id,bool allowed,usize depth,ref mut UnsafeBudget c){
    if(id==0){return true;}let s=statement_at(p,id);
    if(depth>=64 || c.work>=1048576){return unsafe_fail(c,2,s.start,s.end);}c.work+=1;
    return unsafe_expression(p,s.expression,allowed,depth+1,c) && unsafe_expression(p,s.target,allowed,depth+1,c)
        && (s.body==0 || unsafe_block(p,s.body,allowed,depth+1,c))
        && unsafe_statement(p,s.initial,allowed,depth+1,c) && unsafe_statement(p,s.post,allowed,depth+1,c);
}
fn bool unsafe_block(ref Program p,usize id,bool inherited,usize depth,ref mut UnsafeBudget c){
    let b=block_at(p,id);if(depth>=64 || c.work>=1048576){return unsafe_fail(c,2,b.start,b.end);}c.work+=1;
    let bool allowed=inherited || b.unsafe_body;var usize i=0;
    while(i<b.edge_count){if(!unsafe_statement(p,edge_at(p,b.edge_start+i),allowed,depth+1,c)){return false;}i+=1;}
    return unsafe_expression(p,b.tail,allowed,depth+1,c);
}
pub fn Problem unsafe_check(ref Program p){
    var UnsafeBudget c=UnsafeBudget{.calls=[0;4096],.work=0,.problem=Problem{.code=0,.start=0,.end=0}};var usize i=1;var bool needed=false;
    while(i<=binding_count(p)){let b=binding_at(p,i);if(b.unsafe_call && binding_code(b.kind)!=1 && binding_code(b.kind)!=5 && binding_code(b.kind)!=11){discard unsafe_fail(ref mut c,4,b.start,b.end);return c.problem;}i+=1;}
    i=1;while(i<=binding_count(p)){needed=needed || binding_at(p,i).unsafe_call;i+=1;}
    i=1;while(i<=expression_count(p)){let e=expression_at(p,i);needed=needed || expression_code(e.kind)==28 || expression_code(e.kind)==29 || expression_code(e.kind)==6 && e.boolean;i+=1;}
    if(!needed){return c.problem;}
    i=0;while(i<function_count(p)){let f=function_at(p,i);if(!unsafe_block(p,f.body,binding_at(p,f.binding).unsafe_call,0,ref mut c)){return c.problem;}i+=1;}
    i=1;while(i<=expression_count(p)){let e=expression_at(p,i);if((expression_code(e.kind)==6 || (expression_code(e.kind)==28 || expression_code(e.kind)==29)) && c.calls[i-1]==0){discard unsafe_fail(ref mut c,4,e.start,e.end);return c.problem;}i+=1;}return c.problem;
}
