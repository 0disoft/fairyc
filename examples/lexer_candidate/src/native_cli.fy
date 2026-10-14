module native_cli;
import native_cli_options as options;
import project_emit as emit;
import project_emit_text as diagnostics;
import project_lock_policy as locks;
import native_packet_file as packet;
import native_scratch as scratch;
import windows_host_paths as paths;
import windows_run_capture as capture;
import windows_run_control as control;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import project_compiler_cached as cached;
import project_compiler_cache_path as cache_path;

@effects(console) fn i32 failure(ref str message,i32 code){return if(io::write_stderr(message).is_ok){code}else{4};}
@effects(console) fn bool stderr(ref u8[] data){return switch(io::write_stderr_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}};}
@effects(console) fn bool stdout(ref u8[] data){return switch(io::write_bytes(data)){case .ok(n):{n==data.len}case .err(_):{false}};}
fn Result<bytes::Buffer,u32> joined(ref u8[] root,ref u8[] leaf){var bytes::Buffer out=bytes::empty();if(root.len==0 || !bytes::append(ref mut out,root).is_ok){return .err(1);}if(root[root.len-1]!=47 && root[root.len-1]!=92 && !bytes::append_byte(ref mut out,47).is_ok){return .err(1);}if(!bytes::append(ref mut out,leaf).is_ok){return .err(1);}return .ok(out);}
fn bool argument(ref mut v::Vec<bytes::Buffer> out,ref u8[] value){let copied=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{return false;}};return v::push<bytes::Buffer>(out,copied).is_ok;}
fn Result<bytes::Buffer,u32> build_path(ref emit::Plan plan){var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,b"build/").is_ok || !bytes::append(ref mut out,emit::package_name(plan)).is_ok || !bytes::append(ref mut out,b".exe").is_ok){return .err(1);}return .ok(out);}
@effects(console) fn i32 bridge_failure(ref u8[] data){var bytes::Buffer out=bytes::empty();let prefix=b"candidate build failed: ";let detail=if(data.len>=prefix.len && bytes::equal(data[..prefix.len],prefix)){data[prefix.len..]}else{data};if(!bytes::append(ref mut out,b"fairyc: build failed: ").is_ok || !bytes::append(ref mut out,detail).is_ok || detail.len==0 && !bytes::append(ref mut out,b"native driver failed\n").is_ok){return 4;}return if(stderr(bytes::view(ref out))){6}else{4};}
@effects(filesystem,console) fn i32 built(scratch::Directory directory,ref emit::Plan plan,bool release){switch(scratch::close(directory)){case .ok(_):{}case .err(_):{return failure("fairyc: build failed: temporary directory cleanup failed\n",6);}};let path=switch(build_path(plan)){case .ok(p):{p}case .err(_):{return failure("fairyc: internal output processing failed\n",4);}};var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,b"built: ").is_ok || !bytes::append(ref mut out,bytes::view(ref path)).is_ok || !bytes::append(ref mut out,b" (x86_64-windows-msvc, ").is_ok || !bytes::append(ref mut out,if(release){b"release"}else{b"development"}).is_ok || !bytes::append(ref mut out,b", lock ").is_ok){return 4;}let label=locks::state_code(emit::lock_state(plan));if(!bytes::append(ref mut out,if(label==1){b"unchanged"}else if(label==2){b"updated"}else{b"verified"}).is_ok || !bytes::append(ref mut out,b")\n").is_ok){return 4;}return if(stdout(bytes::view(ref out))){0}else{failure("fairyc: output write failed\n",4)};}
@effects(filesystem,console,environment,clock) fn i32 executed(scratch::Directory directory,ref u8[] program,ref v::Vec<bytes::Buffer> arguments){let outcome=switch(capture::run(program,arguments,scratch::path(ref directory),60000)){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: program execution failed\n",6);}};switch(scratch::close(directory)){case .ok(_):{}case .err(_):{return failure("fairyc: build failed: temporary directory cleanup failed\n",6);}};if(!stdout(capture::stdout(ref outcome))){return failure("fairyc: output write failed\n",4);}if(!stderr(capture::stderr(ref outcome))){return 4;}let trap=capture::trap(ref outcome);if(trap!=0){var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,b"trap: ").is_ok || !bytes::append(ref mut out,control::category(trap)).is_ok || !bytes::append_byte(ref mut out,10).is_ok){return 4;}if(!stderr(bytes::view(ref out))){return 4;}return 5;}return capture::exit_bits(capture::status(ref outcome));}
@effects(filesystem,console,environment,clock) pub fn i32 run(bool execute){
    let selected=switch(options::from_process(execute)){case .ok(p):{p}case .err(e):{var bytes::Buffer out=bytes::empty();if(!options::diagnostic(execute,e,ref mut out)){return 4;}return if(stderr(bytes::view(ref out))){options::exit_code(e)}else{4};}};
    let root=switch(paths::current()){case .ok(p):{p}case .err(e):{return failure("fairyc: build failed: current directory query failed\n",if(e.kind==2){4}else{6});}};
    var observed=cached::empty();let result=if(options::cache_root(ref selected).len==0){emit::evaluate(bytes::view(ref root),options::locked(ref selected))}else{emit::evaluate_cached(bytes::view(ref root),options::locked(ref selected),options::cache_root(ref selected),cache_path::command_limits(),ref mut observed)};
    if(options::cache_status(ref selected)){let ref str text=if(observed.path==2){"cache: hit\n"}else if(observed.path==3){"cache: mixed\n"}else if(observed.path==4){"cache: units hit\n"}else if(observed.path==1 && observed.stored){"cache: cold stored\n"}else if(observed.path==1){"cache: cold uncached\n"}else{"cache: skipped\n"};if(!io::write_stderr(text).is_ok){discard result;return 4;}}
    let plan=switch(result){case .ok(p):{p}case .err(e):{var bytes::Buffer out=bytes::empty();if(!diagnostics::error(ref e,ref mut out)){return 4;}return if(stderr(bytes::view(ref out))){diagnostics::exit_code(ref e)}else{4};}};
    let temp=switch(paths::temporary()){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: temporary directory query failed\n",6);}};let directory=switch(scratch::create(bytes::view(ref temp))){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: temporary directory creation failed\n",6);}};
    let input=switch(joined(scratch::path(ref directory),b"packet.bin")){case .ok(p):{p}case .err(_):{return 4;}};switch(packet::write(emit::output_bundle(ref plan),bytes::view(ref input))){case .ok(_):{}case .err(e):{return failure("fairyc: build failed: native packet preparation failed\n",if(e.stage<=2){4}else{6});}};
    let driver=switch(paths::driver()){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: native driver path query failed\n",6);}};
    let output=if(execute){switch(joined(scratch::path(ref directory),b"program.exe")){case .ok(p):{p}case .err(_):{return 4;}}}else{switch(build_path(ref plan)){case .ok(p):{p}case .err(_):{return 4;}}};
    var v::Vec<bytes::Buffer> arguments=v::empty<bytes::Buffer>();if(!argument(ref mut arguments,b"--packet-input") || !argument(ref mut arguments,bytes::view(ref input)) || !argument(ref mut arguments,b"--output") || !argument(ref mut arguments,bytes::view(ref output)) || !argument(ref mut arguments,b"--compile-timeout") || !argument(ref mut arguments,b"1m") || options::release(ref selected) && !argument(ref mut arguments,b"--release") || execute && !argument(ref mut arguments,b"--run-trap-control")){return 4;}
    if(!execute && (!argument(ref mut arguments,b"--output-root") || !argument(ref mut arguments,bytes::view(ref root)))){return 4;}
    let built_image=switch(capture::run(bytes::view(ref driver),ref arguments,bytes::view(ref root),600000)){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: native driver execution failed\n",6);}};if(capture::status(ref built_image)!=0 || capture::trap(ref built_image)!=0){return bridge_failure(capture::stderr(ref built_image));}
    return if(execute){executed(directory,bytes::view(ref output),options::arguments(ref selected))}else{built(directory,ref plan,options::release(ref selected))};
}
