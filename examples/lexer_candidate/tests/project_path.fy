module tests::project_path;
import project_path_unicode_data as unicode;
import project_path as paths;
import std::bytes as bytes;
fn bool equal(ref u8[] a,ref u8[] b){var usize work=0;return switch(paths::equal_fold(a,b,ref mut work)){case .ok(value):{value}case .err(_):{false}};}
@test
fn void matches_go_unicode_lower_and_fold_full_range_checksums(){
    var u32 rune=0;var usize lower=0;var usize fold=0;while(rune<=1114111){lower=(lower*33+(unicode::lower(rune) as usize))%16777213;fold=(fold*33+(unicode::fold(rune) as usize))%16777213;rune+=1;}assert(lower==2243456 && fold==3088718);
}
@test
fn void preserves_utf8_and_distinguishes_lowercase_from_equal_fold(){
    assert(equal(b"SRC",b"src") && equal(b"K",b"\xe2\x84\xaa") && equal(b"S",b"\xc5\xbf") && equal(b"\xce\xa3",b"\xcf\x82") && equal(b"\xf0\x90\x90\x80",b"\xf0\x90\x90\xa8") && !equal(b"i",b"\xc4\xb0"));
    var bytes::Buffer output=bytes::empty();var usize work=0;assert(paths::append_lower(ref mut output,b"AZ-\xe2\x84\xaa-\xc5\xbf-\xce\xa3-\xc4\xb0-\xc8\xba-\xf0\x90\x90\x80",ref mut work).is_ok);assert(bytes::equal(bytes::view(ref output),b"az-k-\xc5\xbf-\xcf\x83-i-\xe2\xb1\xa5-\xf0\x90\x90\xa8"));
    assert(!paths::equal_fold(b"\xff",b"x",ref mut work).is_ok && !paths::append_lower(ref mut output,b"\xc0\x80",ref mut work).is_ok);work=16777216;assert(!paths::equal_fold(b"a",b"A",ref mut work).is_ok);
}
