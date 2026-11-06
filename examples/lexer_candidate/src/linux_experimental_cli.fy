module linux_experimental_cli;
import linux_fetch_cli as fetching;
import project_atlas_cli as atlas;
import project_mcp_cli as mcp;
import project_catalog_cli as catalog;
import linux_cli_data as public;
import std::bytes as bytes;
import std::process as process;

// Only implemented Linux commands enter shared policy owners. Fetch retains
// unknown/help/overlong-argument precedence and is the sole network command.
pub fn u32 command(ref u8[] name){return if(bytes::equal(name,b"atlas-impact")){1}else if(bytes::equal(name,b"atlas-mcp")){2}else if(bytes::equal(name,b"atlas-symbols")){3}else{0};}
@effects(filesystem,console,environment,network,clock) pub fn i32 arguments(){
    if(process::arg_count()>=3){
        var u8[4096] raw=[0;4096];let n=switch(process::arg(2,ref mut raw)){case .ok(p):{p}case .err(_):{return fetching::arguments();}};
        let ref u8[] name=ref raw;let mode=command(name[..n]);if(mode==1){return atlas::arguments_with_help(public::help());}if(mode==2){return mcp::arguments_with_help(public::help());}if(mode==3){return catalog::arguments_with_help(public::help());}
    }
    return fetching::arguments();
}
