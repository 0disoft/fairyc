module tests::closure_unit_c;
import module_unit_c_emit as emit;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_body_ir_lower as lower;
import module_unit_checked_bodies as checked;
import c_text as w;
import std::io as io;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn bool contains(ref u8[] text,ref u8[] part){var usize i=0;while(part.len<=text.len && i<=text.len-part.len){if(bytes::equal(text[i..i+part.len],part)){return true;}i+=1;}return false;}
resource Owned {ir::Program program;usize entry;cleanup{}}
fn Result<Owned,u32> retained(ref u8[] text){
    let input=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{return .err(11);}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{return .err(1001);}};
    let source=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{return .err(1001);}};if(syntax::problem_count(ref source)!=0){return .err(syntax::problem_at(ref source,0).code);}
    var usize declaration=0;var usize i=0;while(i<syntax::declaration_count(ref source)){let d=syntax::declaration_at(ref source,i);if(d.kind==3 && bytes::equal(syntax::source_view(ref source)[d.name_start..d.name_end],b"main")){declaration=i+1;}i+=1;}if(declaration==0){return .err(1001);}
    var sources=v::empty<syntax::Parsed>();if(!v::push<syntax::Parsed>(ref mut sources,source).is_ok){return .err(11);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let owned=switch(checked::build(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let p=switch(lower::lower_checked(ref sources,ref owned)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    var usize entry=0;i=1;while(i<=ir::count(ref p)){let f=clean::frame_at(ir::cleanup_ir(ref p),i);if(f.kind==3 && f.module_index==0 && f.declaration==declaration){entry=i;}i+=1;}if(entry==0){return .err(1001);}return .ok(Owned{.program=p,.entry});
}
// Source checking this probe does not execute its generated-C assertions.
pub fn void verify(ref u8[] text,bool closure_value){
    let owned=switch(retained(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=ref owned.program;let entry=owned.entry;
    var bool invoked=false;var usize i=1;while(i<=ir::count(p)){var usize n=0;while(n<ir::call_count(ir::at(p,i))){invoked=invoked || ir::call_at(ir::at(p,i),n).kind==4;n+=1;}i+=1;}
    var bytes::Buffer out=bytes::empty();assert(emit::render(p,entry,ref mut out).code==0);let c=bytes::view(ref out);
    assert(contains(c,b"fy_invoke")==closure_value);assert(contains(c,b"fy_captures[4]")==closure_value);
    if(!closure_value){assert(!contains(c,b"fy_unit_closure_"));assert(switch(emit::emit_closures(p,entry,1,true)){case .ok(_):{false}case .err(e):{e.code==1001}});return;}
    assert(contains(c,b"int32_t const *fy_env,int32_t fy_arg"));assert(contains(c,b".fy_invoke(")==invoked);
    let selected=switch(emit::emit_closures(p,entry,1,true)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let all=switch(emit::emit_closures(p,entry,2,false)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(emit::emission_stats(ref selected).selected<=emit::emission_stats(ref all).selected);
    assert(contains(emit::emission_header(ref selected),b"fy_captures[4]"));
    // Selected constructor adapters and their real lifted bodies are both kept.
    var usize adapters=0;i=0;while(i<emit::emission_unit_count(ref selected)){if(contains(emit::emission_source(ref selected,i),b"(int32_t const *fy_env,int32_t fy_arg)")){adapters+=1;}i+=1;}assert(adapters!=0);
    // Closure admission stays separate from the older aggregate/project modes.
    assert(switch(emit::emit_aggregates(p,entry,1,true)){case .ok(_):{false}case .err(e):{e.code==1001}});
}

// A decimal length plus exact bytes makes embedded newlines harmless. All
// emission succeeds before stdout publication; diagnostics use stderr only.
fn bool part(ref mut bytes::Buffer packet,ref mut w::State status,ref u8[] text){return w::number(packet,status,text.len) && w::text(packet,status,b"\n") && w::text(packet,status,text);}
@effects(console) fn void member(ref u8[] text){assert(io::write_usize(text.len).is_ok && io::write("\n").is_ok && io::write_bytes(text).is_ok);}
@effects(console) pub fn void produce(ref u8[] name,ref u8[] text){
    let owned=switch(retained(text)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(name);discard io::write_stderr(" unit C lower code=");discard io::write_stderr_u32(e);discard io::write_stderr("\n");assert(false);return;}};
    var bytes::Buffer single=bytes::empty();let error=emit::render(ref owned.program,owned.entry,ref mut single);if(error.code!=0){discard io::write_stderr_bytes(name);discard io::write_stderr(" unit C single code=");discard io::write_stderr_u32(error.code);assert(false);return;}
    let split=switch(emit::emit_closures(ref owned.program,owned.entry,1,true)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(name);discard io::write_stderr(" unit C partition code=");discard io::write_stderr_u32(e.code);assert(false);return;}};
    var bytes::Buffer packet=bytes::empty();var w::State status=w::State{.code=0};assert(w::text(ref mut packet,ref mut status,b"FYCPARTS1\n") && part(ref mut packet,ref mut status,emit::emission_header(ref split)) && part(ref mut packet,ref mut status,emit::emission_literals(ref split)) && w::number(ref mut packet,ref mut status,emit::emission_unit_count(ref split)) && w::text(ref mut packet,ref mut status,b"\n"));
    var usize i=0;while(i<emit::emission_unit_count(ref split)){assert(part(ref mut packet,ref mut status,emit::emission_source(ref split,i)));i+=1;}
    member(bytes::view(ref single));member(bytes::view(ref packet));
}
