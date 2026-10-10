module tests::module_unit_sdk_authority;
import module_unit_sdk_authority as authority;
import module_unit_sdk_sources as sdk_sources;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lower as lower;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as m;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import module_unit_interfaces as interfaces;
import scanner as scanner;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(filesystem) fn bytes::Buffer input(usize index){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(index,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return bytes::empty();}};return switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);bytes::empty()}};}
@effects(console) fn void error(u32 code,usize start,usize end){discard io::write_u32(code);discard io::write(" at ");discard io::write_usize(start);discard io::write("..");discard io::write_usize(end);discard io::write("\n");assert(false);}
@effects(console) fn usize authorized(ref ir::Program p,usize index){var usize work=0;return switch(authority::rule(p,index,ref mut work)){case .ok(r):{r}case .err(e):{error(e.code,e.start,e.end);0}};}
@effects(console) fn ir::Program retained(ref u8[] root,ref u8[] sdk){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok && v::push<syntax::Parsed>(ref mut sources,parsed(sdk)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,sdk);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,2)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,sdk);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,sdk);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,sdk);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);return retained(root,sdk);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{error(e.code,e.start,e.end);retained(root,sdk)}};
}
@effects(console) fn usize selected(ref ir::Program p,ref u8[] link){var usize i=0;while(i<ir::callable_count(p)){if(bytes::equal(ir::linker_name(p,i),link)){return i+1;}i+=1;}i=0;while(i<ir::callable_count(p)){let c=ir::callable_at(p,i);discard io::write_usize(c.module_index);discard io::write("/");discard io::write_usize(c.declaration);discard io::write(" ");discard io::write_bytes(ir::linker_name(p,i));discard io::write("\n");i+=1;}assert(false);return 0;}
fn u32 mutation(ref ir::Program p,ref syntax::Parsed original,usize sdk_id,usize id,u32 mode){var c=ir::callable_at(p,id-1);let link=ir::linker_name(p,id-1);
    if(mode==1){c.module_index=0;}if(mode==2){c.declaration=0;}if(mode==3){c.declaration=syntax::declaration_count(original)+1;}if(mode==4){c.foreign=false;}if(mode==5){c.unsafe_call=false;}if(mode==6){c.template=true;}if(mode==7){c.effects^=1;}if(mode==8){c.must_use=!c.must_use;}
    if(mode==9){var usize i=1;while(i<=clean::type_count(ir::cleanup_ir(p))){let t=clean::type_at(ir::cleanup_ir(p),i);if(t.kind==1 && t.scalar==1){c.result=i;break;}i+=1;}}
    if(mode==10){c.parameter_count+=1;}if(mode==11){c.parameter_start=ir::parameter_count(p)+1;}
    if(mode==15){assert(c.parameter_count==1);let expected=ir::parameter_at(p,c.parameter_start);var usize i=0;var bool found=false;while(i<ir::parameter_count(p)){if(ir::parameter_at(p,i)!=expected){c.parameter_start=i;found=true;break;}i+=1;}assert(found);}
    var usize work=if(mode==14){1048576}else{0};return switch(authority::check_foreign(p,original,1,if(mode==13){0}else{sdk_id},c,if(mode==12){b"fy_forged_v0"}else{link},ref mut work)){case .ok(_):{0}case .err(e):{e.code}};
}
@test @effects(console,filesystem) fn void process_scalar_sdk_declaration_and_signature_are_reauthenticated(){let sdk=input(1);let p=retained(b"module main;import std::process as p;fn i32 main(){discard p::arg_count();return 0;}",bytes::view(ref sdk));let original=parsed(bytes::view(ref sdk));let id=selected(ref p,b"fy_std_process_arg_count_v0");assert(authorized(ref p,id-1)==1);
    assert(mutation(ref p,ref original,6,id,0)==0);var u32 mode=1;while(mode<=14){let actual=mutation(ref p,ref original,6,id,mode);let u32 expected=if(mode==14){1002}else{1001};if(actual!=expected){error(actual,mode as usize,expected as usize);}mode+=1;}
    var bytes::Buffer forged=bytes::empty();assert(bytes::append(ref mut forged,bytes::view(ref sdk)).is_ok && bytes::append_byte(ref mut forged,32).is_ok);let changed=parsed(bytes::view(ref forged));assert(mutation(ref p,ref changed,6,id,0)==1001);
}
@test @effects(console,filesystem) fn void arena_pointer_sdk_declarations_are_reauthenticated_after_parser_release(){let sdk=input(2);let p=retained(b"module main;import std::alloc as alloc;fn i32 main(){return switch(alloc::create()){case .ok(_):{0}case .err(_):{1}};}",bytes::view(ref sdk));let original=parsed(bytes::view(ref sdk));let init=selected(ref p,b"fy_std_arena_initialize_v0");let drop=selected(ref p,b"fy_std_arena_drop_v0");var usize work=0;assert(switch(authority::rule(ref p,init-1,ref mut work)){case .ok(r):{r==70}case .err(_):{false}});work=0;assert(switch(authority::rule(ref p,drop-1,ref mut work)){case .ok(r):{r==71}case .err(_):{false}});assert(mutation(ref p,ref original,1,init,0)==0 && mutation(ref p,ref original,1,init,10)==1001 && mutation(ref p,ref original,1,init,15)==1001);assert(sdk_sources::count(ir::sdk_units(ref p))==1);}
@test @effects(console,filesystem) fn void console_record_result_sdk_signature_keeps_exact_nominal_identity(){let sdk=input(3);let p=retained(b"module main;import std::io as io;@effects(console)fn i32 main(){discard io::write_u32(7);return 0;}",bytes::view(ref sdk));let original=parsed(bytes::view(ref sdk));let id=selected(ref p,b"fy_std_stdout_write_u32_v0");var usize work=0;assert(switch(authority::rule(ref p,id-1,ref mut work)){case .ok(r):{r==26}case .err(_):{false}});assert(mutation(ref p,ref original,4,id,0)==0 && mutation(ref p,ref original,4,id,9)==1001);}
