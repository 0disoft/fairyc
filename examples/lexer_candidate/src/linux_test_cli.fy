module linux_test_cli;
import project_compiler_core as compiler;
import project_compiler_input as input;
import project_test_options as options;
import module_project_complete_c as complete_c;
import module_project_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_emit as c;
import project_path as portable;
import project_source_file as source_file;
import native_packet_file as packet;
import native_test_result as result;
import native_run_control as control;
import linux_build_cli as building;
import linux_build_scratch as scratch;
import linux_host_paths as host;
import linux_capture as backend;
import linux_run_capture as capture;
import linux_test_capture as execution;
import std::bytes as bytes;
import std::vec as v;
import std::text as text;
import std::fs as fs;
import std::io as io;

@effects(console) fn i32 failure(ref str text,i32 code){return if(io::write_stderr(text).is_ok){code}else{4};}
fn Result<bytes::Buffer,u32> joined(ref u8[] root,ref u8[] leaf){var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,root).is_ok || !bytes::append_byte(ref mut out,47).is_ok || !bytes::append(ref mut out,leaf).is_ok || bytes::len(ref out)>4095){return .err(4);}return .ok(out);}
fn usize test_entry(ref compiler::Complete complete,ref options::TestIdentity test){let p=identified::ir_program(compiler::checked_program(complete));var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.kind==3 && f.module_index==options::test_module(test) && f.declaration==options::test_declaration(test)+1){return i;}i+=1;}return 0;}
@effects(console) fn i32 input_error(ref u8[] path,u32 kind){var bytes::Buffer message=bytes::empty();if(!options::test_input_diagnostic(path,kind,ref mut message)){return 4;}return if(io::write_stderr_bytes(bytes::view(ref message)).is_ok){7}else{4};}
@effects(filesystem) fn bool replay(ref u8[] path,ref u8[] input){
    let valid=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return false;}};
    let name=switch(text::cstring_from_text(valid)){case .ok(p):{p}case .err(_):{return false;}};
    var fs::File file=switch(fs::create_new(text::cstring_as_cstr(ref name))){case .ok(p):{p}case .err(_):{return false;}};
    let wrote=switch(fs::write_all(ref mut file,input)){case .ok(n):{n==input.len}case .err(_):{false}};
    let closed=fs::close(file).is_ok;return wrote && closed;
}
pub fn u32 timeout(ref options::TestOptions selected){var u32 time=0;var usize elapsed=0;while(elapsed<options::timeout_ms(selected)){time+=1000;elapsed+=1000;}return time;}
@effects(filesystem,console,environment,clock)
fn Result<capture::Captured,i32> test_execute(ref c::Emission bundle,ref options::TestIdentity test,ref options::TestOptions selected,ref u8[] root,ref u8[] input){
    let temp=switch(host::temporary()){case .ok(p):{p}case .err(_):{return .err(failure("fairyc: build failed: temporary directory query failed\n",6));}};
    let directory=switch(scratch::create(bytes::view(ref temp))){case .ok(p):{p}case .err(_):{return .err(failure("fairyc: build failed: private test staging creation failed\n",6));}};
    let packet_path=switch(joined(scratch::path(ref directory),b"packet.bin")){case .ok(p):{p}case .err(_):{return .err(4);}};
    let program=switch(joined(scratch::path(ref directory),b"program")){case .ok(p):{p}case .err(_):{return .err(4);}};
    let input_path=switch(joined(scratch::path(ref directory),b"input.bin")){case .ok(p):{p}case .err(_):{return .err(4);}};
    if(!packet::write_emission(bundle,bytes::view(ref packet_path)).is_ok){return .err(failure("fairyc: build failed: native test packet preparation failed\n",6));}
    let image=switch(host::executable()){case .ok(p):{p}case .err(_):{return .err(failure("fairyc: build failed: executable path query failed\n",6));}};
    let driver=switch(building::companion(bytes::view(ref image))){case .ok(p):{p}case .err(_):{return .err(failure("fairyc: build failed: native test companion path query failed\n",6));}};
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();
    if(!options::test_add(ref mut args,b"--packet-input") || !options::test_add(ref mut args,bytes::view(ref packet_path)) || !options::test_add(ref mut args,b"--stage-root") || !options::test_add(ref mut args,scratch::path(ref directory)) || !options::test_add(ref mut args,b"--output-root") || !options::test_add(ref mut args,scratch::path(ref directory)) || !options::test_add(ref mut args,b"--output") || !options::test_add(ref mut args,b"program") || !options::test_add(ref mut args,b"--run-trap-control") || options::flags(selected).release && !options::test_add(ref mut args,b"--release")){return .err(4);}
    let built=switch(backend::run_for(bytes::view(ref driver),ref args,root,600000,65536,65536)){case .ok(p):{p}case .err(_):{return .err(failure("fairyc: build failed: native test companion execution failed\n",6));}};
    if(backend::kind(ref built)!=1 || backend::value(ref built)!=0){if(!io::write_stderr("fairyc: build failed: ").is_ok || !io::write_stderr_bytes(backend::stderr(ref built)).is_ok || backend::stderr(ref built).len==0 && !io::write_stderr("native test backend failed or timed out\n").is_ok){return .err(4);}return .err(6);}
    let has_file=options::stdin_file(selected).len!=0;
    if(has_file && !replay(bytes::view(ref input_path),input)){return .err(failure("fairyc: build failed: native test input preparation failed\n",6));}
    let captured=switch(execution::run_for(bytes::view(ref program),options::arguments(selected),scratch::path(ref directory),timeout(selected),if(options::flags(selected).stdin){0}else if(has_file){2}else{1},bytes::view(ref input_path))){case .ok(p):{p}case .err(_):{return .err(failure("fairyc: build failed: native test execution or trap control failed\n",6));}};
    if(!scratch::close(directory).is_ok){return .err(failure("fairyc: build failed: private test staging cleanup failed\n",6));}
    if(capture::kind(ref captured)==3){var bytes::Buffer message=bytes::empty();if(!options::test_timeout_diagnostic_for(options::test_name(test),options::timeout_ms(selected),b"Linux GNU",ref mut message)){return .err(4);}return .err(if(io::write_stderr_bytes(bytes::view(ref message)).is_ok){6}else{4});}
    if(capture::kind(ref captured)==4){return .err(failure("fairyc: build failed: native test output exceeds 65536 bytes\n",6));}
    if(capture::kind(ref captured)!=1){return .err(failure("fairyc: build failed: native test terminated by signal\n",6));}
    return .ok(captured);
}
@effects(filesystem,console,environment,clock)
pub fn i32 tests(compiler::Complete initial,ref input::Input projected,ref options::TestOptions selected,ref v::Vec<options::TestIdentity> tests,ref v::Vec<usize> chosen,ref u8[] root,u32 lock_code){
    var bytes::Buffer input=bytes::empty();
    if(options::stdin_file(selected).len!=0){let path=options::stdin_file(selected);if(!portable::source_relative(path)){return input_error(path,2);}var usize work=0;input=switch(source_file::read(root,path,65536,ref mut work)){case .ok(p):{p}case .err(e):{return input_error(path,if(e.code==3 || e.code==4){3}else if(e.code==8 || e.code==9){4}else{1});}};}
    var v::Vec<result::Captured> results=v::empty<result::Captured>();
    if(v::len<usize>(chosen)==0){return options::test_native_report(selected,tests,chosen,projected,ref results,lock_code,b"x86_64-linux-gnu");}
    let session=switch(complete_c::prepare_tests(initial)){case .ok(p):{p}case .err(_):{return failure("fairyc: unsupported subset capability: native test C emission\n",3);}};
    var usize i=0;while(i<v::len<usize>(chosen)){
        let test=v::at<options::TestIdentity>(tests,*v::at<usize>(chosen,i));let entry=test_entry(complete_c::test_program(ref session),test);
        if(entry==0){return failure("fairyc: internal native test entry invariant failed\n",4);}
        let bundle=switch(complete_c::emit_prepared_test(ref session,entry,128)){case .ok(p):{p}case .err(_):{return failure("fairyc: unsupported subset capability: native test C emission\n",3);}};
        let captured=switch(test_execute(ref bundle,test,selected,root,bytes::view(ref input))){case .ok(p):{p}case .err(code):{return code;}};
        if(!options::flags(selected).capture && (capture::stdout(ref captured).len!=0 || capture::stderr(ref captured).len!=0)){var bytes::Buffer message=bytes::empty();if(!options::test_append(ref mut message,b"fairyc: requested workspace is outside the native test subset: ") || !options::test_append(ref mut message,options::test_name(test)) || !options::test_append(ref mut message,b" produced program output\n")){return 4;}return if(io::write_stderr_bytes(bytes::view(ref message)).is_ok){2}else{4};}
        let outcome=switch(control::classify(capture::value(ref captured),capture::receipt(ref captured),capture::pid(ref captured))){case .ok(p):{p}case .err(_):{return failure("fairyc: build failed: invalid native test trap control\n",6);}};
        if(outcome.trap==0 && outcome.status!=0){return failure("fairyc: build failed: native test exited without valid trap control\n",6);}
        let retained=switch(result::copy(outcome.trap,capture::stdout(ref captured),capture::stderr(ref captured))){case .ok(p):{p}case .err(_):{return 4;}};
        if(!v::push<result::Captured>(ref mut results,retained).is_ok){return 4;}i+=1;
    }
    return options::test_native_report(selected,tests,chosen,projected,ref results,lock_code,b"x86_64-linux-gnu");
}
