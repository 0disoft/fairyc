module tests::module_project_layout_index;
import module_project_compile as compile;
import module_project_metadata as layouts;
import module_project_types as graph;
import module_types as types;
import module_unit_body_ir_model as ir;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(bytes::Buffer source){let s=switch(scanner::collect(source,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}

@test @effects(console) fn void repeated_provider_layouts_keep_indexed_choices_and_field_maps(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize module_index=0;
    while(module_index<16){var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module p").is_ok);assert(bytes::append_usize(ref mut text,module_index).is_ok);assert(bytes::append(ref mut text,b";").is_ok);var usize i=0;while(i<32){assert(bytes::append(ref mut text,b"pub resource R").is_ok);assert(bytes::append_usize(ref mut text,i).is_ok);assert(bytes::append(ref mut text,b"{i32 n;cleanup{}}pub fn void use").is_ok);assert(bytes::append_usize(ref mut text,i).is_ok);assert(bytes::append(ref mut text,b"(ref R").is_ok);assert(bytes::append_usize(ref mut text,i).is_ok);assert(bytes::append(ref mut text,b" n){discard n.n;}").is_ok);i+=1;}assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);module_index+=1;}
    let catalog=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var v::Vec<ir::Program> units=v::empty<ir::Program>();var v::Vec<types::Graph> graphs=v::empty<types::Graph>();var usize unit=0;
    while(unit<128){let provider=unit%16;let p=switch(compile::range(ref sources,ref catalog,provider,provider+1)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);assert(false);return;}};assert(ir::layout_count(ref p)==32 && ir::field_count(ref p)==32);let g=switch(graph::from_ir(ref catalog,ref p)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<types::Graph>(ref mut graphs,g).is_ok);assert(v::push<ir::Program>(ref mut units,p).is_ok);unit+=1;}
    let g=switch(graph::build(ref catalog,ref graphs)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);assert(false);return;}};let p=switch(layouts::build(ref units,ref g)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);assert(false);return;}};
    assert(layouts::layout_count(ref p)==512 && layouts::field_count(ref p)==512 && layouts::work(ref p)<65536);unit=0;
    while(unit<128){let source=v::at<ir::Program>(ref units,unit);var usize i=0;while(i<32){let old=ir::layout_at(source,i).value;let type=graph::mapped(ref g,unit,old.type);let choice=layouts::choice(ref p,type);assert(choice.unit==unit%16 && choice.type==old.type && choice.layout==unit%16*32+i+1);let merged=layouts::layout_at(ref p,choice.layout-1).value;assert(merged.type==type && merged.module_index==unit%16 && merged.field_count==1);let field=layouts::mapped_field(ref p,unit,old.field_start+1);assert(field==merged.field_start+1 && layouts::field_at(ref p,field-1).module_index==unit%16);i+=1;}unit+=1;}
    discard io::write("units 128 layouts 512 fields 512 work ");discard io::write_usize(layouts::work(ref p));discard io::write("\n");
}
