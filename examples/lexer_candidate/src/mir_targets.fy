module mir_targets;
import hir_model as h;
import mir_model as m;
import mir_state as state;
import std::vec as v;

pub fn bool prepare(ref h::Program input, ref mut v::Vec<m::Target> targets, ref mut v::Vec<m::Signature> signatures, ref mut state::Context c) {
    var usize id = 1;
    while (id <= h::binding_count(input)) {
        let h::Binding b = h::binding_at(input,id); let u32 kind = h::binding_code(b.kind);
        if (kind == 1 || kind == 5 || kind==9 || kind==11) {
            if (v::len<m::Target>(targets) >= 512 || b.signature_count > 65536 - v::len<m::Signature>(signatures)) { discard state::fail(c,2,b.start,b.end); return false; }
            let usize start = v::len<m::Signature>(signatures); var usize j = 0;
            while (j < b.signature_count) { if (!v::push<m::Signature>(signatures,m::Signature { .type = h::type_code(h::signature_at(input,b.signature_start + j)) }).is_ok) { discard state::fail(c,3,b.start,b.end); return false; } j += 1; }
            if (!v::push<m::Target>(targets,m::Target { .foreign=kind==11,.link_start=b.link_start,.link_end=b.link_end, .unsafe_call=b.unsafe_call, .effects = b.effects, .binding = id, .module_index = b.module_index, .declaration = b.declaration, .result = h::type_code(b.type), .signature_start = start, .signature_count = b.signature_count, .start = b.start, .end = b.end, .external = kind == 5 || kind==11 }).is_ok) { discard state::fail(c,3,b.start,b.end); return false; }
            c.targets[id] = v::len<m::Target>(targets);
        }
        id += 1;
    }
    return true;
}
