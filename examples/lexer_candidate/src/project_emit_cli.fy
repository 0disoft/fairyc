module project_emit_cli;
import project_emit as emit;
import project_emit_text as output;
import module_project_cli as transport;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@effects(console) fn i32 failure(ref str message,i32 code){return if(io::write_stderr(message).is_ok){code}else{4};}
@effects(filesystem,console)
pub fn i32 run(ref u8[] root,bool locked){return switch(emit::evaluate(root,locked)){
    case .ok(plan):{transport::packet(emit::output_bundle(ref plan))}
    case .err(e):{var bytes::Buffer text=bytes::empty();if(!output::error(ref e,ref mut text)){return failure("fairyc: internal compiler processing failed\n",4);}return if(switch(io::write_stderr_bytes(bytes::view(ref text))){case .ok(n):{n==bytes::len(ref text)}case .err(_):{false}}){output::exit_code(ref e)}else{4};}
};}
@effects(filesystem,console)
pub fn i32 arguments(){let argc=process::arg_count();if(argc!=3 && argc!=4){return failure("usage: lexer-candidate --c-workspace <root> [--locked]\n",2);}var bool locked=false;var u8[4096] data=[0;4096];
    if(argc==4){let n=switch(process::arg(3,ref mut data)){case .ok(n):{n}case .err(_):{return failure("usage: lexer-candidate --c-workspace <root> [--locked]\n",2);}};let ref u8[] view=ref data;if(!bytes::equal(view[..n],b"--locked")){return failure("usage: lexer-candidate --c-workspace <root> [--locked]\n",2);}locked=true;}
    let n=switch(process::arg(2,ref mut data)){case .ok(n):{n}case .err(_):{return failure("fairyc: workspace path exceeds 4095 UTF-8 bytes\n",2);}};if(n==0 || n>4095){return failure("fairyc: invalid workspace path\n",2);}let ref u8[] view=ref data;return run(view[..n],locked);
}
