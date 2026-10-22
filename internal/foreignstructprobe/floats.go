package foreignstructprobe

const floatingFairySource = `
@repr(C) struct fairyc_fixture_float_packet_v0 { u8 lead; f32 single; f64 wide; u32 marker; }
@repr(C) struct fairyc_fixture_float_envelope_v0 { fairyc_fixture_float_packet_v0 packet; }
@repr(C) struct fairyc_fixture_float_small_v0 { f32 value; }
@repr(C) struct fairyc_fixture_float_pair_v0 { f64 first; f64 second; }
@repr(C) struct fairyc_fixture_float_mixed_v0 { u32 marker; f32 single; f64 wide; }
@link_name("fairyc_fixture_float_make_v0")
unsafe extern C fn fairyc_fixture_float_packet_v0 float_make(i32 choice);
@link_name("fairyc_fixture_float_change_v0")
unsafe extern C fn fairyc_fixture_float_packet_v0 float_change(fairyc_fixture_float_packet_v0 value);
@link_name("fairyc_fixture_float_small_v0")
unsafe extern C fn fairyc_fixture_float_small_v0 float_small(fairyc_fixture_float_small_v0 value);
@link_name("fairyc_fixture_float_pair_v0")
unsafe extern C fn fairyc_fixture_float_pair_v0 float_pair(fairyc_fixture_float_pair_v0 value);
@link_name("fairyc_fixture_float_mixed_v0")
unsafe extern C fn fairyc_fixture_float_mixed_v0 float_mixed(fairyc_fixture_float_mixed_v0 value);
@link_name("fairyc_fixture_float_observe_v0")
unsafe extern C fn i32 float_observe(i32 choice, fairyc_fixture_float_packet_v0 original, fairyc_fixture_float_packet_v0 rebuilt,
    fairyc_fixture_float_packet_v0 changed, fairyc_fixture_float_small_v0 small, fairyc_fixture_float_pair_v0 pair,
    fairyc_fixture_float_mixed_v0 mixed, i32 tail);
fn f32 float_copy_single(f32 value){return value;}
fn f64 float_copy_wide(f64 value){return value;}
fn bool check_float_case(i32 choice){
    let fairyc_fixture_float_packet_v0 original=unsafe{float_make(choice)};
    let fairyc_fixture_float_envelope_v0 envelope=fairyc_fixture_float_envelope_v0{.packet=original};
    let fairyc_fixture_float_small_v0 small=unsafe{float_small(fairyc_fixture_float_small_v0{.value=float_copy_single(envelope.packet.single)})};
    let fairyc_fixture_float_pair_v0 pair=unsafe{float_pair(fairyc_fixture_float_pair_v0{.first=float_copy_wide(envelope.packet.wide),.second=original.wide})};
    let fairyc_fixture_float_mixed_v0 mixed=unsafe{float_mixed(fairyc_fixture_float_mixed_v0{.marker=7,.single=original.single,.wide=original.wide})};
    let fairyc_fixture_float_packet_v0 rebuilt=fairyc_fixture_float_packet_v0{.lead=envelope.packet.lead,.single=small.value,.wide=pair.first,.marker=envelope.packet.marker};
    let fairyc_fixture_float_packet_v0 changed=unsafe{float_change(rebuilt)};
    return unsafe{float_observe(choice,original,rebuilt,changed,small,pair,mixed,19)}==42;
}
fn bool check_floats(){return check_float_case(0)&&check_float_case(1)&&check_float_case(2)&&check_float_case(3)&&check_float_case(4);}
`

