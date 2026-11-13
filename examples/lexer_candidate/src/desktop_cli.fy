module desktop_cli;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import std::text as text;
import windows_host_paths as paths;
import windows_run_capture as capture;
import native_cli as native;
import native_cli_options as arguments;
import desktop_package_options as options;
import public_cli_data as public;
import project_json_string as quoted;

@effects(console) fn i32 failed(){discard io::write_stderr("fairyc: native desktop helper execution failed\n");return 6;}
fn bool argument(ref mut v::Vec<bytes::Buffer> out,ref u8[] value){let copied=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{return false;}};return v::push<bytes::Buffer>(out,copied).is_ok;}
@effects(filesystem,console,environment,clock) fn i32 invoke(ref u8[] operation,ref u8[] first,ref u8[] second,bool json){
    let driver=switch(paths::driver()){case .ok(p):{p}case .err(_):{return failed();}};
    let root=switch(paths::current()){case .ok(p):{p}case .err(_):{return failed();}};
    var v::Vec<bytes::Buffer> argv=v::empty<bytes::Buffer>();
    if(!argument(ref mut argv,b"--desktop") || !argument(ref mut argv,operation) || first.len!=0 && !argument(ref mut argv,first) || second.len!=0 && !argument(ref mut argv,second) || json && !argument(ref mut argv,b"--json")){return 4;}
    let result=switch(capture::run(bytes::view(ref driver),ref argv,bytes::view(ref root),600000)){case .ok(p):{p}case .err(_):{return failed();}};
    if(capture::trap(ref result)!=0){return failed();}
    let out=capture::stdout(ref result);let diagnostic=capture::stderr(ref result);
    if(!switch(io::write_bytes(out)){case .ok(n):{n==out.len}case .err(_):{false}} || !switch(io::write_stderr_bytes(diagnostic)){case .ok(n):{n==diagnostic.len}case .err(_):{false}}){return 4;}
    let status=capture::status(ref result);if(status!=0 && status!=2 && status!=4 && status!=6 && status!=7){return failed();}
    if(status==2 && (!io::write_stderr("\n").is_ok || !io::write_stderr(public::help()).is_ok)){return 4;}return capture::exit_bits(status);
}
@effects(filesystem,console,environment,clock) pub fn i32 create(ref u8[] name,ref u8[] uid){return invoke(b"new",name,uid,false);}
@effects(console) fn i32 invalid(ref u8[] value,bool duplicate){
    var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,b"fairyc: command \"package\" ").is_ok){return 4;}
    if(duplicate){if(!bytes::append(ref mut out,b"received ").is_ok || !bytes::append(ref mut out,value).is_ok || !bytes::append(ref mut out,b" more than once").is_ok){return 4;}}
    else{var usize meter=0;let word=switch(quoted::quote_diagnostic(value,ref mut meter)){case .ok(p):{p}case .err(_):{return 4;}};if(!bytes::append(ref mut out,b"does not accept flag ").is_ok || !bytes::append(ref mut out,bytes::view(ref word)).is_ok){return 4;}}
    if(!bytes::append(ref mut out,b"\n\n").is_ok || !bytes::append(ref mut out,text::as_bytes(public::help())).is_ok){return 4;}
    return if(switch(io::write_stderr_bytes(bytes::view(ref out))){case .ok(n):{n==bytes::len(ref out)}case .err(_):{false}}){2}else{4};
}
@effects(filesystem,console,environment,clock) pub fn i32 package(){
    let input=switch(arguments::read(2)){case .ok(p):{p}case .err(_):{return 4;}};
    let selected=switch(options::parse(ref input)){case .ok(p):{p}case .err(e):{return invalid(bytes::view(v::at<bytes::Buffer>(ref input,e.index)),e.duplicate);}};
    let ready=invoke(b"validate",b"",b"",false);if(ready!=0){return ready;}
    let built=native::desktop_release(selected.locked);if(built!=0){return built;}
    return invoke(b"package",b"",b"",selected.json);
}
