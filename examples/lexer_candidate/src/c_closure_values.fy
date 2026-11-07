module c_closure_values;
import mir_model as m;
import hir_shapes as shapes;
import nominal_identity as nominal;
import pointer_types as pointers;
import reference_types as refs;
import array_types as arrays;
import tagged_types as tagged;
import c_flow as flow;
import c_text as w;
import std::bytes as bytes;

// This representation is private to emitted code. MIR stores target IDs and
// copied locals only; native invoke addresses are bound here, never serialized.
pub fn bool present(ref m::Program p){
    if(m::closure_count(p)!=0){return true;}
    var usize i=1;while(i<=m::local_count(p)){if(m::local_at(p,i).type==13){return true;}i+=1;}
    i=1;while(i<=m::target_count(p)){if(m::target_at(p,i).result==13){return true;}i+=1;}
    i=0;while(i<m::signature_count(p)){if(m::signature_at(p,i)==13){return true;}i+=1;}
    i=1;while(i<=shapes::field_count(m::nominal(p))){if(shapes::field_at(m::nominal(p),i).type==13){return true;}i+=1;}return false;
}
fn bool contains(ref m::Program p,u32 type,ref bool[] marked,usize depth){
    if(type==13){return true;}if(depth>=64){return false;}
    if(pointers::valid(type)){return contains(p,pointers::inner(type),marked,depth+1);}
    if(refs::valid(type)){return contains(p,refs::inner(type),marked,depth+1);}
    if(tagged::option(type)){return contains(p,tagged::inner(type),marked,depth+1);}
    if(tagged::result(type)){return contains(p,tagged::success(type),marked,depth+1) || contains(p,tagged::failure(type),marked,depth+1);}
    if(arrays::valid(type) && !arrays::owned(type)){return contains(p,arrays::element(type),marked,depth+1);}
    let usize slot=shapes::find(m::nominal(p),type);return slot!=0 && marked[slot-1];
}
fn bool ordinary(ref m::Program p,u32 type){return nominal::valid(type) && shapes::abi_find(m::nominal(p),type)==0;}
fn bool transport(ref m::Program p,u32 type,ref bool[] marked){return !contains(p,type,marked,0) || type==13 || ordinary(p,type);}
// Keep wrapper and foreign/raw transports closed independently of the frontend.
// A fixed point avoids repeatedly walking shared nested record subgraphs.
pub fn bool admitted(ref m::Program p){
    if(!present(p)){return true;}
    let ref shapes::Table table=m::nominal(p);var bool[512] marked=[false;512];var bool changed=true;var usize round=0;
    while(changed && round<=shapes::count(table)){
        changed=false;var usize i=0;while(i<shapes::count(table)){let shape=shapes::at(table,i);var usize j=0;
            while(!marked[i] && j<shape.field_count){if(contains(p,shapes::field_at(table,shape.field_start+j+1).type,ref marked[..],0)){marked[i]=true;changed=true;}j+=1;}i+=1;
        }round+=1;
    }
    var usize i=0;while(i<shapes::count(table)){let shape=shapes::at(table,i);
        if(marked[i]){if(!ordinary(p,shape.type)){return false;}var usize j=0;while(j<shape.field_count){let u32 type=shapes::field_at(table,shape.field_start+j+1).type;
            // Nested closure carriers remain outside the current shared native
            // transport boundary. Direct closure fields and inert siblings copy.
            if(!transport(p,type,ref marked[..]) || type!=13 && contains(p,type,ref marked[..],0)){return false;}j+=1;}}i+=1;
    }
    i=1;while(i<=m::local_count(p)){if(!transport(p,m::local_at(p,i).type,ref marked[..])){return false;}i+=1;}
    i=1;while(i<=m::target_count(p)){let t=m::target_at(p,i);
        if(!transport(p,t.result,ref marked[..]) || t.foreign && contains(p,t.result,ref marked[..],0)){return false;}
        var usize j=0;while(j<t.signature_count){let u32 type=m::signature_at(p,t.signature_start+j);if(!transport(p,type,ref marked[..]) || t.foreign && contains(p,type,ref marked[..],0)){return false;}j+=1;}i+=1;
    }
    i=0;while(i<m::closure_count(p)){let c=m::closure_at(p,i);if(c.target==0 || c.target>m::target_count(p) || c.captures>4){return false;}
        let t=m::target_at(p,c.target);if(t.external || t.foreign || t.unsafe_call || t.effects!=0 || t.result!=1 || t.signature_count!=c.captures+1){return false;}
        var usize j=0;while(j<t.signature_count){if(m::signature_at(p,t.signature_start+j)!=1){return false;}j+=1;}
        var bool body=false;j=1;while(j<=m::function_count(p)){body=body || m::function_at(p,j).target==c.target;j+=1;}if(!body){return false;}
        j=0;while(j<i){if(m::closure_at(p,j).target==c.target){return false;}j+=1;}i+=1;
    }
    i=1;while(i<=m::operation_count(p)){let o=m::operation_at(p,i);
        // No layout query, raw access, cast, comparison or wrapper operation may
        // observe the private representation, including through nested structs.
        if((contains(p,o.type,ref marked[..],0) || contains(p,o.owner_type,ref marked[..],0)) && o.kind!=2 && o.kind!=6 && o.kind!=7 && o.kind!=8 && o.kind!=40){return false;}
        if(o.kind==40){var bool found=false;var usize j=0;while(j<m::closure_count(p)){let c=m::closure_at(p,j);found=found || c.target==(o.operator as usize) && c.captures==o.argument_count;j+=1;}if(!found || o.type!=13 || o.owner_type!=13 || o.argument_count>4){return false;}}
        i+=1;
    }return true;
}
pub fn bool definition(ref m::Program p,ref mut bytes::Buffer out,ref mut w::State s){
    return !present(p) || w::text(out,s,b"struct fy_closure_i32{int32_t (*fy_invoke)(const int32_t *,int32_t);int32_t fy_captures[4];};\n");
}
fn bool adapter(ref mut bytes::Buffer out,ref mut w::State s,usize target){return w::text(out,s,b"fy_closure_") && w::number(out,s,target);}
// All actual helper prototypes precede these adapters. Construction itself is
// a target reachability edge; do not infer helper use from direct calls alone.
pub fn bool adapters(ref m::Program p,ref flow::Plan plan,ref mut bytes::Buffer out,ref mut w::State s){
    var usize i=0;while(i<m::closure_count(p)){let c=m::closure_at(p,i);
        if(!w::text(out,s,b"int32_t ") || !adapter(out,s,c.target) || !w::text(out,s,b"(const int32_t *fy_captures,int32_t fy_argument){\n")){return false;}
        if(c.captures==0 && !w::text(out,s,b"    (void)fy_captures;\n")){return false;}
        if(!w::text(out,s,if(flow::stops(plan,c.target)){b"    "}else{b"    return "}) || !w::function(out,s,c.target) || !w::text(out,s,b"(")){return false;}
        var usize j=0;while(j<c.captures){if(!w::text(out,s,b"fy_captures[") || !w::number(out,s,j) || !w::text(out,s,b"],")){return false;}j+=1;}
        if(!w::text(out,s,b"fy_argument);\n}\n")){return false;}i+=1;
    }return true;
}
pub fn bool create(ref m::Program p,m::Operation o,ref mut bytes::Buffer out,ref mut w::State s){
    if(!w::text(out,s,b"    ") || !w::local(out,s,o.result) || !w::text(out,s,b"=(struct fy_closure_i32){&") || !adapter(out,s,o.operator as usize) || !w::text(out,s,b",{")){return false;}
    var usize i=0;while(i<4){if(i!=0 && !w::text(out,s,b",")){return false;}
        if(i<o.argument_count){if(!w::local(out,s,m::edge_at(p,o.argument_start+i))){return false;}}else if(!w::text(out,s,b"INT32_C(0)")){return false;}i+=1;
    }return w::text(out,s,b"}};\n");
}
pub fn bool call(m::Block b,ref mut bytes::Buffer out,ref mut w::State s){
    // The MIR inputs are already evaluated locals; snapshots make their order
    // explicit and read the callable exactly once without C argument ordering.
    return w::text(out,s,b"    {\n        struct fy_closure_i32 fy_callable=") && w::local(out,s,b.value)
        && w::text(out,s,b";\n        int32_t fy_argument=") && w::local(out,s,b.left)
        && w::text(out,s,b";\n        ") && w::local(out,s,b.destination)
        && w::text(out,s,b"=fy_callable.fy_invoke(fy_callable.fy_captures,fy_argument);\n    }\n    ") && w::jump(out,s,b.target);
}
