module tests::hir_arrays;
import tests::hir_nominal as support;
import hir_model as h;
import array_types as arrays;

@test
fn void owned_array_types_and_ordered_elements() {
    let h::Program p = support::retained(b"module main;fn i32 main(){var i64[3] a=[4000000000,2,3];a[0]=7;let b=[true;2];assert(b.len==2);return 0;}");
    var usize id = 1; var usize constructors = 0; var usize indexes = 0; var usize lengths = 0;
    while (id <= h::expression_count(ref p)) {
        let e = h::expression_at(ref p,id); let u32 kind = h::expression_code(e.kind);
        if (kind == 12) {
            constructors += 1; assert(arrays::valid(h::type_code(e.type)));
            assert(e.edge_count == if (e.boolean) { 1 } else { arrays::length(h::type_code(e.type)) });
            var usize i = 0; while (i < e.edge_count) { assert(h::edge_at(ref p,e.edge_start+i)<id); i += 1; }
        } else if (kind == 13) { indexes += 1; assert(e.left<id && e.right<id); }
        else if (kind == 14) { lengths += 1; assert(h::type_code(e.type)==5 && e.left<id); }
        id += 1;
    }
    assert(constructors==2 && indexes==1 && lengths==1);
    assert(!arrays::valid(1032) && !arrays::valid(1039) && !arrays::valid(1545));
    assert(arrays::make(1,0)==0 && arrays::make(1,65)==0 && arrays::make(256,2)==0);
    assert(arrays::count(b"A")==0 && arrays::count(b"-1")==0 && arrays::count(b"0xff")==0);
    assert(arrays::count(b"64")==64 && arrays::count(b"0x40")==64);
}
