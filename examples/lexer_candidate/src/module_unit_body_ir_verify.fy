module module_unit_body_ir_verify;
import module_unit_body_ir_model as ir;
import candidate_limits as limits;
import literal_types as literals;
import module_unit_body_ir_enums as enums;
import module_unit_body_ir_switches as switches;
import module_unit_sdk_sources as sdk_sources;
import module_unit_sdk_authority as sdk_authority;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_storage as storage;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_cleanup_mir_verify as cleaning;
import module_unit_value_model as budget;
import module_unit_value_place as places;
import module_unit_value_builtin as builtins;
import pointer_types as pointers;
import raw_view_intrinsics as raw;
import std::bytes as bytes;
import std::vec as v;

fn clean_ir::Problem fail(u32 code,usize module_index,usize start,usize end){return clean_ir::Problem{.code,.module_index,.start,.end};}
fn bool span(usize start,usize end,usize limit){return start<=end && end<=limit;}
fn bool expression(ref ir::Body b,usize id){return id==0 || lookup::node(b,id)!=0;}
fn bool statement(ref ir::Body b,usize id){return id==0 || lookup::statement(b,id)!=0;}
pub fn bool unsafe_scope(ref ir::Program p,ref ir::Body b,usize frame,budget::Expression e,ref mut usize work){
    let f=clean_ir::frame_at(ir::cleanup_ir(p),frame);var usize i=0;
    if(f.kind!=2){while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return false;}let c=ir::callable_at(p,i);if(c.module_index==f.module_index && c.declaration==f.declaration && !c.foreign && c.unsafe_call){return true;}i+=1;}}
    i=0;while(i<ir::node_count(b)){if(!budget::charge(work,1)){return false;}let owner=ir::node_at(b,i).value;if(owner.kind==31 && owner.start<=e.start && e.end<=owner.end){return true;}i+=1;}return false;
}
// Revalidate pointer casts from retained operation, canonical types and the
// enclosing unsafe expression or original unsafe callable, without Parsed.
pub fn clean_ir::Problem check_casts(ref ir::Program p,ref ir::Body b,usize frame,ref mut usize work){
    if(frame==0 || frame>clean_ir::frame_count(ir::cleanup_ir(p))){return fail(1001,0,0,0);}let f=clean_ir::frame_at(ir::cleanup_ir(p),frame);var usize i=0;
    while(i<ir::node_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let e=ir::node_at(b,i).value;
        if(e.kind==23 && e.state==1 && (e.target==0 || e.target>clean_ir::type_count(ir::cleanup_ir(p)) || e.type==0 || e.type>clean_ir::type_count(ir::cleanup_ir(p)))){return fail(1001,f.module_index,e.start,e.end);}
        if(e.kind==23 && e.state==1 && clean_ir::type_at(ir::cleanup_ir(p),e.target).kind==12){
            let left=lookup::node(b,e.left);if(left==0 || e.type!=e.target || e.right==0 || e.builtin.kind!=0 || !e.builtin.unsafe_context || !bytes::equal(ir::node_text(b,i),b"as")){return fail(1001,f.module_index,e.start,e.end);}let source=ir::node_at(b,left-1).value;if(source.state!=1 || source.type==0){return fail(1001,f.module_index,e.start,e.end);}let t=clean_ir::type_at(ir::cleanup_ir(p),source.type);let target=clean_ir::type_at(ir::cleanup_ir(p),e.target);let referent=clean_ir::type_at(ir::cleanup_ir(p),if(t.kind==7 || t.kind==8){t.first}else{source.type});
            if(!places::raw_cast(t,target,referent) || !unsafe_scope(p,b,frame,e,work)){return fail(if(*work>=1048576){1002}else{1001},f.module_index,e.start,e.end);}
            if(places::opaque(target)){var usize root=left;var usize depth=0;while(ir::node_at(b,root-1).value.kind==3){if(depth>=64 || !budget::charge(work,1)){return fail(1002,f.module_index,e.start,e.end);}root=lookup::node(b,ir::node_at(b,root-1).value.left);if(root==0){return fail(1001,f.module_index,e.start,e.end);}depth+=1;}let operand=ir::node_at(b,root-1).value;let owner=lookup::node(b,operand.left);if(!(operand.kind==2 && operand.binding!=0 || (operand.kind==21 || operand.kind==22) && owner!=0 && ir::node_at(b,owner-1).value.kind==2 && ir::node_at(b,owner-1).value.binding!=0)){return fail(1001,f.module_index,e.start,e.end);}}
        }i+=1;
    }return fail(0,0,0,0);
}
pub struct RawParts {bool valid;usize owner;usize data;usize length;}
pub fn RawParts raw_arguments(ref ir::Body b,budget::Expression e,ref mut usize work){var RawParts args=RawParts{.valid=false,.owner=0,.data=0,.length=0};var usize link=e.right;var usize count=0;while(link!=0){if(count>=3 || !budget::charge(work,1)){return args;}let row=lookup::node(b,link);if(row==0){return args;}let pair=ir::node_at(b,row-1).value;let child=lookup::node(b,pair.left);if(pair.kind!=7 || child==0){return args;}if(count==0){args.length=child;}else if(count==1){args.data=child;}else{args.owner=child;}count+=1;link=pair.right;}args.valid=count==3;return args;}
pub fn clean_ir::Problem check_raw_parts(ref ir::Program p,ref ir::Body b,usize frame,ref mut usize work){let f=clean_ir::frame_at(ir::cleanup_ir(p),frame);var usize i=0;while(i<ir::node_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let e=ir::node_at(b,i).value;
    if(e.kind==6 && e.builtin.kind>=9 && e.builtin.kind<=12){let callee=lookup::node(b,e.left);let args=raw_arguments(b,e,work);if(e.state!=1 || callee==0 || !e.builtin.unsafe_context || !unsafe_scope(p,b,frame,e,work) || !args.valid){return fail(if(*work>=1048576){1002}else{1001},f.module_index,e.start,e.end);}let target=ir::node_at(b,callee-1).value;
        if(target.kind!=2 || raw::named(ir::node_text(b,callee-1))!=e.builtin.kind || target.builtin.kind!=e.builtin.kind){return fail(1001,f.module_index,e.start,e.end);}if(ir::node_at(b,args.owner-1).value.state!=1 || ir::node_at(b,args.data-1).value.state!=1 || ir::node_at(b,args.length-1).value.state!=1){return fail(1001,f.module_index,e.start,e.end);}
        let owner=ir::node_at(b,args.owner-1).value;let data=ir::node_at(b,args.data-1).value;let length=ir::node_at(b,args.length-1).value;let ot=clean_ir::type_at(ir::cleanup_ir(p),owner.type);let dt=clean_ir::type_at(ir::cleanup_ir(p),data.type);let lt=clean_ir::type_at(ir::cleanup_ir(p),length.type);let rt=clean_ir::type_at(ir::cleanup_ir(p),e.type);
        if(!builtins::borrowed(ot) || builtins::mutable(ot)!=(e.builtin.kind==11) || dt.kind!=12 || dt.first!=0 || (pointers::inner(dt.scalar)!=3 && pointers::inner(dt.scalar)!=7) || e.builtin.kind==11 && !pointers::mutable(dt.scalar) || lt.kind!=1 || lt.scalar!=5 || rt.kind!=if(e.builtin.kind==9){9}else if(e.builtin.kind==10){10}else if(e.builtin.kind==11){11}else{14} || !places::same(e.place,builtins::projected(e,owner.place))){return fail(1001,f.module_index,e.start,e.end);}
    }i+=1;}return fail(0,0,0,0);
}
pub fn clean_ir::Problem check_body(ref ir::Body b,ref clean_ir::Program cleaned,usize frame,ref mut usize work){
    if(frame==0 || frame>clean_ir::frame_count(cleaned)){return fail(1001,0,0,0);}let f=clean_ir::frame_at(cleaned,frame);let nt=clean_ir::type_count(cleaned);
    if(ir::node_count(b)>limits::program_expressions() || ir::statement_count(b)>4096 || ir::text_bytes(b)>1048576 || ir::region_count(b)>4096 || ir::reset_count(b)>16384 || ir::placement_count(b)>16384){return fail(1002,f.module_index,0,0);}
    if(!span(ir::start(b),ir::end(b),ir::source_bytes(b)) || ir::body(b)==0 || lookup::statement(b,ir::body(b))==0){return fail(1001,f.module_index,0,0);}
    let root=ir::statement_at(b,lookup::statement(b,ir::body(b))-1).value;
    if(root.kind!=1 || root.start<ir::start(b) || root.end>ir::end(b)){return fail(1001,f.module_index,root.start,root.end);}
    var usize previous=0;var usize i=0;
    while(i<ir::node_count(b)){
        if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let n=ir::node_at(b,i);let e=n.value;let tree=n.syntax;
        if(e.source_index<=previous || e.kind==0 || e.kind>33 || e.state>2 || e.type>nt || e.expected>nt || e.binding>ir::local_count(b) || e.place.root>ir::local_count(b) || !span(e.start,e.end,ir::source_bytes(b)) || e.start<ir::start(b) || e.end>ir::end(b) || !span(n.text_start,n.text_end,ir::text_bytes(b)) || !span(n.cooked_start,n.cooked_end,ir::text_bytes(b)) || e.kind!=tree.kind || e.start!=tree.start || e.end!=tree.end || e.left!=tree.left || e.right!=tree.right){return fail(1001,f.module_index,e.start,e.end);}
        if((e.kind==10 || e.kind==23) && (e.target==0 || e.target>nt)){return fail(1001,f.module_index,e.start,e.end);}
        if(e.kind==1 && (n.literal_kind==0 || n.literal_kind>9)){return fail(1001,f.module_index,e.start,e.end);}if(e.state==1 && e.type==0){return fail(1001,f.module_index,e.start,e.end);}if(e.kind!=1 && (n.literal_kind!=0 || n.cooked_start!=n.cooked_end)){return fail(1001,f.module_index,e.start,e.end);}
        if(e.kind==1 && n.literal_kind>=7){let data=ir::node_cooked(b,i);if(!budget::charge(work,data.len+1)){return fail(1002,f.module_index,e.start,e.end);}if(n.literal_kind==7 && !literals::text(data) || n.literal_kind==9 && !literals::cstring(data)){return fail(1001,f.module_index,e.start,e.end);}if(e.state==1 && clean_ir::type_at(cleaned,e.type).kind!=if(n.literal_kind==7){9}else if(n.literal_kind==8){10}else{14}){return fail(1001,f.module_index,e.start,e.end);}}
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
    i=0;while(i<ir::region_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let r=ir::region_at(b,i);if(r.parent>i || r.capability==0 || r.capability>ir::local_count(b) || r.source_mode<1 || r.source_mode>3 || r.statement==0 || r.body==0 || r.source==0 || !statement(b,r.statement) || !statement(b,r.body) || !expression(b,r.source) || r.arena_owner>ir::local_count(b) || r.source_binding>ir::local_count(b) || r.arena_type==0 || r.arena_type>nt || r.source_type==0 || r.source_type>nt){return fail(1001,f.module_index,r.start,r.end);}let s=ir::statement_at(b,lookup::statement(b,r.statement)-1).value;let body=ir::statement_at(b,lookup::statement(b,r.body)-1).value;let local=ir::local_at(b,r.capability);let source=ir::node_at(b,lookup::node(b,r.source)-1).value;let type=clean_ir::type_at(cleaned,r.source_type);
        if(s.kind!=17 || s.start!=r.start || s.end!=r.end || s.expression!=r.source || s.first!=r.body || s.target!=local.source_index || body.kind!=1 || body.start!=r.body_start || body.end!=r.body_end || local.kind!=6 || local.scope!=r.scope || clean_ir::type_at(cleaned,local.type).kind!=19 || source.type!=r.source_type || type.kind!=8 || type.first!=r.arena_type || r.source_mode==1 && (r.arena_owner==0 || ir::local_at(b,r.arena_owner).type!=r.arena_type)){return fail(1001,f.module_index,r.start,r.end);}if(r.parent!=0){let parent=ir::region_at(b,r.parent-1);if(r.start<parent.body_start || r.end>parent.body_end){return fail(1001,f.module_index,r.start,r.end);}}i+=1;}
    var v::Vec<u8> placed=v::empty<u8>();i=0;while(i<ir::node_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}if(!v::push<u8>(ref mut placed,0).is_ok){return fail(1003,f.module_index,0,0);}i+=1;}
    i=0;while(i<ir::placement_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let x=ir::placement_at(b,i);let call=lookup::node(b,x.expression);let argument=lookup::node(b,x.argument);if(call==0 || argument==0 || x.region_id>ir::region_count(b) || x.capability==0 || x.capability>ir::local_count(b) || x.result_type==0 || x.result_type>nt || x.borrow_type==0 || x.borrow_type>nt || x.owner_type==0 || x.owner_type>nt){return fail(1001,f.module_index,x.start,x.end);}if(*v::at<u8>(ref placed,call-1)!=0){return fail(1001,f.module_index,x.start,x.end);}let slot=v::at_mut<u8>(ref mut placed,call-1);*slot=1;let e=ir::node_at(b,call-1).value;let cap=ir::local_at(b,x.capability);let result=clean_ir::type_at(cleaned,x.result_type);let borrow=clean_ir::type_at(cleaned,x.borrow_type);
        if(e.kind!=6 || e.builtin.kind!=300 || e.type!=x.result_type || e.start!=x.start || e.end!=x.end || result.kind!=6 || result.first!=x.borrow_type || result.second!=x.error_type || borrow.kind!=7 || borrow.first!=x.owner_type || ir::node_at(b,argument-1).value.type!=x.owner_type || clean_ir::type_at(cleaned,cap.type).kind!=19 || (x.region_id==0 && cap.kind!=1) || (x.region_id!=0 && ir::region_at(b,x.region_id-1).capability!=x.capability)){return fail(1001,f.module_index,x.start,x.end);}i+=1;}
    i=0;while(i<ir::node_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let e=ir::node_at(b,i).value;if(e.kind==6 && e.builtin.kind==300 && *v::at<u8>(ref placed,i)==0){return fail(1001,f.module_index,e.start,e.end);}i+=1;}
    i=0;while(i<ir::reset_count(b)){if(!budget::charge(work,1)){return fail(1002,f.module_index,0,0);}let e=ir::reset_at(b,i);if(e.region_id==0 || e.region_id>ir::region_count(b) || e.kind==0 || e.kind>5 || !statement(b,e.loop) || !span(e.start,e.end,ir::source_bytes(b)) || e.group==0 || e.group>ir::reset_count(b) || e.order>=64){return fail(1001,f.module_index,e.start,e.end);}let r=ir::region_at(b,e.region_id-1);if(e.kind==1){if(e.order!=0 || e.start!=r.start || e.end!=r.end){return fail(1001,f.module_index,e.start,e.end);}}else if(e.start<r.body_start || e.end>r.body_end){return fail(1001,f.module_index,e.start,e.end);}if(i==0){if(e.group!=1 || e.order!=0){return fail(1001,f.module_index,e.start,e.end);}}else{let prior=ir::reset_at(b,i-1);if(e.group==prior.group){if(e.order!=prior.order+1 || e.kind!=prior.kind || e.loop!=prior.loop || e.start!=prior.start || e.end!=prior.end || ir::region_at(b,prior.region_id-1).parent!=e.region_id){return fail(1001,f.module_index,e.start,e.end);}}else if(e.group!=prior.group+1 || e.order!=0){return fail(1001,f.module_index,e.start,e.end);}}i+=1;}
    return fail(0,0,0,0);
}
pub fn clean_ir::Problem check(ref ir::Program p){
    let cleaned=cleaning::check(ir::cleanup_ir(p));if(cleaned.code!=0){return cleaned;}
    if(ir::count(p)!=clean_ir::frame_count(ir::cleanup_ir(p))){return fail(1001,0,0,0);}if(ir::work(p)>1048576 || ir::name_bytes(p)>1048576 || ir::layout_count(p)>1024 || ir::field_count(p)>limits::metadata_fields() || ir::callable_count(p)>1024){return fail(1002,0,0,0);}
    // Producer work remains capped above. Revalidation has its own bounded
    // traversal, so its capacity does not depend on the producer's history.
    var usize work=0;var usize i=1;let nt=clean_ir::type_count(ir::cleanup_ir(p));
    let sdk=sdk_sources::check(ir::sdk_units(p),ref mut work);if(sdk.code!=0){return sdk;}
    let modules=sdk_sources::module_count(ir::sdk_units(p));
    if(modules!=0){
        while(i<=ir::count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let f=clean_ir::frame_at(ir::cleanup_ir(p),i);if(f.module_index>=modules){return fail(1001,f.module_index,0,0);}let unit=sdk_sources::find(ir::sdk_units(p),f.module_index);if(unit!=0 && ir::source_bytes(ir::at(p,i))!=sdk_sources::source_at(ir::sdk_units(p),unit-1).len){return fail(1001,f.module_index,0,0);}i+=1;}
        i=1;while(i<=nt){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let t=clean_ir::type_at(ir::cleanup_ir(p),i);if((t.kind==2 || t.kind==3 || t.kind==4 || t.kind==13) && t.module_index>=modules){return fail(1001,t.module_index,0,0);}i+=1;}
        i=0;while(i<ir::layout_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}if(ir::layout_at(p,i).value.module_index>=modules){return fail(1001,0,0,0);}i+=1;}
        i=0;while(i<ir::field_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}if(ir::field_at(p,i).module_index>=modules){return fail(1001,0,0,0);}i+=1;}
        i=0;while(i<ir::callable_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}if(ir::callable_at(p,i).module_index>=modules){return fail(1001,0,0,0);}i+=1;}
    }i=1;
    while(i<=ir::count(p)){let result=check_body(ir::at(p,i),ir::cleanup_ir(p),i,ref mut work);if(result.code!=0){return result;}let casts=check_casts(p,ir::at(p,i),i,ref mut work);if(casts.code!=0){return casts;}let raw=check_raw_parts(p,ir::at(p,i),i,ref mut work);if(raw.code!=0){return raw;}let stored=storage::check(p,ir::at(p,i),i,ref mut work);if(stored.code!=0){return stored;}i+=1;}
    i=0;while(i<ir::layout_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let row=ir::layout_at(p,i);let s=row.value;if(s.type==0 || s.type>nt || s.field_start>ir::field_count(p) || s.field_count>ir::field_count(p)-s.field_start || !span(row.name_start,row.name_end,ir::name_bytes(p))){return fail(1001,s.module_index,s.start,s.end);}let t=clean_ir::type_at(ir::cleanup_ir(p),s.type);if(t.module_index!=s.module_index || t.declaration!=s.declaration){return fail(1001,s.module_index,s.start,s.end);}i+=1;}
    i=0;while(i<ir::field_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let f=ir::field_at(p,i);if(f.type>nt || f.payload_start>ir::field_count(p) || f.payload_count>ir::field_count(p)-f.payload_start || !span(f.name_start,f.name_end,ir::name_bytes(p))){return fail(1001,f.module_index,f.start,f.end);}i+=1;}
    i=0;while(i<ir::callable_count(p)){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let c=ir::callable_at(p,i);if(c.result>nt || c.parameter_start>ir::parameter_count(p) || c.parameter_count>ir::parameter_count(p)-c.parameter_start || !span(c.link_start,c.link_end,ir::name_bytes(p))){return fail(1001,c.module_index,0,0);}var usize j=0;while(j<c.parameter_count){if(!budget::charge(ref mut work,1)){return fail(1002,c.module_index,0,0);}if(ir::parameter_at(p,c.parameter_start+j)>nt){return fail(1001,c.module_index,0,0);}j+=1;}i+=1;}
    let authenticated=sdk_authority::check(p,ref mut work);if(authenticated.code!=0){return authenticated;}
    let enum_proof=enums::check(p,ref mut work);if(enum_proof.code!=0){return enum_proof;}
    i=1;while(i<=ir::count(p)){let matched=switches::check_body(p,ir::at(p,i),i,ref mut work);if(matched.code!=0){return matched;}i+=1;}
    i=1;while(i<=clean_ir::operand_count(ir::cleanup_ir(p))){if(!budget::charge(ref mut work,1)){return fail(1002,0,0,0);}let o=clean_ir::operand_at(ir::cleanup_ir(p),i);let b=ir::at(p,o.frame);if(o.kind==1){if(o.id>ir::local_count(b) || ir::local_at(b,o.id).type!=o.type){return fail(1001,0,o.start,o.end);}}else{let n=lookup::node(b,o.id);if(n==0 || ir::node_at(b,n-1).value.type!=o.type){return fail(1001,0,o.start,o.end);}}i+=1;}
    return fail(0,0,0,0);
}
