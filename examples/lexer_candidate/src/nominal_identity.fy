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

// A program-owned path binds one exact declaration in the joined arena.
// Numeric layout tags remain local to this complete source-set owner.
pub fn u32 bound(ref syntax::Parsed p,usize root){if(root==0){return 0;}let n=syntax::type_at(p,root);if(n.kind!=1 || n.left==0 || n.left>syntax::declaration_count(p) || n.left>512){return 0;}let d=syntax::declaration_at(p,n.left-1);return if(d.kind==4){255+state::small_id(n.left)}else if(d.kind==6){98303+state::small_id(n.left)}else if(d.kind==11){131071+state::small_id(n.left)}else{0};}
