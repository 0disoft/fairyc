module hir_state;
import candidate_limits as limits;
import scalar_views as views;
import c_abi_types as c_types;
import pointer_types as pointers;
import reference_types as refs;
import array_types as arrays;
import resource_identity as resources;
import enum_types as enums;
import tagged_types as tagged;
import hir_model as h;
import std::bytes as bytes;
import std::vec as v;

// Temporary input-to-semantic links, discarded before publication.
pub struct Link { usize use; usize entry; usize function; usize iteration_view; usize iteration_index; }
pub struct Context { h::Problem problem; usize owner; usize next_scope; usize loop_target; usize scope_count; usize[128] scopes; }
pub fn Context empty() { return Context { .problem = h::Problem { .code = 0, .start = 0, .end = 0 }, .owner = 0, .next_scope = 1, .loop_target = 0, .scope_count = 0, .scopes = [0;128] }; }
pub fn usize fail(ref mut Context c, u32 code, usize start, usize end) { if (c.problem.code == 0) { c.problem = h::Problem { .code, .start, .end }; } return 0; }
pub fn usize scope(ref Context c) { return if (c.scope_count == 0) { 0 } else { c.scopes[c.scope_count - 1] }; }
pub fn bool enter(ref mut Context c) { if (c.scope_count >= 128) { discard fail(c,2,0,0); return false; } let usize slot = c.scope_count; let usize next = c.next_scope; c.scopes[slot] = next; c.scope_count += 1; c.next_scope += 1; return true; }
pub fn bool visible(ref Context c, h::Binding b) { if (h::binding_code(b.kind) == h::binding_code(.Function) || h::binding_code(b.kind) == h::binding_code(.External) || h::binding_code(b.kind)==11) { return true; } if (b.owner != c.owner) { return false; } var usize i = 0; while (i < c.scope_count) { if (c.scopes[i] == b.scope) { return true; } i += 1; } return false; }
pub fn h::Type type(u32 code) { return if (code == 1) { .I32 } else if (code == 2) { .I64 } else if (code == 3) { .U8 } else if (code == 4) { .U32 } else if (code == 5) { .Usize } else if (code == 6) { .Bool } else if (code == 7) { .Void } else if (code == 8) { .Callable } else if (code == 9) { .Text } else if (code == 10) { .Bytes } else if (code == 11) { .MutableBytes } else if(code==12){.CStr} else if(views::extended(code)){.ScalarView{.id=code}}else if(c_types::valid(code)){.CScalar{.id=code}}else if(pointers::valid(code)){.Pointer{.id=code}}else if(refs::valid(code)){.Borrowed{.id=code}}else if(resources::valid(code)){.Resource{.id=code}} else if (code >= 256 && code <= 767) { .Nominal { .id = code } } else if(tagged::option(code)){ .Option { .id = code } } else if(tagged::result(code)){ .Result { .id = code } } else if(enums::valid(code)){ .Enumeration { .id = code } } else if (arrays::valid(code)) { .Array { .id = code } } else { .Invalid }; }
pub fn h::BindingKind binding_kind(u32 code) { return if (code == 1) { .Function } else if (code == 2) { .Parameter } else if (code == 3 || code==12) { .Let } else if (code == 4) { .Var } else if (code == 6) { .Struct } else if (code == 7) { .Enum } else if(code==9){.Resource}else if(code==10){.DropReceiver} else if(code==11){.Foreign}else { .External }; }
pub fn h::Operator operator(ref u8[] raw) {
    if (bytes::equal(raw,b"+" ) || bytes::equal(raw,b"+=")) { return .Plus; }
    if (bytes::equal(raw,b"-" ) || bytes::equal(raw,b"-=")) { return .Minus; }
    if (bytes::equal(raw,b"!")) { return .Not; }
    if (bytes::equal(raw,b"~")) { return .Complement; }
    if (bytes::equal(raw,b"*") || bytes::equal(raw,b"*=")) { return .Multiply; }
    if (bytes::equal(raw,b"/") || bytes::equal(raw,b"/=")) { return .Divide; }
    if (bytes::equal(raw,b"%") || bytes::equal(raw,b"%=")) { return .Remainder; }
    if (bytes::equal(raw,b"&") || bytes::equal(raw,b"&=")) { return .BitAnd; }
    if (bytes::equal(raw,b"|") || bytes::equal(raw,b"|=")) { return .BitOr; }
    if (bytes::equal(raw,b"^") || bytes::equal(raw,b"^=")) { return .BitXor; }
    if (bytes::equal(raw,b"<<") || bytes::equal(raw,b"<<=")) { return .ShiftLeft; }
    if (bytes::equal(raw,b">>") || bytes::equal(raw,b">>=")) { return .ShiftRight; }
    if (bytes::equal(raw,b"==")) { return .Equal; }
    if (bytes::equal(raw,b"!=")) { return .NotEqual; }
    if (bytes::equal(raw,b"<")) { return .Less; }
    if (bytes::equal(raw,b"<=")) { return .LessEqual; }
    if (bytes::equal(raw,b">")) { return .Greater; }
    if (bytes::equal(raw,b">=")) { return .GreaterEqual; }
    if (bytes::equal(raw,b"&&")) { return .And; }
    if (bytes::equal(raw,b"||")) { return .Or; }
    if (bytes::equal(raw,b"=")) { return .Assign; }
    return .None;
}
pub fn h::Expression typed_expression(h::ExpressionKind kind,h::Type type,h::Operator operator,usize start,usize end) { return h::Expression { .kind, .type, .operator, .start, .end, .left = 0, .right = 0, .alternative = 0, .block = 0, .binding = 0, .edge_start = 0, .edge_count = 0, .payload_start = 0, .payload_end = 0, .boolean = false }; }
pub fn h::Expression expression(usize start,usize end){return typed_expression(.Integer,.Invalid,.None,start,end);}
pub fn h::Statement statement(usize start, usize end, usize scope) { return h::Statement { .kind = .Expression, .start, .end, .scope, .expression = 0, .target = 0, .binding = 0, .operator = .None, .body = 0, .initial = 0, .post = 0, .loop_target = 0 }; }
pub fn usize push_expression(ref mut v::Vec<h::Expression> arena, h::Expression e, ref mut Context c) { if (c.problem.code != 0) { return 0; } if (v::len<h::Expression>(arena) >= limits::program_expressions()) { return fail(c,2,e.start,e.end); } if (!v::push<h::Expression>(arena,e).is_ok) { return fail(c,3,e.start,e.end); } return v::len<h::Expression>(arena); }
pub fn usize push_statement(ref mut v::Vec<h::Statement> arena, h::Statement s, ref mut Context c) { if (c.problem.code != 0) { return 0; } if (v::len<h::Statement>(arena) >= 8192) { return fail(c,2,s.start,s.end); } if (!v::push<h::Statement>(arena,s).is_ok) { return fail(c,3,s.start,s.end); } return v::len<h::Statement>(arena); }
pub fn bool edge(ref mut v::Vec<usize> arena, usize id, ref mut Context c) { if (c.problem.code != 0) { return false; } if (id == 0) { discard fail(c,4,0,0); return false; } if (v::len<usize>(arena) >= 32768) { discard fail(c,2,0,0); return false; } if (!v::push<usize>(arena,id).is_ok) { discard fail(c,3,0,0); return false; } return true; }
pub fn usize reserve_edges(ref mut v::Vec<usize> arena, usize count, ref mut Context c) { let usize start = v::len<usize>(arena); var usize i = 0; while (i < count) { if (!edge(arena,1,c)) { return 0; } i += 1; } return start; }
pub fn void set_edge(ref mut v::Vec<usize> arena, usize index, usize id) { let ref mut usize entry = v::at_mut<usize>(arena,index); *entry = id; }
pub fn usize use(ref v::Vec<Link> links, usize id) { return if (id == 0 || id > v::len<Link>(links)) { 0 } else { v::at<Link>(links,id - 1).use }; }
pub fn usize entry(ref v::Vec<Link> links, usize id) { return if (id == 0 || id > v::len<Link>(links)) { 0 } else { v::at<Link>(links,id - 1).entry }; }
pub fn usize function(ref v::Vec<Link> links, usize id) { return if (id == 0 || id > v::len<Link>(links)) { 0 } else { v::at<Link>(links,id - 1).function }; }
// Literal magnitudes never pass through a signed machine value. Exact canonical
// decimal bytes therefore cover both i64 MIN and unsigned usize MAX on AMD64.
pub fn bool integer(ref mut bytes::Buffer payload, ref u8[] raw, bool negative, ref mut Context c) {
    if (bytes::len(payload) > limits::program_expressions()*21 - 21) { discard fail(c,2,0,0); return false; }
    var usize base = 10; var usize i = 0;
    if (raw.len >= 2 && raw[0] == 48) { if (raw[1] == 120 || raw[1] == 88) { base = 16; i = 2; } else if (raw[1] == 98 || raw[1] == 66) { base = 2; i = 2; } }
    var usize magnitude = 0; var usize digits = 0;
    while (i < raw.len) {
        let u8 byte = raw[i]; i += 1; if (byte == 95) { continue; }
        var usize digit = 16;
        if (byte >= 48 && byte <= 57) { digit = (byte - 48) as usize; }
        else if (byte >= 65 && byte <= 70) { digit = (byte - 65 + 10) as usize; }
        else if (byte >= 97 && byte <= 102) { digit = (byte - 97 + 10) as usize; }
        if (digit >= base || magnitude > (18446744073709551615 - digit) / base) { discard fail(c,4,0,0); return false; }
        magnitude = magnitude * base + digit; digits += 1;
    }
    if (digits == 0) { discard fail(c,4,0,0); return false; }
    if (negative && !bytes::append_byte(payload,45).is_ok) { discard fail(c,3,0,0); return false; }
    if (!bytes::append_usize(payload,magnitude).is_ok) { discard fail(c,3,0,0); return false; }
    return true;
}
