module tests::module_unit_origin_call_places;
import module_unit_origin_returns as origins;
import module_unit_origin_inputs as inputs;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

import module_unit_origin_diagnostics as diagnostics;
fn void check(ref u8[] text,u32 expected){
    let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let source=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref source)==0);
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,source).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let ready=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let primary=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let plan=switch(origins::build(ref sources,ref metadata,ref primary)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let report=switch(diagnostics::collect(ref plan)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    if(expected==0){assert(diagnostics::exit_code(ref report)==0);return;}
    var bool found=false;var usize i=0;while(i<diagnostics::count(ref report)){if(diagnostics::at(ref report,i).code==expected){found=true;}i+=1;}assert(found);
}
@test fn void returned_reference_field_places_preserve_origins_and_reject_escapes(){
    check(b"module main;struct Cell{i32 value;}fn ref Cell identity(ref Cell cell){return cell;}fn ref i32 project(ref Cell cell){return ref identity(cell).value;}",0);
    check(b"module main;struct Cell{i32 value;}fn ref Cell identity(ref Cell cell){return cell;}fn ref i32 bad(){let Cell cell=Cell{.value=1};return ref identity(ref cell).value;}",2102);
}
@test fn void projected_call_borrows_keep_the_owner_loan_until_last_use(){
    check(b"module main;struct Cell{i32 value;}fn ref Cell identity(ref Cell cell){return cell;}fn i32 main(){var Cell cell=Cell{.value=0};let ref i32 view=ref identity(ref cell).value;cell.value=1;return *view;}",2101);
}
