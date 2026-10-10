module module_project_body_equal;
import module_unit_body_ir_model as ir;
import module_unit_value_model as m;
import module_unit_region_model as regions;
import syntax as syntax;
import std::bytes as bytes;
fn bool place_same(m::Place a,m::Place b){return a.kind==b.kind && a.root==b.root && a.scope==b.scope && a.projection==b.projection && a.mutable==b.mutable && a.indirect==b.indirect && a.resource_field==b.resource_field && a.borrowable==b.borrowable;}
fn bool field_same(m::Field a,m::Field b){return a.id==b.id && a.owner==b.owner && a.module_index==b.module_index && a.declaration==b.declaration && a.source_index==b.source_index;}
fn bool builtin_same(m::Builtin a,m::Builtin b){return a.kind==b.kind && a.element==b.element && a.slice==b.slice && a.unsafe_context==b.unsafe_context;}
fn bool expression_same(m::Expression a,m::Expression b){return a.source_index==b.source_index && a.binding==b.binding && a.kind==b.kind && a.start==b.start && a.end==b.end && a.left==b.left && a.right==b.right && a.target==b.target && a.expected==b.expected && a.type==b.type && a.state==b.state && a.flexible==b.flexible && a.minimum==b.minimum && field_same(a.field,b.field) && place_same(a.place,b.place) && builtin_same(a.builtin,b.builtin);}
fn bool syntax_same(syntax::Expression a,syntax::Expression b){return a.kind==b.kind && a.start==b.start && a.end==b.end && a.token==b.token && a.left==b.left && a.right==b.right;}
fn bool node_same(ir::Node a,ir::Node b){return a.literal_kind==b.literal_kind && a.text_start==b.text_start && a.text_end==b.text_end && a.cooked_start==b.cooked_start && a.cooked_end==b.cooked_end && expression_same(a.value,b.value) && syntax_same(a.syntax,b.syntax);}
fn bool statement_value_same(syntax::Statement a,syntax::Statement b){return a.kind==b.kind && a.start==b.start && a.end==b.end && a.expression==b.expression && a.target==b.target && a.first==b.first && a.second==b.second && a.entry==b.entry && a.operator==b.operator;}
fn bool statement_same(ir::Statement a,ir::Statement b){return a.source_index==b.source_index && a.operator_start==b.operator_start && a.operator_end==b.operator_end && statement_value_same(a.value,b.value);}
fn bool local_same(m::Local a,m::Local b){return a.header_start==b.header_start && a.header_end==b.header_end && a.kind==b.kind && a.source_index==b.source_index && a.initializer==b.initializer && a.expected==b.expected && a.type==b.type && a.scope==b.scope && a.start==b.start && a.end==b.end && place_same(a.origin,b.origin);}
fn bool call_same(m::Invocation a,m::Invocation b){return a.kind==b.kind && a.expression==b.expression && a.source_index==b.source_index && a.target_module==b.target_module && a.target_declaration==b.target_declaration && a.instance==b.instance && a.argument_start==b.argument_start && a.argument_count==b.argument_count && a.start==b.start && a.end==b.end && a.checked==b.checked;}
fn bool argument_same(m::Argument a,m::Argument b){return a.invocation==b.invocation && a.position==b.position && a.expression==b.expression && a.source_index==b.source_index && a.expected==b.expected && a.actual==b.actual && a.start==b.start && a.end==b.end && a.label_start==b.label_start && a.label_end==b.label_end && a.checked==b.checked && a.shared==b.shared;}
fn bool scope_same(regions::Scope a,regions::Scope b){return a.parent==b.parent && a.capability==b.capability && a.scope==b.scope && a.statement==b.statement && a.body==b.body && a.source==b.source && a.source_binding==b.source_binding && a.arena_owner==b.arena_owner && a.arena_type==b.arena_type && a.source_type==b.source_type && a.source_mode==b.source_mode && a.start==b.start && a.end==b.end && a.body_start==b.body_start && a.body_end==b.body_end;}
fn bool placement_same(regions::Placement a,regions::Placement b){return a.region_id==b.region_id && a.capability==b.capability && a.expression==b.expression && a.argument==b.argument && a.owner_type==b.owner_type && a.borrow_type==b.borrow_type && a.result_type==b.result_type && a.error_type==b.error_type && a.start==b.start && a.end==b.end;}
fn bool exit_same(regions::Exit a,regions::Exit b){return a.region_id==b.region_id && a.kind==b.kind && a.group==b.group && a.order==b.order && a.loop==b.loop && a.start==b.start && a.end==b.end;}
// Exact comparison after ID rebinding; lexical IDs and retained bytes stay
// local to the original declaration. No digest substitutes for equality.
pub fn bool same(ref ir::Body a,ref ir::Body b,ref mut usize work){
    if(ir::body(a)!=ir::body(b) || ir::source_bytes(a)!=ir::source_bytes(b) || ir::start(a)!=ir::start(b) || ir::end(a)!=ir::end(b) || ir::node_count(a)!=ir::node_count(b) || ir::statement_count(a)!=ir::statement_count(b) || ir::local_count(a)!=ir::local_count(b) || ir::call_count(a)!=ir::call_count(b) || ir::argument_count(a)!=ir::argument_count(b) || ir::region_count(a)!=ir::region_count(b) || ir::placement_count(a)!=ir::placement_count(b) || ir::reset_count(a)!=ir::reset_count(b)){return false;}
    if(!m::charge(work,ir::text_bytes(a)+1) || !bytes::equal(ir::text_view(a),ir::text_view(b))){return false;}
    var usize i=0;
    i=0;while(i<ir::node_count(a)){if(!m::charge(work,1) || !node_same(ir::node_at(a,i),ir::node_at(b,i))){return false;}i+=1;}
    i=0;while(i<ir::statement_count(a)){if(!m::charge(work,1) || !statement_same(ir::statement_at(a,i),ir::statement_at(b,i))){return false;}i+=1;}
    i=1;while(i<=ir::local_count(a)){if(!m::charge(work,1) || !local_same(ir::local_at(a,i),ir::local_at(b,i))){return false;}i+=1;}
    i=0;while(i<ir::call_count(a)){if(!m::charge(work,1) || !call_same(ir::call_at(a,i),ir::call_at(b,i))){return false;}i+=1;}
    i=0;while(i<ir::argument_count(a)){if(!m::charge(work,1) || !argument_same(ir::argument_at(a,i),ir::argument_at(b,i))){return false;}i+=1;}
    i=0;while(i<ir::region_count(a)){if(!m::charge(work,1) || !scope_same(ir::region_at(a,i),ir::region_at(b,i))){return false;}i+=1;}
    i=0;while(i<ir::placement_count(a)){if(!m::charge(work,1) || !placement_same(ir::placement_at(a,i),ir::placement_at(b,i))){return false;}i+=1;}
    i=0;while(i<ir::reset_count(a)){if(!m::charge(work,1) || !exit_same(ir::reset_at(a,i),ir::reset_at(b,i))){return false;}i+=1;}
    return true;
}
