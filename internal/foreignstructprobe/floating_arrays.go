package foreignstructprobe

const floatingArrayFairySource = `
@repr(C) struct fairyc_fixture_floating_array_packet_v0 {u8 lead;f32[3] single;f64[3] wide;u8 tail;}
@repr(C) struct fairyc_fixture_floating_array_envelope_v0 {fairyc_fixture_floating_array_packet_v0 packet;}
@repr(C) struct fairyc_fixture_floating_array_small_v0 {f32[1] value;}
@repr(C) struct fairyc_fixture_floating_array_double_v0 {f64[1] value;}
@link_name("fairyc_fixture_floating_array_make_v0")
unsafe extern C fn fairyc_fixture_floating_array_packet_v0 floating_array_make(i32 choice);
@link_name("fairyc_fixture_floating_array_change_v0")
unsafe extern C fn fairyc_fixture_floating_array_packet_v0 floating_array_change(i32 choice,fairyc_fixture_floating_array_packet_v0 value,i32 tail);
@link_name("fairyc_fixture_floating_array_observe_v0")
unsafe extern C fn i32 floating_array_observe(i32 choice,fairyc_fixture_floating_array_packet_v0 original,
 fairyc_fixture_floating_array_envelope_v0 saved,fairyc_fixture_floating_array_packet_v0 rebuilt,
 fairyc_fixture_floating_array_packet_v0 changed,fairyc_fixture_floating_array_packet_v0 direct,
 fairyc_fixture_floating_array_packet_v0 replaced,i32 tail);
@link_name("fairyc_fixture_floating_array_small_make_v0")
unsafe extern C fn fairyc_fixture_floating_array_small_v0 floating_array_small_make(i32 choice);
@link_name("fairyc_fixture_floating_array_double_make_v0")
unsafe extern C fn fairyc_fixture_floating_array_double_v0 floating_array_double_make(i32 choice);
@link_name("fairyc_fixture_floating_array_small_change_v0")
unsafe extern C fn fairyc_fixture_floating_array_small_v0 floating_array_small_change(fairyc_fixture_floating_array_small_v0 value);
@link_name("fairyc_fixture_floating_array_double_change_v0")
unsafe extern C fn fairyc_fixture_floating_array_double_v0 floating_array_double_change(fairyc_fixture_floating_array_double_v0 value);
@link_name("fairyc_fixture_floating_array_small_observe_v0")
unsafe extern C fn i32 floating_array_small_observe(i32 choice,fairyc_fixture_floating_array_small_v0 original,
 fairyc_fixture_floating_array_small_v0 rebuilt,fairyc_fixture_floating_array_small_v0 changed,
 fairyc_fixture_floating_array_double_v0 wide,fairyc_fixture_floating_array_double_v0 wide_copy,
 fairyc_fixture_floating_array_double_v0 wide_changed);
fn f32[3] floating_array_copy_single(f32[3] value){return value;}
fn f64[3] floating_array_copy_wide(f64[3] value){return value;}
fn f32[1] floating_array_copy_small(f32[1] value){return value;}
fn f64[1] floating_array_copy_double(f64[1] value){return value;}
fn void floating_array_replace(ref mut fairyc_fixture_floating_array_packet_v0 p,fairyc_fixture_floating_array_packet_v0 q){p.single=q.single;p.wide=q.wide;}
fn bool check_floating_array_case(i32 choice){
 let fairyc_fixture_floating_array_packet_v0 original=unsafe{floating_array_make(choice)};
 let fairyc_fixture_floating_array_envelope_v0 saved=fairyc_fixture_floating_array_envelope_v0{.packet=original};
 let ref fairyc_fixture_floating_array_envelope_v0 view=ref saved;
 let f32[3] single=floating_array_copy_single(view.packet.single);
 let wide=floating_array_copy_wide(original.wide);
 let fairyc_fixture_floating_array_packet_v0 rebuilt=fairyc_fixture_floating_array_packet_v0{.lead=original.lead,.single=single,.wide=wide,.tail=original.tail};
 let fairyc_fixture_floating_array_packet_v0 changed=unsafe{floating_array_change(choice,rebuilt,19)};
 var fairyc_fixture_floating_array_packet_v0 direct=original;
 direct.single=changed.single;direct.wide=changed.wide;
 var fairyc_fixture_floating_array_packet_v0 replaced=original;
 floating_array_replace(ref mut replaced,changed);
 let fairyc_fixture_floating_array_small_v0 small=unsafe{floating_array_small_make(choice)};
 let fairyc_fixture_floating_array_small_v0 small_copy=fairyc_fixture_floating_array_small_v0{.value=floating_array_copy_small(small.value)};
 let fairyc_fixture_floating_array_small_v0 small_changed=unsafe{floating_array_small_change(small_copy)};
 let fairyc_fixture_floating_array_double_v0 double=unsafe{floating_array_double_make(choice)};
 let fairyc_fixture_floating_array_double_v0 double_copy=fairyc_fixture_floating_array_double_v0{.value=floating_array_copy_double(double.value)};
 let fairyc_fixture_floating_array_double_v0 double_changed=unsafe{floating_array_double_change(double_copy)};
 return unsafe{floating_array_observe(choice,original,saved,rebuilt,changed,direct,replaced,23)}==42 &&
  unsafe{floating_array_small_observe(choice,small,small_copy,small_changed,double,double_copy,double_changed)}==42;
}
fn bool check_floating_arrays(){return check_floating_array_case(0)&&check_floating_array_case(1)&&check_floating_array_case(2)&&check_floating_array_case(3)&&check_floating_array_case(4);}
`

