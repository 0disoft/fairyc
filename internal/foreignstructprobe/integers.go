package foreignstructprobe

const integerFairySource = `
@repr(C) struct fairyc_fixture_integer_packet_v0 { u8 lead; i8 tiny; i16 small; u16 word; u64 wide; isize offset; }
@repr(C) struct fairyc_fixture_integer_envelope_v0 { fairyc_fixture_integer_packet_v0 packet; }
@repr(C) struct fairyc_fixture_integer_i8_v0 { i8 value; }
@repr(C) struct fairyc_fixture_integer_i16_v0 { i16 value; }
@repr(C) struct fairyc_fixture_integer_u16_v0 { u16 value; }
@repr(C) struct fairyc_fixture_integer_u64_v0 { u64 value; }
@repr(C) struct fairyc_fixture_integer_isize_v0 { isize value; }
@link_name("fairyc_fixture_integer_make_v0")
unsafe extern C fn fairyc_fixture_integer_packet_v0 integer_make(i32 choice);
@link_name("fairyc_fixture_integer_change_v0")
unsafe extern C fn fairyc_fixture_integer_packet_v0 integer_change(i32 choice, fairyc_fixture_integer_packet_v0 value, i32 tail);
@link_name("fairyc_fixture_integer_i8_v0")
unsafe extern C fn fairyc_fixture_integer_i8_v0 integer_i8(fairyc_fixture_integer_i8_v0 value);
@link_name("fairyc_fixture_integer_i16_v0")
unsafe extern C fn fairyc_fixture_integer_i16_v0 integer_i16(fairyc_fixture_integer_i16_v0 value);
@link_name("fairyc_fixture_integer_u16_v0")
unsafe extern C fn fairyc_fixture_integer_u16_v0 integer_u16(fairyc_fixture_integer_u16_v0 value);
@link_name("fairyc_fixture_integer_u64_v0")
unsafe extern C fn fairyc_fixture_integer_u64_v0 integer_u64(fairyc_fixture_integer_u64_v0 value);
@link_name("fairyc_fixture_integer_isize_v0")
unsafe extern C fn fairyc_fixture_integer_isize_v0 integer_isize(fairyc_fixture_integer_isize_v0 value);
@link_name("fairyc_fixture_integer_observe_v0")
unsafe extern C fn i32 integer_observe(i32 choice, fairyc_fixture_integer_packet_v0 original,
    fairyc_fixture_integer_envelope_v0 envelope, fairyc_fixture_integer_packet_v0 rebuilt,
    fairyc_fixture_integer_packet_v0 changed, i32 tail);
fn i8 integer_copy_i8(i8 value){return value;}
fn i16 integer_copy_i16(i16 value){return value;}
fn u16 integer_copy_u16(u16 value){return value;}
fn u64 integer_copy_u64(u64 value){return value;}
fn isize integer_copy_isize(isize value){return value;}
fn bool check_integer_case(i32 choice){
    let fairyc_fixture_integer_packet_v0 original=unsafe{integer_make(choice)};
    let fairyc_fixture_integer_envelope_v0 envelope=fairyc_fixture_integer_envelope_v0{.packet=original};
    let fairyc_fixture_integer_i8_v0 tiny=unsafe{integer_i8(fairyc_fixture_integer_i8_v0{.value=integer_copy_i8(envelope.packet.tiny)})};
    let fairyc_fixture_integer_i16_v0 small=unsafe{integer_i16(fairyc_fixture_integer_i16_v0{.value=integer_copy_i16(envelope.packet.small)})};
    let fairyc_fixture_integer_u16_v0 word=unsafe{integer_u16(fairyc_fixture_integer_u16_v0{.value=integer_copy_u16(envelope.packet.word)})};
    let fairyc_fixture_integer_u64_v0 wide=unsafe{integer_u64(fairyc_fixture_integer_u64_v0{.value=integer_copy_u64(envelope.packet.wide)})};
    let fairyc_fixture_integer_isize_v0 offset=unsafe{integer_isize(fairyc_fixture_integer_isize_v0{.value=integer_copy_isize(envelope.packet.offset)})};
    let fairyc_fixture_integer_packet_v0 rebuilt=fairyc_fixture_integer_packet_v0{
        .lead=original.lead,.tiny=tiny.value,.small=small.value,.word=word.value,.wide=wide.value,.offset=offset.value};
    let fairyc_fixture_integer_packet_v0 changed=unsafe{integer_change(choice,rebuilt,19)};
    return unsafe{integer_observe(choice,original,envelope,rebuilt,changed,23)}==42;
}
fn bool check_integers(){return check_integer_case(0)&&check_integer_case(1)&&check_integer_case(2)&&check_integer_case(3);}
`

