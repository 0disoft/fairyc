package foreignstructprobe

const arrayFairySource = `
@repr(C) struct fairyc_fixture_array_packet_v0 {
    u8 lead;
    u32[3] words;
    i64[2] wide;
    i32[2] codes;
    usize[2] sizes;
    u8[4] bytes;
    u8 tail;
}
@repr(C) struct fairyc_fixture_array_small_v0 { u8 lead; u8[3] bytes; u32 tail; }
@link_name("fairyc_fixture_array_change_v0")
unsafe extern C fn fairyc_fixture_array_packet_v0 array_change(i32 before, fairyc_fixture_array_packet_v0 value, i32 after);
@link_name("fairyc_fixture_array_small_change_v0")
unsafe extern C fn fairyc_fixture_array_small_v0 small_change(fairyc_fixture_array_small_v0 value);
fn void replace_array_words(ref mut fairyc_fixture_array_packet_v0 value) { value.words = [31,32,33]; }
fn bool check_arrays() {
    var fairyc_fixture_array_packet_v0 value = fairyc_fixture_array_packet_v0 {
        .lead=11, .words=[1,2,3], .wide=[-9,10], .codes=[-3,4], .sizes=[5,6], .bytes=[7,8,9,10], .tail=12
    };
    let fairyc_fixture_array_packet_v0 changed = unsafe { array_change(17, value, 19) };
    let u32[3] copied = value.words;
    replace_array_words(ref mut value);
    let fairyc_fixture_array_small_v0 small = fairyc_fixture_array_small_v0 { .lead=1, .bytes=[2,3,4], .tail=5 };
    let fairyc_fixture_array_small_v0 changed_small = unsafe { small_change(small) };
    return copied[0]==1 && copied[1]==2 && copied[2]==3 && value.words[0]==31 && value.words[2]==33 &&
        value.lead==11 && value.tail==12 && value.wide[0]==-9 && value.wide[1]==10 &&
        value.codes[0]==-3 && value.codes[1]==4 && value.sizes[0]==5 && value.sizes[1]==6 &&
        value.bytes[0]==7 && value.bytes[1]==8 && value.bytes[2]==9 && value.bytes[3]==10 &&
        changed.lead==11 && changed.tail==66 && changed.words[0]==101 && changed.words[1]==2 && changed.words[2]==3 &&
        changed.wide[0]==-9 && changed.wide[1]==200 && changed.codes[0]==-3 && changed.codes[1]==-300 &&
        changed.sizes[0]==5 && changed.sizes[1]==400 && changed.bytes[0]==7 && changed.bytes[1]==8 && changed.bytes[2]==9 && changed.bytes[3]==55 &&
        small.lead==1 && small.bytes[0]==2 && small.bytes[1]==3 && small.bytes[2]==4 && small.tail==5 &&
        changed_small.lead==1 && changed_small.bytes[0]==2 && changed_small.bytes[1]==13 && changed_small.bytes[2]==4 && changed_small.tail==15;
}
`

const arrayCompanionC = `
#include <stddef.h>
struct fairyc_fixture_array_packet_v0 {
    uint8_t lead;
    uint32_t words[3];
    int64_t wide[2];
    int32_t codes[2];
    uintptr_t sizes[2];
    uint8_t bytes[4];
    uint8_t tail;
};
struct fairyc_fixture_array_small_v0 { uint8_t lead; uint8_t bytes[3]; uint32_t tail; };
_Static_assert(sizeof(struct fairyc_fixture_array_packet_v0) == 64, "array packet size");
_Static_assert(_Alignof(struct fairyc_fixture_array_packet_v0) == 8, "array packet alignment");
_Static_assert(offsetof(struct fairyc_fixture_array_packet_v0, words) == 4, "array packet words");
_Static_assert(offsetof(struct fairyc_fixture_array_packet_v0, tail) == 60, "array packet tail");
_Static_assert(sizeof(struct fairyc_fixture_array_small_v0) == 8, "small packet size");

struct fairyc_fixture_array_packet_v0 fairyc_fixture_array_change_v0(
    int32_t before, struct fairyc_fixture_array_packet_v0 value, int32_t after
) {
    if (before != 17 || after != 19 || value.lead != 11 || value.tail != 12 ||
        value.words[0] != 1 || value.words[1] != 2 || value.words[2] != 3 ||
        value.wide[0] != -9 || value.wide[1] != 10 || value.codes[0] != -3 || value.codes[1] != 4 ||
        value.sizes[0] != 5 || value.sizes[1] != 6 || value.bytes[0] != 7 || value.bytes[1] != 8 ||
        value.bytes[2] != 9 || value.bytes[3] != 10) {
        struct fairyc_fixture_array_packet_v0 failed = {0};
        return failed;
    }
    value.words[0] = 101;
    value.wide[1] = 200;
    value.codes[1] = -300;
    value.sizes[1] = 400;
    value.bytes[3] = 55;
    value.tail = 66;
    return value;
}

struct fairyc_fixture_array_small_v0 fairyc_fixture_array_small_change_v0(struct fairyc_fixture_array_small_v0 value) {
    if (value.lead != 1 || value.bytes[0] != 2 || value.bytes[1] != 3 || value.bytes[2] != 4 || value.tail != 5) {
        struct fairyc_fixture_array_small_v0 failed = {0};
        return failed;
    }
    value.bytes[1] = 13;
    value.tail = 15;
    return value;
}
`
