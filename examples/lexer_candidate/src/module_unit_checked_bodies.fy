module module_unit_checked_bodies;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_result_flow as results;
import module_unit_moves as moves;
import module_unit_origin_returns as origins;
import module_unit_origin_diagnostics as diagnostics;
import module_unit_cleanup_policy as policy;
import module_unit_interfaces as interfaces;
import module_signatures as signatures;
import project_path as budget;
import syntax as syntax;
import sha256 as sha;
import std::vec as v;

pub struct Problem {u32 stage;u32 code;usize module_index;usize start;usize end;usize expected;usize actual;usize declaration_start;usize declaration_end;usize move_start;usize move_end;usize header_start;usize header_end;usize field_module;usize field_start;usize field_end;}
struct Stamp {usize bytes;u32[8] digest;}
// The factory owns every body-check result, including cleanup authority. This
// owner does not confer restored-IR, identity or Complete-publication authority.
pub resource Plan {closure::Ready ready;values::Plan typed;moves::Plan moved;policy::Plan cleanup_policy;v::Vec<Stamp> sources;cleanup{}}
pub fn ref values::Plan checked_values(ref Plan p){return ref p.typed;}
pub fn ref moves::Plan checked_moves(ref Plan p){return ref p.moved;}
pub fn ref interfaces::Interfaces checked_interfaces(ref Plan p){return closure::imported(ref p.ready);}
pub fn ref policy::Plan cleanup_policy(ref Plan p){return ref p.cleanup_policy;}
fn Problem failed(u32 stage,u32 code,usize module_index,usize start,usize end,usize expected,usize actual){return Problem{.stage,.code,.module_index,.start,.end,.expected,.actual,.declaration_start=0,.declaration_end=0,.move_start=0,.move_end=0,.header_start=0,.header_end=0,.field_module=0,.field_start=0,.field_end=0};}
pub fn Result<Plan,Problem> build(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop){
    let bound=switch(binding::bind(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{return .err(failed(2,e.code,e.module_index,e.start,e.end,0,0));}};
    let ready=switch(closure::build_unit(sources,metadata,ref bound)){case .ok(p):{p}case .err(e):{return .err(failed(3,e.code,e.module_index,e.start,e.end,0,0));}};
    let typed=switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(e):{return .err(failed(4,e.code,e.module_index,e.start,e.end,e.expected,e.actual));}};
    // Type, ownership and borrow phases retain their independent histories.
    let moved=switch(moves::build(sources,ref typed,0)){case .ok(p):{p}case .err(e):{return .err(failed(5,e.code,e.module_index,e.start,e.end,e.expected,e.actual));}};
    if(moves::count(ref moved)!=0){let e=moves::at(ref moved,0);var Problem p=failed(5,e.code,e.module_index,e.start,e.end,0,0);p.declaration_start=e.declaration_start;p.declaration_end=e.declaration_end;p.move_start=e.move_start;p.move_end=e.move_end;p.header_start=e.header_start;p.header_end=e.header_end;p.field_module=e.field_module;p.field_start=e.field_start;p.field_end=e.field_end;return .err(p);}
    let inferred=switch(origins::build(sources,metadata,ref typed)){case .ok(p):{p}case .err(e):{return .err(failed(7,e.code,e.module_index,e.start,e.end,e.expected,e.actual));}};
    let report=switch(diagnostics::collect(ref inferred)){case .ok(p):{p}case .err(e):{return .err(failed(7,e.code,e.module_index,e.start,e.end,e.expected,e.actual));}};
    var usize i=0;while(i<diagnostics::count(ref report)){if(diagnostics::visible_at(ref report,i)){let e=diagnostics::at(ref report,i);return .err(failed(7,e.code,e.module_index,e.start,e.end,0,0));}i+=1;}
    // Preserve stage 6 and the original move-plus-policy work history.
    let checked=switch(policy::build(sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{return .err(failed(6,e.code,e.module_index,e.start,e.end,0,0));}};
    if(policy::count(ref checked)!=0){let e=policy::at(ref checked,0);var Problem p=failed(6,e.code,e.module_index,e.start,e.end,0,0);p.field_module=e.field_module;p.field_start=e.field_start;p.field_end=e.field_end;return .err(p);}
    let result_flow=results::check(sources,ref typed);if(result_flow.code!=0){return .err(failed(9,result_flow.code,result_flow.module_index,result_flow.start,result_flow.end,0,0));}
    if(v::len<syntax::Parsed>(sources)>512){return .err(failed(6,10,0,0,0,0,0));}var v::Vec<Stamp> stamps=v::empty<Stamp>();i=0;
    while(i<v::len<syntax::Parsed>(sources)){if(!v::push<Stamp>(ref mut stamps,Stamp{.bytes=syntax::source_view(v::at<syntax::Parsed>(sources,i)).len,.digest=signatures::source_digest(metadata,i)}).is_ok){return .err(failed(6,11,i,0,0,0,0));}i+=1;}
    return .ok(Plan{.ready,.typed,.moved,.cleanup_policy=checked,.sources=stamps});
}
// Freshness has a separate caller-owned workspace history; source hashing does
// not consume or reset the completed cleanup policy's 1MiB lowering history.
pub fn Problem validate_sources(ref Plan p,ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    if(!budget::charge(work,0)){return failed(6,10,0,0,0,0,0);}if(v::len<syntax::Parsed>(sources)!=v::len<Stamp>(ref p.sources)){return failed(6,12,0,0,0,0,0);}var usize i=0;
    while(i<v::len<Stamp>(ref p.sources)){let stamp=v::at<Stamp>(ref p.sources,i);let raw=syntax::source_view(v::at<syntax::Parsed>(sources,i));if(!budget::charge(work,raw.len+1)){return failed(6,10,i,0,0,0,0);}if(raw.len!=stamp.bytes){return failed(6,12,i,0,0,0,0);}let digest=sha::digest(raw);var usize j=0;while(j<8){if(digest[j]!=stamp.digest[j]){return failed(6,12,i,0,0,0,0);}j+=1;}i+=1;
    }return failed(0,0,0,0,0,0,0);
}

// A source-authenticated catalog already retains the exact immutable bytes and
// their digest. Reuse that proof rather than hashing every source per unit.
pub fn Problem validate_sources_catalog(ref Plan p,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref mut usize work){
    if(!budget::charge(work,0)){return failed(6,10,0,0,0,0,0);}if(v::len<syntax::Parsed>(sources)!=v::len<Stamp>(ref p.sources) || signatures::module_count(metadata)!=v::len<Stamp>(ref p.sources)){return failed(6,12,0,0,0,0,0);}var usize i=0;
    while(i<v::len<Stamp>(ref p.sources)){let stamp=v::at<Stamp>(ref p.sources,i);let source=v::at<syntax::Parsed>(sources,i);let raw=syntax::source_view(source);if(!budget::charge(work,raw.len+1)){return failed(6,10,i,0,0,0,0);}if(raw.len!=stamp.bytes || !signatures::source_matches(metadata,i,source)){return failed(6,12,i,0,0,0,0);}let digest=signatures::source_digest(metadata,i);var usize j=0;while(j<8){if(digest[j]!=stamp.digest[j]){return failed(6,12,i,0,0,0,0);}j+=1;}i+=1;
    }return failed(0,0,0,0,0,0,0);
}


// Error-only projection retains every typed origin diagnostic and related span.
pub fn Result<diagnostics::Report,Problem> origin_report(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,usize first,usize stop){
    let bound=switch(binding::bind(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{return .err(failed(2,e.code,e.module_index,e.start,e.end,0,0));}};
    let ready=switch(closure::build_unit(sources,metadata,ref bound)){case .ok(p):{p}case .err(e):{return .err(failed(3,e.code,e.module_index,e.start,e.end,0,0));}};
    let typed=switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(e):{return .err(failed(4,e.code,e.module_index,e.start,e.end,e.expected,e.actual));}};
    let moved=switch(moves::build(sources,ref typed,0)){case .ok(p):{p}case .err(e):{return .err(failed(5,e.code,e.module_index,e.start,e.end,e.expected,e.actual));}};
    if(moves::count(ref moved)!=0){let e=moves::at(ref moved,0);return .err(failed(5,e.code,e.module_index,e.start,e.end,0,0));}
    let inferred=switch(origins::build(sources,metadata,ref typed)){case .ok(p):{p}case .err(e):{return .err(failed(7,e.code,e.module_index,e.start,e.end,e.expected,e.actual));}};
    return switch(diagnostics::collect(ref inferred)){case .ok(p):{.ok(p)}case .err(e):{.err(failed(7,e.code,e.module_index,e.start,e.end,e.expected,e.actual))}};
}
