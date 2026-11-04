module project_experimental_cli;
import project_fetch_cli as fetching;
import project_atlas_cli as atlas;
import project_mcp_cli as mcp;
import std::bytes as bytes;
import std::process as process;
@effects(filesystem,console,environment,network,clock) pub fn i32 arguments(){if(process::arg_count()>=3){var u8[4096] raw=[0;4096];let n=switch(process::arg(2,ref mut raw)){case .ok(p):{p}case .err(_):{0}};let ref u8[] command=ref raw;if(bytes::equal(command[..n],b"atlas-impact")){return atlas::arguments();}if(bytes::equal(command[..n],b"atlas-mcp")){return mcp::arguments();}}return fetching::arguments();}
