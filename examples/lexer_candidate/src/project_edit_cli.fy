module project_edit_cli;
import project_edit_options as options;
import project_edit_preview as preview;
import project_edit_body as edit;
import project_remote_fs as files;
import windows_rooted_file as host;
import public_cli_data as public;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@effects(console) fn i32 error(i32 code,ref str text){if(!io::write_stderr("fairyc: ").is_ok || !io::write_stderr(text).is_ok || !io::write_stderr("\n").is_ok){return 4;}return code;}
@effects(console) fn i32 failure(preview::Problem p){return error(if(p.stage==2 || p.stage==3){7}else{2},if(p.stage==2){"edit workspace or current lock unavailable"}else if(p.stage==3){"edit workspace validation failed"}else if(p.stage==4){"edit subject, source hash or replacement body rejected"}else if(p.stage==5){"edited workspace failed semantic/MIR validation"}else if(p.stage==6){"edit report exceeds resource policy"}else{"edit requires one exact ordinary root function"});}
@effects(filesystem,environment,console) pub fn i32 run(ref u8[] root,ref options::Options o){
    // Apply remains closed until the publication owner is connected. It can
    // never silently degrade to a successful preview.
    if(options::apply(o)){return error(2,"edit apply publication is not connected");}
    let selected=switch(preview::select(root,options::uid(o),options::symbol(o))){case .ok(p):{p}case .err(p):{return failure(p);}};
    if(!preview::body_allowed(ref selected,options::body_file(o))){return error(7,"edit body file must be a portable workspace file outside production roots");}
    let opened=switch(host::open(root)){case .ok(p):{p}case .err(_):{return error(7,"edit body root unavailable");}};
    let body=switch(files::read(ref opened,options::body_file(o),edit::maximum_body())){case .ok(p):{p}case .err(_):{return error(7,"edit body file read, identity or byte policy failed");}};
    let result=switch(preview::finish(selected,options::expected(o),bytes::view(ref body))){case .ok(p):{p}case .err(p):{return failure(p);}};
    return if(switch(io::write_bytes(preview::report(ref result))){case .ok(n):{n==preview::report(ref result).len}case .err(_):{false}}){0}else{error(4,"write output failed")};
}
@effects(filesystem,environment,console) pub fn i32 arguments(){
    if(process::arg_count()==4){var u8[4096] raw=[0;4096];let n=switch(process::arg(3,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] value=ref raw;if(bytes::equal(value[..n],b"--help") || bytes::equal(value[..n],b"-h") || bytes::equal(value[..n],b"help")){return if(io::write(public::help()).is_ok){0}else{4};}}
    let o=switch(options::arguments()){case .ok(p):{p}case .err(_):{return error(2,"invalid experimental atlas-edit replace-body arguments");}};
    return run(b".",ref o);
}
