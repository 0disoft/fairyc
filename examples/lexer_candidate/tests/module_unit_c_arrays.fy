module tests::module_unit_c_arrays;
import module_unit_c_emit as emit;
import module_unit_body_ir_model as ir;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
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

@test @effects(console) fn void array_c_copies_values_across_calls_and_returns(){
    let p=retained(b"module main;import helper as h;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){let i32[3] row=[1,2,3];var i32[3] copy=h::echo(row);copy[0]=9;let i32[3] restored=h::echo(row);var i32 total=0;let usize extent=row.len;for(var usize i=0;i<extent;i+=1){total+=row[i];}assert(row[0]==1 && copy[0]==9 && restored[2]==3 && total==6 && h::identity<i32>(row[0])==1);unsafe{trace(row[0]);trace(copy[0]);trace(restored[2]);trace(total);}return 0;}",b"module helper;pub fn i32[3] echo(i32[3] row){return row;}pub fn T identity<T>(T row){return row;}");output(ref p,entry(ref p));
}
@test @effects(console) fn void array_c_evaluates_repeated_values_once_and_orders_literal_effects(){
    let p=retained(b"module main;struct Point{i32 x;i32 y;}struct Packet{i32[2] data;}fn i32 next(ref mut i32 n){*n+=1;return *n;}@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){var i32 n=0;let i32[3] first=[next(ref mut n),next(ref mut n),next(ref mut n)];assert(first[0]==1 && first[1]==2 && first[2]==3);let i32[3] repeated=[next(ref mut n);3];assert(n==4 && repeated[0]==4 && repeated[2]==4);var Point[2] points=[Point{.x=1,.y=2};2];points[0].x=9;assert(points[1].x==1);let Packet packet=Packet{.data=[7,8]};assert(packet.data[1]==8);let i64[4096] large=[9223372036854775807;4096];assert(large.len==4096 && large[4095]==9223372036854775807);let bool[2] flags=[true,false];let u8[2] bytes=[255,0];let u32[1] unsigned=[4294967295];let usize[1] sizes=[18446744073709551615];assert(flags[0] && !flags[1] && bytes[0]==255 && unsigned[0]==4294967295 && sizes[0]==18446744073709551615);unsafe{trace(n);trace(first[0]);trace(first[2]);trace(repeated[1]);trace(points[0].x);trace(points[1].x);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void array_c_preserves_indexed_storage_and_old_values_before_rhs(){
    let p=retained(b"module main;fn usize select(ref mut usize calls){*calls+=1;return 1;}fn i32 mutate(ref mut i32 n){*n=40;return 3;}fn void set(ref mut i32 n){*n=8;}@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);struct Container{i32[2] values;}@effects(console)resource R{i32 id;cleanup{unsafe{trace(this.id);}}}@effects(console)fn i32 main(){var i32[3] row=[1,2,3];var usize calls=0;row[select(ref mut calls)]+=mutate(ref mut row[1]);assert(row[1]==5 && calls==1);var usize index=0;row[index]={index=2;77};assert(row[0]==77 && row[2]==3);set(ref mut row[2]);{let ref i32 selected=ref row[2];assert(*selected==8);}var Container r=Container{.values=[6,7]};let R owner=R{.id=8};r.values[0]+=2;assert(r.values[0]==8);unsafe{trace(row[0]);trace(row[1]);trace(row[2]);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void array_c_traps_read_at_extent(){
    let p=retained(b"module main;fn i32 main(){let i32[2] row=[1,2];var usize index=row.len;return row[index];}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void array_c_checks_target_bounds_before_rhs_effects(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn void signal(i32 n){unsafe{trace(n);}}@effects(console)fn i32 rhs(i32 n){unsafe{trace(99);}return n;}@effects(console)resource R{i32 id;cleanup{unsafe{trace(this.id);}}}@effects(console)fn i32 main(){let R owner=R{.id=5};defer signal(88);var i32[2] row=[1,2];var usize index=row.len;unsafe{trace(1);}row[index]=rhs(9);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void array_c_reads_borrowed_fields_without_recursive_stack_copies(){
    let p=retained(b"module main;struct Context{usize[1024] values;usize n;}fn usize recurse(ref Context c,usize depth){var usize total=0;total+=c.values[0];total+=c.values[1];total+=c.values[2];total+=c.values[3];total+=c.values[4];total+=c.values[5];total+=c.values[6];total+=c.values[7];total+=c.values[8];total+=c.values[9];total+=c.values[10];total+=c.values[11];total+=c.values[12];total+=c.values[13];total+=c.values[14];total+=c.values[15];total+=c.values[16];total+=c.values[17];total+=c.values[18];total+=c.values[19];total+=c.values[20];total+=c.values[21];total+=c.values[22];total+=c.values[23];if(depth==0){return total+c.n;}return recurse(c,depth-1)+1;}fn i32 main(){let Context c=Context{.values=[0;1024],.n=42};assert(recurse(ref c,32)==74);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void array_c_keeps_borrowed_field_snapshot_before_effectful_index(){
    let p=retained(b"module main;struct Context{usize[2] values;}fn usize index(ref mut Context c){c.values[0]=99;return 0;}fn usize read(ref mut Context c){return c.values[index(c)];}fn i32 main(){var Context c=Context{.values=[1,2]};assert(read(ref mut c)==1);assert(c.values[0]==99);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void array_c_traps_borrowed_field_index_at_extent(){
    let p=retained(b"module main;struct Context{usize[2] values;}fn usize read(ref Context c,usize index){return c.values[index];}fn i32 main(){let Context c=Context{.values=[1,2]};discard read(ref c,2);return 0;}",b"");output(ref p,entry(ref p));
}
