module tests::module_project_frames_capacity;
import module_unit_c_callable_index as contracts;
import module_unit_c_frame_index as targets;
import module_unit_cleanup_mir_model as clean;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_verify as proof;
import module_project_frames as frames;
import module_project_types as graph;
import module_types as types;
import module_signatures as signatures;
import module_unit_body_ir_model as ir;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(bytes::Buffer source){let s=switch(scanner::collect(source,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}

@test @effects(console) fn void ordinary_project_frames_above_one_thousand_keep_distinct_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize module_index=0;
    while(module_index<9){var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module p").is_ok);assert(bytes::append_usize(ref mut text,module_index).is_ok);assert(bytes::append(ref mut text,b";").is_ok);var usize i=0;while(i<120){assert(bytes::append(ref mut text,b"pub fn void f").is_ok);assert(bytes::append_usize(ref mut text,i).is_ok);assert(bytes::append(ref mut text,b"(){}").is_ok);i+=1;}assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);module_index+=1;}
    module_index=0;while(module_index<9){var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module q").is_ok);assert(bytes::append_usize(ref mut text,module_index).is_ok);assert(bytes::append(ref mut text,b";import p").is_ok);assert(bytes::append_usize(ref mut text,module_index).is_ok);assert(bytes::append(ref mut text,b" as p;pub fn void run(){").is_ok);var usize i=0;while(i<120){assert(bytes::append(ref mut text,b"p::f").is_ok);assert(bytes::append_usize(ref mut text,i).is_ok);assert(bytes::append(ref mut text,b"();").is_ok);i+=1;}if(module_index==0){assert(bytes::append(ref mut text,b"let i32 n=1;unsafe{").is_ok);var usize cast=0;while(cast<1000){assert(bytes::append(ref mut text,b"discard ref n as ptr i32;").is_ok);cast+=1;}assert(bytes::append(ref mut text,b"};").is_ok);}assert(bytes::append(ref mut text,b"}").is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);module_index+=1;}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var v::Vec<ir::Program> units=v::empty<ir::Program>();var v::Vec<types::Graph> inputs=v::empty<types::Graph>();module_index=0;
    while(module_index<18){let unit=switch(compile::range(ref sources,ref metadata,module_index,module_index+1)){case .ok(p):{p}case .err(e):{discard io::write("compile ");discard io::write_u32(e.code);assert(false);return;}};assert(ir::count(ref unit)==if(module_index<9){120}else{1});let types=switch(graph::from_ir(ref metadata,ref unit)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<types::Graph>(ref mut inputs,types).is_ok);assert(v::push<ir::Program>(ref mut units,unit).is_ok);module_index+=1;}
    let types=switch(graph::build(ref metadata,ref inputs)){case .ok(p):{p}case .err(_):{assert(false);return;}};let p=switch(frames::build(ref units,ref types)){case .ok(p):{p}case .err(e):{discard io::write("frames ");discard io::write_u32(e.code);assert(false);return;}};assert(frames::count(ref p)==1089 && frames::work(ref p)<=1048576);module_index=0;
    while(module_index<9){var usize i=1;while(i<=120){let id=frames::mapped(ref p,module_index,i);assert(id==module_index*120+i);let f=frames::at(ref p,id);let origin=frames::origin(ref p,id);assert(f.module_index==module_index && f.context==id && origin.unit==module_index && origin.frame==i);i+=1;}module_index+=1;}
    let linked=switch(link::build(ref metadata,ref units)){case .ok(p):{p}case .err(e):{discard io::write("link ");discard io::write_u32(e.code);assert(false);return;}};assert(ir::count(ref linked)==1089 && ir::callable_count(ref linked)==1080 && proof::check(ref linked).code==0);
    var usize lookup_work=0;let indexed=switch(contracts::build(ref linked,ref mut lookup_work)){case .ok(index):{index}case .err(_):{assert(false);return;}};var usize row=0;var u32 code=0;while(row<ir::callable_count(ref linked)){let c=ir::callable_at(ref linked,row);assert(contracts::find(ref linked,ref indexed,c.module_index,c.declaration,ref mut lookup_work,ref mut code)==row+1);row+=1;}assert(code==0 && lookup_work<20000);assert(contracts::find(ref linked,ref indexed,255,1048576,ref mut lookup_work,ref mut code)==0 && code==0);lookup_work=1048576;let first=ir::callable_at(ref linked,0);assert(contracts::find(ref linked,ref indexed,first.module_index,first.declaration,ref mut lookup_work,ref mut code)==0 && code==1002);lookup_work=1048576;assert(switch(contracts::build(ref linked,ref mut lookup_work)){case .ok(_):{false}case .err(e):{e==1002}});

    lookup_work=0;let body_index=switch(targets::build(ref linked,ref indexed,ref mut lookup_work)){case .ok(index):{index}case .err(_):{assert(false);return;}};code=0;row=1;while(row<=ir::count(ref linked)){let f=clean::frame_at(ir::cleanup_ir(ref linked),row);assert(targets::find(ref linked,ref body_index,f.module_index,f.declaration,f.context,false,ref mut lookup_work,ref mut code)==row);row+=1;}assert(code==0 && lookup_work<30000);assert(targets::find(ref linked,ref body_index,255,1048576,1,true,ref mut lookup_work,ref mut code)==0 && code==0);lookup_work=1048576;let f=clean::frame_at(ir::cleanup_ir(ref linked),1);assert(targets::find(ref linked,ref body_index,f.module_index,f.declaration,0,false,ref mut lookup_work,ref mut code)==0 && code==1002);lookup_work=1048576;assert(switch(targets::build(ref linked,ref indexed,ref mut lookup_work)){case .ok(_):{false}case .err(e):{e==1002}});
    discard io::write("frames ");discard io::write_usize(frames::count(ref p));discard io::write(" work ");discard io::write_usize(frames::work(ref p));discard io::write(" callables ");discard io::write_usize(ir::callable_count(ref linked));discard io::write("\n");
}
