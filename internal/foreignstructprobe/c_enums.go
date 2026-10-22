package foreignstructprobe

const enumFairySource = `
@repr(c::uchar) enum FixtureTiny { Off; On; }
@repr(c::uint) enum FixtureState { Ready; Busy; Done; }
@repr(c::ushort) enum FixtureShort { Off; On; }
@repr(c::longlong) enum FixtureWide { Zero; One; Two; }
@repr(c::long) enum FixtureLong { Zero; One; Two; }
@repr(C) struct fairyc_fixture_enum_packet_v0 {
    u8 lead; FixtureTiny tiny; FixtureState state; FixtureWide wide; FixtureLong platform; u8 tail;
}
@repr(C) struct fairyc_fixture_enum_small_v0 { FixtureTiny tiny; u8 tail; FixtureShort short_state; }
@link_name("fairyc_fixture_enum_change_v0")
unsafe extern C fn fairyc_fixture_enum_packet_v0 enum_change(i32 before, fairyc_fixture_enum_packet_v0 value, i32 after);
@link_name("fairyc_fixture_enum_small_change_v0")
unsafe extern C fn fairyc_fixture_enum_small_v0 enum_small_change(fairyc_fixture_enum_small_v0 value);
fn i32 enum_state_code(FixtureState value) {
    return switch(value) { case .Ready:{0} case .Busy:{1} case .Done:{2} };
}
fn bool check_c_enums() {
    var fairyc_fixture_enum_packet_v0 value=fairyc_fixture_enum_packet_v0{.lead=7,.tiny=.On,.state=.Busy,.wide=.Two,.platform=.One,.tail=9};
    let fairyc_fixture_enum_packet_v0 copy=value;
    value.state=.Done;
    let fairyc_fixture_enum_packet_v0 changed=unsafe{enum_change(17,value,19)};
    let fairyc_fixture_enum_small_v0 small=fairyc_fixture_enum_small_v0{.tiny=.On,.tail=11,.short_state=.On};
    let fairyc_fixture_enum_small_v0 changed_small=unsafe{enum_small_change(small)};
    return enum_state_code(copy.state)==1 && enum_state_code(value.state)==2 &&
        enum_state_code(changed.state)==0 && changed.lead==7 && changed.tail==33 && value.tail==9 &&
        switch(value.tiny){case .Off:{false}case .On:{true}} &&
        switch(changed.tiny){case .Off:{true}case .On:{false}} &&
        switch(value.wide){case .Zero:{false}case .One:{false}case .Two:{true}} &&
        switch(changed.wide){case .Zero:{false}case .One:{true}case .Two:{false}} &&
        switch(value.platform){case .Zero:{false}case .One:{true}case .Two:{false}} &&
        switch(changed.platform){case .Zero:{false}case .One:{false}case .Two:{true}} &&
        small.tail==11 && changed_small.tail==22 &&
        switch(small.short_state){case .Off:{false}case .On:{true}} &&
        switch(changed_small.short_state){case .Off:{true}case .On:{false}} &&
        switch(small.tiny){case .Off:{false}case .On:{true}} &&
        switch(changed_small.tiny){case .Off:{true}case .On:{false}};
}
`

const enumCompanionC = `
struct fairyc_fixture_enum_packet_v0 {
    uint8_t lead; unsigned char tiny; unsigned int state; long long wide; long platform; uint8_t tail;
};
struct fairyc_fixture_enum_small_v0 { unsigned char tiny; uint8_t tail; unsigned short short_state; };
_Static_assert(sizeof(struct fairyc_fixture_enum_packet_v0) == (sizeof(long) == 8 ? 32 : 24), "enum packet size");
_Static_assert(_Alignof(struct fairyc_fixture_enum_packet_v0) == 8, "enum packet alignment");
_Static_assert(offsetof(struct fairyc_fixture_enum_packet_v0, tiny) == 1, "enum tiny offset");
_Static_assert(offsetof(struct fairyc_fixture_enum_packet_v0, state) == 4, "enum state offset");
_Static_assert(offsetof(struct fairyc_fixture_enum_packet_v0, wide) == 8, "enum wide offset");
_Static_assert(offsetof(struct fairyc_fixture_enum_packet_v0, platform) == 16, "enum long offset");
_Static_assert(offsetof(struct fairyc_fixture_enum_packet_v0, tail) == 16 + sizeof(long), "enum tail offset");
_Static_assert(sizeof(struct fairyc_fixture_enum_small_v0) == 4, "small enum packet size");
_Static_assert(offsetof(struct fairyc_fixture_enum_small_v0, short_state) == 2, "short enum offset");
struct fairyc_fixture_enum_packet_v0 fairyc_fixture_enum_change_v0(
    int32_t before, struct fairyc_fixture_enum_packet_v0 value, int32_t after
) {
    if (before != 17 || after != 19 || value.lead != 7 || value.tiny != 1 || value.state != 2 ||
        value.wide != 2 || value.platform != 1 || value.tail != 9) {
        struct fairyc_fixture_enum_packet_v0 failed = {0};
        return failed;
    }
    value.tiny = 0;
    value.state = 0;
    value.wide = 1;
    value.platform = 2;
    value.tail = 33;
    return value;
}
struct fairyc_fixture_enum_small_v0 fairyc_fixture_enum_small_change_v0(struct fairyc_fixture_enum_small_v0 value) {
    if (value.tiny != 1 || value.tail != 11 || value.short_state != 1) {
        struct fairyc_fixture_enum_small_v0 failed = {0};
        return failed;
    }
    value.tiny = 0;
    value.short_state = 0;
    value.tail = 22;
    return value;
}
`