const floatingCompanionC = `
#include <string.h>
struct fairyc_fixture_float_packet_v0 { uint8_t lead; float single; double wide; uint32_t marker; };
struct fairyc_fixture_float_small_v0 { float value; };
struct fairyc_fixture_float_pair_v0 { double first; double second; };
struct fairyc_fixture_float_mixed_v0 { uint32_t marker; float single; double wide; };
_Static_assert(sizeof(struct fairyc_fixture_float_packet_v0)==24, "float packet size");
_Static_assert(_Alignof(struct fairyc_fixture_float_packet_v0)==8, "float packet alignment");
_Static_assert(offsetof(struct fairyc_fixture_float_packet_v0,single)==4, "float field offset");
_Static_assert(offsetof(struct fairyc_fixture_float_packet_v0,wide)==8, "double field offset");
_Static_assert(sizeof(struct fairyc_fixture_float_small_v0)==4, "small float size");
_Static_assert(sizeof(struct fairyc_fixture_float_pair_v0)==16, "SSE pair size");
_Static_assert(sizeof(struct fairyc_fixture_float_mixed_v0)==16, "mixed float size");
static uint32_t fairyc_fixture_f32_bits_v0(float value){uint32_t bits;memcpy(&bits,&value,sizeof bits);return bits;}
static uint64_t fairyc_fixture_f64_bits_v0(double value){uint64_t bits;memcpy(&bits,&value,sizeof bits);return bits;}
static uint32_t fairyc_fixture_float_expected_single_v0(int32_t choice){
    switch(choice){
    case 0:return UINT32_C(0x3fa00000);
    case 1:return UINT32_C(0x80000000);
    case 2:return UINT32_C(0x7f800000);
    case 3:return UINT32_C(0x7fc00001);
    default:return UINT32_C(1);
    }
}
static uint64_t fairyc_fixture_float_expected_wide_v0(int32_t choice){
    switch(choice){
    case 0:return UINT64_C(0xc004000000000000);
    case 1:return UINT64_C(0x8000000000000000);
    case 2:return UINT64_C(0xfff0000000000000);
    case 3:return UINT64_C(0x7ff8000000000001);
    default:return UINT64_C(1);
    }
}
static int fairyc_fixture_float_valid_v0(int32_t choice,float single,double wide){
    uint32_t a=fairyc_fixture_f32_bits_v0(single);
    uint64_t b=fairyc_fixture_f64_bits_v0(wide);
    if(choice==3){
        return (a&UINT32_C(0x7fc00000))==UINT32_C(0x7fc00000) &&
            (b&UINT64_C(0x7ff8000000000000))==UINT64_C(0x7ff8000000000000);
    }
    return a==fairyc_fixture_float_expected_single_v0(choice) && b==fairyc_fixture_float_expected_wide_v0(choice);
}
struct fairyc_fixture_float_packet_v0 fairyc_fixture_float_make_v0(int32_t choice){
    struct fairyc_fixture_float_packet_v0 value={11,0.0f,0.0,17};
    uint32_t single=fairyc_fixture_float_expected_single_v0(choice);
    uint64_t wide=fairyc_fixture_float_expected_wide_v0(choice);
    memcpy(&value.single,&single,sizeof single);
    memcpy(&value.wide,&wide,sizeof wide);
    return value;
}
struct fairyc_fixture_float_packet_v0 fairyc_fixture_float_change_v0(struct fairyc_fixture_float_packet_v0 value){
    value.single=-3.5f;value.wide=6.25;value.marker=77;return value;
}
struct fairyc_fixture_float_small_v0 fairyc_fixture_float_small_v0(struct fairyc_fixture_float_small_v0 value){return value;}
struct fairyc_fixture_float_pair_v0 fairyc_fixture_float_pair_v0(struct fairyc_fixture_float_pair_v0 value){return value;}
struct fairyc_fixture_float_mixed_v0 fairyc_fixture_float_mixed_v0(struct fairyc_fixture_float_mixed_v0 value){return value;}
int32_t fairyc_fixture_float_observe_v0(int32_t choice,struct fairyc_fixture_float_packet_v0 original,
    struct fairyc_fixture_float_packet_v0 rebuilt,struct fairyc_fixture_float_packet_v0 changed,
    struct fairyc_fixture_float_small_v0 small,struct fairyc_fixture_float_pair_v0 pair,
    struct fairyc_fixture_float_mixed_v0 mixed,int32_t tail){
    return tail==19 && original.lead==11 && original.marker==17 && rebuilt.lead==11 && rebuilt.marker==17 &&
        changed.lead==11 && changed.marker==77 && mixed.marker==7 &&
        fairyc_fixture_float_valid_v0(choice,original.single,original.wide) &&
        fairyc_fixture_float_valid_v0(choice,rebuilt.single,rebuilt.wide) &&
        fairyc_fixture_float_valid_v0(choice,small.value,pair.first) &&
        fairyc_fixture_float_valid_v0(choice,small.value,pair.second) &&
        fairyc_fixture_float_valid_v0(choice,mixed.single,mixed.wide) &&
        fairyc_fixture_f32_bits_v0(changed.single)==UINT32_C(0xc0600000) &&
        fairyc_fixture_f64_bits_v0(changed.wide)==UINT64_C(0x4019000000000000) ? 42 : 0;
}
`
