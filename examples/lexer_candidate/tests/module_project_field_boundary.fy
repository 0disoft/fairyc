module tests::module_project_field_boundary;
import module_project_compile as compile;
import module_project_link as link;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_body_ir_verify as proof;
import module_signatures as signatures;
import candidate_limits as limits;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(bytes::Buffer source){let s=switch(scanner::collect(source,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}

@test @effects(console) fn void project_fields_admit_8192_and_reject_8193(){
    assert(limits::metadata_fields()==2048 && limits::project_fields()==8192);
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize module_index=0;
    while(module_index<17){var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,b"module p").is_ok);assert(bytes::append_usize(ref mut text,module_index).is_ok);assert(bytes::append(ref mut text,b";").is_ok);var usize i=0;let usize resources=if(module_index<16){8}else{1};let usize fields=if(module_index<16){64}else{1};
        while(i<resources){assert(bytes::append(ref mut text,b"pub resource R").is_ok);assert(bytes::append_usize(ref mut text,i).is_ok);assert(bytes::append(ref mut text,b"{").is_ok);var usize field=0;while(field<fields){assert(bytes::append(ref mut text,b"i32 n").is_ok);assert(bytes::append_usize(ref mut text,field).is_ok);assert(bytes::append(ref mut text,b";").is_ok);field+=1;}assert(bytes::append(ref mut text,b"cleanup{}}pub fn void use").is_ok);assert(bytes::append_usize(ref mut text,i).is_ok);assert(bytes::append(ref mut text,b"(ref R").is_ok);assert(bytes::append_usize(ref mut text,i).is_ok);assert(bytes::append(ref mut text,b" n){discard n.n0;}").is_ok);i+=1;}
        assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);module_index+=1;
    }
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var v::Vec<ir::Program> units=v::empty<ir::Program>();module_index=0;
    while(module_index<16){let unit=switch(compile::range(ref sources,ref metadata,module_index,module_index+1)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);assert(false);return;}};assert(ir::field_count(ref unit)==512);assert(v::push<ir::Program>(ref mut units,unit).is_ok);module_index+=1;}
    {let p=switch(link::build(ref metadata,ref units)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);assert(false);return;}};assert(ir::field_count(ref p)==8192 && clean::field_count(ir::cleanup_ir(ref p))==8192 && proof::check(ref p).code==0);}
    let last=switch(compile::range(ref sources,ref metadata,16,17)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(ir::field_count(ref last)==1);assert(v::push<ir::Program>(ref mut units,last).is_ok);
    switch(link::build(ref metadata,ref units)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1002 && e.module_index==16);}};
    discard io::write("project fields 8192 accepted; 8193 rejected; unit cap 2048\n");
}
