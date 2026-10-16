module tests::module_unit_enum_work_budgets;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_enums as enums;
import module_unit_cleanup_mir_model as clean;
import module_types as types;
import module_unit_interfaces as interfaces;
import candidate_limits as limits;
import std::vec as v;
import std::bytes as bytes;

fn ir::Program empty(bool project){
    var v::Vec<types::Type> graph=v::empty<types::Type>();assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=7,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);
    var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
    // Choose the scope before transferring the owned vectors.
    let cleaned=switch(if(project){clean::assemble_project(graph,v::empty<clean::Frame>(),glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),0)}else{clean::assemble(graph,v::empty<clean::Frame>(),glues,v::empty<clean::Field>(),v::empty<clean::Operand>(),v::empty<clean::Drop>(),0)}){case .ok(p):{p}case .err(_):{assert(false);return empty(project);}};
    var p=switch(ir::assemble(cleaned,v::empty<ir::Body>(),v::empty<ir::Layout>(),v::empty<interfaces::Field>(),v::empty<interfaces::Callable>(),v::empty<usize>(),bytes::empty(),0)){case .ok(p):{p}case .err(_):{assert(false);return empty(project);}};
    return p;
}
@test fn void project_enum_prefix_keeps_exact_aggregate_and_unit_ceilings(){
    let p=empty(true);var usize work=1048575;var usize total=1;
    assert(enums::check_metered(ref p,ref mut work,ref mut total).code==0 && work==0 && total==1048577);
    work=0;total=limits::project_work()-1;assert(enums::check_metered(ref p,ref mut work,ref mut total).code==0 && work==0 && total==limits::project_work());
    work=0;total=limits::project_work();assert(enums::check_metered(ref p,ref mut work,ref mut total).code==1002 && work==1 && total==limits::project_work());
    work=1048576;total=0;assert(enums::check_metered(ref p,ref mut work,ref mut total).code==1002 && work==1048576 && total==0);
}
@test fn void unit_enum_scan_retains_its_shared_work_budget(){
    let p=empty(false);var usize work=1048575;var usize total=7;
    assert(enums::check_metered(ref p,ref mut work,ref mut total).code==0 && work==1048576 && total==7);
    assert(enums::check_metered(ref p,ref mut work,ref mut total).code==1002 && work==1048576 && total==7);
}
