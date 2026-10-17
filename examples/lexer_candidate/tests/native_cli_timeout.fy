module tests::native_cli_timeout;
import native_cli as cli;
import std::bytes as bytes;

@test fn void timeout_diagnostic_retains_identity_and_canonical_seconds(){
    var bytes::Buffer out=bytes::empty();
    assert(cli::test_timeout_diagnostic(b"tests::timeout::hangs",1000,ref mut out));
    assert(bytes::equal(bytes::view(ref out),b"fairyc: build failed: execute native test tests::timeout::hangs: MSVC target executable exceeded 1s\n"));
    assert(bytes::resize_zeroed(ref mut out,0).is_ok);
    assert(cli::test_timeout_diagnostic(b"pkg::nested::slow",2000,ref mut out));
    assert(bytes::equal(bytes::view(ref out),b"fairyc: build failed: execute native test pkg::nested::slow: MSVC target executable exceeded 2s\n"));
    assert(bytes::resize_zeroed(ref mut out,0).is_ok);
    assert(cli::test_timeout_diagnostic(b"tests::upper_seconds",59000,ref mut out));
    assert(bytes::equal(bytes::view(ref out),b"fairyc: build failed: execute native test tests::upper_seconds: MSVC target executable exceeded 59s\n"));
    assert(bytes::resize_zeroed(ref mut out,0).is_ok);
    assert(cli::test_timeout_diagnostic(b"tests::default_timeout",60000,ref mut out));
    assert(bytes::equal(bytes::view(ref out),b"fairyc: build failed: execute native test tests::default_timeout: MSVC target executable exceeded 1m0s\n"));
}
