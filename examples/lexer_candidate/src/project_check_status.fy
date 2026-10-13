module project_check_status;
import project_check as check;
import project_workspace_lock as loader;
import project_module_visibility as visibility;
import project_instance_graph as instances;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_package_input as packages;

// The loader preserves original parser/lexer failures separately from package
// admission. Stable check maps source diagnostics to 3 and package/lock to 7.
pub fn i32 package_input(ref packages::Error e){let p=packages::error(e);if(p.phase==7 || p.phase==6 && p.reason==7 || p.phase==5 && p.reason==2){return 4;}if(p.phase==6 && (p.reason==2 || p.reason==3)){return 3;}return 7;}
pub fn i32 load(ref loader::Error e){
    let phase=loader::phase(e);if(phase==3){return 4;}if(phase==2){return 7;}
    let visible=loader::input_error(e);let kind=visibility::load_phase(visible);if(kind==3){return 4;}
    if(kind==2){return if(visibility::error(visibility::visibility_error(visible)).code==3){4}else{7};}
    if(kind==4){return 7;}let local=visibility::input_error(visible);let local_phase=instances::load_phase(local);if(local_phase==3){return 4;}if(local_phase==2){return if(instances::error(instances::graph_error(local)).code==3){4}else{7};}
    let input=instances::input_error(local);let problem=workspace::error(input);if(problem.code==7){return 4;}if(!workspace::has_catalog_error(input)){return 7;}
    let c=workspace::nested_catalog_error(input);if(catalog::error(c).code==10){return 4;}if(!catalog::has_input_error(c)){return 7;}return package_input(catalog::input_error(c));
}
pub fn i32 error(ref check::Error e){let phase=check::phase(e);if(phase==1){return load(check::load_error(e));}if(phase==2 || phase==4){return 4;}let p=check::compiler_error(e);if(p.phase==4 || p.phase==6 || p.phase==5 && p.code==9 || p.code==10 || p.code==11 || p.code==12 || p.code==1001 || p.code==1002 || p.code==1003 || p.code==4503){return 4;}return 3;}
