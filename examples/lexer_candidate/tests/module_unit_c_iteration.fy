module tests::module_unit_c_iteration;
import module_unit_c_emit as emit;
import module_unit_body_ir_model as ir;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_cfg_iteration as iteration;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_moves as moves;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import module_types as types;
import module_unit_value_model as m;
import module_unit_region_model as regions;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn ir::Program retained(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,v::len<syntax::Parsed>(ref sources))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" at ");discard io::write_usize(e.start);discard io::write(".. ");discard io::write_usize(e.end);discard io::write(" lower\n");assert(false);retained(root,provider)}};
}
fn usize entry(ref ir::Program p){var usize found=0;var usize i=1;while(i<=ir::count(p)){let f=clean_ir::frame_at(ir::cleanup_ir(p),i);let type=clean_ir::type_at(ir::cleanup_ir(p),f.result);if(f.module_index==0 && f.kind==3 && type.kind==1 && type.scalar==1){var usize j=1;var usize params=0;let b=ir::at(p,i);while(j<=ir::local_count(b)){if(ir::local_at(b,j).kind==1){params+=1;}j+=1;}if(params==0){assert(found==0);found=i;}}i+=1;}return found;}
@effects(console) fn void output(ref ir::Program p,usize frame){var bytes::Buffer c=bytes::empty();let e=emit::render(p,frame,ref mut c);if(e.code!=0){discard io::write_u32(e.code);discard io::write(" C at ");discard io::write_usize(e.start);discard io::write("\n");}assert(e.code==0);assert(bytes::view(ref c).len>100);assert(io::write_bytes(bytes::view(ref c)).is_ok);}

@test @effects(console) fn void byte_loop_c_evaluates_view_once_empty_and_nested_control(){let p=retained(b"module main;fn ref u8[] select(ref mut usize calls,ref u8[] data){*calls+=1;return data;}fn usize extent(ref mut usize calls){*calls+=1;return 3;}fn i32 first(ref u8[] data){for(let u8 byte in data){return byte as i32;}return -1;}fn i32 main(){let ref u8[] data=b\"\\x01\\x02\\x03\";var usize calls=0;var usize total=0;let ref u8[] selected=select(ref mut calls,data);for(let u8 byte in selected){total+=byte as usize;}assert(calls==1 && total==6);let ref u8[] empty=b\"\";for(let u8 byte in empty){assert(false);}var usize sliced=0;for(let u8 byte in data[1..extent(ref mut calls)]){sliced+=byte as usize;}assert(calls==2 && sliced==5 && first(data)==1 && first(b\"\")==-1);var usize nested=0;let ref u8[] outer_bytes=b\"\\x01\\x02\";for(let u8 outer in outer_bytes){for(let u8 inner in data){if(inner==2){continue;}nested+=(outer as usize)+(inner as usize);if(inner==3){break;}}}assert(nested==14);return 0;}",b"");output(ref p,entry(ref p));}
@test @effects(console) fn void byte_loop_c_preserves_iteration_defer_and_owner_cleanup(){let p=retained(b"module main;@link_name(\"test_iteration_report\")unsafe extern C fn void report(i32 n);fn void show(i32 n){unsafe{report(n);}}resource R{i32 n;cleanup{show(this.n);}}fn i32 main(){let R outer=R{.n=9};let ref u8[] data=b\"\\x01\\x02\\x03\\x04\";for(let u8 byte in data){let i32 n=byte as i32;let R owner=R{.n=n};defer show(100+n);if(n==2){continue;}if(n==3){break;}}return 0;}",b"");output(ref p,entry(ref p));}
@test @effects(console) fn void byte_loop_c_rejects_forged_iteration_cfg(){let p=retained(b"module main;fn i32 main(){var usize count=0;let ref u8[] data=b\"abc\";for(let u8 byte in data){count+=byte as usize;}assert(count==294);return 0;}",b"");let frame=entry(ref p);let g=switch(flow::lower(ref p,frame)){case .ok(g):{g}case .err(_):{assert(false);return;}};assert(cfg::check(ref p,ref g).code==0);var usize capture=0;var usize step=0;var usize i=1;while(i<=cfg::count(ref g)){let n=cfg::at(ref g,i);if(n.kind==20){capture=i;}if(n.kind==21){step=i;}i+=1;}assert(capture!=0 && step!=0);var usize mode=0;while(mode<6){var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();i=1;while(i<=cfg::count(ref g)){var n=cfg::at(ref g,i);if(i==capture){if(mode==0){n.source=0;}if(mode==1){n.target=0;}if(mode==2){n.kind=7;n.source=0;n.target=0;}if(mode==5){n.first=cfg::at(ref g,step).second;}}assert(v::push<cfg::Block>(ref mut blocks,n).is_ok);i+=1;}if(mode==4){assert(v::push<cfg::Block>(ref mut blocks,cfg::at(ref g,capture)).is_ok);}let checked=cfg::assemble(ref p,frame,if(mode==3){step}else{cfg::entry(ref g)},blocks,0);assert(switch(checked){case .ok(_):{false}case .err(e):{e.code==1001}});mode+=1;}var usize work=1048576;assert(iteration::check(ref p,ref g,ref mut work).code==1002);}
