module tests::module_normalize;
import module_registry as registry;
import scanner as scanner;
import candidate_limits as limits;
import std::bytes as bytes;

fn void normalized(ref u8[] source, u32 code, ref u8[] expected) {
    var bytes::Buffer out = bytes::empty();
    assert(registry::normalize(source, ref mut out) == code);
    assert(bytes::equal(bytes::view(ref out), expected));
}
@test fn void canonical_paths_preserve_identifier_and_keyword_rules() {
    assert(scanner::canonical_path(b"name") && scanner::canonical_path(b"_A9::b_0::C"));
    assert(!scanner::canonical_path(b"") && !scanner::canonical_path(b"9name"));
    assert(!scanner::canonical_path(b"a:") && !scanner::canonical_path(b"a::"));
    assert(!scanner::canonical_path(b"::a") && !scanner::canonical_path(b"a:::b"));
    assert(!scanner::canonical_path(b"a b") && !scanner::canonical_path(b"a/* comment */::b"));
    assert(!scanner::canonical_path(b"fn") && !scanner::canonical_path(b"a::defer"));
    assert(!scanner::canonical_path(b"module") && !scanner::canonical_path(b"import") && !scanner::canonical_path(b"as"));
    normalized(b"name", 0, b"name");normalized(b"_A9::b_0::C", 0, b"_A9::b_0::C");
    var bytes::Buffer appended = bytes::empty();assert(bytes::append(ref mut appended,b"prefix:").is_ok);
    assert(registry::normalize(b"a::b",ref mut appended)==0 && bytes::equal(bytes::view(ref appended),b"prefix:a::b"));
}
@test fn void normalization_retains_scanner_fallback_and_failure_bytes() {
    normalized(b"module a::b;", 0, b"a::b");
    normalized(b"import a /* comment */ :: b as alias;", 0, b"a::b");
    normalized(b"a::\r\nb", 0, b"a::b");
    normalized(b"a::import::b", 0, b"a::::b");
    normalized(b"a::defer", 9, b"a::");normalized(b"fn", 9, b"");
    normalized(b"module", 9, b"");normalized(b"import", 9, b"");normalized(b"as", 9, b"");
    normalized(b"a+", 9, b"a");normalized(b"a\r", 9, b"");
    normalized(b"a/*", 9, b"");normalized(b"a\xFF", 9, b"");
    // The fast path must not admit an over-limit identifier before scanning.
    var bytes::Buffer long = bytes::empty();assert(bytes::resize_zeroed(ref mut long,limits::source_bytes()+1).is_ok);
    var usize i=0;while(i<bytes::len(ref long)){bytes::view_mut(ref mut long)[i]=97;i+=1;}
    assert(!scanner::canonical_path(bytes::view(ref long)));
    normalized(bytes::view(ref long),9,b"");
}
