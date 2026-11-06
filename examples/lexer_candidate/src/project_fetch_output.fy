module project_fetch_output;
import project_json_string as quoted;
import std::bytes as bytes;
import std::io as io;

// Shared presentation accepts scalar stages/counts, not a platform's owned
// filesystem or HTTPS problem type. Numeric native causes stay in API results.
@effects(console) pub fn i32 invocation(ref str message,ref str help){if(!io::write_stderr("fairyc: ").is_ok || !io::write_stderr(message).is_ok || !io::write_stderr("\n\n").is_ok || !io::write_stderr(help).is_ok){return 4;}return 2;}
@effects(console) pub fn i32 output_error(){discard io::write_stderr("fairyc: write output failed\n");return 4;}
@effects(console) pub fn i32 show_help(ref str help){return if(io::write(help).is_ok){0}else{output_error()};}
@effects(console) pub fn i32 unknown(ref u8[] name,ref str help){var usize meter=0;let text=switch(quoted::quote_diagnostic(name,ref mut meter)){case .ok(value):{value}case .err(_):{return 4;}};
    if(!io::write_stderr("fairyc: unknown experimental command ").is_ok || !io::write_stderr_bytes(bytes::view(ref text)).is_ok || !io::write_stderr("\n\n").is_ok || !io::write_stderr(help).is_ok){return 4;}return 2;
}
fn ref str reason(u32 stage,u32 code){
    if(stage==1){return "cannot read source plan or root manifest";}
    if(stage==2){return "invalid source plan, manifest or workspace binding";}
    if(stage==3){return "cannot acquire exclusive package fetch lock";}
    if(stage==4){if(code==5){return "offline source cache miss";}if(code==3){return "source cache byte or entry limit exceeded";}if(code==4){return "source cache path is noncanonical, linked or not a regular file";}return "source cache read failed";}
    if(stage==5){return "source archive SHA-256 mismatch";}
    if(stage==6){if(code==3){return "source ZIP file, byte or work limit exceeded";}return "invalid source ZIP or package identity";}
    if(stage==7){if(code==4){return "existing package conflicts with pinned archive";}return "cannot publish or reuse source package";}
    if(stage==8){return "package fetch lock cleanup failed";}
    if(stage==9){return "package fetch deadline exceeded";}
    if(stage==10){if(code==5){return "source download byte limit exceeded";}if(code==4){return "source HTTPS response status or encoding rejected";}return "source HTTPS request failed or timed out";}
    if(stage==11){return "cannot publish immutable source archive cache";}
    return "source package preparation failed";
}
@effects(console) pub fn i32 failure(u32 stage,u32 code,u32 cleanup_code){
    if(!io::write_stderr("package fetch failed: ").is_ok || !io::write_stderr(reason(stage,code)).is_ok){return 4;}
    if(cleanup_code!=0 && !io::write_stderr("; cleanup failed; inspect leftover fetch lock or staging directory").is_ok){return 4;}
    return if(io::write_stderr("\n").is_ok){7}else{4};
}
@effects(console) pub fn i32 success(usize packages,usize downloaded,usize cache_hits,usize installed,usize reused){var text=bytes::empty();
    if(!bytes::append(ref mut text,b"packages fetched: ").is_ok || !bytes::append_usize(ref mut text,packages).is_ok || !bytes::append(ref mut text,b" package(s), ").is_ok || !bytes::append_usize(ref mut text,downloaded).is_ok || !bytes::append(ref mut text,b" downloaded, ").is_ok || !bytes::append_usize(ref mut text,cache_hits).is_ok || !bytes::append(ref mut text,b" cache hit(s), ").is_ok || !bytes::append_usize(ref mut text,installed).is_ok || !bytes::append(ref mut text,b" installed, ").is_ok || !bytes::append_usize(ref mut text,reused).is_ok || !bytes::append(ref mut text,b" reused\n").is_ok){return output_error();}
    return if(switch(io::write_bytes(bytes::view(ref text))){case .ok(n):{n==bytes::len(ref text)}case .err(_):{false}}){0}else{output_error()};
}
