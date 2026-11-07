module windows_cancel;
import windows_command as command;
import std::bytes as bytes;

@effects(filesystem) @link_name("fairyc_private_windows_cancel_create_v1") unsafe extern C fn ptr mut void create_event(ptr u8 name);
@effects(filesystem) @link_name("fairyc_private_windows_cancel_open_v1") unsafe extern C fn ptr mut void open_event(ptr u8 name);
@effects(filesystem) @link_name("fairyc_private_windows_cancel_signal_v1") unsafe extern C fn i32 set_event(ptr mut void handle);
@effects(filesystem) @link_name("fairyc_private_windows_cancel_local_v1") unsafe extern C fn ptr mut void local_event();
@effects(filesystem) @link_name("fairyc_private_windows_cancel_console_bind_v1") unsafe extern C fn ptr mut void bind_console(ptr mut void event);
@effects(filesystem) @link_name("fairyc_private_windows_cancel_console_close_v1") unsafe extern C fn i32 unbind_console(ptr mut void registration);
@effects(filesystem) @link_name("fairyc_private_windows_wait_v1") unsafe extern C fn u32 wait_event(ptr mut void handle,u32 timeout);
@effects(filesystem) @link_name("fairyc_private_windows_close_v1") unsafe extern C fn i32 close_handle(ptr mut void handle);
@link_name("fairyc_private_windows_error_v1") unsafe extern C fn u32 last_error();

// The module alone constructs or inspects this owner. A null owner performs no
// native work. Named events are manual-reset, initially clear, and never reset.
@effects(filesystem) pub resource Token{
    ptr mut void handle;
    cleanup{if(this.handle!=null){unsafe{discard close_handle(this.handle);};}}
}
// 1 invalid input/none signal, 2 encoding allocation, 3 create (including 183
// collision), 4 open, 5 signal, 6 poll, 7 local create, 8 bind, 9 unbind.
// Native causes survive cleanup unchanged.
pub struct Problem{u32 kind;u32 os_code;}
fn Problem fail(u32 kind,u32 os_code){return Problem{.kind,.os_code};}
pub fn Token none(){return Token{.handle=null};}
// Independent unnamed event; there is no discoverable name or inherited handle.
@effects(filesystem) pub fn Result<Token,Problem> local(){
    let handle=unsafe{local_event()};if(handle==null){return .err(fail(7,unsafe{last_error()}));}return .ok(Token{.handle});
}
// Native binding owns a duplicate, not a borrow of Token's handle. One binding
// per process consumes Ctrl-C/Break; other console controls retain OS handling.
// No-console (OS error 6) is an explicit inactive owner, not a failed command.
@effects(filesystem) pub resource Console{
    ptr mut void registration;
    bool bound;
    cleanup{if(this.bound){unsafe{discard unbind_console(this.registration);};}}
}
pub fn bool active(ref Console binding){return binding.bound;}
@effects(filesystem) pub fn Result<Console,Problem> console(ref Token token){
    if(token.handle==null){return .err(fail(1,0));}
    let registration=unsafe{bind_console(token.handle)};
    if(registration!=null){return .ok(Console{.registration,.bound=true});}
    let code=unsafe{last_error()};return if(code==6){.ok(Console{.registration=null,.bound=false})}else{.err(fail(8,code))};
}
// Consume/clear even on native unregister failure. Native cleanup never leaves
// a callback pointing at the consumed registration or its released handle.
@effects(filesystem) pub fn Result<bool,Problem> close_console(Console input){
    var binding=input;if(!binding.bound){return .ok(true);}binding.bound=false;
    return if(unsafe{unbind_console(binding.registration)}!=0){.ok(true)}else{.err(fail(9,unsafe{last_error()}))};
}

fn bool valid(ref u8[] name){
    let prefix=b"Local\\fairyc-cancel-";
    if(name.len<=prefix.len || name.len>prefix.len+128 || !bytes::equal(name[..prefix.len],prefix)){return false;}
    var usize i=prefix.len;
    while(i<name.len){let c=name[i];if(!(c>=65 && c<=90 || c>=97 && c<=122 || c>=48 && c<=57 || c==46 || c==95 || c==45)){return false;}i+=1;}
    return true;
}
fn Result<bytes::Buffer,Problem> wide(ref u8[] name){
    if(!valid(name)){return .err(fail(1,0));}
    return switch(command::path(name)){case .ok(p):{.ok(p)}case .err(_):{.err(fail(2,0))}};
}
@effects(filesystem) pub fn Result<Token,Problem> create(ref u8[] name){
    let encoded=switch(wide(name)){case .ok(p):{p}case .err(e):{return .err(e);}};
    // The private exclusive-create adapter closes only a colliding reference
    // and restores ERROR_ALREADY_EXISTS before reporting failure.
    let handle=unsafe{create_event(bytes::view(ref encoded) as ptr u8)};
    if(handle==null){return .err(fail(3,unsafe{last_error()}));}
    return .ok(Token{.handle});
}
@effects(filesystem) pub fn Result<Token,Problem> open(ref u8[] name){
    let encoded=switch(wide(name)){case .ok(p):{p}case .err(e):{return .err(e);}};
    // SYNCHRONIZE | EVENT_MODIFY_STATE; inheritance is explicitly disabled.
    let handle=unsafe{open_event(bytes::view(ref encoded) as ptr u8)};
    if(handle==null){return .err(fail(4,unsafe{last_error()}));}
    return .ok(Token{.handle});
}
@effects(filesystem) pub fn Result<bool,Problem> signal(ref Token token){
    if(token.handle==null){return .err(fail(1,0));}
    if(unsafe{set_event(token.handle)}==0){return .err(fail(5,unsafe{last_error()}));}
    return .ok(true);
}
@effects(filesystem) pub fn Result<bool,Problem> requested(ref Token token){
    if(token.handle==null){return .ok(false);}
    let status=unsafe{wait_event(token.handle,0)};
    if(status==0){return .ok(true);}
    if(status==258){return .ok(false);}
    return .err(fail(6,unsafe{last_error()}));
}