const integerCompanionC = `
struct fairyc_fixture_integer_packet_v0 { uint8_t lead; int8_t tiny; int16_t small; uint16_t word; uint64_t wide; intptr_t offset; };
struct fairyc_fixture_integer_envelope_v0 { struct fairyc_fixture_integer_packet_v0 packet; };
struct fairyc_fixture_integer_i8_v0 { int8_t value; };
struct fairyc_fixture_integer_i16_v0 { int16_t value; };
struct fairyc_fixture_integer_u16_v0 { uint16_t value; };
struct fairyc_fixture_integer_u64_v0 { uint64_t value; };
struct fairyc_fixture_integer_isize_v0 { intptr_t value; };
_Static_assert(sizeof(intptr_t)==8, "integer pointer width");
_Static_assert(sizeof(struct fairyc_fixture_integer_packet_v0)==24, "integer packet size");
_Static_assert(_Alignof(struct fairyc_fixture_integer_packet_v0)==8, "integer packet alignment");
_Static_assert(offsetof(struct fairyc_fixture_integer_packet_v0,tiny)==1, "integer tiny offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_packet_v0,small)==2, "integer small offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_packet_v0,word)==4, "integer word offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_packet_v0,wide)==8, "integer wide offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_packet_v0,offset)==16, "integer offset offset");
struct fairyc_fixture_integer_packet_v0 fairyc_fixture_integer_make_v0(int32_t choice){
    struct fairyc_fixture_integer_packet_v0 value={11,0,0,0,0,0};
    switch(choice){
    case 0: break;
    case 1: value.tiny=INT8_MIN;value.small=INT16_MIN;value.word=UINT16_MAX;value.wide=UINT64_MAX;value.offset=INTPTR_MIN;break;
    case 2: value.tiny=INT8_MAX;value.small=INT16_MAX;value.word=UINT16_C(32768);value.wide=UINT64_C(0x8000000000000001);value.offset=INTPTR_MAX;break;
    default: value.tiny=-1;value.small=-1;value.word=17;value.wide=UINT64_C(0x0123456789abcdef);value.offset=-65539;break;
    }
    return value;
}
static int fairyc_fixture_integer_equal_v0(struct fairyc_fixture_integer_packet_v0 a, struct fairyc_fixture_integer_packet_v0 b){
    return a.lead==b.lead && a.tiny==b.tiny && a.small==b.small && a.word==b.word && a.wide==b.wide && a.offset==b.offset;
}
struct fairyc_fixture_integer_packet_v0 fairyc_fixture_integer_change_v0(int32_t choice, struct fairyc_fixture_integer_packet_v0 value, int32_t tail){
    if(tail!=19 || !fairyc_fixture_integer_equal_v0(value,fairyc_fixture_integer_make_v0(choice))){
        struct fairyc_fixture_integer_packet_v0 failed={0,0,0,0,0,0};return failed;
    }
    value.lead=99;value.tiny=-7;value.small=-1234;value.word=60000;value.wide=UINT64_C(0xfedcba9876543210);value.offset=-123456789;return value;
}
struct fairyc_fixture_integer_i8_v0 fairyc_fixture_integer_i8_v0(struct fairyc_fixture_integer_i8_v0 value){return value;}
struct fairyc_fixture_integer_i16_v0 fairyc_fixture_integer_i16_v0(struct fairyc_fixture_integer_i16_v0 value){return value;}
struct fairyc_fixture_integer_u16_v0 fairyc_fixture_integer_u16_v0(struct fairyc_fixture_integer_u16_v0 value){return value;}
struct fairyc_fixture_integer_u64_v0 fairyc_fixture_integer_u64_v0(struct fairyc_fixture_integer_u64_v0 value){return value;}
struct fairyc_fixture_integer_isize_v0 fairyc_fixture_integer_isize_v0(struct fairyc_fixture_integer_isize_v0 value){return value;}
int32_t fairyc_fixture_integer_observe_v0(int32_t choice, struct fairyc_fixture_integer_packet_v0 original,
    struct fairyc_fixture_integer_envelope_v0 envelope, struct fairyc_fixture_integer_packet_v0 rebuilt,
    struct fairyc_fixture_integer_packet_v0 changed, int32_t tail){
    struct fairyc_fixture_integer_packet_v0 expected=fairyc_fixture_integer_make_v0(choice);
    return tail==23 && fairyc_fixture_integer_equal_v0(original,expected) &&
        fairyc_fixture_integer_equal_v0(envelope.packet,expected) && fairyc_fixture_integer_equal_v0(rebuilt,expected) &&
        changed.lead==99 && changed.tiny==-7 && changed.small==-1234 && changed.word==60000 &&
        changed.wide==UINT64_C(0xfedcba9876543210) && changed.offset==-123456789 ? 42 : 0;
}
`
