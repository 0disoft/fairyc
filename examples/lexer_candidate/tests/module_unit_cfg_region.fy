module tests::module_unit_cfg_region;
import module_unit_body_ir_model as ir;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_verify as proof;
import module_unit_sdk_sources as sdk_sources;
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
import std::process as process;
import file_source as files;
fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console,filesystem) fn ir::Program retained(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,allocator()).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" at ");discard io::write_usize(e.start);discard io::write(".. ");discard io::write_usize(e.end);discard io::write(" lower\n");assert(false);retained(root,provider)}};
}


@effects(console) fn cfg::Graph lowered(ref ir::Program p,usize frame){return switch(flow::lower(p,frame)){case .ok(g):{g}case .err(e):{discard io::write_u32(e.code);discard io::write(" cfg at ");discard io::write_usize(e.start);discard io::write(".. ");discard io::write_usize(e.end);discard io::write("\n");assert(false);lowered(p,frame)}};}
fn usize declared(ref ir::Program p,usize declaration){var usize i=1;while(i<=ir::count(p)){let f=clean_ir::frame_at(ir::cleanup_ir(p),i);if(f.kind==3 && f.declaration==declaration){return i;}i+=1;}return 0;}
fn usize containing(ref ir::Program p,u32 kind){var usize f=1;while(f<=ir::count(p)){let b=ir::at(p,f);var usize i=0;while(i<ir::node_count(b)){if(ir::node_at(b,i).value.kind==kind){return f;}i+=1;}f+=1;}return 0;}

fn usize with_defer(ref ir::Program p){var usize f=1;while(f<=ir::count(p)){let b=ir::at(p,f);var usize i=0;while(i<ir::statement_count(b)){if(ir::statement_at(b,i).value.kind==13){return f;}i+=1;}f+=1;}return 0;}

@effects(filesystem) fn syntax::Parsed allocator(){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let size=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return allocator();}};let data=switch(files::load(bytes::view(ref path)[..size])){case .ok(b):{b}case .err(_):{assert(false);return allocator();}};return parsed(bytes::view(ref data));}
fn usize selected(ref ir::Program p){var usize i=1;while(i<=ir::count(p)){if(ir::region_count(ir::at(p,i))!=0){return i;}i+=1;}return 0;}
fn usize event(ref ir::Program p,ref cfg::Graph g,cfg::Block n,ref u8[] source){if(n.kind==14){return 100+n.target;}if(n.kind==16){return 200+ir::reset_at(ir::at(p,cfg::frame(g)),n.target-1).region_id;}if(n.kind==5){let o=clean_ir::operand_at(ir::cleanup_ir(p),clean_ir::drop_at(ir::cleanup_ir(p),n.drop-1).operand);let name=source[o.start..o.end];return if(bytes::equal(name,b"inner")){3}else if(bytes::equal(name,b"mid")){2}else if(bytes::equal(name,b"outside")){1}else{999};}return 0;}

