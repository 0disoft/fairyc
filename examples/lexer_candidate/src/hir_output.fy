module hir_output;
import hir_model as h;
import hir_shapes as shapes;
import array_types as arrays;
import literal_types as literals;
import tagged_types as tagged;
import std::bytes as bytes;

fn bool put(ref mut bytes::Buffer out, ref u8[] text) { return bytes::append(out,text).is_ok; }
fn bool tab(ref mut bytes::Buffer out) { return put(out,b"\t"); }
fn bool number(ref mut bytes::Buffer out, usize value) { return tab(out) && bytes::append_usize(out,value).is_ok; }
fn bool type_name(ref mut bytes::Buffer out, h::Type type) {
    return tab(out) && switch(type) {
        case .Invalid: { put(out,b"invalid") } case .I32: { put(out,b"i32") }
        case .I64: { put(out,b"i64") } case .U8: { put(out,b"u8") } case .U32: { put(out,b"u32") }
        case .Usize: { put(out,b"usize") } case .Bool: { put(out,b"bool") }
        case .Void: { put(out,b"void") } case .Callable: { put(out,b"callable") }
        case .Text: { put(out,b"ref-str") } case .Bytes: { put(out,b"ref-u8-slice") } case .MutableBytes: { put(out,b"ref-mut-u8-slice") }
        case .Array { .id = id }: { put(out,b"array:") && bytes::append_u32(out,id).is_ok }
        case .Option { .id = id }: { put(out,b"option:") && bytes::append_u32(out,id-196608).is_ok }
        case .Result { .id = id }: { put(out,b"result:") && bytes::append_u32(out,tagged::success(id)).is_ok && put(out,b":") && bytes::append_u32(out,tagged::failure(id)).is_ok }
        case .Enumeration { .id = id }: { put(out,b"enum:") && bytes::append_u32(out,id).is_ok }
        case .Nominal { .id = id }: { put(out,b"nominal:") && bytes::append_u32(out,id).is_ok }
    };
}
fn bool binding_kind(ref mut bytes::Buffer out, h::BindingKind kind) {
    return tab(out) && switch(kind) {
        case .Function: { put(out,b"function") } case .Parameter: { put(out,b"parameter") }
        case .Let: { put(out,b"let") } case .Var: { put(out,b"var") } case .External: { put(out,b"external") }
        case .Struct: { put(out,b"struct") } case .Enum: { put(out,b"enum") }
    };
}
fn bool expression_kind(ref mut bytes::Buffer out, h::ExpressionKind kind) {
    return tab(out) && switch(kind) {
        case .Integer: { put(out,b"integer") } case .Boolean: { put(out,b"boolean") }
        case .Reference: { put(out,b"reference") } case .Unary: { put(out,b"unary") }
        case .Binary: { put(out,b"binary") } case .Call: { put(out,b"call") }
        case .Block: { put(out,b"block") } case .If: { put(out,b"if") }
        case .Construct: { put(out,b"construct") } case .Field: { put(out,b"field") } case .Initializer: { put(out,b"initializer") } case .Array: { put(out,b"array") } case .Index: { put(out,b"index") } case .Length: { put(out,b"length") } case .Borrow: { put(out,b"borrow-array") } case .Slice: { put(out,b"slice") }
        case .Variant: { put(out,b"variant") } case .Switch: { put(out,b"switch") } case .Arm: { put(out,b"arm") } case .Capture: { put(out,b"capture") }
        case .Propagate: { put(out,b"propagate") }
        case .Literal: { put(out,b"literal") }
    };
}
fn bool statement_kind(ref mut bytes::Buffer out, h::StatementKind kind) {
    return tab(out) && switch(kind) {
        case .Binding: { put(out,b"binding") } case .Return: { put(out,b"return") }
        case .Assignment: { put(out,b"assignment") } case .Expression: { put(out,b"expression") }
        case .While: { put(out,b"while") } case .For: { put(out,b"for") }
        case .Break: { put(out,b"break") } case .Continue: { put(out,b"continue") }
        case .Assert: { put(out,b"assert") } case .Discard: { put(out,b"discard") } case .Defer: { put(out,b"defer") }
    };
}
fn bool operator(ref mut bytes::Buffer out, h::Operator op) {
    return tab(out) && switch(op) {
        case .None: { put(out,b"-") } case .Plus: { put(out,b"+") } case .Minus: { put(out,b"-") }
        case .Not: { put(out,b"!") } case .Complement: { put(out,b"~") } case .Multiply: { put(out,b"*") }
        case .Divide: { put(out,b"/") } case .Remainder: { put(out,b"%") } case .BitAnd: { put(out,b"&") }
        case .BitOr: { put(out,b"|") } case .BitXor: { put(out,b"^") } case .ShiftLeft: { put(out,b"<<") }
        case .ShiftRight: { put(out,b">>") } case .Equal: { put(out,b"==") } case .NotEqual: { put(out,b"!=") }
        case .Less: { put(out,b"<") } case .LessEqual: { put(out,b"<=") } case .Greater: { put(out,b">") }
        case .GreaterEqual: { put(out,b">=") } case .And: { put(out,b"&&") } case .Or: { put(out,b"||") }
        case .Assign: { put(out,b"=") }
    };
}

