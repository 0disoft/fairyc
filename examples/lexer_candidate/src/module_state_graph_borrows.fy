module module_state_graph_borrows;
import module_state_graph_model as m;
import module_state_graph_values as values;
import module_state_layout as state;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// Block-local owner metadata only. Key spans refer to the immutable graph data;
// views retain their original storage identity, never a runtime memory address.
struct Borrow{usize id;u32 codec;usize field;usize key_start;usize key_end;bool mapped;bool valid;}
// Nonowning plain metadata: resource-field mutable projections stay closed.
// Only the sealed core factory creates this ledger for a verified graph block.
pub struct Ledger{usize count;usize[1024] ids;u32[1024] codecs;usize[1024] fields;usize[1024] starts;usize[1024] ends;bool[1024] mapped;bool[1024] valid;}
pub fn Ledger empty(){return Ledger{.count=0,.ids=[0;1024],.codecs=[0;1024],.fields=[0;1024],.starts=[0;1024],.ends=[0;1024],.mapped=[false;1024],.valid=[false;1024]};}

fn Result<Borrow,u32> lookup(ref Ledger handles,usize id,ref mut usize work){
    if(handles.count>1024){return .err(1001);}var usize i=0;while(i<handles.count){
        if(!budget::charge(work,1)){return .err(1002);}let row=Borrow{.id=handles.ids[i],.codec=handles.codecs[i],.field=handles.fields[i],.key_start=handles.starts[i],.key_end=handles.ends[i],.mapped=handles.mapped[i],.valid=handles.valid[i]};
        if(row.id==id){return if(row.valid){.ok(row)}else{.err(1001)};}i+=1;
    }return .err(1001);
}
fn u32 ordinary(ref mut v::Vec<m::Value> all,ref mut v::Vec<m::Value> live,usize id,u32 codec,ref mut usize work){
    let row=m::Value{.id,.codec};let code=values::define(all,row,work);if(code!=0){return code;}
    if(!budget::charge(work,1)){return 1002;}return if(v::push<m::Value>(live,row).is_ok){0}else{1003};
}
fn u32 define(ref mut v::Vec<m::Value> all_borrows,ref mut Ledger handles,Borrow row,ref mut usize work){
    // Borrow IDs and ordinary value IDs are deliberately separate namespaces.
    if(handles.count>=1024){return 1002;}
    let code=values::define(all_borrows,m::Value{.id=row.id,.codec=row.codec},work);if(code!=0){return code;}
    if(!budget::charge(work,1)){return 1002;}let i=handles.count;
    handles.ids[i]=row.id;handles.codecs[i]=row.codec;handles.fields[i]=row.field;handles.starts[i]=row.key_start;handles.ends[i]=row.key_end;handles.mapped[i]=row.mapped;handles.valid[i]=row.valid;handles.count+=1;return 0;
}

pub fn u32 apply(m::Operation op,ref m::Graph graph,ref state::Plan layouts,ref mut v::Vec<m::Value> all,ref mut v::Vec<m::Value> live,ref mut v::Vec<m::Value> all_borrows,ref mut Ledger handles,bool helper,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}let header=m::header(graph);
    if(handles.count>1024){return 1001;}
    if(helper || op.kind<7 || op.kind>10 || header.capability==0 || op.capability!=header.capability || op.cost!=1 || op.callee!=0 || !values::absent(op.value) || op.view_offset>4294967295 || op.view_length>4294967295){return 1001;}
    if(op.kind==7 || op.kind==8){
        if(header.layout>=state::count(layouts) || (header.effects&1)!=1){return 1001;}
        let layout=state::at(layouts,header.layout);if(op.field>=layout.field_count){return 1001;}
        let field=state::field_at(layouts,layout.field_start+op.field);
        if(op.result!=0 || op.found==0 || op.borrow_input!=0 || op.borrow_result==0 || op.view_offset!=0 || op.view_length!=0){return 1001;}
        if(op.kind==7){if(field.kind!=1 || !values::absent(op.key)){return 1001;}}
        else{
            if(field.kind!=2 || op.key.input!=0){return 1001;}
            let code=values::operand(op.key,field.key,m::data(graph),live,work);if(code!=0){return code;}
        }
        let row=Borrow{.id=op.borrow_result,.codec=field.value,.field=op.field,.key_start=op.key.start,.key_end=op.key.end,.mapped=op.kind==8,.valid=true};
        let code=define(all_borrows,handles,row,work);if(code!=0){return code;}
        return ordinary(all,live,op.found,1,work);
    }
    if(op.field!=0 || !values::absent(op.key)){return 1001;}
    let source=switch(lookup(handles,op.borrow_input,work)){case .ok(row):{row}case .err(code):{return code;}};
    if(op.kind==9){
        if(op.result!=0 || op.found!=0 || op.borrow_result==0 || op.view_length==0){return 1001;}
        // Go validates the nonempty view shape here. Actual range validity,
        // including offset+length overflow and encoded widths, is runtime work.
        var row=source;row.id=op.borrow_result;return define(all_borrows,handles,row,work);
    }
    if(op.result==0 || op.found!=0 || op.borrow_result!=0 || op.view_offset!=0 || op.view_length!=0){return 1001;}
    return ordinary(all,live,op.result,source.codec,work);
}

// Called only after the core has validated the ordinary mutation and operand.
// Dynamic keys conservatively reject any same-field live borrow, even unused.
pub fn u32 invalidate(m::Operation op,ref m::Graph graph,ref mut Ledger handles,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}
    if(handles.count>1024){return 1001;}
    if(op.kind!=2 && op.kind!=3 && op.kind!=5 && op.kind!=6){return 0;}
    let mapped=op.kind==5 || op.kind==6;var usize i=0;
    while(i<handles.count){
        if(!budget::charge(work,1)){return 1002;}
        if(handles.valid[i] && handles.fields[i]==op.field){
            if(mapped){
                if(op.key.input!=0){return 1001;}let raw=m::data(graph);
                if(!handles.mapped[i] || handles.starts[i]>handles.ends[i] || handles.ends[i]>raw.len || op.key.start>op.key.end || op.key.end>raw.len){return 1001;}
                let key=raw[handles.starts[i]..handles.ends[i]];let changed=raw[op.key.start..op.key.end];
                if(!budget::charge(work,key.len+changed.len+1)){return 1002;}
                if(!bytes::equal(key,changed)){i+=1;continue;}
            }
            handles.valid[i]=false;
        }i+=1;
    }return 0;
}
// The program owner invokes this only for verified transitive write effects.
pub fn u32 invalidate_all(ref mut Ledger handles,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(handles.count>1024){return 1001;}var usize i=0;
    while(i<handles.count){
        if(!budget::charge(work,1)){return 1002;}handles.valid[i]=false;i+=1;
    }return 0;
}
