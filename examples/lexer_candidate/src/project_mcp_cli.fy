module project_mcp_cli;
import project_mcp_options as policy;
import project_mcp_protocol as protocol;
import project_mcp_stream as stream;
import project_atlas_options as options;
import project_atlas_source as source;
import public_cli_data as public;
import std::bytes as bytes;
import std::process as process;
import std::io as io;

@effects(console) fn i32 error(i32 code,ref str text){if(!io::write_stderr("fairyc: ").is_ok || !io::write_stderr(text).is_ok || !io::write_stderr("\n").is_ok){return 4;}return code;}
@effects(filesystem,environment) fn Result<bytes::Buffer,u32> query(ref u8[] root,policy::Policy p,ref protocol::Reply request){let selected=switch(options::selected(protocol::uid(request),protocol::symbol(request),p.packages,p.calls,p.depth,p.symbols,p.output-1024)){case .ok(value):{value}case .err(_):{return protocol::finish(request,b"",false,p.output);}};let data=switch(source::query(root,ref selected)){case .ok(value):{value}case .err(_):{return protocol::finish(request,b"",false,p.output);}};return protocol::finish(request,bytes::view(ref data),true,p.output);}
// One invocation supplies the root/capability policy. The read-only tool always
// re-enters the actual Fairy locked-source verifier; no child supplies facts.
@effects(filesystem,environment,console) pub fn i32 serve(ref u8[] root,policy::Policy p){if(!policy::valid(p)){return error(2,"invalid MCP session policy");}var reader=switch(stream::open()){case .ok(value):{value}case .err(_):{return error(2,"MCP session failed its framing, I/O or resource policy");}};var state=protocol::empty();var usize count=0;
 while(true){var bytes::Buffer line=bytes::empty();let more=switch(stream::next(ref mut reader,ref mut line)){case .ok(value):{value}case .err(_):{return error(2,"MCP session failed its framing, I/O or resource policy");}};if(!more){return 0;}if(count>=p.requests){return error(2,"MCP session failed its framing, I/O or resource policy");}count+=1;
 let reply=switch(protocol::step(ref mut state,line,p.output)){case .ok(value):{value}case .err(_):{return error(2,"MCP session failed its framing, I/O or resource policy");}};
 if(protocol::pending(ref reply)){let result=switch(query(root,p,ref reply)){case .ok(value):{value}case .err(_):{return error(2,"MCP session failed its framing, I/O or resource policy");}};if(stream::write(bytes::view(ref result))!=0){return error(4,"write output failed");}}
 else if(stream::write(protocol::packet(ref reply))!=0){return error(4,"write output failed");}
 }
 return 0;
}
@effects(filesystem,environment,console) pub fn i32 arguments(){if(process::arg_count()==4){var u8[4096] raw=[0;4096];let n=switch(process::arg(3,ref mut raw)){case .ok(value):{value}case .err(_):{0}};let ref u8[] flag=ref raw;if(bytes::equal(flag[..n],b"--help") || bytes::equal(flag[..n],b"-h") || bytes::equal(flag[..n],b"help")){return if(io::write(public::help()).is_ok){0}else{4};}}
 let p=switch(policy::arguments()){case .ok(value):{value}case .err(_):{return error(2,"invalid experimental atlas-mcp arguments");}};return serve(b".",p);
}