// Render an already complete HIR into a caller-owned Buffer. No console effects
// and no output publication on failure. All records are under 1024 bytes; the
// guard reserves a whole record below the 2 MiB CLI transport ceiling.
// 3 allocation failure, 4 output limit. Enum names and IDs are private v1 data.
pub fn u32 render(ref h::Program p, ref mut bytes::Buffer out) {
    let ref shapes::Table nominal = h::nominal(p);
    var bool array = false; var usize id = 1; while (id <= h::binding_count(p)) { array = array || arrays::valid(h::type_code(h::binding_at(p,id).type)); id += 1; }
    id = 1; while (id <= h::expression_count(p)) { array = array || arrays::valid(h::type_code(h::expression_at(p,id).type)); id += 1; }
    if (!put(out,if (shapes::has_payloads(nominal)) { b"fairyc.enum-payload-hir/v1\nP" } else if (shapes::has_enums(nominal)) { b"fairyc.enum-hir/v1\nP" } else if (h::has_views(p)) { b"fairyc.view-hir/v1\nP" } else if (h::has_literals(p)) { b"fairyc.literal-hir/v1\nP" } else if (array) { b"fairyc.array-hir/v1\nP" } else if (shapes::count(nominal) == 0) { b"fairyc.scalar-hir/v1\nP" } else { b"fairyc.nominal-hir/v1\nP" }) || !number(out,h::module_index(p)) || !put(out,b"\n")) { return 3; }
    var usize index = 0;
    while (index < shapes::count(nominal)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let shapes::Shape shape = shapes::at(nominal,index);
        if (!put(out,b"N") || !number(out,shape.type as usize) || !number(out,shape.binding) || !number(out,shape.field_start) || !number(out,shape.field_count) || !number(out,if (shape.public) { 1 } else { 0 }) || !put(out,b"\n")) { return 3; } index += 1;
    }
    index = 1;
    while (index <= shapes::field_count(nominal)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let shapes::Field field = shapes::field_at(nominal,index);
        if (!put(out,b"D") || !number(out,index) || !number(out,field.owner as usize) || !number(out,field.type as usize) || !number(out,field.start) || !number(out,field.end) || (shapes::has_payloads(nominal) && (!number(out,field.payload_start) || !number(out,field.payload_count))) || !put(out,b"\n")) { return 3; } index += 1;
    }
    index = 0;
    while (index < h::function_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let h::Function f = h::function_at(p,index);
        if (!put(out,b"F") || !number(out,index + 1) || !number(out,f.binding) || !number(out,f.start) || !number(out,f.end) || !number(out,f.scope) || !number(out,f.body) || !number(out,f.parameter_start) || !number(out,f.parameter_count) || !number(out,if (f.public) { 1 } else { 0 }) || !put(out,b"\n")) { return 3; }
        index += 1;
    }
    index = 1;
    while (index <= h::binding_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let h::Binding b = h::binding_at(p,index);
        if (!put(out,b"B") || !number(out,index) || !binding_kind(out,b.kind) || !type_name(out,b.type) || !number(out,b.start) || !number(out,b.end) || !number(out,b.owner) || !number(out,b.scope) || !number(out,b.module_index) || !number(out,b.declaration) || !number(out,b.signature_start) || !number(out,b.signature_count) || !put(out,b"\n")) { return 3; }
        index += 1;
    }
    index = 1;
    while (index <= h::expression_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let h::Expression e = h::expression_at(p,index);
        if (h::expression_code(e.kind) == 15 && h::literal_bytes(p,index).len > (2097152 - bytes::len(out) - 1024) / 2) { return 4; }
        if (!put(out,b"E") || !number(out,index) || !expression_kind(out,e.kind) || !type_name(out,e.type) || !operator(out,e.operator) || !number(out,e.start) || !number(out,e.end) || !number(out,e.left) || !number(out,e.right) || !number(out,e.alternative) || !number(out,e.block) || !number(out,e.binding) || !number(out,e.edge_start) || !number(out,e.edge_count) || !number(out,if (e.boolean) { 1 } else { 0 }) || !tab(out)) { return 3; }
        if (h::expression_code(e.kind) == h::expression_code(.Integer)) { if (!put(out,h::integer_text(p,index))) { return 3; } }
        else if (h::expression_code(e.kind) == 15) { if (!literals::hex(out,h::literal_bytes(p,index))) { return 3; } }
        else if (!put(out,b"-")) { return 3; }
        if (!put(out,b"\n")) { return 3; } index += 1;
    }
    index = 1;
    while (index <= h::statement_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let h::Statement s = h::statement_at(p,index);
        if (!put(out,b"S") || !number(out,index) || !statement_kind(out,s.kind) || !number(out,s.start) || !number(out,s.end) || !number(out,s.scope) || !number(out,s.expression) || !number(out,s.target) || !number(out,s.binding) || !operator(out,s.operator) || !number(out,s.body) || !number(out,s.initial) || !number(out,s.post) || !number(out,s.loop_target) || !put(out,b"\n")) { return 3; }
        index += 1;
    }
    index = 1;
    while (index <= h::block_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        let h::Block b = h::block_at(p,index);
        if (!put(out,b"K") || !number(out,index) || !number(out,b.start) || !number(out,b.end) || !number(out,b.owner) || !number(out,b.scope) || !number(out,b.parent_scope) || !number(out,b.edge_start) || !number(out,b.edge_count) || !number(out,b.tail) || !put(out,b"\n")) { return 3; } index += 1;
    }
    index = 0;
    while (index < h::edge_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        if (!put(out,b"A") || !number(out,index) || !number(out,h::edge_at(p,index)) || !put(out,b"\n")) { return 3; } index += 1;
    }
    index = 0;
    while (index < h::signature_count(p)) {
        if (bytes::len(out) > 2097152 - 1024) { return 4; }
        if (!put(out,b"T") || !number(out,index) || !type_name(out,h::signature_at(p,index)) || !put(out,b"\n")) { return 3; } index += 1;
    }
    return 0;
}
