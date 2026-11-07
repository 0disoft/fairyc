module module_state_command_windows;
import module_state_command_arguments as arguments;
import module_state_command_options as command;
import module_state_command_source as source;
import module_state_graph_source as authored;
import syntax as syntax;
import module_state_command_windows_file as files;
import module_state_command_snapshot_report as report;
import module_state_graph_transport as transport;
import module_state_graph_native_assets as runtime;
import module_state_graph_frame_assets as frame;
import module_state_graph_windows_build as builder;
import module_state_graph_windows_run as runner;
import module_state_graph_windows_persistent as persistent;
import module_state_graph_request as request;
import module_state_local_windows as store;
import module_state_local_codec as codec;
import project_compiler_core as core;
import project_compiler_state_graph as graph;
import module_state_graph_program as program;
import module_state_graph_model as model;
import module_state_layout as layout;
import windows_cancel as cancellation;
import windows_host_paths as paths;
import windows_msvc_toolchain as tools;
import native_msvc_build as msvc;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

pub fn ref u8[] runtime_leaf(){return b"fairyc-state-graph-runtime.pack";}
pub fn ref u8[] frame_leaf(){return b"fairyc-state-graph-frame.pack";}
@effects(console) fn i32 help(){
    return if(io::write_bytes(b"Experimental authored local Fairy state runner\n\nUsage: fairyc-state <init|execute|snapshot> <existing-store-directory> <source.fy> --engine native-graph --profile low|optimized\n\nWindows amd64 MSVC; fresh bounded source; one transaction and one attempt.\ninit/snapshot never build or run the transaction. The Go-MIR form is unsupported.\nExit codes: 0 success, 2 invocation, 3 source rejection, 4 I/O/backend failure,\n6 transaction rejection, 7 conflict/layout/busy, 8 language trap.\n").is_ok){0}else{4};
}
@effects(console) fn i32 problem(i32 exit,ref u8[] phase,u32 stage,u32 code,u32 os){
    if(!io::write_stderr_bytes(b"fairyc-state: ").is_ok || !io::write_stderr_bytes(phase).is_ok || !io::write_stderr_bytes(b" stage=").is_ok || !io::write_stderr_u32(stage).is_ok || !io::write_stderr_bytes(b" code=").is_ok || !io::write_stderr_u32(code).is_ok || !io::write_stderr_bytes(b" os=").is_ok || !io::write_stderr_u32(os).is_ok || !io::write_stderr_bytes(b"\n").is_ok){return 4;}return exit;
}
@effects(console) fn i32 arguments_problem(arguments::Failure error){
    return switch(error){
    case .Invocation{.error=e}:{switch(e){case .Invalid:{problem(2,b"arguments (use --help)",0,1,0)}case .UnsupportedGoMIR:{problem(2,b"Go-MIR form is unsupported",0,2,0)}case .TooLarge:{problem(2,b"argument limit",0,3,0)}case .Nul:{problem(2,b"argument NUL",0,4,0)}}}
    case .Host{.status=status,.index=index,.needed=needed}:{
        discard problem(4,b"argument host",0,status,0);discard io::write_stderr_bytes(b"argument index=");discard io::write_stderr_usize(index);discard io::write_stderr_bytes(b" needed=");discard io::write_stderr_usize(needed);discard io::write_stderr_bytes(b"\n");4
    }
    case .Allocation:{problem(4,b"argument allocation",0,1003,0)}
    };
}
@effects(filesystem,console) fn bool checkpoint(ref cancellation::Token token){
    return switch(cancellation::requested(token)){case .ok(stop):{if(stop){discard problem(4,b"canceled",0,1009,0);}!stop}case .err(e):{discard problem(4,b"cancel observation",e.kind,1010,e.os_code);false}};
}
fn i32 storage_exit(store::Failure cause){return if(cause.stage==6 && (cause.code==1004 || cause.code==1005) || cause.stage==3 && cause.os_code==33){7}else{4};}
@effects(console) fn i32 storage_problem(store::Failure cause){return problem(storage_exit(cause),b"store",cause.stage,cause.code,cause.os_code);}
@effects(console) fn i32 execution_problem(ref persistent::Failure error){
    let cause=persistent::cause(error);
    if(cause.stage==2 || cause.stage==7){return storage_problem(cause.storage);}
    if(cause.stage==4){
        let code=problem(4,b"runner",cause.runner_stage,cause.runner_status,cause.runner_os_code);
        switch(persistent::failure_runner(error)){case .ok(e):{discard io::write_stderr_bytes(runner::diagnostics(e));}case .err(_):{}};return code;
    }
    if(cause.stage==8){discard io::write_stderr_bytes(b"fairyc-state: runtime status=");discard io::write_stderr_usize(cause.runtime_status);discard io::write_stderr_bytes(b"\n");return 4;}
    return problem(4,b"execution",cause.stage,cause.code,cause.cancellation_os_code);
}
@effects(console) fn i32 output(ref u8[] data,i32 exit){
    return switch(io::write_bytes(data)){case .ok(_):{exit}case .err(e):{
        discard io::write_stderr_bytes(b"fairyc-state: report output failed; an executed commit may already be published; status=");discard io::write_stderr_u32(e.status);discard io::write_stderr_bytes(b" written=");discard io::write_stderr_usize(e.written);discard io::write_stderr_bytes(b"\n");4
    }};
}
fn ref layout::Plan schemas(ref transport::Prepared prepared){return program::layouts(graph::program(core::checked_state_graph(transport::checked(prepared))));}
fn usize selected(ref transport::Prepared prepared){return model::header(program::root(graph::program(core::checked_state_graph(transport::checked(prepared))))).layout;}
fn bool source_name(ref u8[] path,ref core::Complete checked){
    var usize start=0;var usize i=0;while(i<path.len){if(path[i]==47 || path[i]==92){start=i+1;}i+=1;}
    if(path.len-start<=3){return false;}let stem=path[start..path.len-3];i=0;
    while(i<stem.len){let c=stem[i];if(!(c==95 || c>=65 && c<=90 || c>=97 && c<=122 || i!=0 && c>=48 && c<=57)){return false;}i+=1;}
    let parsed=authored::parsed(graph::source(core::checked_state_graph(checked)));let declaration=syntax::declaration_at(parsed,0);
    return bytes::equal(stem,syntax::source_view(parsed)[declaration.name_start..declaration.name_end]);
}

