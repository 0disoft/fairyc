module project_check_cli;
import project_check as check;
import project_check_status as status;
import project_check_text as output;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
import project_check_options as options;
import project_compiler_cached as cached;
import project_compiler_cache_path as cache_path;

@effects(console)
fn i32 failure(ref str message,i32 code){return if(io::write_stderr(message).is_ok){code}else{4};}
@effects(console)
fn i32 render(Result<check::Stats,check::Error> result){var bytes::Buffer text=bytes::empty();switch(result){
    case .ok(stats):{if(!output::success(stats,ref mut text)){return failure("fairyc: internal compiler processing failed\n",4);}return if(switch(io::write_bytes(bytes::view(ref text))){case .ok(count):{count==bytes::len(ref text)}case .err(_):{false}}){0}else{4};}
    case .err(e):{let code=status::error(ref e);if(!output::error(ref e,ref mut text)){return failure("fairyc: internal compiler processing failed\n",4);}return if(switch(io::write_stderr_bytes(bytes::view(ref text))){case .ok(count):{count==bytes::len(ref text)}case .err(_):{false}}){code}else{4};}
};return 4;}
@effects(filesystem,console)
pub fn i32 run(ref u8[] root,bool locked){return render(check::evaluate(root,locked));}
@effects(filesystem,console,environment,clock)
pub fn i32 run_cached(ref u8[] root,bool locked,ref u8[] cache_root,bool status){var observed=cached::empty();
    let result=check::evaluate_cached(root,locked,cache_root,cache_path::command_limits(),ref mut observed);
    if(status){let ref str text=if(observed.path==2){"cache: hit\n"}else if(observed.path==3){"cache: mixed\n"}else if(observed.path==4){"cache: units hit\n"}else if(observed.path==1 && observed.stored){"cache: cold stored\n"}else if(observed.path==1){"cache: cold uncached\n"}else{"cache: skipped\n"};if(!io::write_stderr(text).is_ok){discard result;return 4;}}
    return render(result);
}
@effects(filesystem,console,environment,clock)
pub fn i32 arguments(){let selected=switch(options::from_process()){case .ok(p):{p}case .err(e):{return failure(options::diagnostic(e),if(e.code==6){4}else{2});}};
    return if(options::cache_root(ref selected).len==0){run(b".",options::locked(ref selected))}else{run_cached(b".",options::locked(ref selected),options::cache_root(ref selected),options::status(ref selected))};
}
