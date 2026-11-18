module tests::copied_closure_calls;
import scanner as scanner;
import syntax as syntax;
import module_signatures as signatures;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_unit_calls as calls;
import module_unit_checked_bodies as checking;
import std::vec as v;
import std::bytes as bytes;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
@effects(console) pub fn void checked(ref u8[] raw,usize count){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_stderr("signature ");discard io::write_stderr_u32(e.code);assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{discard io::write_stderr("bind ");discard io::write_stderr_u32(e.code);assert(false);return;}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_stderr("calls ");discard io::write_stderr_u32(e.code);discard io::write_stderr_usize(e.start);discard io::write_stderr("/");discard io::write_stderr_usize(e.end);assert(false);return;}};
    let checked=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_stderr("types ");discard io::write_stderr_u32(e.code);discard io::write_stderr_usize(e.start);discard io::write_stderr("/");discard io::write_stderr_usize(e.end);assert(false);return;}};assert((count==0 || values::count(ref checked)==count) && values::unresolved(ref checked)==0);
}

fn bool rejected(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return false;}};let stream=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return false;}};return switch(syntax::parse(stream,512)){case .ok(p):{syntax::problem_count(ref p)!=0}case .err(_):{true}};}
@effects(console) pub fn void reject(ref u8[] raw){if(!rejected(raw)){discard io::write_stderr_bytes(raw);assert(false);}}

// Use the actual authenticated SDK source, never a fabricated Vec declaration.
@effects(console) pub fn u32 checked_vector(ref u8[] raw,ref u8[] sdk){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(sdk)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_stderr("vector metadata ");discard io::write_stderr_u32(e.code);assert(false);return 99;}};assert(signatures::sdk_vector_source(ref metadata,1));
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{discard io::write_stderr("vector bind ");discard io::write_stderr_u32(e.code);assert(false);return 99;}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_stderr("vector calls ");discard io::write_stderr_u32(e.code);assert(false);return 99;}};assert(calls::count(closure::call_plan(ref ready))>0);
    let result=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_stderr("vector Values ");discard io::write_stderr_u32(e.code);discard io::write_stderr("/");discard io::write_stderr_usize(e.start);discard io::write_stderr("/");discard io::write_stderr_usize(e.end);assert(false);return e.code;}};assert(values::unresolved(ref result)==0);
    switch(checking::build(ref sources,ref metadata,0,1)){case .ok(_):{}case .err(e):{discard io::write_stderr("vector checking ");discard io::write_stderr_u32(e.stage);discard io::write_stderr("/");discard io::write_stderr_u32(e.code);discard io::write_stderr("/");discard io::write_stderr_usize(e.start);discard io::write_stderr("/");discard io::write_stderr_usize(e.end);assert(false);}};return 0;
}

@effects(console) pub fn void rejected_vector(ref u8[] raw,ref u8[] sdk,u32 code){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(sdk)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(checking::build(ref sources,ref metadata,0,1)){case .ok(_):{assert(false);}case .err(e):{if(e.code!=code){discard io::write_stderr("vector refusal ");discard io::write_stderr_u32(e.code);discard io::write_stderr(" expected ");discard io::write_stderr_u32(code);}assert(e.code==code);}};
}

pub fn void reject_typed(ref u8[] raw){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{return;}};let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{return;}};assert(switch(values::build(ref sources,ref metadata,ref ready)){case .ok(_):{false}case .err(_):{true}});}
