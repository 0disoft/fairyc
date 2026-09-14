module cleanup_syntax;
import scanner as scanner;
import std::vec as v;
import bodies as bodies;
import body_state as state;
import types as types;
import attributes as attributes;

pub struct Attempt { bool ok; usize body; usize start; usize end; usize attribute_start; usize attribute_count; }
pub fn Attempt parse(ref scanner::Stream stream, ref mut state::Cursor cursor, ref mut v::Vec<state::Node> nodes, ref mut v::Vec<bodies::Entry> entries, ref mut v::Vec<state::Statement> statements, ref mut v::Vec<types::Node> type_nodes, ref mut v::Vec<attributes::Attribute> attribute_nodes, ref mut v::Vec<attributes::Argument> attribute_arguments, usize attribute_base, usize argument_base) {
    var Attempt out = Attempt { .ok = false, .body = 0, .start = 0, .end = 0, .attribute_start = 0, .attribute_count = 0 };
    if (state::word(stream,cursor.index,b"@")) {
        out.attribute_start = attribute_base;
        let attributes::Attempt a = attributes::parse(stream,cursor.index,out.attribute_start,argument_base,attribute_nodes,attribute_arguments);
        cursor.index = a.next; cursor.previous_end = a.end; cursor.status = a.status; cursor.expected = a.expected;
        if (!a.ok) { return out; }
        out.attribute_count = a.count - out.attribute_start;
    }
    out.start = state::current(stream,cursor).start;
    if (!state::expect(stream,cursor,b"cleanup",24)) { return out; }
    out.body = bodies::block(stream,cursor,nodes,entries,statements,type_nodes,0,false);
    out.end = cursor.previous_end; out.ok = out.body != 0;
    return out;
}
