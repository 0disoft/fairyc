module c_guard_facts;
import mir_model as m;

// Conservative facts from unique immutable definitions, copies and signed unary
// literals. Mutable/parameter/join values stay unknown. Signed magnitudes avoid
// overflowing the compiler while proving a target-width trap.
struct Integer { bool known; bool negative; usize magnitude; }
fn Integer unknown() { return Integer{.known=false,.negative=false,.magnitude=0}; }
fn Integer literal(ref m::Program p, usize definition) {
    if (definition == 0) { return Integer{.known=false,.negative=false,.magnitude=0}; }
    let o=m::operation_at(p,definition);
    if (o.kind != 1 || o.type < 1 || o.type > 5) { return Integer{.known=false,.negative=false,.magnitude=0}; }
    let raw=m::literal_text(p,definition); let bool negative=raw.len>0 && raw[0]==45;
    var usize i=if(negative){1}else{0}; var usize magnitude=0;
    while(i<raw.len){
        if(raw[i]<48 || raw[i]>57){return Integer{.known=false,.negative=false,.magnitude=0};}
        let usize digit=(raw[i]-48) as usize;
        if(magnitude>(18446744073709551615-digit)/10){return Integer{.known=false,.negative=false,.magnitude=0};}
        magnitude=magnitude*10+digit;i+=1;
    }
    return Integer{.known=true,.negative=negative && magnitude!=0,.magnitude};
}
fn Integer value(ref m::Program p, usize id, usize depth) {
    if(id==0 || depth>=64 || m::local_at(p,id).mutable || m::local_at(p,id).parameter){return unknown();}
    var usize at=1;var usize definition=0;
    while(at<=m::operation_count(p)){
        let o=m::operation_at(p,at);
        if(o.result==id){if(definition!=0){return unknown();}definition=at;}at+=1;
    }
    if(definition==0){return unknown();}
    let o=m::operation_at(p,definition);
    if(o.kind==1){return literal(p,definition);}
    if(o.kind==2){return value(p,o.left,depth+1);}
    if(o.kind==3 && (o.operator==1 || o.operator==2)){
        let a=value(p,o.left,depth+1);if(!a.known){return a;}
        let bool negative=if(o.operator==2 && a.magnitude!=0){!a.negative}else{a.negative};
        return if(outside(o.type,negative,a.magnitude)){unknown()}else{Integer{.known=true,.negative,.magnitude=a.magnitude}};
    }
    return unknown();
}
fn usize maximum(u32 type) {
    return if(type==1){2147483647}else if(type==2){9223372036854775807}
        else if(type==3){255}else if(type==4){4294967295}else{18446744073709551615};
}
fn bool outside(u32 type, bool negative, usize magnitude) {
    return if(negative){if(type==1 || type==2){magnitude>maximum(type)+1}else{magnitude!=0}}
        else{magnitude>maximum(type)};
}
pub fn u32 failure(ref m::Program p, m::Block b) {
    if(b.kind!=4 || b.operator==0 || b.type<1 || b.type>5){return 0;}
    let Integer a=value(p,b.left,0);let Integer z=value(p,b.right,0);
    if(b.right==0){return if(a.known && outside(b.type,!a.negative,a.magnitude)){1}else{0};}
    if(b.operator==6 || b.operator==7){
        if(z.known && z.magnitude==0){return 2;}
        return if(a.known && z.known && (b.type==1 || b.type==2) && a.negative
            && a.magnitude==maximum(b.type)+1 && z.negative && z.magnitude==1){3}else{0};
    }
    if(b.operator==11 || b.operator==12){
        if(!z.known){return 0;}
        let usize width=if(b.type==3){8}else if(b.type==1 || b.type==4){32}else{64};
        if(z.negative || z.magnitude>=width){return 4;}
        return if(b.operator==11 && a.known && (a.negative || a.magnitude>(maximum(b.type)>>z.magnitude))){1}else{0};
    }
    if(!a.known || !z.known){return 0;}
    if(b.operator==5){
        let bool negative=a.negative!=z.negative;
        if(negative && b.type!=1 && b.type!=2){return if(a.magnitude!=0 && z.magnitude!=0){1}else{0};}
        let usize limit=maximum(b.type)+if(negative){1}else{0};
        return if(z.magnitude!=0 && a.magnitude>limit/z.magnitude){1}else{0};
    }
    if(b.operator!=1 && b.operator!=2){return 0;}
    let bool negative=if(b.operator==2 && z.magnitude!=0){!z.negative}else{z.negative};
    if(a.negative==negative){
        if(outside(b.type,a.negative,a.magnitude) || outside(b.type,negative,z.magnitude)){return 1;}
        let usize limit=maximum(b.type)+if(a.negative){1}else{0};
        return if(a.magnitude>limit-z.magnitude){1}else{0};
    }
    return if(a.magnitude>=z.magnitude){if(outside(b.type,a.negative,a.magnitude-z.magnitude)){1}else{0}}
        else{if(outside(b.type,negative,z.magnitude-a.magnitude)){1}else{0}};
}
