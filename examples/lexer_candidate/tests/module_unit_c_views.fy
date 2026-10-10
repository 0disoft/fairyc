module tests::module_unit_c_views;
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


@test @effects(console) fn void view_c_mutates_owner_and_shares_call_arguments(){
    let p=retained(b"module main;fn void add(ref mut i32[] v){v[0]+=2;v[1]+=2;}fn i32 read(ref i32[] v){return v[0];}fn ref i32[] suffix(ref i32[] v){return v[1..];}@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){var i32[4] row=[1,2,3,4];add(ref mut row[1..3]);assert(row[1]==4 && row[2]==5);{let ref mut i32[] whole=ref mut row;assert(read(whole)==1);whole[0]=7;let ref i32[] tail=suffix(whole);assert(tail.len==3 && tail[0]==4);}unsafe{trace(row[0]);trace(row[1]);trace(row[2]);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void view_c_selects_original_plain_struct_field_storage(){
    let p=retained(b"module main;struct Point{i32 x;i32 y;}struct Box{Point[3] points;}fn void change(ref mut Point[] v){v[0].x+=5;}fn ref Point[] tail(ref Point[] v){return v[1..2];}@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){var Box b=Box{.points=[Point{.x=1,.y=0},Point{.x=2,.y=0},Point{.x=3,.y=0}]};change(ref mut b.points[1..]);{let ref Point[] prefix=ref b.points[..2];assert(prefix.len==2 && prefix[1].x==7);}{let ref mut Point[] whole=ref mut b.points[..];change(whole);let ref Point[] part=tail(whole);assert(part.len==1 && part[0].x==7);}assert(b.points[0].x==6 && b.points[1].x==7);unsafe{trace(b.points[0].x);trace(b.points[1].x);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void view_c_executes_generic_scalar_view_specializations(){
    let p=retained(b"module main;fn void set<T>(ref mut T[] v,T value){v[0]=value;}fn T first<T>(ref T[] v){return v[0];}fn i32 main(){var i32[1] small=[0];var i64[1] wide=[0];var u8[1] bytes=[0];var u32[1] words=[0];var usize[1] sizes=[0];var bool[1] flags=[false];set<i32>(ref mut small,7);set<i64>(ref mut wide,9223372036854775807);set<u8>(ref mut bytes,255);set<u32>(ref mut words,4294967295);set<usize>(ref mut sizes,18446744073709551615);set<bool>(ref mut flags,true);assert(first<i32>(ref small)==7 && first<i64>(ref wide)==9223372036854775807 && first<u8>(ref bytes)==255 && first<u32>(ref words)==4294967295 && first<usize>(ref sizes)==18446744073709551615 && first<bool>(ref flags));return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void view_c_orders_bounds_and_retains_nested_empty_ranges(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn usize start(ref mut i32 count){*count+=1;unsafe{trace(11);}return 1;}@effects(console)fn usize end(ref mut i32 count){*count+=1;unsafe{trace(22);}return 3;}@effects(console)fn i32 main(){let u8[4] row=[1,2,3,4];var i32 count=0;let ref u8[] selected=ref row[start(ref mut count)..end(ref mut count)];assert(count==2 && selected.len==2 && selected[0]==2 && selected[1]==3);let ref u8[] full=ref row;let empty=full[full.len..];assert(empty.is_empty && empty.len==0);let prefix=full[..2];let middle=full[1..3][1..2];assert(prefix.len==2 && prefix[1]==2 && middle.len==1 && middle[0]==3);unsafe{trace(count);trace(33);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void view_c_traps_reversed_slice_without_unwinding(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)resource R{i32 id;cleanup{unsafe{trace(this.id);}}}@effects(console)fn i32 main(){let R owner=R{.id=9};let u8[3] row=[1,2,3];let ref u8[] whole=ref row;unsafe{trace(1);}let invalid=whole[3..1];return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void view_c_traps_upper_bound_beyond_view_extent(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){let u8[3] row=[1,2,3];let ref u8[] whole=ref row;unsafe{trace(2);}let invalid=whole[..4];return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void view_c_traps_index_into_empty_end_view(){
    let p=retained(b"module main;fn i32 main(){let i32[2] row=[1,2];let ref i32[] whole=ref row;let empty=whole[whole.len..];assert(empty.is_empty);return empty[0];}",b"");output(ref p,entry(ref p));
}
