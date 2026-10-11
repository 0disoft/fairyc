module tests::sha256_stream;
import sha256_stream as stream;
import sha256 as single;
import std::bytes as bytes;

fn void check(usize size,ref u8[] expected){var u8[129] data=[97;129];let ref u8[] input=ref data;var stream::State state=stream::empty();var usize at=0;
    while(at<size){let usize next=if(size-at>7){at+7}else{size};assert(stream::update(ref mut state,input[at..next]));at=next;}assert(stream::valid(ref state) && state.length==size && state.used==size%64 && stream::matches(ref state,expected) && single::matches(input[..size],expected));
    var bytes::Buffer hex=bytes::empty();assert(stream::append_hex(ref state,ref mut hex) && bytes::equal(bytes::view(ref hex),expected));
}
@test
fn void matches_reference_vectors_across_padding_and_chunk_boundaries(){
    check(0,b"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855");check(1,b"ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb");check(55,b"9f4390f8d30c2dd92ec9f095b65e2b9ae9b0a925a5258e241c9f1e910f734318");check(56,b"b35439a4ac6f0948b6d6f9e3c6af0f5f590ce20f1bde7090ef7970686ec6738a");check(63,b"7d3e74a05d7db15bce4ad9ec0658ea98e3f06eeecf16b4c6fff2da457ddc2f34");check(64,b"ffe054fe7ae0cb6dc65c3af9b61d5209f439851db43d0ba5997337df154668eb");check(65,b"635361c48bb9eab14198e76ea8ab7f1a41685d6ad62aa9146d301d4f17eb0ae0");check(127,b"c57e9278af78fa3cab38667bef4ce29d783787a2f731d4e12200270f0c32320a");check(128,b"6836cf13bac400e9105071cd6af47084dfacad4e5e302c94bfed24e013afb73e");check(129,b"c12cb024a2e5551cca0e08fce8f1c5e314555cc3fef6329ee994a3db752166ae");
    var stream::State state=stream::empty();assert(stream::update(ref mut state,b"a"));assert(stream::matches(ref state,b"ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"));assert(stream::update(ref mut state,b"bc"));assert(stream::matches(ref state,b"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad") && state.length==3 && state.used==3);
}
@test
fn void streams_beyond_old_input_cap_and_rejects_invalid_state_without_update(){
    var u8[4096] block=[97;4096];let ref u8[] data=ref block;var stream::State state=stream::empty();var usize count=0;while(count<257){assert(stream::update(ref mut state,data));count+=1;}assert(state.length==1052672 && stream::matches(ref state,b"5c8287eaf42d038c965c0bc4face4222cbaf8b6cf50362be965090f9ce71a9ff"));
    var stream::State invalid=stream::empty();invalid.length=100663296;assert(!stream::update(ref mut invalid,b"a") && invalid.length==100663296 && invalid.used==0);invalid.used=64;assert(!stream::valid(ref invalid) && !stream::update(ref mut invalid,b"") && invalid.used==64);invalid=stream::empty();invalid.length=1;assert(!stream::valid(ref invalid) && !stream::update(ref mut invalid,b"a") && invalid.length==1 && invalid.used==0);
}
