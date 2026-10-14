module tests::module_catalog_capacity;
import module_catalog as catalog;
import module_signatures as signatures;
import candidate_limits as limits;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(bytes::Buffer data){let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};return switch(syntax::parse(stream,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(bytes::empty())}};}
fn syntax::Parsed module_source(usize index,usize count){var bytes::Buffer data=bytes::empty();assert(bytes::append(ref mut data,b"module p").is_ok && bytes::append_usize(ref mut data,index).is_ok && bytes::append_byte(ref mut data,59).is_ok);var usize i=0;while(i<count){assert(bytes::append(ref mut data,b"fn void f").is_ok && bytes::append_usize(ref mut data,i).is_ok && bytes::append(ref mut data,b"(){}").is_ok);i+=1;}return parsed(data);}
@test fn void signatures_accept_the_complete_catalog_and_reject_overflow(){
    assert(limits::project_callables()==limits::project_declarations());var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;
    while(i<64){assert(v::push<syntax::Parsed>(ref mut sources,module_source(i,128)).is_ok);i+=1;}
    {let p=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(signatures::count(ref p)==8192 && signatures::declaration_count(ref p)==8192);let first=signatures::at(ref p,1);let last=signatures::at(ref p,8192);assert(first.module_index==0 && last.module_index==63 && first.parameter_count==0 && last.parameter_count==0);}
    assert(v::push<syntax::Parsed>(ref mut sources,module_source(64,1)).is_ok);assert(switch(signatures::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==10 && e.module_index==64}});
}
@test fn void declaration_catalog_accepts_exact_capacity_and_rejects_one_more(){assert(limits::project_declarations()==8192);var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var usize i=0;while(i<64){assert(v::push<syntax::Parsed>(ref mut sources,module_source(i,128)).is_ok);i+=1;}{let result=switch(catalog::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(catalog::declaration_count(ref result)==8192 && catalog::module_count(ref result)==64);let first=catalog::lookup(ref result,0,b"f127");let last=catalog::lookup(ref result,63,b"f127");assert(first.code==0 && last.code==0 && first.item!=last.item);}assert(v::push<syntax::Parsed>(ref mut sources,module_source(64,1)).is_ok);assert(switch(catalog::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==10 && e.module_index==64}});}
@test fn void module_local_duplicate_guards_keep_import_and_declaration_conflicts(){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,module_source(0,128)).is_ok);var bytes::Buffer data=bytes::empty();assert(bytes::append(ref mut data,b"module other;fn void f127(){}fn void f127(){}").is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(data)).is_ok);assert(switch(catalog::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==2 && e.module_index==1}});assert(switch(v::pop<syntax::Parsed>(ref mut sources)){case .some(_):{true}case .none:{false}});data=bytes::empty();assert(bytes::append(ref mut data,b"module other;import p0 as f127;fn void f127(){}").is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(data)).is_ok);assert(switch(catalog::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==2 && e.module_index==1}});}
