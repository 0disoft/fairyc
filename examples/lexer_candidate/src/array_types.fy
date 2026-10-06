module array_types;
import syntax as syntax;
import type_state as state;

// Private canonical IDs preserve the rank-one scalar IDs. Rank two uses a
// disjoint radix-1024 outer length and one rank-one element ID. Each dimension
// is 1..64; higher ranks and nominal/view elements stay closed.
pub fn bool flat(u32 type) { return type >= 1033 && type <= 1542 && (type - 1024) % 8 >= 1 && (type - 1024) % 8 <= 6; }
pub fn bool valid(u32 type) { return flat(type) || type >= 5120 && type <= 70150 && (type - 4096) / 1024 >= 1 && (type - 4096) / 1024 <= 64 && flat(1024 + (type - 4096) % 1024); }
pub fn u32 element(u32 type) { assert(valid(type)); return if (flat(type)) { (type - 1024) % 8 } else { 1024 + (type - 4096) % 1024 }; }
pub fn usize length(u32 type) { assert(valid(type)); return if (flat(type)) { ((type - 1024) / 8) as usize } else { ((type - 4096) / 1024) as usize }; }
pub fn u32 make(u32 element, usize count) {
    if (count < 1 || count > 64) { return 0; }
    return if (element >= 1 && element <= 6) { 1024 + state::small_id(count) * 8 + element } else if (flat(element)) { 4096 + state::small_id(count) * 1024 + element - 1024 } else { 0 };
}
pub fn usize count(ref u8[] text) {
    if (text.len == 0) { return 0; }
    var usize base = 10; var usize i = 0; var usize n = 0;
    if (text.len >= 2 && text[0] == 48) { if (text[1] == 120 || text[1] == 88) { base = 16; i = 2; } else if (text[1] == 98 || text[1] == 66) { base = 2; i = 2; } }
    while (i < text.len) {
        let u8 b = text[i]; i += 1; if (b == 95) { continue; }
        if (!(b >= 48 && b <= 57) && !(b >= 65 && b <= 70) && !(b >= 97 && b <= 102)) { return 0; }
        let usize digit = if (b >= 48 && b <= 57) { (b - 48) as usize } else if (b >= 65 && b <= 70) { (b - 65 + 10) as usize } else { (b - 97 + 10) as usize };
        if (digit >= base || n > (64 - digit) / base) { return 0; }
        n = n * base + digit;
    }
    return if (n >= 1 && n <= 64) { n } else { 0 };
}
pub fn u32 resolve(ref syntax::Parsed p, usize root) {
    let node = syntax::type_at(p,root);
    if (node.kind != 8 || syntax::type_at(p,node.right).kind != 9) { return 0; }
    let child = syntax::type_at(p,node.left); var u32 element = 0;
    if (child.kind == 1) { element = state::primitive(p,node.left); }
    else if (child.kind == 8 && syntax::type_at(p,child.left).kind == 1 && syntax::type_at(p,child.right).kind == 9) { element = make(state::primitive(p,child.left),count(syntax::type_text(p,child.right))); }
    return make(element,count(syntax::type_text(p,node.right)));
}