const floatingArrayCompanionC = `
struct fairyc_fixture_floating_array_packet_v0 {uint8_t lead;float single[3];double wide[3];uint8_t tail;};
struct fairyc_fixture_floating_array_envelope_v0 {struct fairyc_fixture_floating_array_packet_v0 packet;};
struct fairyc_fixture_floating_array_small_v0 {float value[1];};
struct fairyc_fixture_floating_array_double_v0 {double value[1];};
_Static_assert(sizeof(struct fairyc_fixture_floating_array_packet_v0)==48,"floating array packet size");
_Static_assert(_Alignof(struct fairyc_fixture_floating_array_packet_v0)==8,"floating array alignment");
_Static_assert(offsetof(struct fairyc_fixture_floating_array_packet_v0,single)==4,"floating array single offset");
_Static_assert(offsetof(struct fairyc_fixture_floating_array_packet_v0,wide)==16,"floating array wide offset");
_Static_assert(offsetof(struct fairyc_fixture_floating_array_packet_v0,tail)==40,"floating array tail offset");
_Static_assert(sizeof(struct fairyc_fixture_floating_array_small_v0)==4,"floating array small size");
_Static_assert(sizeof(struct fairyc_fixture_floating_array_double_v0)==8,"floating array double size");
struct fairyc_fixture_floating_array_packet_v0 fairyc_fixture_floating_array_make_v0(int32_t choice){
 struct fairyc_fixture_floating_array_packet_v0 value={0};value.lead=11;value.tail=13;
 for(unsigned i=0;i<3;++i){
  int32_t item=(choice+(int32_t)i)%5;
  uint32_t single=fairyc_fixture_float_expected_single_v0(item);
  uint64_t wide=fairyc_fixture_float_expected_wide_v0(item);
  memcpy(&value.single[i],&single,sizeof single);memcpy(&value.wide[i],&wide,sizeof wide);
 }return value;
}
static int fairyc_fixture_floating_array_valid_v0(int32_t choice,struct fairyc_fixture_floating_array_packet_v0 value){
 if(value.lead!=11||value.tail!=13){return 0;}
 for(unsigned i=0;i<3;++i){if(!fairyc_fixture_float_valid_v0((choice+(int32_t)i)%5,value.single[i],value.wide[i])){return 0;}}
 return 1;
}
struct fairyc_fixture_floating_array_packet_v0 fairyc_fixture_floating_array_change_v0(int32_t choice,struct fairyc_fixture_floating_array_packet_v0 value,int32_t tail){
 if(tail!=19||!fairyc_fixture_floating_array_valid_v0(choice,value)){struct fairyc_fixture_floating_array_packet_v0 failed={0};return failed;}
 struct fairyc_fixture_floating_array_packet_v0 next=fairyc_fixture_floating_array_make_v0((choice+1)%5);
 for(unsigned i=0;i<3;++i){value.single[i]=next.single[i];value.wide[i]=next.wide[i];}return value;
}
int32_t fairyc_fixture_floating_array_observe_v0(int32_t choice,struct fairyc_fixture_floating_array_packet_v0 original,
 struct fairyc_fixture_floating_array_envelope_v0 saved,struct fairyc_fixture_floating_array_packet_v0 rebuilt,
 struct fairyc_fixture_floating_array_packet_v0 changed,struct fairyc_fixture_floating_array_packet_v0 direct,
 struct fairyc_fixture_floating_array_packet_v0 replaced,int32_t tail){
 return tail==23&&fairyc_fixture_floating_array_valid_v0(choice,original)&&fairyc_fixture_floating_array_valid_v0(choice,saved.packet)&&
  fairyc_fixture_floating_array_valid_v0(choice,rebuilt)&&fairyc_fixture_floating_array_valid_v0((choice+1)%5,changed)&&
  fairyc_fixture_floating_array_valid_v0((choice+1)%5,direct)&&fairyc_fixture_floating_array_valid_v0((choice+1)%5,replaced)?42:0;
}
struct fairyc_fixture_floating_array_small_v0 fairyc_fixture_floating_array_small_make_v0(int32_t choice){
 struct fairyc_fixture_floating_array_small_v0 value={{0}};uint32_t bits=fairyc_fixture_float_expected_single_v0(choice);
 memcpy(&value.value[0],&bits,sizeof bits);return value;
}
struct fairyc_fixture_floating_array_double_v0 fairyc_fixture_floating_array_double_make_v0(int32_t choice){
 struct fairyc_fixture_floating_array_double_v0 value={{0}};uint64_t bits=fairyc_fixture_float_expected_wide_v0(choice);
 memcpy(&value.value[0],&bits,sizeof bits);return value;
}
struct fairyc_fixture_floating_array_small_v0 fairyc_fixture_floating_array_small_change_v0(struct fairyc_fixture_floating_array_small_v0 value){value.value[0]=-3.5f;return value;}
struct fairyc_fixture_floating_array_double_v0 fairyc_fixture_floating_array_double_change_v0(struct fairyc_fixture_floating_array_double_v0 value){value.value[0]=6.25;return value;}
int32_t fairyc_fixture_floating_array_small_observe_v0(int32_t choice,struct fairyc_fixture_floating_array_small_v0 original,
 struct fairyc_fixture_floating_array_small_v0 rebuilt,struct fairyc_fixture_floating_array_small_v0 changed,
 struct fairyc_fixture_floating_array_double_v0 wide,struct fairyc_fixture_floating_array_double_v0 wide_copy,
 struct fairyc_fixture_floating_array_double_v0 wide_changed){
 return fairyc_fixture_float_valid_v0(choice,original.value[0],wide.value[0])&&
  fairyc_fixture_float_valid_v0(choice,rebuilt.value[0],wide_copy.value[0])&&
  fairyc_fixture_f32_bits_v0(changed.value[0])==UINT32_C(0xc0600000)&&
  fairyc_fixture_f64_bits_v0(wide_changed.value[0])==UINT64_C(0x4019000000000000)?42:0;
}
`
