package foreignstructprobe

const integerArrayFairySource = `
@repr(C) struct fairyc_fixture_integer_array_packet_v0 { u8 lead; i8[3] tiny;i16[3] small;u16[3] word;u64[3] wide;isize[3] offset; u8 tail; }
@repr(C) struct fairyc_fixture_integer_array_envelope_v0 { fairyc_fixture_integer_array_packet_v0 packet; }
@link_name("fairyc_fixture_integer_array_make_v0") unsafe extern C fn fairyc_fixture_integer_array_packet_v0 integer_array_make(i32 choice);
@link_name("fairyc_fixture_integer_array_change_v0") unsafe extern C fn fairyc_fixture_integer_array_packet_v0 integer_array_change(i32 choice,fairyc_fixture_integer_array_packet_v0 value,i32 tail);
@link_name("fairyc_fixture_integer_array_observe_v0") unsafe extern C fn i32 integer_array_observe(i32 choice,fairyc_fixture_integer_array_packet_v0 original,fairyc_fixture_integer_array_envelope_v0 saved,fairyc_fixture_integer_array_packet_v0 rebuilt,fairyc_fixture_integer_array_packet_v0 changed,fairyc_fixture_integer_array_packet_v0 replaced,i32 tail);
fn i8[3] integer_array_copy_tiny(i8[3] value){return value;}
@repr(C) struct fairyc_fixture_integer_array_tiny_v0 { i8[1] value; }
@link_name("fairyc_fixture_integer_array_tiny_v0_make") unsafe extern C fn fairyc_fixture_integer_array_tiny_v0 integer_array_tiny_make(i32 choice);
@link_name("fairyc_fixture_integer_array_tiny_v0_change") unsafe extern C fn fairyc_fixture_integer_array_tiny_v0 integer_array_tiny_change(fairyc_fixture_integer_array_tiny_v0 value);
@link_name("fairyc_fixture_integer_array_tiny_v0_observe") unsafe extern C fn i32 integer_array_tiny_observe(i32 choice,fairyc_fixture_integer_array_tiny_v0 original,fairyc_fixture_integer_array_tiny_v0 rebuilt,fairyc_fixture_integer_array_tiny_v0 changed);
fn i16[3] integer_array_copy_small(i16[3] value){return value;}
@repr(C) struct fairyc_fixture_integer_array_small_v0 { i16[1] value; }
@link_name("fairyc_fixture_integer_array_small_v0_make") unsafe extern C fn fairyc_fixture_integer_array_small_v0 integer_array_small_make(i32 choice);
@link_name("fairyc_fixture_integer_array_small_v0_change") unsafe extern C fn fairyc_fixture_integer_array_small_v0 integer_array_small_change(fairyc_fixture_integer_array_small_v0 value);
@link_name("fairyc_fixture_integer_array_small_v0_observe") unsafe extern C fn i32 integer_array_small_observe(i32 choice,fairyc_fixture_integer_array_small_v0 original,fairyc_fixture_integer_array_small_v0 rebuilt,fairyc_fixture_integer_array_small_v0 changed);
fn u16[3] integer_array_copy_word(u16[3] value){return value;}
@repr(C) struct fairyc_fixture_integer_array_word_v0 { u16[1] value; }
@link_name("fairyc_fixture_integer_array_word_v0_make") unsafe extern C fn fairyc_fixture_integer_array_word_v0 integer_array_word_make(i32 choice);
@link_name("fairyc_fixture_integer_array_word_v0_change") unsafe extern C fn fairyc_fixture_integer_array_word_v0 integer_array_word_change(fairyc_fixture_integer_array_word_v0 value);
@link_name("fairyc_fixture_integer_array_word_v0_observe") unsafe extern C fn i32 integer_array_word_observe(i32 choice,fairyc_fixture_integer_array_word_v0 original,fairyc_fixture_integer_array_word_v0 rebuilt,fairyc_fixture_integer_array_word_v0 changed);
fn u64[3] integer_array_copy_wide(u64[3] value){return value;}
@repr(C) struct fairyc_fixture_integer_array_wide_v0 { u64[1] value; }
@link_name("fairyc_fixture_integer_array_wide_v0_make") unsafe extern C fn fairyc_fixture_integer_array_wide_v0 integer_array_wide_make(i32 choice);
@link_name("fairyc_fixture_integer_array_wide_v0_change") unsafe extern C fn fairyc_fixture_integer_array_wide_v0 integer_array_wide_change(fairyc_fixture_integer_array_wide_v0 value);
@link_name("fairyc_fixture_integer_array_wide_v0_observe") unsafe extern C fn i32 integer_array_wide_observe(i32 choice,fairyc_fixture_integer_array_wide_v0 original,fairyc_fixture_integer_array_wide_v0 rebuilt,fairyc_fixture_integer_array_wide_v0 changed);
fn isize[3] integer_array_copy_offset(isize[3] value){return value;}
@repr(C) struct fairyc_fixture_integer_array_offset_v0 { isize[1] value; }
@link_name("fairyc_fixture_integer_array_offset_v0_make") unsafe extern C fn fairyc_fixture_integer_array_offset_v0 integer_array_offset_make(i32 choice);
@link_name("fairyc_fixture_integer_array_offset_v0_change") unsafe extern C fn fairyc_fixture_integer_array_offset_v0 integer_array_offset_change(fairyc_fixture_integer_array_offset_v0 value);
@link_name("fairyc_fixture_integer_array_offset_v0_observe") unsafe extern C fn i32 integer_array_offset_observe(i32 choice,fairyc_fixture_integer_array_offset_v0 original,fairyc_fixture_integer_array_offset_v0 rebuilt,fairyc_fixture_integer_array_offset_v0 changed);
fn void integer_array_replace(ref mut fairyc_fixture_integer_array_packet_v0 p,fairyc_fixture_integer_array_packet_v0 q){p.tiny=q.tiny;p.small=q.small;p.word=q.word;p.wide=q.wide;p.offset=q.offset;}
fn bool check_integer_array_case(i32 choice){
    let fairyc_fixture_integer_array_packet_v0 original=unsafe{integer_array_make(choice)};
    let fairyc_fixture_integer_array_envelope_v0 saved=fairyc_fixture_integer_array_envelope_v0{.packet=original};
    let ref fairyc_fixture_integer_array_envelope_v0 view=ref saved;
    let fairyc_fixture_integer_array_packet_v0 rebuilt=fairyc_fixture_integer_array_packet_v0{.lead=original.lead,.tiny=integer_array_copy_tiny(view.packet.tiny),.small=integer_array_copy_small(view.packet.small),.word=integer_array_copy_word(view.packet.word),.wide=integer_array_copy_wide(view.packet.wide),.offset=integer_array_copy_offset(view.packet.offset),.tail=original.tail};
    let fairyc_fixture_integer_array_packet_v0 changed=unsafe{integer_array_change(choice,rebuilt,19)};
    var fairyc_fixture_integer_array_packet_v0 replaced=original;
    replaced.tiny=changed.tiny;
    integer_array_replace(ref mut replaced,changed);
    let fairyc_fixture_integer_array_tiny_v0 tiny=unsafe{integer_array_tiny_make(choice)};
    let fairyc_fixture_integer_array_tiny_v0 tiny_copy=fairyc_fixture_integer_array_tiny_v0{.value=tiny.value};
    let fairyc_fixture_integer_array_tiny_v0 tiny_changed=unsafe{integer_array_tiny_change(tiny_copy)};
    let fairyc_fixture_integer_array_small_v0 small=unsafe{integer_array_small_make(choice)};
    let fairyc_fixture_integer_array_small_v0 small_copy=fairyc_fixture_integer_array_small_v0{.value=small.value};
    let fairyc_fixture_integer_array_small_v0 small_changed=unsafe{integer_array_small_change(small_copy)};
    let fairyc_fixture_integer_array_word_v0 word=unsafe{integer_array_word_make(choice)};
    let fairyc_fixture_integer_array_word_v0 word_copy=fairyc_fixture_integer_array_word_v0{.value=word.value};
    let fairyc_fixture_integer_array_word_v0 word_changed=unsafe{integer_array_word_change(word_copy)};
    let fairyc_fixture_integer_array_wide_v0 wide=unsafe{integer_array_wide_make(choice)};
    let fairyc_fixture_integer_array_wide_v0 wide_copy=fairyc_fixture_integer_array_wide_v0{.value=wide.value};
    let fairyc_fixture_integer_array_wide_v0 wide_changed=unsafe{integer_array_wide_change(wide_copy)};
    let fairyc_fixture_integer_array_offset_v0 offset=unsafe{integer_array_offset_make(choice)};
    let fairyc_fixture_integer_array_offset_v0 offset_copy=fairyc_fixture_integer_array_offset_v0{.value=offset.value};
    let fairyc_fixture_integer_array_offset_v0 offset_changed=unsafe{integer_array_offset_change(offset_copy)};
    return unsafe{integer_array_observe(choice,original,saved,rebuilt,changed,replaced,23)}==42 &&
        unsafe{integer_array_tiny_observe(choice,tiny,tiny_copy,tiny_changed)}==42 &&
        unsafe{integer_array_small_observe(choice,small,small_copy,small_changed)}==42 &&
        unsafe{integer_array_word_observe(choice,word,word_copy,word_changed)}==42 &&
        unsafe{integer_array_wide_observe(choice,wide,wide_copy,wide_changed)}==42 &&
        unsafe{integer_array_offset_observe(choice,offset,offset_copy,offset_changed)}==42;
}
fn bool check_integer_arrays(){return check_integer_array_case(0)&&check_integer_array_case(1)&&check_integer_array_case(2)&&check_integer_array_case(3);}
`

