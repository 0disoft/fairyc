module nominal_identity;
import syntax as syntax;
import resource_identity as resources;
import type_state as state;
import std::bytes as bytes;

// Declaration identity is shared by ordinary structs and tagged payload leaves.
pub fn bool valid(u32 type) { return type >= 256 && type <= 767; }
pub fn u32 named(ref syntax::Parsed p, ref u8[] spelling) {
    var usize i = 0;
    while (i < syntax::declaration_count(p)) {
        let d = syntax::declaration_at(p,i);
        if (d.kind == 4 && bytes::equal(syntax::source_view(p)[d.name_start..d.name_end],spelling)) {
            return 256 + state::small_id(i);
        }
        i += 1;
    }
    return resources::named(p,spelling);
}
