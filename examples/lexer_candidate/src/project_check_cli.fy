module project_check_cli;
import project_check as check;
import project_check_status as status;
import project_check_text as output;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@effects(console)
fn i32 failure(ref str message,i32 code){return if(io::write_stderr(message).is_ok){code}else{4};}
@effects(filesystem,console)
pub fn i32 run(ref u8[] root,bool locked){var bytes::Buffer text=bytes::empty();switch(check::evaluate(root,locked)){
    case .ok(stats):{if(!output::success(stats,ref mut text)){return failure("fairyc: internal compiler processing failed\n",4);}return if(switch(io::write_bytes(bytes::view(ref text))){case .ok(count):{count==bytes::len(ref text)}case .err(_):{false}}){0}else{4};}
    case .err(e):{let code=status::error(ref e);if(!output::error(ref e,ref mut text)){return failure("fairyc: internal compiler processing failed\n",4);}return if(switch(io::write_stderr_bytes(bytes::view(ref text))){case .ok(count):{count==bytes::len(ref text)}case .err(_):{false}}){code}else{4};}
};return 4;}
@effects(filesystem,console)
pub fn i32 arguments(){let count=process::arg_count();if(count!=2 && count!=3){return failure("fairyc: command \"check\" accepts only optional --locked\n",2);}var bool locked=false;
    if(count==3){var u8[4096] data=[0;4096];let size=switch(process::arg(2,ref mut data)){case .ok(n):{n}case .err(_):{return failure("fairyc: command \"check\" accepts only optional --locked\n",2);}};let ref u8[] view=ref data;if(!bytes::equal(view[..size],b"--locked")){return failure("fairyc: command \"check\" accepts only optional --locked\n",2);}locked=true;}
    return run(b".",locked);
}
