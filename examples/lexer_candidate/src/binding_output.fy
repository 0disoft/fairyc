module binding_output;
import name_resolution as names;
import std::bytes as bytes;
import std::text as text;
import std::io as io;

fn bool number(ref mut bytes::Buffer buffer, usize value) {
    return bytes::append(buffer,b"\t").is_ok && bytes::append_usize(buffer,value).is_ok;
}
@effects(console)
pub fn i32 run(ref names::Resolved resolved) {
    var bytes::Buffer buffer = bytes::empty();
    if (!bytes::append(ref mut buffer,b"record\tid\tkind\tstart\tend\tscope\towner\ttarget\n").is_ok) {
        discard io::write_stderr("error: buffer allocation failed\n"); return 2;
    }
    var usize id = 1;
    while (id <= names::symbol_count(resolved)) {
        let names::Symbol symbol = names::symbol_at(resolved,id);
        let ref str label = if (symbol.kind == 1) { "function" } else if (symbol.kind == 2) { "parameter" }
            else if (symbol.kind == 3) { "local-let" } else { "local-var" };
        if (!bytes::append(ref mut buffer,b"symbol").is_ok || !number(ref mut buffer,id) ||
            !bytes::append(ref mut buffer,b"\t").is_ok || !bytes::append(ref mut buffer,text::as_bytes(label)).is_ok ||
            !number(ref mut buffer,symbol.start) || !number(ref mut buffer,symbol.end) ||
            !number(ref mut buffer,symbol.scope) || !number(ref mut buffer,symbol.owner) ||
            !number(ref mut buffer,0) || !bytes::append(ref mut buffer,b"\n").is_ok) {
            discard io::write_stderr("error: buffer allocation failed\n"); return 2;
        }
        id += 1;
    }
    var usize index = 0;
    while (index < names::reference_count(resolved)) {
        let names::Reference use = names::reference_at(resolved,index);
        if (!bytes::append(ref mut buffer,b"reference").is_ok || !number(ref mut buffer,use.expression) ||
            !bytes::append(ref mut buffer,b"\tname").is_ok || !number(ref mut buffer,use.start) ||
            !number(ref mut buffer,use.end) || !number(ref mut buffer,0) || !number(ref mut buffer,0) ||
            !number(ref mut buffer,use.symbol) || !bytes::append(ref mut buffer,b"\n").is_ok) {
            discard io::write_stderr("error: buffer allocation failed\n"); return 2;
        }
        index += 1;
    }
    if (bytes::len(ref buffer) > 1048576) {
        discard io::write_stderr("error: output limit (1048576 bytes)\n"); return 2;
    }
    let ref u8[] data = bytes::view(ref buffer);
    return if (switch(io::write_bytes(data)) { case .ok(count): { count == data.len } case .err(_): { false } }) { 0 }
        else { discard io::write_stderr("error: stdout write failed\n"); 2 };
}
