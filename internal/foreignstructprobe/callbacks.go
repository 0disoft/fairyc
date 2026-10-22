package foreignstructprobe

const callbackFairySource = `
@link_name("fairyc_fixture_callback_twice_v0")
unsafe extern C fn i32 callback_twice(i32 value, unsafe extern C fn i32(i32, ptr mut void) callback, ptr mut void data);
@link_name("fairyc_fixture_callback_bump_v0")
unsafe extern C fn i32 callback_bump(i32 value, ptr mut void data);
@link_name("fairyc_fixture_callback_context_v0")
unsafe extern C fn ptr mut void callback_context();
@link_name("fairyc_fixture_callback_observe_v0")
unsafe extern C fn i32 callback_observe();
unsafe fn i32 callback_pure(i32 value, ptr mut void data){return value+1;}
unsafe fn i32 callback_counted(i32 value, ptr mut void data){return callback_bump(value,data);}
fn bool check_callbacks(){
    let i32 pure=unsafe{callback_twice(20,callback_pure,null)};
    var i32 count=0;
    let i32 counted=unsafe{callback_twice(20,callback_counted,ref mut count as ptr mut void)};
    let i32 nested=unsafe{callback_twice(20,callback_counted,callback_context())};
    return pure==42 && counted==42 && count==2 && nested==42 && unsafe{callback_observe()}==42;
}
`

const callbackCompanionC = `
typedef int32_t (*fairyc_fixture_callback_v0)(int32_t, void *);
static int32_t fairyc_fixture_callback_count_v0=0;
static int32_t fairyc_fixture_callback_calls_v0=0;
int32_t fairyc_fixture_callback_twice_v0(int32_t value, fairyc_fixture_callback_v0 callback, void *data){
    if(!callback) return 0;
    /* Direct same-thread calls complete before returning to Fairy. */
    fairyc_fixture_callback_calls_v0++;
    int32_t first=callback(value,data);
    fairyc_fixture_callback_calls_v0++;
    int32_t second=callback(value,data);
    return first+second;
}
int32_t fairyc_fixture_callback_bump_v0(int32_t value, void *data){
    if(!data) return 0;
    (*(int32_t *)data)++;
    return value+1;
}
void *fairyc_fixture_callback_context_v0(void){fairyc_fixture_callback_count_v0=0;return &fairyc_fixture_callback_count_v0;}
int32_t fairyc_fixture_callback_observe_v0(void){
    return fairyc_fixture_callback_count_v0==2 && fairyc_fixture_callback_calls_v0==6 ? 42 : 0;
}
`
