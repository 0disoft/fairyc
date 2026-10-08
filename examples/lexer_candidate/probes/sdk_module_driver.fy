module main;
import module_cli as modules;
import std::bytes as bytes;
import std::process as process;

// Opt-in observation entry. The authored CLI keeps its existing full-AST path.
@effects(filesystem,console)
fn i32 main(){
    if(process::arg_count()!=3){return 2;}
    var bytes::Buffer arg=bytes::empty();if(!bytes::resize_zeroed(ref mut arg,4096).is_ok){return 2;}
    let usize flag=switch(process::arg(1,bytes::view_mut(ref mut arg))){case .ok(n):{n}case .err(_):{return 2;}};
    if(!bytes::equal(bytes::view(ref arg)[..flag],b"--c-modules-file")){return 2;}
    let usize length=switch(process::arg(2,bytes::view_mut(ref mut arg))){case .ok(n):{n}case .err(_):{return 2;}};
    return modules::emit_sdk(bytes::view(ref arg)[..length]);
}
