module tests::linux_gnu_companion;
import linux_gnu_companion as companion;
import std::bytes as bytes;
import std::vec as v;

fn void argument(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let p=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(args,p).is_ok);}
fn v::Vec<bytes::Buffer> base(bool run){var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();argument(ref mut args,b"--packet-input");argument(ref mut args,b"/stage/packet.bin");argument(ref mut args,b"--stage-root");argument(ref mut args,b"/stage");argument(ref mut args,b"--output-root");argument(ref mut args,if(run){b"/stage"}else{b"/root"});argument(ref mut args,b"--output");argument(ref mut args,if(run){b"program"}else{b"build/program"});return args;}
@test fn void accepts_build_run_and_explicit_compiler_options(){
    var args=base(false);argument(ref mut args,b"--release");argument(ref mut args,b"--cc");argument(ref mut args,b"clang");
    let p=switch(companion::parse(ref args)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(companion::release(ref p) && !companion::trap(ref p) && companion::compiler(ref p)==2 && bytes::equal(companion::stage(ref p),b"/stage") && bytes::equal(companion::root(ref p),b"/root") && bytes::equal(companion::output(ref p),b"build/program"));
    var run=base(true);argument(ref mut run,b"--run-trap-control");let selected=switch(companion::parse(ref run)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(companion::trap(ref selected) && !companion::release(ref selected));
    let asset=switch(companion::asset_path(b"/opt/bin/fairyc-candidate-linux-build")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref asset),b"/opt/bin/fairyc-linux-runtime-assets.pack"));assert(!companion::asset_path(b"fairyc").is_ok);
    var v::Vec<bytes::Buffer> help=v::empty<bytes::Buffer>();argument(ref mut help,b"--help");let option=switch(companion::parse(ref help)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(companion::help(ref option));
}
@test fn void rejects_ambiguous_flags_and_nonprivate_trap_outputs(){
    var u32 variant=0;while(variant<6){var args=base(false);
        if(variant==0){argument(ref mut args,b"--release");argument(ref mut args,b"--release");}
        else if(variant==1){argument(ref mut args,b"--cc");}
        else if(variant==2){argument(ref mut args,b"--cc");argument(ref mut args,b"other");}
        else if(variant==3){argument(ref mut args,b"--run-trap-control");}
        else if(variant==4){argument(ref mut args,b"--help");}
        else{argument(ref mut args,b"--stage-root");argument(ref mut args,b"/elsewhere");}
        assert(!companion::parse(ref args).is_ok);variant+=1;
    }
    let empty=v::empty<bytes::Buffer>();assert(!companion::parse(ref empty).is_ok);
}
