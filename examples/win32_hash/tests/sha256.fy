module tests::sha256;
import sha256_stream as sha;
import std::bytes as bytes;

fn void vector(usize length,ref u8[] expected) {
    var u8[129] data=[97;129];let ref u8[] input=ref data;
    var sha::State state=sha::empty();var usize offset=0;
    while(offset<length){let next=if(length-offset>7){offset+7}else{length};assert(sha::update(ref mut state,input[offset..next]));offset=next;}
    assert(sha::matches(ref state,expected));
}
@test fn void sha_padding_and_chunk_boundaries() {
    vector(0,b"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855");
    vector(55,b"9f4390f8d30c2dd92ec9f095b65e2b9ae9b0a925a5258e241c9f1e910f734318");
    vector(56,b"b35439a4ac6f0948b6d6f9e3c6af0f5f590ce20f1bde7090ef7970686ec6738a");
    vector(63,b"7d3e74a05d7db15bce4ad9ec0658ea98e3f06eeecf16b4c6fff2da457ddc2f34");
    vector(64,b"ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb");
    vector(65,b"635361c48bb9eab14198e76ea8ab7f1a41685d6ad62aa9146d301d4f17eb0ae0");
    vector(127,b"c57e9278af78fa3cab38667bef4ce29d783787a2f731d4e12200270f0c32320a");
    vector(128,b"6836cf13bac400e9105071cd6af47084dfacad4e5e302c94bfed24e013afb73e");
    vector(129,b"c12cb024a2e5551cca0e08fce8f1c5e314555cc3fef6329ee994a3db752166ae");
}
@test fn void standard_abc_and_length_overflow() {
    var sha::State state=sha::empty();assert(sha::update(ref mut state,b"a") && sha::update(ref mut state,b"bc"));
    assert(sha::matches(ref state,b"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"));
    state.length=2305843009213693951;state.used=63;assert(!sha::update(ref mut state,b"x"));
}
@test fn void reject_out_of_range_word_state() {
    var sha::State state=sha::empty();state.words[0]=4294967296;
    assert(!sha::valid(ref state));assert(!sha::update(ref mut state,b"x"));
    state=sha::empty();assert(sha::update(ref mut state,b"abc"));
    assert(sha::matches(ref state,b"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"));
}