@test @effects(console,filesystem) fn void region_return_and_propagation_interleave_owner_drops_with_inner_first_resets(){
    let text=b"module main;import std::alloc as alloc;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void done(i32 n){}fn Result<i32,alloc::AllocError> f(ref mut alloc::Arena a,ref mut alloc::Arena b){let R outside=make();defer done(1);region r in a{let R mid=make();defer done(2);region s in b{let R inner=make();defer done(3);discard s.place(1)?;return .ok(7);}}}";
    let p=retained(text,b"");let frame=selected(ref p);let body=ir::at(ref p,frame);assert(ir::region_count(body)==2 && ir::placement_count(body)==1);let placement=ir::placement_at(body,0);assert(placement.region_id==2 && placement.capability==ir::region_at(body,1).capability);let g=lowered(ref p,frame);let usize[8] expected=[103,102,101,3,202,2,201,1];var u32 mode=0;while(mode<2){var usize id=cfg::entry(ref g);var usize index=0;var usize steps=0;var usize begins=0;var bool ended=false;while(id!=0){assert(steps<512);steps+=1;let n=cfg::at(ref g,id);if(n.kind==15){begins+=1;}let code=event(ref p,ref g,n,text);if(code!=0){assert(index<8 && code==expected[index]);index+=1;}if(n.kind==6 || n.kind==9){ended=true;assert(n.kind==if(mode==0){6}else{9});}id=if(n.kind==4 && mode==1){n.second}else{n.first};}assert(index==8 && begins==2 && ended);mode+=1;}
}
@test @effects(console,filesystem) fn void region_fallthrough_defers_and_drops_inside_before_reset_and_outer_cleanup(){
    let text=b"module main;import std::alloc as alloc;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void done(i32 n){}fn void f(ref mut alloc::Arena a){let R outside=make();defer done(1);region r in a{let R inner=make();defer done(2);discard r.place(3);}}";let p=retained(text,b"");let g=lowered(ref p,selected(ref p));let usize[5] expected=[102,3,201,101,1];var usize id=cfg::entry(ref g);var usize index=0;var usize steps=0;while(id!=0){assert(steps<512);steps+=1;let n=cfg::at(ref g,id);let code=event(ref p,ref g,n,text);if(code!=0){assert(index<5 && code==expected[index]);index+=1;}id=n.first;}assert(index==5);
}
@test @effects(console,filesystem) fn void region_loop_exit_resets_only_crossed_region_and_preserves_outer_scope(){
    let text=b"module main;import std::alloc as alloc;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void done(i32 n){}fn void f(ref mut alloc::Arena a,ref mut alloc::Arena b,bool c){let R outside=make();defer done(1);region r in a{let R mid=make();defer done(2);while(c){region s in b{let R inner=make();defer done(3);if(c){continue;}else{break;}}}}}";let p=retained(text,b"");let g=lowered(ref p,selected(ref p));let usize[8] expected=[103,3,202,102,2,201,101,1];var u32 mode=0;while(mode<2){var usize id=cfg::entry(ref g);var usize index=0;var usize branches=0;var usize steps=0;while(id!=0){assert(steps<512);steps+=1;let n=cfg::at(ref g,id);let code=event(ref p,ref g,n,text);if(code!=0){assert(index<8 && code==expected[index]);index+=1;}if(n.kind==3){branches+=1;if(mode==1 && branches==3){break;}id=if(branches==1 || mode==1){n.first}else{n.second};}else{id=n.first;}}assert(index==if(mode==0){8}else{3});mode+=1;}
}
fn u32 malformed(ref ir::Program p,ref cfg::Graph g,u32 mode){var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();var v::Vec<cfg::Deferred> sites=v::empty<cfg::Deferred>();var v::Vec<cfg::Capture> captures=v::empty<cfg::Capture>();var usize i=1;var usize reset=0;var bool changed=false;
    while(i<=cfg::count(g)){var n=cfg::at(g,i);if(n.kind==16){reset=i;}if(!changed && n.kind==15){if(mode==1){n.target=999999;changed=true;}if(mode==2){n.source=0;changed=true;}if(mode==7){n.kind=7;n.source=0;n.target=0;changed=true;}if(mode==8){n.start+=1;changed=true;}}if(!changed && n.kind==16){if(mode==3){n.target=999999;changed=true;}if(mode==4){n.start+=1;changed=true;}if(mode==5){n.kind=7;n.target=0;changed=true;}}assert(v::push<cfg::Block>(ref mut blocks,n).is_ok);i+=1;}if(mode==6){assert(reset!=0 && v::push<cfg::Block>(ref mut blocks,cfg::at(g,reset)).is_ok);}i=1;while(i<=cfg::deferred_count(g)){assert(v::push<cfg::Deferred>(ref mut sites,cfg::deferred_at(g,i)).is_ok);i+=1;}i=1;while(i<=cfg::capture_count(g)){assert(v::push<cfg::Capture>(ref mut captures,cfg::capture_at(g,i)).is_ok);i+=1;}return switch(cfg::assemble_deferred(p,cfg::frame(g),cfg::entry(g),blocks,sites,captures,cfg::work(g))){case .ok(_):{0}case .err(e):{e.code}};
}
@test @effects(console,filesystem) fn void region_handoff_rejects_missing_duplicate_and_wrong_checkpoint_reset_operations(){let p=retained(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){region r in a{discard r.place(1);}}",b"");let g=lowered(ref p,selected(ref p));assert(malformed(ref p,ref g,0)==0);var u32 mode=1;while(mode<=8){assert(malformed(ref p,ref g,mode)==1001);mode+=1;}}

fn u32 altered_body(ref ir::Program p,u32 mode){let frame=selected(p);let b=ir::at(p,frame);var v::Vec<ir::Node> nodes=v::empty<ir::Node>();var v::Vec<ir::Statement> statements=v::empty<ir::Statement>();var v::Vec<m::Local> locals=v::empty<m::Local>();var v::Vec<m::Invocation> calls=v::empty<m::Invocation>();var v::Vec<m::Argument> arguments=v::empty<m::Argument>();var v::Vec<regions::Scope> scopes=v::empty<regions::Scope>();var v::Vec<regions::Placement> placements=v::empty<regions::Placement>();var v::Vec<regions::Exit> resets=v::empty<regions::Exit>();var usize i=0;
    while(i<ir::node_count(b)){assert(v::push<ir::Node>(ref mut nodes,ir::node_at(b,i)).is_ok);i+=1;}i=0;while(i<ir::statement_count(b)){assert(v::push<ir::Statement>(ref mut statements,ir::statement_at(b,i)).is_ok);i+=1;}i=1;while(i<=ir::local_count(b)){assert(v::push<m::Local>(ref mut locals,ir::local_at(b,i)).is_ok);i+=1;}i=0;while(i<ir::call_count(b)){assert(v::push<m::Invocation>(ref mut calls,ir::call_at(b,i)).is_ok);i+=1;}i=0;while(i<ir::argument_count(b)){assert(v::push<m::Argument>(ref mut arguments,ir::argument_at(b,i)).is_ok);i+=1;}
    i=0;while(i<ir::region_count(b)){var r=ir::region_at(b,i);if(mode==4){r.body_start+=1;}if(mode==8){r.capability=0;}if(mode==9){r.source_mode=1;}if(mode==10){r.parent=i+1;}assert(v::push<regions::Scope>(ref mut scopes,r).is_ok);i+=1;}
    i=0;while(i<ir::placement_count(b)){var x=ir::placement_at(b,i);if(mode==2){x.owner_type=clean_ir::type_count(ir::cleanup_ir(p))+1;}if(mode==3){x.region_id=0;}if(mode==11){x.expression=999999;}if(mode!=1){assert(v::push<regions::Placement>(ref mut placements,x).is_ok);}i+=1;}
    i=0;while(i<ir::reset_count(b)){var e=ir::reset_at(b,i);if(mode==5){e.group=0;}if(mode==6){e.order=1;}if(mode==7){e.kind=0;}assert(v::push<regions::Exit>(ref mut resets,e).is_ok);i+=1;}
    let text=switch(bytes::copy(ir::text_view(b))){case .ok(p):{p}case .err(_):{assert(false);return 0;}};let changed=ir::assemble_body_regions(ir::body(b),ir::source_bytes(b),ir::start(b),ir::end(b),nodes,statements,locals,calls,arguments,scopes,placements,resets,text);var usize work=0;return proof::check_body(ref changed,ir::cleanup_ir(p),frame,ref mut work).code;
}
@test @effects(console,filesystem) fn void region_body_snapshot_rejects_missing_placement_and_invalid_scope_reset_ledgers(){let p=retained(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){region r in a{discard r.place(1);}}",b"");assert(altered_body(ref p,0)==0);var u32 mode=1;while(mode<=11){assert(altered_body(ref p,mode)==1001);mode+=1;}}

@test @effects(console,filesystem) fn void owned_region_program_retains_sdk_bytes_after_parser_release(){
    let p=retained(b"module main;import std::alloc as alloc;fn void f(ref mut alloc::Arena a){region r in a{discard r.place(1);}}",b"");
    let original=allocator();assert(sdk_sources::module_count(ir::sdk_units(ref p))==2 && sdk_sources::count(ir::sdk_units(ref p))==1);
    assert(sdk_sources::find(ir::sdk_units(ref p),1)==1 && bytes::equal(sdk_sources::source_at(ir::sdk_units(ref p),0),syntax::source_view(ref original)));
    assert(proof::check(ref p).code==0);
}
