module resource_identity;
import syntax as syntax;
import type_state as state;
import std::bytes as bytes;

// A disjoint declaration identity range, outside scalar arrays and user enums.
// The declaration arena has at most 512 entries; spelling/field equality cannot
// turn one resource into another or into an inert struct.
pub fn bool valid(u32 type) { return type>=98304 && type<=98815; }
pub fn u32 named(ref syntax::Parsed p,ref u8[] spelling) {
    var usize i=0;while(i<syntax::declaration_count(p) && i<512){let d=syntax::declaration_at(p,i);
        if(d.kind==6 && bytes::equal(syntax::source_view(p)[d.name_start..d.name_end],spelling)){return 98304+state::small_id(i);}i+=1;
    }return 0;
}
