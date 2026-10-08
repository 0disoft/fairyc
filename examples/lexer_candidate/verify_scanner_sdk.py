"""Run original scanner Token/cooked/diagnostic storage through candidate C."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "scanner-sdk"
observer.SDK = observer.REPO / "examples/lexer_candidate/src/scanner.fy"
observer.DEPENDENCIES = [
    observer.REPO / "examples/lexer_candidate/src" / f"{name}.fy"
    for name in ("candidate_limits", "numbers", "diagnostics", "utf8", "strings", "budget")
] + [
    observer.REPO / "internal/sdkstdlib/sources/std/bytes.fy",
    observer.REPO / "internal/sdkstdlib/sources/std/io.fy",
    observer.REPO / "internal/sdkstdlib/sources/optional/std/vec.fy",
]
observer.CASE_OBSERVER = Path(__file__)
observer.PREAMBLE = "module main;import scanner as s;import std::bytes as b;"
observer.RUNTIME_NAMES += ("stage0_buffer.c", "stage0_windows_console.c")
observer.CASES = {
    "owned-tokens-cooked-diagnostics-and-limits": observer.FOREIGN + r'''
fn i32 exercise(){
    let ref u8[] source=b"true \"a\\n\" b\"\\x41\" 0x1f /* unclosed";
    let input=switch(b::copy(source)){case .ok(x):{x}case .err(_):{return 0;}};
    let stream=switch(s::collect(input,false,65536,1048576,256,16777216)){case .ok(x):{x}case .err(_):{return 0;}};
    assert(b::equal(s::source_view(ref stream),source));
    var usize position=0;var usize i=0;var bool keyword=false;var bool string=false;var bool bytes=false;var bool integer=false;
    while(i<s::token_count(ref stream)){
        let info=s::token_info(ref stream,i);let raw=s::token_raw(ref stream,i);let cooked=s::token_cooked(ref stream,i);
        assert(info.start==position && info.end>=info.start && b::equal(raw,source[info.start..info.end]));position=info.end;
        if(info.kind==4){keyword=true;assert(b::equal(cooked,b"true"));}
        // The private Stream stores cooked literal payloads as hex text, as in
        // the scanner report protocol consumed by the parser/literal modules.
        if(info.kind==7){string=true;assert(b::equal(cooked,b"610a"));}
        if(info.kind==8){bytes=true;assert(b::equal(cooked,b"41"));}
        if(info.kind==5){integer=true;assert(b::equal(raw,b"0x1f"));}
        if(info.kind==1){assert(i+1==s::token_count(ref stream) && info.start==source.len && info.end==source.len);}
        i+=1;
    }
    assert(position==source.len && keyword && string && bytes && integer);
    assert(s::diagnostic_count(ref stream)==1);let problem=s::diagnostic_info(ref stream,0);
    assert(problem.code==1 && problem.exit_repair && b::equal(s::diagnostic_token(ref stream,0),b"*/"));
    let storage=s::storage(ref stream);assert(storage.source_bytes==source.len && storage.cooked_bytes>=3 && storage.diagnostic_bytes==2 && storage.diagnostics==1);
    s::release(stream);
    let limited_input=switch(b::copy(b"abcdef")){case .ok(x):{x}case .err(_):{return 0;}};
    let limited=switch(s::collect(limited_input,false,3,1048576,256,16777216)){case .ok(x):{x}case .err(_):{return 0;}};
    assert(s::diagnostic_count(ref limited)==1 && s::diagnostic_info(ref limited,0).code==10);
    assert(s::token_count(ref limited)==2 && b::equal(s::token_raw(ref limited,0),b"abcdef") && s::token_info(ref limited,1).kind==1);
    s::release(limited);return 7;
}
fn i32 main(){assert(exercise()==7);assert(unsafe{fixture_live()}==0);return 7;}
''',
}
observer.REJECT = {}

if __name__ == "__main__":
    observer.main()
