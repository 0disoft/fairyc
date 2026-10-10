module tests::module_unit_c_references;
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
@test @effects(console) fn void reference_c_snapshots_root_and_projected_values_before_rhs_mutation(){
    let p=retained(b"module main;struct Box{i32 value;}fn i32 bump(ref mut i32 n){*n=10;return 2;}fn i32 box_bump(ref mut Box n){n.value=40;return 2;}@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){var i32 n=1;n+=bump(ref mut n);assert(n==3);var Box box=Box{.value=4};box.value+=box_bump(ref mut box);assert(box.value==6);unsafe{trace(n);trace(box.value);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void reference_c_mutates_borrowed_resource_without_dropping_the_reference(){
    let p=retained(b"module main;@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)resource R{i32 id;cleanup{unsafe{trace(this.id);}}}fn void adjust(ref mut R owner){owner.id+=2;}@effects(console)fn i32 main(){var R r=R{.id=1};adjust(ref mut r);let ref R view=ref r;assert(view.id==3);unsafe{trace(10);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void reference_c_retains_shared_projection_and_generic_reference_returns(){
    let p=retained(b"module main;struct Box{i32 value;}fn ref i32 select(ref Box n){return ref n.value;}fn ref T identity<T>(ref T n){return n;}fn i32 main(){let Box b=Box{.value=7};let ref i32 chosen=select(ref b);assert(*chosen==7);let i32 n=9;let ref i32 value=identity<i32>(ref n);assert(*value==9);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void reference_c_reborrows_fields_from_shared_and_mutable_call_results(){
    let p=retained(b"module main;struct Item{i32 n;}fn ref Item alias(ref Item p){return p;}fn ref mut Item exclusive(ref mut Item p){return p;}fn ref i32 shared(ref Item p){return ref alias(p).n;}fn ref mut i32 mutable(ref mut Item p){return ref mut exclusive(p).n;}fn i32 main(){var Item value=Item{.n=7};let ref i32 first=shared(ref value);assert(*first==7);let ref mut i32 second=mutable(ref mut value);*second=9;assert(*second==9);assert(value.n==9);let ref i32 third=ref alias(ref value).n;assert(*third==9);return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void reference_c_snapshots_indirect_compound_target_before_aliasing_rhs(){
    let p=retained(b"module main;fn i32 mutate(ref mut i32 n){*n=20;return 4;}fn i32 compound(ref mut i32 n){*n+=mutate(n);return *n;}@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){var i32 n=1;assert(compound(ref mut n)==5);assert(n==5);unsafe{trace(n);}return 0;}",b"");output(ref p,entry(ref p));
}
@test @effects(console) fn void reference_c_allows_sequential_mutable_and_shared_scopes(){
    let p=retained(b"module main;fn void write(ref mut i32 n,i32 next){*n=next;}@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 main(){var i32 n=0;write(ref mut n,5);{let ref i32 borrow=ref n;assert(*borrow==5);}write(ref mut n,8);assert(n==8);unsafe{trace(n);}return 0;}",b"");output(ref p,entry(ref p));
}
fn u32 mutation(ref ir::Program p,ref cfg::Graph g,u32 mode){
    var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();var bool changed=false;var usize i=1;while(i<=cfg::count(g)){var n=cfg::at(g,i);if(!changed && n.kind==17 && mode!=0){changed=true;if(mode==1){n.source=0;}else if(mode==2){n.target=0;}else if(mode==3){n.start+=1;}else if(mode==4){n.kind=7;n.source=0;n.target=0;}else if(mode==5){assert(v::push<cfg::Block>(ref mut blocks,n).is_ok);}}assert(v::push<cfg::Block>(ref mut blocks,n).is_ok);i+=1;}
    assert(mode==0 || changed);return switch(cfg::assemble(p,cfg::frame(g),cfg::entry(g),blocks,cfg::work(g))){case .ok(_):{0}case .err(e):{e.code}};
}
@test @effects(console) fn void reference_c_rejects_missing_duplicate_and_malformed_compound_snapshots(){
    let p=retained(b"module main;fn i32 main(){var i32 n=1;n+=2;return n;}",b"");let g=switch(flow::lower(ref p,entry(ref p))){case .ok(g):{g}case .err(_):{assert(false);return;}};assert(mutation(ref p,ref g,0)==0);var u32 mode=1;while(mode<=5){assert(mutation(ref p,ref g,mode)==1001);mode+=1;}
}
