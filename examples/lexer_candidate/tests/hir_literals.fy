module tests::hir_literals;
import tests::hir_nominal as support;
import hir_model as h;
import hir_output as output;
import std::bytes as bytes;

@test
fn void cooked_static_literals_survive_parser_and_keep_distinct_types() {
    let h::Program p = support::retained(b"module main;fn i32 main(){let ref str s=\"한글\\n\\0\\u{1f600}\";let ref u8[] b=b\"\\xff\\0\\t\";let empty=\"\";assert(s.len==12 && !s.is_empty && empty.is_empty && b[0]==255);return 0;}");
    var usize i = 1; var usize count = 0;
    while (i <= h::expression_count(ref p)) {
        let e = h::expression_at(ref p,i);
        if (h::expression_code(e.kind) == 15) {
            if (count == 0) { assert(h::type_code(e.type)==9 && bytes::equal(h::literal_bytes(ref p,i),b"한글\n\0😀")); }
            else if (count == 1) { assert(h::type_code(e.type)==10 && bytes::equal(h::literal_bytes(ref p,i),b"\xff\0\t")); }
            else { assert(h::type_code(e.type)==9 && h::literal_bytes(ref p,i).len==0); }
            count += 1;
        } i += 1;
    }
    assert(count==3);
    var bytes::Buffer rendered=bytes::empty();assert(output::render(ref p,ref mut rendered)==0);
    assert(bytes::equal(bytes::view(ref rendered)[..23],b"fairyc.literal-hir/v1\nP"));
}