// Actual command orchestration; injected paths/token are explicit trusted host
// inputs, never an interpreter, arbitrary executable or provider substitution.
@effects(filesystem,console,environment,clock)
pub fn i32 run(ref arguments::Invocation invocation,ref u8[] runtime_path,ref u8[] frame_path,ref cancellation::Token token){
    return switch(arguments::parsed(invocation)){case .Help:{help()}case .Native{.options=p}:{run_native(invocation,p,runtime_path,frame_path,token)}};
}
@effects(filesystem,console,environment,clock)
fn i32 run_native(ref arguments::Invocation invocation,command::Options options,ref u8[] runtime_path,ref u8[] frame_path,ref cancellation::Token token){
    if(!checkpoint(token)){return 4;}
    let argv=arguments::arguments(invocation);let source_path=bytes::view(v::at<bytes::Buffer>(argv,options.source_index));let store_path=bytes::view(v::at<bytes::Buffer>(argv,options.store_index));
    var usize file_work=0;
    let actual=switch(files::load(source_path,65536,token,ref mut file_work)){case .ok(p):{p}case .err(e):{return problem(if(e.code==1002){3}else{4},b"source file",e.stage,e.code,e.os_code);}};
    if(!checkpoint(token)){return 4;}
    var usize source_work=0;let checked=switch(source::prepare(bytes::view(ref actual),ref mut source_work)){case .ok(p):{p}case .err(code):{return problem(3,b"source",0,code,0);}};
    if(!source_name(source_path,ref checked)){return problem(3,b"source module name",0,1001,0);}
    let runtime_bytes=switch(files::load(runtime_path,1048576,token,ref mut file_work)){case .ok(p):{p}case .err(e):{return problem(4,b"runtime file",e.stage,e.code,e.os_code);}};
    var usize asset_work=0;let runtime=switch(runtime::load(runtime_bytes,ref mut asset_work)){case .ok(p):{p}case .err(code):{return problem(4,b"runtime pin",0,code,0);}};
    let frame_bytes=switch(files::load(frame_path,1048576,token,ref mut file_work)){case .ok(p):{p}case .err(e):{return problem(4,b"frame file",e.stage,e.code,e.os_code);}};
    var usize frame_work=0;let framing=switch(frame::load(frame_bytes,ref mut frame_work)){case .ok(p):{p}case .err(code):{return problem(4,b"frame pin",0,code,0);}};
    if(!checkpoint(token)){return 4;}
    var prepare_work=transport::empty_work();let prepared=switch(transport::prepare(checked,ref runtime,ref framing,b"x86_64-windows-msvc",ref mut prepare_work)){case .ok(p):{p}case .err(code):{return problem(3,b"transport",0,code,0);}};
    let policy=codec::Limits{.bytes=16777216,.entries=65536,.value=65536};
    if(options.action==2){
        if(!checkpoint(token)){return 4;}
        let toolchain=switch(tools::resolve()){case .ok(p):{p}case .err(e):{return problem(4,b"toolchain",e.stage,e.status,e.os_code);}};
        var usize build_work=0;let built=switch(builder::build_cancel(ref toolchain,prepared,ref runtime,msvc::Options{.release=options.release,.timeout_ms=60000,.stack_reserve=16777216},token,ref mut build_work)){case .ok(p):{p}case .err(e):{let code=problem(4,b"build",builder::stage(ref e),builder::status(ref e),builder::os_code(ref e));discard io::write_stderr_bytes(builder::diagnostics(ref e));return code;}};
        if(!checkpoint(token)){return 4;}
        var work=persistent::empty_work();let current=builder::prepared(ref built);
        let instance=switch(store::open(store_path,schemas(current),selected(current),policy,false,ref mut work.local)){case .ok(p):{p}case .err(e):{return storage_problem(e);}};
        let result=switch(persistent::run_cancel(ref built,ref instance,request::Limits{.writes=1024,.bytes=1048576,.value=65536,.key=1024},runner::Options{.timeout_ms=60000},token,ref mut work)){case .ok(p):{p}case .err(e):{return execution_problem(ref e);}};
        // Never poll after a possible publication. Even failed JSON delivery
        // cannot revoke an acknowledged store operation or justify a retry.
        var usize report_work=0;let data=switch(report::execute(options,current,ref result,ref mut report_work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_bytes(b"fairyc-state: report construction failed; an executed commit may already be published\n");return problem(4,b"report",0,code,0);}};
        return output(bytes::view(ref data),report::outcome_exit(ref result));
    }
    if(!checkpoint(token)){return 4;}
    // init enters an uncancellable create/snapshot/report phase, preserving a
    // known initial publication even when a signal arrives during create.
    var local=codec::empty_work();let instance=switch(store::open(store_path,schemas(ref prepared),selected(ref prepared),policy,options.action==1,ref mut local)){case .ok(p):{p}case .err(e):{return storage_problem(e);}};
    let hash=layout::at(schemas(ref prepared),selected(ref prepared)).hash;
    let state=switch(store::snapshot(ref instance,hash,ref mut local)){case .ok(p):{p}case .err(e):{return storage_problem(e);}};
    var usize report_work=0;let data=switch(report::snapshot(options,ref prepared,ref state,ref mut report_work)){case .ok(p):{p}case .err(code):{return problem(4,b"snapshot report",0,code,0);}};
    return output(bytes::view(ref data),0);
}
@effects(filesystem,console,environment,clock) pub fn i32 run_process(){
    let invocation=switch(arguments::read()){case .ok(p):{p}case .err(e):{return arguments_problem(e);}};
    switch(arguments::parsed(ref invocation)){case .Help:{return help();}case .Native{.options=p}:{}};
    let token=switch(cancellation::local()){case .ok(p):{p}case .err(e):{return problem(4,b"cancel event",e.kind,0,e.os_code);}};
    let console=switch(cancellation::console(ref token)){case .ok(p):{p}case .err(e):{return problem(4,b"console registration",e.kind,0,e.os_code);}};
    let image=switch(paths::executable()){case .ok(p):{p}case .err(e):{return problem(4,b"image path",e.kind,0,e.os_code);}};
    let runtime=switch(paths::sibling(bytes::view(ref image),runtime_leaf())){case .ok(p):{p}case .err(e):{return problem(4,b"runtime path",e.kind,0,e.os_code);}};
    let frame=switch(paths::sibling(bytes::view(ref image),frame_leaf())){case .ok(p):{p}case .err(e):{return problem(4,b"frame path",e.kind,0,e.os_code);}};
    let result=run(ref invocation,bytes::view(ref runtime),bytes::view(ref frame),ref token);
    // Registration cleanup is required, but cannot reinterpret known state
    // publication as cancellation or trigger any retry.
    return switch(cancellation::close_console(console)){case .ok(_):{result}case .err(e):{
        discard io::write_stderr_bytes(b"fairyc-state: console cleanup failed; state may already be published\n");problem(4,b"console cleanup",e.kind,0,e.os_code)
    }};
}
