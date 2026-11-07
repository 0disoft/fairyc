module project_compiler_command;
import project_compiler_core as core;
import project_compiler_input as input;
import project_compiler_source_mode as mode;
import module_public_type_text as coordinates;
import generic_signature_diagnostics as diagnostics;
import std::vec as v;

// Commands choose only from the current projected source. Legacy ordinary
// factories and explicit artifact decoders retain their closed admission modes.
pub fn Result<bool,core::Problem> aggregates(ref input::Input sources,ref mut usize work){
    return switch(mode::aggregates(sources,work)){
        case .ok(p):{.ok(p)}
        case .err(_):{var p=core::empty();p.phase=5;p.code=10;.err(p)}
    };
}
pub fn Result<bool,core::Problem> closures(ref input::Input sources,ref mut usize work){
    return switch(mode::closures(sources,work)){case .ok(p):{.ok(p)}case .err(_):{var p=core::empty();p.phase=5;p.code=10;.err(p)}};
}
pub fn Result<core::Complete,core::Problem> build(ref input::Input sources,ref coordinates::Coordinates table,ref mut v::Vec<diagnostics::Reports> reports,ref mut usize source_work,ref mut usize link_work){
    if(v::len<diagnostics::Reports>(reports)!=0){return core::build_complete_reporting(sources,table,reports,source_work,link_work);}
    let aggregate=switch(aggregates(sources,source_work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    return if(aggregate){core::build_complete_aggregate_sources(input::sources(sources),input::owners(sources),input::bindings(sources),table,source_work,link_work)}else{core::build_complete_reporting(sources,table,reports,source_work,link_work)};
}
