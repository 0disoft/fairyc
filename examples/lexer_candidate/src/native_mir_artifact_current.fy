module native_mir_artifact_current;
import native_mir_artifact_decode as decoder;
import native_mir_artifact_encode as encoder;
import candidate_artifact_wire as wire;
import candidate_limits as limits;
import project_cache_encoding as budget;
import scanner as scanner;
import syntax as syntax;
import name_resolution as names;
import type_check as typing;
import hir_lower as hir;
import hir_model as h;
import mir_lower as mir;
import mir_model as m;
import std::bytes as bytes;

// Errors match the private codec: 1001 invalid/stale source or MIR, 1002
// bound/work, 1003 allocation. Caller work is a shared, never-reset 16MiB ledger.
// Frontend APIs retain their own depth/arena/pass limits. Their admission fees
// below count input bytes and arena rows, NOT unobservable internal CPU steps.
// All fees precede their phase, including failures; comparisons pay len+1.
fn u32 frontend_code(i32 code){return if(code==3){1003}else if(code==4){1002}else{1001};}
fn u32 lower_code(u32 code){return if(code==2){1002}else if(code==3){1003}else{1001};}

// Drop this detached owner before allocating the current-source frontend. Its
// verifier proves structure/flow only; no saved helper/body becomes authority.
fn u32 stored_source(ref u8[] raw,ref u8[] current,usize module_index,ref mut usize work){
    let saved=switch(decoder::decode(raw,work)){case .ok(p):{p}case .err(code):{return code;}};
    if(m::module_index(ref saved)!=module_index || m::source_view(ref saved).len!=current.len){return 1001;}
    if(!budget::charge(work,current.len+1)){return 1002;}
    return if(bytes::equal(m::source_view(ref saved),current)){0}else{1001};
}

fn usize syntax_rows(ref syntax::Parsed p){
    return syntax::declaration_count(p)+syntax::expression_count(p)+syntax::statement_count(p)
        +syntax::entry_count(p)+syntax::type_count(p)+syntax::field_count(p)
        +syntax::attribute_count(p)+syntax::attribute_argument_count(p);
}

fn Result<m::Program,u32> current_program(ref u8[] current,usize module_index,ref mut usize work){
    if(!budget::charge(work,current.len+1)){return .err(1002);}
    let input=switch(bytes::copy(current)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    if(!budget::charge(work,current.len+1)){return .err(1002);}
    let stream=switch(scanner::collect(input,false,limits::program_bytes(),1048576,256,16777216)){
        case .ok(p):{p}case .err(code):{return .err(frontend_code(code));}
    };
    if(!budget::charge(work,current.len+scanner::token_count(ref stream)+1)){return .err(1002);}
    // Use the executable parser, including private closure preparation from
    // original bytes. Never parse a saved helper or synthesize its identity.
    let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(code):{return .err(frontend_code(code));}};
    if(syntax::problem_count(ref parsed)!=0){return .err(1001);}
    if(!budget::charge(work,current.len+syntax_rows(ref parsed)+1)){return .err(1002);}
    let resolved=names::resolve(parsed);
    let name_code=names::problem(ref resolved).code;
    if(name_code!=0){return .err(if(name_code==4){1002}else if(name_code==5){1003}else{1001});}
    let rows=syntax_rows(names::parsed(ref resolved));
    if(!budget::charge(work,rows+names::symbol_count(ref resolved)+names::reference_count(ref resolved)+1)){return .err(1002);}
    let typed=typing::check(resolved);
    let type_code=typing::problem(ref typed).code;
    if(type_code!=0){return .err(if(type_code==10){1002}else if(type_code==12){1003}else{1001});}
    if(!budget::charge(work,current.len+rows+names::symbol_count(typing::resolved(ref typed))+1)){return .err(1002);}
    let high=switch(hir::lower(typed,module_index)){case .ok(p):{p}case .err(e):{return .err(lower_code(e.code));}};
    if(!budget::charge(work,h::source_view(ref high).len+h::binding_count(ref high)+h::signature_count(ref high)
        +h::function_count(ref high)+h::expression_count(ref high)+h::statement_count(ref high)
        +h::block_count(ref high)+h::edge_count(ref high)+h::drop_count(ref high)+1)){return .err(1002);}
    // mir::lower assembles and runs mir_verify before publishing this owner.
    return switch(mir::lower(high)){case .ok(p):{.ok(p)}case .err(e):{.err(lower_code(e.code))}};
}

// current is caller-owned ORIGINAL source, not a decoded or rewritten snapshot.
// Rebuild safe lifted helpers, captures, signatures, effects and bodies through
// the source pipeline, then require every canonical byte (including the exact
// FYNMIR00/FYNMIR01 feature gate) to match. Return only the newly checked owner.
pub fn Result<m::Program,u32> restore(ref u8[] raw,ref u8[] current,usize module_index,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}
    if(current.len>limits::program_bytes() || raw.len>wire::byte_limit()){return .err(1002);}
    if(module_index>=limits::source_files()){return .err(1001);}
    let code=stored_source(raw,current,module_index,work);if(code!=0){return .err(code);}
    let trusted=switch(current_program(current,module_index,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let canonical=switch(encoder::encode(ref trusted,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let expected=wire::content(ref canonical);
    if(expected.len!=raw.len){return .err(1001);}
    if(!budget::charge(work,raw.len+1)){return .err(1002);}
    if(!bytes::equal(expected,raw)){return .err(1001);}
    return .ok(trusted);
}
