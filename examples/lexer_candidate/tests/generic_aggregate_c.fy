module tests::generic_aggregate_c;
import module_unit_body_ir_model as body_ir;
import module_unit_c_emit as emit;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_aggregate as aggregate;
import c_text as w;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) pub fn void verify(ref body_ir::Program p,ref v::Vec<syntax::Parsed> sources){
 var usize entry=0;var usize i=1;while(i<=body_ir::count(p)){let f=clean::frame_at(body_ir::cleanup_ir(p),i);let source=v::at<syntax::Parsed>(sources,f.module_index);let d=syntax::declaration_at(source,f.declaration-1);if(f.kind==3 && bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],b"main")){assert(entry==0);entry=i;}i+=1;}assert(entry!=0);
 // Argument tuples can never acquire a private C value spelling.
 i=1;while(i<=clean::type_count(body_ir::cleanup_ir(p))){if(clean::type_at(body_ir::cleanup_ir(p),i).kind==22){var text=bytes::empty();var w::State state=w::State{.code=0};assert(!aggregate::type(p,i,ref mut text,ref mut state) && state.code==1004);}i+=1;}
 var output=bytes::empty();let code=emit::render(p,entry,ref mut output);if(code.code!=0){discard io::write_stderr("aggregate C ");discard io::write_stderr_u32(code.code);discard io::write_stderr(" at ");discard io::write_stderr_usize(code.start);}assert(code.code==0);
 assert(io::write_usize(bytes::len(ref output)).is_ok && io::write("\n").is_ok && io::write_bytes(bytes::view(ref output)).is_ok);
}
