module module_unit_body_ir_switches;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_enums as enums;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as m;
import std::bytes as bytes;
import std::vec as v;

fn bool charge(ref mut usize work,usize amount,ref mut u32 code){if(!m::charge(work,amount)){*code=1002;return false;}return true;}
fn bool mark(ref mut v::Vec<usize> owners,usize id,usize owner){if(id==0 || id>v::len<usize>(owners) || *v::at<usize>(owners,id-1)!=0){return false;}let slot=v::at_mut<usize>(owners,id-1);*slot=owner;return true;}
fn bool zeros(ref mut v::Vec<usize> out,usize count,ref mut usize work,ref mut u32 code){var usize i=0;while(i<count){if(!charge(work,1,code)){return false;}if(!v::push<usize>(out,0).is_ok){*code=1003;return false;}i+=1;}return true;}
fn bool same(m::Field a,m::Field b){return a.id==b.id && a.owner==b.owner && a.module_index==b.module_index && a.declaration==b.declaration && a.source_index==b.source_index;}
fn bool name(ref ir::Body b,usize id,ref u8[] expected,ref mut usize work,ref mut u32 code){return charge(work,expected.len+1,code) && bytes::equal(ir::node_text(b,id-1),expected);}
fn bool binding(ref ir::Body b,usize original,usize type,m::Field field,usize arm,ref mut v::Vec<usize> owners,ref mut v::Vec<usize> locals,ref mut usize work,ref mut u32 code){
    let id=lookup::node(b,original);if(!mark(owners,id,arm) || !charge(work,1,code)){return false;}let e=ir::node_at(b,id-1).value;
    if(e.kind!=17 || e.state!=2 || e.type!=0 || e.target!=type || e.expected!=type || !same(e.field,field) || e.left!=0 || e.right!=0){return false;}
    let root=e.place.root;if(root==0 || root>ir::local_count(b) || *v::at<usize>(locals,root-1)!=0){return false;}let local=ir::local_at(b,root);
    if(local.kind!=4 || local.source_index!=original || local.type!=type || local.expected!=type || local.initializer!=0 || local.scope!=e.place.scope || e.place.mutable || e.place.indirect){return false;}
    let slot=v::at_mut<usize>(locals,root-1);*slot=arm;return true;
}
fn bool pattern(ref ir::Program p,ref ir::Body b,usize id,usize subject,usize arm,ref mut v::Vec<usize> owners,ref mut v::Vec<usize> locals,ref mut usize work,ref mut u32 code,ref mut usize variant){
    if(!mark(owners,id,arm) || !charge(work,1,code)){return false;}let e=ir::node_at(b,id-1).value;let t=clean::type_at(ir::cleanup_ir(p),subject);
    if(e.state!=2 || e.type!=0 || e.expected!=subject){return false;}
    if(t.kind==5 || t.kind==6){
        if(e.kind!=16 || e.right!=0 || e.field.id!=0 || e.field.owner!=0){return false;}var usize payload=0;
        if(name(b,id,if(t.kind==5){b"some"}else{b"ok"},work,code)){*variant=1;payload=t.first;}
        else if(name(b,id,if(t.kind==5){b"none"}else{b"err"},work,code)){*variant=2;payload=if(t.kind==5){0}else{t.second};}else{return false;}
        if(payload==0){return e.left==0;}return binding(b,e.left,payload,m::Field{.id=0,.owner=subject,.module_index=0,.declaration=0,.source_index=0},arm,owners,locals,work,code);
    }
    if(t.kind!=4 || (e.kind!=16 && e.kind!=32) || !enums::member(p,e.field,subject,3)){return false;}let selected=ir::field_at(p,e.field.id-1);
    if(!name(b,id,ir::field_name(p,e.field.id-1),work,code) || e.left!=0){return false;};*variant=e.field.id;
    if(e.kind==16){return selected.payload_count==0 && e.right==0;}
    var v::Vec<usize> seen=v::empty<usize>();if(!zeros(ref mut seen,selected.payload_count,work,code)){return false;}var usize link=e.right;var usize count=0;
    while(link!=0){if(count>=selected.payload_count || !charge(work,1,code)){return false;}let pair=lookup::node(b,link);if(!mark(owners,pair,arm)){return false;}let row=ir::node_at(b,pair-1).value;
        if(row.kind!=33 || row.state!=2 || row.type!=0 || !enums::member(p,row.field,subject,4) || row.field.id<=selected.payload_start || row.field.id>selected.payload_start+selected.payload_count){return false;}
        let field=ir::field_at(p,row.field.id-1);let position=row.field.id-selected.payload_start;
        if(!mark(ref mut seen,position,arm) || row.expected!=field.type || !name(b,pair,ir::field_name(p,row.field.id-1),work,code) || !binding(b,row.left,field.type,row.field,arm,owners,locals,work,code)){return false;}count+=1;link=row.right;
    }return count==selected.payload_count;
}
// The reference's narrow no-tail exception is a direct final return (Result
// only) or assert(false) (Option/Result). It is not a general dominance proof.
fn bool no_tail(ref ir::Program p,ref ir::Body b,usize block,usize subject,usize result,ref mut usize work,ref mut u32 code){
    let output=clean::type_at(ir::cleanup_ir(p),result);if(output.kind==1 && output.scalar==7){return true;}let t=clean::type_at(ir::cleanup_ir(p),subject);if(t.kind!=5 && t.kind!=6){return false;}let body=ir::statement_at(b,block-1).value;
    if(body.first==0 || body.second==0 || !charge(work,2,code)){return false;}let pair=lookup::statement(b,body.first);if(pair==0){return false;}let row=ir::statement_at(b,pair-1).value;let last=lookup::statement(b,row.first);if(row.kind!=7 || last==0){return false;}let s=ir::statement_at(b,last-1).value;if(t.kind==6 && s.kind==3){return true;}let assertion=lookup::node(b,s.expression);if(s.kind!=11 || assertion==0){return false;}let e=ir::node_at(b,assertion-1).value;return e.kind==1 && name(b,assertion,b"false",work,code);
}
// All pattern records have exactly one owning arm. Bindings are associated
// with that arm's retained lexical symbol, so inactive payloads cannot be
// relabelled as bindings of a different variant or escape into another arm.
pub fn clean::Problem check_body(ref ir::Program p,ref ir::Body b,usize frame,ref mut usize work){
    if(frame==0 || frame>clean::frame_count(ir::cleanup_ir(p))){return clean::Problem{.code=1001,.module_index=0,.start=0,.end=0};}let module_index=clean::frame_at(ir::cleanup_ir(p),frame).module_index;
    var u32 code=0;var v::Vec<usize> owners=v::empty<usize>();var v::Vec<usize> locals=v::empty<usize>();var usize i=0;
    if(!zeros(ref mut owners,ir::node_count(b),work,ref mut code) || !zeros(ref mut locals,ir::local_count(b),work,ref mut code)){return clean::Problem{.code,.module_index,.start=0,.end=0};}
    while(i<ir::node_count(b)){if(!charge(work,1,ref mut code)){return clean::Problem{.code,.module_index,.start=0,.end=0};}let e=ir::node_at(b,i).value;
        if(e.kind==15){
            let subject=lookup::node(b,e.left);var bool valid=subject!=0 && e.state==1 && e.type!=0 && e.target!=0 && e.target<=clean::type_count(ir::cleanup_ir(p));var usize capacity=2;var usize base=0;var usize required=2;
            if(valid){let actual=ir::node_at(b,subject-1).value;let t=clean::type_at(ir::cleanup_ir(p),e.target);valid=actual.state==1 && actual.type==e.target && (t.kind==4 || t.kind==5 || t.kind==6);if(valid && t.kind==4){let shape=enums::layout(p,e.target,work,ref mut code);valid=shape!=0;if(valid){let s=ir::layout_at(p,shape-1).value;capacity=s.field_count;base=s.field_start;required=0;var usize k=0;while(k<capacity){if(!charge(work,1,ref mut code)){valid=false;break;}if(ir::field_at(p,base+k).kind==3){required+=1;}k+=1;}}}}
            var v::Vec<usize> seen=v::empty<usize>();if(valid){valid=zeros(ref mut seen,capacity,work,ref mut code);}var usize link=e.right;var usize count=0;
            while(valid && link!=0){if(count>=required || !charge(work,1,ref mut code)){valid=false;break;}let pair=lookup::node(b,link);if(!mark(ref mut owners,pair,i+1)){valid=false;break;}let row=ir::node_at(b,pair-1).value;let arm=lookup::node(b,row.left);if(row.kind!=19 || row.state!=2 || !mark(ref mut owners,arm,i+1)){valid=false;break;}let a=ir::node_at(b,arm-1).value;let pat=lookup::node(b,a.left);let block=lookup::statement(b,a.right);
                if(a.kind!=18 || a.state!=2 || a.target!=e.target || a.expected!=e.type || a.start<e.start || a.end>e.end || pat==0 || block==0){valid=false;break;}let body=ir::statement_at(b,block-1).value;let pat_value=ir::node_at(b,pat-1).value;
                if(body.kind!=1 || body.start<a.start || body.end>a.end || pat_value.start<a.start || pat_value.end>body.start){valid=false;break;}var usize variant=0;
                valid=pattern(p,b,pat,e.target,arm,ref mut owners,ref mut locals,work,ref mut code,ref mut variant) && variant>base && mark(ref mut seen,variant-base,arm);
                if(body.expression!=0){let result=lookup::node(b,body.expression);valid=valid && result!=0;if(valid){let value=ir::node_at(b,result-1).value;valid=value.state==1 && value.type==e.type;}}
                else{valid=valid && no_tail(p,b,block,e.target,e.type,work,ref mut code);}
                count+=1;link=row.right;
            }
            if(!valid || count!=required || required==0){return clean::Problem{.code=if(code!=0){code}else{1001},.module_index,.start=e.start,.end=e.end};}
        }i+=1;
    }
    // Region bindings share syntax kind 17 with switch payloads.
    // Their owner is the retained lexical statement, rather than a switch arm.
    i=0;while(i<ir::statement_count(b)){if(!charge(work,1,ref mut code)){return clean::Problem{.code,.module_index,.start=0,.end=0};}let s=ir::statement_at(b,i).value;
        if(s.kind==17){let id=lookup::node(b,s.target);if(id==0){return clean::Problem{.code=1001,.module_index,.start=s.start,.end=s.end};}let e=ir::node_at(b,id-1).value;let root=e.place.root;
            if(e.kind!=17 || e.state!=2 || root==0 || root>ir::local_count(b) || !mark(ref mut owners,id,ir::node_count(b)+i+1)){return clean::Problem{.code=1001,.module_index,.start=e.start,.end=e.end};}let local=ir::local_at(b,root);
            if(local.kind!=6 || local.source_index!=s.target || local.scope!=e.place.scope || local.start!=e.start || local.end!=e.end || local.initializer!=0){return clean::Problem{.code=1001,.module_index,.start=e.start,.end=e.end};}
        }i+=1;
    }
    i=0;while(i<ir::node_count(b)){if(!charge(work,1,ref mut code)){return clean::Problem{.code,.module_index,.start=0,.end=0};}let e=ir::node_at(b,i).value;
        if((e.kind==16 || e.kind==17 || e.kind==18 || e.kind==19 || e.kind==32 || e.kind==33) && *v::at<usize>(ref owners,i)==0){return clean::Problem{.code=1001,.module_index,.start=e.start,.end=e.end};}
        if(e.kind==2 && e.binding!=0 && ir::local_at(b,e.binding).kind==4){let arm=*v::at<usize>(ref locals,e.binding-1);if(arm==0){return clean::Problem{.code=1001,.module_index,.start=e.start,.end=e.end};}let a=ir::node_at(b,arm-1).value;let body=ir::statement_at(b,lookup::statement(b,a.right)-1).value;if(e.type!=ir::local_at(b,e.binding).type || e.start<body.start || e.end>body.end){return clean::Problem{.code=1001,.module_index,.start=e.start,.end=e.end};}}
        i+=1;
    }
    i=1;while(i<=ir::local_count(b)){if(!charge(work,1,ref mut code)){return clean::Problem{.code,.module_index,.start=0,.end=0};}let l=ir::local_at(b,i);if(l.kind==4 && *v::at<usize>(ref locals,i-1)==0){return clean::Problem{.code=1001,.module_index,.start=l.start,.end=l.end};}i+=1;}
    return clean::Problem{.code=0,.module_index=0,.start=0,.end=0};
}
