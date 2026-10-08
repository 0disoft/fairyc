module literal_types;
import syntax as syntax;
import std::bytes as bytes;
import utf8 as utf8;

// Private tags 9/10 are immutable literal-capable descriptors. Tag 11
// resolves mutable byte-view annotations, but is never a literal value.
pub fn bool valid(u32 type) { return type == 9 || type == 10 || type == 12; }
pub fn u32 resolve(ref syntax::Parsed p, usize root) {
    let n = syntax::type_at(p,root);
    if (n.kind != 2 && n.kind != 3) { return 0; }
    let child = syntax::type_at(p,n.left);
    if (n.kind == 2 && child.kind == 1 && bytes::equal(syntax::type_text(p,n.left),b"str")) { return 9; }
    if (n.kind == 2 && child.kind == 1 && bytes::equal(syntax::type_text(p,n.left),b"CStr")) { return 12; }
    if (child.kind == 4 && syntax::type_at(p,child.left).kind == 1 && bytes::equal(syntax::type_text(p,child.left),b"u8")) { return if (n.kind == 3) { 11 } else { 10 }; }
    return 0;
}
pub fn bool text(ref u8[] cooked) {
    var usize i = 0; while (i < cooked.len) { let usize n = utf8::width(cooked,i); if (n == 0) { return false; } i += n; } return true;
}
fn u8 nibble(u8 b) { return if (b >= 48 && b <= 57) { b - 48 } else if (b >= 97 && b <= 102) { b - 97 + 10 } else { 255 }; }
// Scanner token cooking uses lowercase hex. Decode that transport exactly once
// when the HIR takes ownership; semantic literal storage contains actual octets.
pub fn u32 decode(ref mut bytes::Buffer payload, ref u8[] encoded) {
    if (encoded.len % 2 != 0) { return 4; }
    if (bytes::len(payload) > 86016 || encoded.len / 2 > 86016 - bytes::len(payload)) { return 2; }
    var usize i = 0; while (i < encoded.len) {
        let u8 hi = nibble(encoded[i]); let u8 lo = nibble(encoded[i+1]); if (hi > 15 || lo > 15) { return 4; }
        if (!bytes::append_byte(payload,hi * 16 + lo).is_ok) { return 3; } i += 2;
    } return 0;
}
// Hex keeps binary payloads, tabs/newlines and embedded NUL out of TSV records.
pub fn bool hex(ref mut bytes::Buffer out, ref u8[] cooked) {
    if (bytes::len(out) > 2097152 || cooked.len > (2097152 - bytes::len(out)) / 2) { return false; }
    let ref u8[] digits = b"0123456789abcdef";
    var usize i = 0; while (i < cooked.len) { let u8 b = cooked[i]; if (!bytes::append_byte(out,digits[(b / 16) as usize]).is_ok || !bytes::append_byte(out,digits[(b % 16) as usize]).is_ok) { return false; } i += 1; } return true;
}

// C literals retain exactly one trailing NUL and no earlier NUL.
pub fn bool cstring(ref u8[] data){if(data.len==0 || data[data.len-1]!=0){return false;}var usize i=0;while(i+1<data.len){if(data[i]==0){return false;}i+=1;}return true;}
