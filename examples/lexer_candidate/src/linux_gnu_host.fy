module linux_gnu_host;
import linux_command as command;
import std::bytes as bytes;
import std::vec as v;

@effects(filesystem) @link_name("fairyc_private_linux_gnu_stage_v1") unsafe extern C fn u32 stage_raw(ptr u8 path,usize size,ptr mut u32 output);
@effects(filesystem) @link_name("fairyc_private_linux_root_file_close_v1") unsafe extern C fn u32 close_raw(u32 file);
@effects(filesystem) @link_name("fairyc_private_linux_gnu_write_v1") unsafe extern C fn u32 write_raw(u32 root,ptr u8 name,usize size,ptr u8 data,usize length);
@effects(filesystem) @link_name("fairyc_private_linux_gnu_read_v1") unsafe extern C fn u32 read_raw(u32 root,ptr u8 name,usize size,ptr mut u8 data,usize capacity,ptr mut usize length);
@effects(filesystem,environment) @link_name("fairyc_private_linux_gnu_find_v1") unsafe extern C fn u32 find_raw(ptr u8 name,usize size,ptr mut u8 output,usize capacity,ptr mut usize length);
@effects(filesystem,environment,console,clock) @link_name("fairyc_private_linux_gnu_run_v1") unsafe extern C fn u32 run_raw(ptr u8 argv,usize size,usize count,ptr u8 cwd,usize cwd_size,u32 timeout_ms,ptr mut u8 output,ptr mut usize output_length,ptr mut u8 error,ptr mut usize error_length,ptr mut u32 kind,ptr mut u32 value);
@effects(filesystem) @link_name("fairyc_private_linux_gnu_publish_v1") unsafe extern C fn u32 publish_raw(ptr u8 root,usize root_size,ptr u8 path,usize size,ptr u8 data,usize length);
@effects(filesystem) pub resource Stage{u32 raw;cleanup{unsafe{discard close_raw(this.raw);};}}
pub resource Captured{u32 kind;u32 value;bytes::Buffer output;bytes::Buffer error;cleanup{}}
pub fn ref u8[] stdout(ref Captured p){return bytes::view(ref p.output);}
pub fn ref u8[] stderr(ref Captured p){return bytes::view(ref p.error);}
pub fn u32 kind(ref Captured p){return p.kind;}
pub fn u32 value(ref Captured p){return p.value;}
fn Result<bytes::Buffer,u32> path(ref u8[] value){return switch(command::path(value)){case .ok(p):{.ok(p)}case .err(_):{.err(1)}};}
@effects(filesystem) pub fn Result<Stage,u32> open(ref u8[] location){
    let encoded=path(location)?;var u32 raw=0;let code=unsafe{stage_raw(bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref mut raw as ptr mut u32)};
    var stage=Stage{.raw};if(code!=0){return .err(code);}if(raw==0 || raw>2147483648){return .err(1);}return .ok(stage);
}
@effects(filesystem) pub fn Result<bool,u32> close(Stage input){var stage=input;let raw=stage.raw;stage.raw=0;let code=unsafe{close_raw(raw)};return if(code==0){.ok(true)}else{.err(code)};}
@effects(filesystem) pub fn Result<bool,u32> write(ref Stage stage,ref u8[] leaf,ref u8[] data){
    let name=path(leaf)?;let code=unsafe{write_raw(stage.raw,bytes::view(ref name) as ptr u8,bytes::len(ref name),data as ptr u8,data.len)};return if(code==0){.ok(true)}else{.err(code)};
}
@effects(filesystem) pub fn Result<bytes::Buffer,u32> read(ref Stage stage,ref u8[] leaf){
    let name=path(leaf)?;var usize size=0;var bytes::Buffer data=bytes::empty();
    let code=unsafe{read_raw(stage.raw,bytes::view(ref name) as ptr u8,bytes::len(ref name),bytes::view_mut(ref mut data) as ptr mut u8,0,ref mut size as ptr mut usize)};
    if(code!=0){return .err(code);}if(size==0 || size>67108864 || !bytes::resize_zeroed(ref mut data,size).is_ok){return .err(1);}
    var usize copied=0;let read=unsafe{read_raw(stage.raw,bytes::view(ref name) as ptr u8,bytes::len(ref name),bytes::view_mut(ref mut data) as ptr mut u8,size,ref mut copied as ptr mut usize)};
    if(read!=0){return .err(read);}if(copied!=size){return .err(1);}return .ok(data);
}
@effects(filesystem,environment) pub fn Result<bytes::Buffer,u32> find(ref u8[] spelling){
    let name=path(spelling)?;var bytes::Buffer data=bytes::empty();if(!bytes::resize_zeroed(ref mut data,4096).is_ok){return .err(1);}var usize size=0;
    let code=unsafe{find_raw(bytes::view(ref name) as ptr u8,bytes::len(ref name),bytes::view_mut(ref mut data) as ptr mut u8,4096,ref mut size as ptr mut usize)};
    if(code!=0){return .err(code);}if(size==0 || size>4095 || bytes::view(ref data)[size]!=0 || !bytes::truncate(ref mut data,size).is_ok){return .err(1);}return .ok(data);
}
@effects(filesystem,environment,console,clock) pub fn Result<Captured,u32> run(ref u8[] program,ref v::Vec<bytes::Buffer> args,ref u8[] directory,u32 timeout_ms){
    let argv=switch(command::build(program,args)){case .ok(p):{p}case .err(_):{return .err(1);}};let cwd=path(directory)?;
    var bytes::Buffer output=bytes::empty();var bytes::Buffer error=bytes::empty();if(!bytes::resize_zeroed(ref mut output,65536).is_ok || !bytes::resize_zeroed(ref mut error,65536).is_ok){return .err(1);}
    var usize out_length=0;var usize err_length=0;var u32 kind=0;var u32 value=0;
    let code=unsafe{run_raw(command::view(ref argv) as ptr u8,command::view(ref argv).len,command::count(ref argv),bytes::view(ref cwd) as ptr u8,bytes::len(ref cwd),timeout_ms,bytes::view_mut(ref mut output) as ptr mut u8,ref mut out_length as ptr mut usize,bytes::view_mut(ref mut error) as ptr mut u8,ref mut err_length as ptr mut usize,ref mut kind as ptr mut u32,ref mut value as ptr mut u32)};
    if(code!=0){return .err(code);}if(out_length>65536 || err_length>65536 || !(kind==1 && value<=255 || kind==2 && value>0 && value<=64 || kind==3 && value==0 || kind==4 && value==0)){return .err(1);}
    if(!bytes::truncate(ref mut output,out_length).is_ok || !bytes::truncate(ref mut error,err_length).is_ok){return .err(1);}return .ok(Captured{.kind,.value,.output,.error});
}
@effects(filesystem) pub fn Result<bool,u32> publish(ref u8[] root,ref u8[] relative,ref u8[] data){
    let parent=path(root)?;let leaf=path(relative)?;let code=unsafe{publish_raw(bytes::view(ref parent) as ptr u8,bytes::len(ref parent),bytes::view(ref leaf) as ptr u8,bytes::len(ref leaf),data as ptr u8,data.len)};return if(code==0){.ok(true)}else{.err(code)};
}