const integerArrayCompanionC = `
struct fairyc_fixture_integer_array_packet_v0 { uint8_t lead; int8_t tiny[3];int16_t small[3];uint16_t word[3];uint64_t wide[3];intptr_t offset[3]; uint8_t tail; };
struct fairyc_fixture_integer_array_envelope_v0 { struct fairyc_fixture_integer_array_packet_v0 packet; };
_Static_assert(sizeof(struct fairyc_fixture_integer_array_packet_v0)==72, "integer array packet size");
_Static_assert(_Alignof(struct fairyc_fixture_integer_array_packet_v0)==8, "integer array packet alignment");
_Static_assert(offsetof(struct fairyc_fixture_integer_array_packet_v0,tiny)==1, "integer array tiny offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_array_packet_v0,small)==4, "integer array small offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_array_packet_v0,word)==10, "integer array word offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_array_packet_v0,wide)==16, "integer array wide offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_array_packet_v0,offset)==40, "integer array offset offset");
_Static_assert(offsetof(struct fairyc_fixture_integer_array_packet_v0,tail)==64, "integer array tail offset");
struct fairyc_fixture_integer_array_packet_v0 fairyc_fixture_integer_array_make_v0(int32_t choice){
    struct fairyc_fixture_integer_array_packet_v0 value={0}; value.lead=11;value.tail=13;
    static const int8_t tiny[4][3]={{0,0,0},{INT8_MIN,INT8_MAX,-7},{INT8_MAX,INT8_MIN,0},{-7,17,-33}};
    for(unsigned i=0;i<3;++i){value.tiny[i]=tiny[(unsigned)choice%4][i];}
    static const int16_t small[4][3]={{0,0,0},{INT16_MIN,INT16_MAX,-1234},{INT16_MAX,INT16_MIN,0},{-1234,12345,-23456}};
    for(unsigned i=0;i<3;++i){value.small[i]=small[(unsigned)choice%4][i];}
    static const uint16_t word[4][3]={{0,0,0},{0,UINT16_MAX,UINT16_C(32768)},{UINT16_MAX,0,0},{UINT16_C(32768),17,60000}};
    for(unsigned i=0;i<3;++i){value.word[i]=word[(unsigned)choice%4][i];}
    static const uint64_t wide[4][3]={{0,0,0},{0,UINT64_MAX,UINT64_C(0x8000000000000001)},{UINT64_MAX,0,0},{UINT64_C(0x8000000000000001),UINT64_C(0xfedcba9876543210),UINT64_C(0x123456789abcdef)}};
    for(unsigned i=0;i<3;++i){value.wide[i]=wide[(unsigned)choice%4][i];}
    static const intptr_t offset[4][3]={{0,0,0},{INTPTR_MIN,INTPTR_MAX,-65539},{INTPTR_MAX,INTPTR_MIN,0},{-65539,1234567,-9876543}};
    for(unsigned i=0;i<3;++i){value.offset[i]=offset[(unsigned)choice%4][i];}
    return value;
}
static int fairyc_fixture_integer_array_equal_v0(struct fairyc_fixture_integer_array_packet_v0 a,struct fairyc_fixture_integer_array_packet_v0 b){
    if(a.lead!=b.lead||a.tail!=b.tail){return 0;}
    for(unsigned i=0;i<3;++i){
        if(a.tiny[i]!=b.tiny[i]||a.small[i]!=b.small[i]||a.word[i]!=b.word[i]||a.wide[i]!=b.wide[i]||a.offset[i]!=b.offset[i]){return 0;}
    } return 1;
}
struct fairyc_fixture_integer_array_packet_v0 fairyc_fixture_integer_array_change_v0(int32_t choice,struct fairyc_fixture_integer_array_packet_v0 value,int32_t tail){
    if(tail!=19||!fairyc_fixture_integer_array_equal_v0(value,fairyc_fixture_integer_array_make_v0(choice))){struct fairyc_fixture_integer_array_packet_v0 failed={0};return failed;}
    return fairyc_fixture_integer_array_make_v0((choice+1)%4);
}
int32_t fairyc_fixture_integer_array_observe_v0(int32_t choice,struct fairyc_fixture_integer_array_packet_v0 original,struct fairyc_fixture_integer_array_envelope_v0 saved,struct fairyc_fixture_integer_array_packet_v0 rebuilt,struct fairyc_fixture_integer_array_packet_v0 changed,struct fairyc_fixture_integer_array_packet_v0 replaced,int32_t tail){
    struct fairyc_fixture_integer_array_packet_v0 expected=fairyc_fixture_integer_array_make_v0(choice);
    struct fairyc_fixture_integer_array_packet_v0 next=fairyc_fixture_integer_array_make_v0((choice+1)%4);
    return tail==23 && fairyc_fixture_integer_array_equal_v0(original,expected) &&
        fairyc_fixture_integer_array_equal_v0(saved.packet,expected) && fairyc_fixture_integer_array_equal_v0(rebuilt,expected) &&
        fairyc_fixture_integer_array_equal_v0(changed,next) && fairyc_fixture_integer_array_equal_v0(replaced,next) ? 42 : 0;
}
struct fairyc_fixture_integer_array_tiny_v0 { int8_t value[1]; };
_Static_assert(sizeof(struct fairyc_fixture_integer_array_tiny_v0)==sizeof(int8_t), "small integer array size");
struct fairyc_fixture_integer_array_tiny_v0 fairyc_fixture_integer_array_tiny_v0_make(int32_t choice){struct fairyc_fixture_integer_array_tiny_v0 v={{0}};struct fairyc_fixture_integer_array_packet_v0 p=fairyc_fixture_integer_array_make_v0(choice);v.value[0]=p.tiny[0];return v;}
struct fairyc_fixture_integer_array_tiny_v0 fairyc_fixture_integer_array_tiny_v0_change(struct fairyc_fixture_integer_array_tiny_v0 value){value.value[0]=17;return value;}
int32_t fairyc_fixture_integer_array_tiny_v0_observe(int32_t choice,struct fairyc_fixture_integer_array_tiny_v0 original,struct fairyc_fixture_integer_array_tiny_v0 rebuilt,struct fairyc_fixture_integer_array_tiny_v0 changed){struct fairyc_fixture_integer_array_tiny_v0 expected=fairyc_fixture_integer_array_tiny_v0_make(choice);return original.value[0]==expected.value[0] && rebuilt.value[0]==original.value[0] && changed.value[0]==17 ? 42:0;}
struct fairyc_fixture_integer_array_small_v0 { int16_t value[1]; };
_Static_assert(sizeof(struct fairyc_fixture_integer_array_small_v0)==sizeof(int16_t), "small integer array size");
struct fairyc_fixture_integer_array_small_v0 fairyc_fixture_integer_array_small_v0_make(int32_t choice){struct fairyc_fixture_integer_array_small_v0 v={{0}};struct fairyc_fixture_integer_array_packet_v0 p=fairyc_fixture_integer_array_make_v0(choice);v.value[0]=p.small[0];return v;}
struct fairyc_fixture_integer_array_small_v0 fairyc_fixture_integer_array_small_v0_change(struct fairyc_fixture_integer_array_small_v0 value){value.value[0]=12345;return value;}
int32_t fairyc_fixture_integer_array_small_v0_observe(int32_t choice,struct fairyc_fixture_integer_array_small_v0 original,struct fairyc_fixture_integer_array_small_v0 rebuilt,struct fairyc_fixture_integer_array_small_v0 changed){struct fairyc_fixture_integer_array_small_v0 expected=fairyc_fixture_integer_array_small_v0_make(choice);return original.value[0]==expected.value[0] && rebuilt.value[0]==original.value[0] && changed.value[0]==12345 ? 42:0;}
struct fairyc_fixture_integer_array_word_v0 { uint16_t value[1]; };
_Static_assert(sizeof(struct fairyc_fixture_integer_array_word_v0)==sizeof(uint16_t), "small integer array size");
struct fairyc_fixture_integer_array_word_v0 fairyc_fixture_integer_array_word_v0_make(int32_t choice){struct fairyc_fixture_integer_array_word_v0 v={{0}};struct fairyc_fixture_integer_array_packet_v0 p=fairyc_fixture_integer_array_make_v0(choice);v.value[0]=p.word[0];return v;}
struct fairyc_fixture_integer_array_word_v0 fairyc_fixture_integer_array_word_v0_change(struct fairyc_fixture_integer_array_word_v0 value){value.value[0]=17;return value;}
int32_t fairyc_fixture_integer_array_word_v0_observe(int32_t choice,struct fairyc_fixture_integer_array_word_v0 original,struct fairyc_fixture_integer_array_word_v0 rebuilt,struct fairyc_fixture_integer_array_word_v0 changed){struct fairyc_fixture_integer_array_word_v0 expected=fairyc_fixture_integer_array_word_v0_make(choice);return original.value[0]==expected.value[0] && rebuilt.value[0]==original.value[0] && changed.value[0]==17 ? 42:0;}
struct fairyc_fixture_integer_array_wide_v0 { uint64_t value[1]; };
_Static_assert(sizeof(struct fairyc_fixture_integer_array_wide_v0)==sizeof(uint64_t), "small integer array size");
struct fairyc_fixture_integer_array_wide_v0 fairyc_fixture_integer_array_wide_v0_make(int32_t choice){struct fairyc_fixture_integer_array_wide_v0 v={{0}};struct fairyc_fixture_integer_array_packet_v0 p=fairyc_fixture_integer_array_make_v0(choice);v.value[0]=p.wide[0];return v;}
struct fairyc_fixture_integer_array_wide_v0 fairyc_fixture_integer_array_wide_v0_change(struct fairyc_fixture_integer_array_wide_v0 value){value.value[0]=UINT64_C(0xfedcba9876543210);return value;}
int32_t fairyc_fixture_integer_array_wide_v0_observe(int32_t choice,struct fairyc_fixture_integer_array_wide_v0 original,struct fairyc_fixture_integer_array_wide_v0 rebuilt,struct fairyc_fixture_integer_array_wide_v0 changed){struct fairyc_fixture_integer_array_wide_v0 expected=fairyc_fixture_integer_array_wide_v0_make(choice);return original.value[0]==expected.value[0] && rebuilt.value[0]==original.value[0] && changed.value[0]==UINT64_C(0xfedcba9876543210) ? 42:0;}
struct fairyc_fixture_integer_array_offset_v0 { intptr_t value[1]; };
_Static_assert(sizeof(struct fairyc_fixture_integer_array_offset_v0)==sizeof(intptr_t), "small integer array size");
struct fairyc_fixture_integer_array_offset_v0 fairyc_fixture_integer_array_offset_v0_make(int32_t choice){struct fairyc_fixture_integer_array_offset_v0 v={{0}};struct fairyc_fixture_integer_array_packet_v0 p=fairyc_fixture_integer_array_make_v0(choice);v.value[0]=p.offset[0];return v;}
struct fairyc_fixture_integer_array_offset_v0 fairyc_fixture_integer_array_offset_v0_change(struct fairyc_fixture_integer_array_offset_v0 value){value.value[0]=1234567;return value;}
int32_t fairyc_fixture_integer_array_offset_v0_observe(int32_t choice,struct fairyc_fixture_integer_array_offset_v0 original,struct fairyc_fixture_integer_array_offset_v0 rebuilt,struct fairyc_fixture_integer_array_offset_v0 changed){struct fairyc_fixture_integer_array_offset_v0 expected=fairyc_fixture_integer_array_offset_v0_make(choice);return original.value[0]==expected.value[0] && rebuilt.value[0]==original.value[0] && changed.value[0]==1234567 ? 42:0;}
`
