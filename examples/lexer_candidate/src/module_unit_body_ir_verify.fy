module module_unit_body_ir_verify;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_cleanup_mir_verify as cleaning;
import module_unit_value_model as budget;
import std::vec as v;

fn clean_ir::Problem fail(u32 code,usize module_index,usize start,usize end){return clean_ir::Problem{.code,.module_index,.start,.end};}
fn bool span(usize start,usize end,usize limit){return start<=end && end<=limit;}
fn bool expression(ref ir::Body b,usize id){return id==0 || lookup::node(b,id)!=0;}
fn bool statement(ref ir::Body b,usize id){return id==0 || lookup::statement(b,id)!=0;}
pub fn clean_ir::Problem check_body(ref ir::Body b,ref clean_ir::Program cleaned,usize frame,ref mut usize work){
    if(frame==0 || frame>clean_ir::frame_count(cleaned)){return fail(1001,0,0,0);}let f=clean_ir::frame_at(cleaned,frame);let nt=clean_ir::type_count(cleaned);
    if(ir::node_count(b)>4160 || ir::statement_count(b)>4096 || ir::text_bytes(b)>1048576){return fail(1002,f.module_index,0,0);}
    if(!span(ir::start(b),ir::end(b),ir::source_bytes(b)) || ir::body(b)==0 || lookup::statement(b,ir::body(b))==0){return fail(1001,f.module_index,0,0);}
    let root=ir::statement_at(b,lookup::statement(b,ir::body(b))-1).value;
    if(root.kind!=1 || root.start<ir::start(b) || root.end>ir::end(b)){return fail(1001,f.module_index,root.start,root.end);}
    var usize previous=0;var usize i=0;
    while(i<ir::node_count(b)){
        if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let n=ir::node_at(b,i);let e=n.value;let tree=n.syntax;
        if(e.source_index<=previous || e.kind==0 || e.kind>33 || e.state>2 || e.type>nt || e.expected>nt || e.binding>ir::local_count(b) || e.place.root>ir::local_count(b) || !span(e.start,e.end,ir::source_bytes(b)) || e.start<ir::start(b) || e.end>ir::end(b) || !span(n.text_start,n.text_end,ir::text_bytes(b)) || !span(n.cooked_start,n.cooked_end,ir::text_bytes(b)) || e.kind!=tree.kind || e.start!=tree.start || e.end!=tree.end || e.left!=tree.left || e.right!=tree.right){return fail(1001,f.module_index,e.start,e.end);}
        if((e.kind==10 || e.kind==23) && (e.target==0 || e.target>nt)){return fail(1001,f.module_index,e.start,e.end);}
        if(e.kind==1 && (n.literal_kind==0 || n.literal_kind>9)){return fail(1001,f.module_index,e.start,e.end);}if(e.state==1 && e.type==0){return fail(1001,f.module_index,e.start,e.end);}if(e.kind!=1 && (n.literal_kind!=0 || n.cooked_start!=n.cooked_end)){return fail(1001,f.module_index,e.start,e.end);}
        let left_statement=e.kind==12 || e.kind==14 || e.kind==31;let right_statement=e.kind==18;let right_type=e.kind==20 || e.kind==23;
        if(!(if(left_statement){statement(b,e.left)}else{expression(b,e.left)}) || !right_type && !(if(right_statement){statement(b,e.right)}else{expression(b,e.right)})){return fail(1001,f.module_index,e.start,e.end);}
        if(!left_statement && e.left!=0 && e.left>=e.source_index || !right_statement && !right_type && e.right!=0 && e.right>=e.source_index){return fail(1001,f.module_index,e.start,e.end);}previous=e.source_index;i+=1;
    }
    previous=0;i=0;
    while(i<ir::statement_count(b)){
        if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let row=ir::statement_at(b,i);let s=row.value;
        if(row.source_index<=previous || s.kind==0 || s.kind>18 || !span(s.start,s.end,ir::source_bytes(b)) || s.start<ir::start(b) || s.end>ir::end(b) || !span(row.operator_start,row.operator_end,ir::text_bytes(b)) || !expression(b,s.expression) || !expression(b,s.target) || !statement(b,s.first) || (s.kind==7 || s.kind==14 || s.kind==15) && !statement(b,s.second) || s.kind!=4 && row.operator_start!=row.operator_end){return fail(1001,f.module_index,s.start,s.end);}
        // Block lists use reverse links; validate exactly the advertised length.
        if(s.kind==1){var usize link=s.first;var usize count=0;while(link!=0){if(!budget::charge(work,1) || count>=4096){return fail(1002,f.module_index,s.start,s.end);}let index=lookup::statement(b,link);if(index==0){return fail(1001,f.module_index,s.start,s.end);}let pair=ir::statement_at(b,index-1).value;if(pair.kind!=7 || pair.first==0 || !statement(b,pair.first)){return fail(1001,f.module_index,s.start,s.end);}link=pair.second;count+=1;}if(count!=s.second){return fail(1001,f.module_index,s.start,s.end);}}
        if(s.first!=0 && s.first>=row.source_index || (s.kind==7 || s.kind==14 || s.kind==15) && s.second!=0 && s.second>=row.source_index){return fail(1001,f.module_index,s.start,s.end);}previous=row.source_index;i+=1;
    }
    i=1;while(i<=ir::local_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let n=ir::local_at(b,i);if(n.type==0 || n.type>nt || n.expected>nt || !span(n.start,n.end,ir::source_bytes(b)) || !expression(b,n.initializer) || n.origin.root>ir::local_count(b)){return fail(1001,f.module_index,n.start,n.end);}i+=1;}
    i=0;while(i<ir::call_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let c=ir::call_at(b,i);if(c.expression==0 || c.expression>ir::node_count(b) || c.argument_start>ir::argument_count(b) || c.argument_count>ir::argument_count(b)-c.argument_start){return fail(1001,f.module_index,c.start,c.end);}let e=ir::node_at(b,c.expression-1).value;if(e.kind!=6 || e.source_index!=c.source_index || e.start!=c.start || e.end!=c.end){return fail(1001,f.module_index,c.start,c.end);}var usize j=0;while(j<c.argument_count){if(!budget::charge(work,1)){return fail(1002,f.module_index,c.start,c.end);}let a=ir::argument_at(b,c.argument_start+j);if(a.invocation!=i+1 || a.position!=j || a.expression==0 || a.expression>ir::node_count(b) || a.expected>nt || a.actual>nt || ir::node_at(b,a.expression-1).value.source_index!=a.source_index){return fail(1001,f.module_index,a.start,a.end);}j+=1;}i+=1;}
    i=0;while(i<ir::region_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let r=ir::region_at(b,i);if(r.parent>i || !statement(b,r.statement) || !statement(b,r.body) || !expression(b,r.source) || r.arena_owner>ir::local_count(b) || r.source_binding>ir::local_count(b) || r.arena_type==0 || r.arena_type>nt || r.source_type==0 || r.source_type>nt){return fail(1001,f.module_index,r.start,r.end);}i+=1;}
    i=0;while(i<ir::reset_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let e=ir::reset_at(b,i);if(e.region_id==0 || e.region_id>ir::region_count(b) || e.kind==0 || e.kind>5 || !statement(b,e.loop) || !span(e.start,e.end,ir::source_bytes(b))){return fail(1001,f.module_index,e.start,e.end);}i+=1;}
    return fail(0,0,0,0);
}
pub fn clean_ir::Problem check(ref ir::Program p){
    let cleaned=cleaning::check(ir::cleanup_ir(p));if(cleaned.code!=0){return cleaned;}
    if(ir::count(p)!=clean_ir::frame_count(ir::cleanup_ir(p))){return fail(1001,0,0,0);}if(ir::work(p)>1048576 || ir::name_bytes(p)>1048576 || ir::layout_count(p)>1024 || ir::field_count(p)>1024 || ir::callable_count(p)>1024){return fail(1002,0,0,0);}
    var usize work=ir::work(p);var usize i=1;let nt=clean_ir::type_count(ir::cleanup_ir(p));
    while(i<=ir::count(p)){let result=check_body(ir::at(p,i),ir::cleanup_ir(p),i,ref mut work);if(result.code!=0){return result;}i+=1;}
    i=0;while(i<ir::layout_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let row=ir::layout_at(p,i);let s=row.value;if(s.type==0 || s.type>nt || s.field_start>ir::field_count(p) || s.field_count>ir::field_count(p)-s.field_start || !span(row.name_start,row.name_end,ir::name_bytes(p))){return fail(1001,s.module_index,s.start,s.end);}let t=clean_ir::type_at(ir::cleanup_ir(p),s.type);if(t.module_index!=s.module_index || t.declaration!=s.declaration){return fail(1001,s.module_index,s.start,s.end);}i+=1;}
    i=0;while(i<ir::field_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let f=ir::field_at(p,i);if(f.type>nt || f.payload_start>ir::field_count(p) || f.payload_count>ir::field_count(p)-f.payload_start || !span(f.name_start,f.name_end,ir::name_bytes(p))){return fail(1001,f.module_index,f.start,f.end);}i+=1;}
    i=0;while(i<ir::callable_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let c=ir::callable_at(p,i);if(c.result>nt || c.parameter_start>ir::parameter_count(p) || c.parameter_count>ir::parameter_count(p)-c.parameter_start || !span(c.link_start,c.link_end,ir::name_bytes(p))){return fail(1001,c.module_index,0,0);}var usize j=0;while(j<c.parameter_count){if(!budget::charge(ref mut work,1)){return fail(1002,c.module_index,0,0);}if(ir::parameter_at(p,c.parameter_start+j)>nt){return fail(1001,c.module_index,0,0);}j+=1;}i+=1;}
    i=1;while(i<=clean_ir::operand_count(ir::cleanup_ir(p))){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let o=clean_ir::operand_at(ir::cleanup_ir(p),i);let b=ir::at(p,o.frame);if(o.kind==1){if(o.id>ir::local_count(b) || ir::local_at(b,o.id).type!=o.type){return fail(1001,0,o.start,o.end);}}else{let n=lookup::node(b,o.id);if(n==0 || ir::node_at(b,n-1).value.type!=o.type){return fail(1001,0,o.start,o.end);}}i+=1;}
    return fail(0,0,0,0);
}
