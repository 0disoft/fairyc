module tests::checksum;
import checksum as checksum;
import std::bytes as bytes;
@test fn void single_record_checksum_formats(){
    var u8[64] expected=[0;64];
    let ref u8[] zero=b"0000000000000000000000000000000000000000000000000000000000000000";
    assert(checksum::parse(zero,ref mut expected));
    assert(checksum::parse(b"0000000000000000000000000000000000000000000000000000000000000000  파일.txt\r\n",ref mut expected));
    assert(checksum::parse(b"0000000000000000000000000000000000000000000000000000000000000000 *file.bin",ref mut expected));
    assert(checksum::parse(b"SHA256 (file.bin) = 0000000000000000000000000000000000000000000000000000000000000000",ref mut expected));
    assert(bytes::equal(ref expected,zero));
    assert(!checksum::parse(b"abc",ref mut expected));
    assert(!checksum::parse(b"0000000000000000000000000000000000000000000000000000000000000000  a\n0000000000000000000000000000000000000000000000000000000000000000  b",ref mut expected));
    assert(checksum::extension(b"C:\\file.SHA256"));
    assert(!checksum::extension(b"file.sha256.json"));
}
