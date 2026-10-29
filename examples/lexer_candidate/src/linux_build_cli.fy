module linux_build_cli;
import project_emit as emit;
import project_emit_text as diagnostics;
import project_lock_policy as locks;
import native_cli_options as options;
import native_packet_file as packet;
import linux_host_paths as host;
import linux_build_scratch as scratch;
import linux_capture as capture;
import linux_run_cli as running;
import project_compiler_cached as cached;
import project_compiler_cache_path as cache_path;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) fn i32 failure(ref str text,i32 code){return if(io::write_stderr(text).is_ok){code}else{4};}
fn Result<bytes::Buffer,u32> join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();if(root.len==0 || !bytes::append(ref mut result,root).is_ok){return .err(1);}if(root[root.len-1]!=47 && !bytes::append_byte(ref mut result,47).is_ok){return .err(1);}if(!bytes::append(ref mut result,leaf).is_ok || bytes::len(ref result)>4095){return .err(1);}return .ok(result);}
fn bool argument(ref mut v::Vec<bytes::Buffer> output,ref u8[] value){let copied=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{return false;}};return v::push<bytes::Buffer>(output,copied).is_ok;}
pub fn Result<bytes::Buffer,u32> companion(ref u8[] image){var usize end=0;var usize i=0;while(i<image.len){if(image[i]==47){end=i+1;}i+=1;}if(end==0){return .err(1);}return join(image[..end],b"fairyc-candidate-linux-build");}
pub fn Result<bytes::Buffer,u32> output(ref u8[] name){return join(b"build",name);}
pub fn Result<bytes::Buffer,u32> output_for(bool execute,ref u8[] name){if(!execute){return output(name);}return switch(bytes::copy(b"program")){case .ok(p):{.ok(p)}case .err(_):{.err(1)}};}
@effects(console) fn i32 bridge_failure(ref capture::Captured result){
    if(!io::write_stderr("fairyc: build failed: ").is_ok){return 4;}
    let data=capture::stderr(result);let prefix=b"candidate build failed: ";let detail=if(data.len>=prefix.len && bytes::equal(data[..prefix.len],prefix)){data[prefix.len..]}else{data};
    if(detail.len==0){return failure("Linux backend failed or timed out\n",6);}return if(io::write_stderr_bytes(detail).is_ok){6}else{4};
}
@effects(filesystem,console,environment,clock) pub fn i32 run(bool execute){
    let selected=switch(options::from_process(execute)){case .ok(p):{p}case .err(e):{var bytes::Buffer message=bytes::empty();if(!options::diagnostic(execute,e,ref mut message)){return 4;}return if(io::write_stderr_bytes(bytes::view(ref message)).is_ok){options::exit_code(e)}else{4};}};
    var observed=cached::empty();
    let evaluated=if(options::cache_root(ref selected).len==0){emit::evaluate_run(b".",options::locked(ref selected),execute)}else{emit::evaluate_cached_run(b".",options::locked(ref selected),options::cache_root(ref selected),cache_path::command_limits(),ref mut observed,execute)};
    if(options::cache_status(ref selected)){let ref str text=if(observed.path==2){"cache: hit\n"}else if(observed.path==3){"cache: mixed\n"}else if(observed.path==4){"cache: units hit\n"}else if(observed.path==1 && observed.stored){"cache: cold stored\n"}else if(observed.path==1){"cache: cold uncached\n"}else{"cache: skipped\n"};if(!io::write_stderr(text).is_ok){discard evaluated;return 4;}}
    let plan=switch(evaluated){case .ok(p):{p}case .err(e):{var bytes::Buffer message=bytes::empty();if(!diagnostics::error(ref e,ref mut message)){return 4;}return if(io::write_stderr_bytes(bytes::view(ref message)).is_ok){diagnostics::exit_code(ref e)}else{4};}};
    let root=switch(host::current()){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: current directory query failed\n",6);}};
    let image=switch(host::executable()){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: executable path query failed\n",6);}};
    let driver=switch(companion(bytes::view(ref image))){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: companion path query failed\n",6);}};
    let temporary=switch(host::temporary()){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: temporary path query failed\n",6);}};
    let directory=switch(scratch::create(bytes::view(ref temporary))){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: private staging creation failed\n",6);}};
    let input=switch(join(scratch::path(ref directory),b"packet.bin")){case .ok(p):{p}case .err(_):{return 4;}};
    if(!packet::write(emit::output_bundle(ref plan),bytes::view(ref input)).is_ok){return failure("fairyc: build failed: packet write failed\n",6);}
    let target=switch(output_for(execute,emit::package_name(ref plan))){case .ok(p):{p}case .err(_):{return 4;}};
    var v::Vec<bytes::Buffer> arguments=v::empty<bytes::Buffer>();
    if(!argument(ref mut arguments,b"--packet-input") || !argument(ref mut arguments,bytes::view(ref input)) || !argument(ref mut arguments,b"--stage-root") || !argument(ref mut arguments,scratch::path(ref directory)) || !argument(ref mut arguments,b"--output-root") || !argument(ref mut arguments,if(execute){scratch::path(ref directory)}else{bytes::view(ref root)}) || !argument(ref mut arguments,b"--output") || !argument(ref mut arguments,bytes::view(ref target))){return 4;}
    if(execute && !argument(ref mut arguments,b"--run-trap-control")){return 4;}
    if(options::release(ref selected) && !argument(ref mut arguments,b"--release")){return 4;}
    let result=switch(capture::run_for(bytes::view(ref driver),ref arguments,bytes::view(ref root),600000,65536,65536)){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: Linux backend companion execution failed\n",6);}};
    if(capture::kind(ref result)!=1 || capture::value(ref result)!=0){return bridge_failure(ref result);}
    if(execute){
        let program=switch(join(scratch::path(ref directory),b"program")){case .ok(p):{p}case .err(_):{return 4;}};
        return running::executed(directory,bytes::view(ref program),options::arguments(ref selected),ref plan,bytes::view(ref root));
    }
    if(!scratch::close(directory).is_ok){return failure("fairyc: build failed: private staging cleanup failed\n",6);}
    let state=locks::state_code(emit::lock_state(ref plan));
    if(!io::write("built: ").is_ok || !io::write_bytes(bytes::view(ref target)).is_ok || !io::write(" (x86_64-linux-gnu, ").is_ok || !io::write(if(options::release(ref selected)){"release"}else{"development"}).is_ok || !io::write(", lock ").is_ok || !io::write(if(state==1){"unchanged"}else if(state==2){"updated"}else{"verified"}).is_ok || !io::write(")\n").is_ok){return 4;}return 0;
}
