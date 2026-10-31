module tests::linux_gnu_build;
import linux_gnu_build as build;
import native_packet_decode as packet;
import std::bytes as bytes;

@link_name("gnu_probe_mode") unsafe extern C fn void mode(u32 value,u32 release,u32 arguments);
@link_name("gnu_probe_counts") unsafe extern C fn u32 counts();
fn packet::Packet input(bool arguments){
    var bytes::Buffer header=bytes::empty();assert(bytes::append(ref mut header,b"#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#define FY_CANDIDATE_PROJECT_HEADER_V1\n/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\n#include <stdint.h>\n").is_ok);
    if(arguments){assert(bytes::append(ref mut header,b"extern int fy_std_process_arg_count_v0(void);\n").is_ok);}assert(bytes::append(ref mut header,b"#endif\n").is_ok);
    let ref u8[] literals=b"/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\ntypedef int literals_present;\n";
    let ref u8[] unit=b"/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\nextern int32_t fy_fn_1(void){return 42;}\nint32_t fy_candidate_entry(void){return fy_fn_1();}\n#ifndef FY_CANDIDATE_NO_MAIN\nint main(void){return (int)fy_candidate_entry();}\n#endif\n";
    var bytes::Buffer data=bytes::empty();assert(bytes::append(ref mut data,b"FYCPARTS1\n").is_ok);assert(bytes::append_usize(ref mut data,bytes::len(ref header)).is_ok && bytes::append_byte(ref mut data,10).is_ok && bytes::append(ref mut data,bytes::view(ref header)).is_ok);
    assert(bytes::append_usize(ref mut data,literals.len).is_ok && bytes::append_byte(ref mut data,10).is_ok && bytes::append(ref mut data,literals).is_ok && bytes::append(ref mut data,b"1\n").is_ok);
    assert(bytes::append_usize(ref mut data,unit.len).is_ok && bytes::append_byte(ref mut data,10).is_ok && bytes::append(ref mut data,unit).is_ok);
    return switch(packet::decode(data)){case .ok(p):{p}case .err(_):{assert(false);return input(arguments);}};
}
@test @effects(filesystem,console,environment,clock) fn void compiles_links_and_publishes_both_profiles(){
    var u32 release=0;while(release<2){unsafe{mode(0,release,0);};let p=input(false);let n=switch(build::build_publish(ref p,b"typedef int runtime_present;\n",b"/stage",build::Options{.release=release==1,.timeout_ms=60000,.compiler=0},b"/root",b"build/program")){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(n==128 && unsafe{counts()}==311);release+=1;}
}
@test @effects(filesystem,console,environment,clock) fn void initializes_arguments_and_resolves_clang_without_go_fallback(){
    unsafe{mode(0,0,1);};let p=input(true);let result=switch(build::build(ref p,b"typedef int runtime_present;\n",b"/stage",build::Options{.release=false,.timeout_ms=60000,.compiler=2})){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(build::object_count(ref result)==4 && build::data(ref result).len==128 && unsafe{counts()}==410);
    unsafe{mode(9,0,0);};let q=input(false);assert(build::build_publish(ref q,b"typedef int runtime_present;\n",b"/stage",build::Options{.release=false,.timeout_ms=60000,.compiler=0},b"/root",b"build/program").is_ok);assert(unsafe{counts()}==311);
}
@test @effects(filesystem,console,environment,clock) fn void preserves_old_image_and_reports_failure_phase(){
    var u32 scenario=1;while(scenario<=14){if(scenario!=9){unsafe{mode(scenario,0,0);};let p=input(false);
        switch(build::build_publish(ref p,b"typedef int runtime_present;\n",b"/stage",build::Options{.release=false,.timeout_ms=60000,.compiler=1},b"/root",b"build/program")){
            case .ok(_):{assert(false);}case .err(e):{let u32 phase=if(scenario==1 || scenario==10 || scenario==11){6}else if(scenario==2){8}else if(scenario==3 || scenario==13){7}else if(scenario==4){9}else if(scenario==5){4}else if(scenario==6 || scenario==14){5}else if(scenario==8){10}else{3};assert(build::stage(ref e)==phase);if(scenario==1){assert(bytes::equal(build::diagnostics(ref e),b"compile error\n"));}if(scenario==2){assert(bytes::equal(build::diagnostics(ref e),b"link error\n"));}}
        };assert(unsafe{counts()}%10==0);
    }scenario+=1;}
    unsafe{mode(0,0,0);};let p=input(false);assert(!build::build(ref p,b"runtime",b"/stage",build::Options{.release=false,.timeout_ms=0,.compiler=0}).is_ok);assert(unsafe{counts()}==0);
}
