module module_state_graph_native_plan;
import module_state_graph_program as program;
import module_state_graph_model as m;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

resource Body{v::Vec<usize> values;v::Vec<usize> borrows;cleanup{}}
struct Span{usize start;usize end;}
// Detached structural preparation only; no Complete or execution authority.
pub resource Plan{v::Vec<Body> bodies;v::Vec<Span> traps;bytes::Buffer data;bytes::Buffer json;cleanup{}}
pub fn usize body_count(ref Plan plan){return v::len<Body>(ref plan.bodies);}
pub fn usize value_count(ref Plan plan,usize body){return v::len<usize>(ref v::at<Body>(ref plan.bodies,body).values);}
pub fn usize borrow_count(ref Plan plan,usize body){return v::len<usize>(ref v::at<Body>(ref plan.bodies,body).borrows);}
pub fn usize trap_count(ref Plan plan){return v::len<Span>(ref plan.traps);}
pub fn ref u8[] trap(ref Plan plan,usize index){let span=*v::at<Span>(ref plan.traps,index);return bytes::view(ref plan.data)[span.start..span.end];}
pub fn ref u8[] trap_json(ref Plan plan){return bytes::view(ref plan.json);}
fn Result<usize,u32> slot(ref v::Vec<usize> ids,usize id,ref mut usize work){
    var usize i=0;while(i<v::len<usize>(ids)){if(!budget::charge(work,1)){return .err(1002);}if(*v::at<usize>(ids,i)==id){return .ok(i);}i+=1;}return .err(1001);
}
pub fn Result<usize,u32> value_slot(ref Plan plan,usize body,usize id,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(body>=body_count(plan)){return .err(1001);}return slot(ref v::at<Body>(ref plan.bodies,body).values,id,work);
}
pub fn Result<usize,u32> borrow_slot(ref Plan plan,usize body,usize id,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(body>=body_count(plan)){return .err(1001);}return slot(ref v::at<Body>(ref plan.bodies,body).borrows,id,work);
}
pub fn Result<usize,u32> trap_id(ref Plan plan,ref u8[] category,ref mut usize work){
    if(category.len>m::byte_limit() || !budget::charge(work,category.len+1)){return .err(1002);}if(category.len==0){return .ok(0);}
    var usize i=0;while(i<trap_count(plan)){let current=trap(plan,i);if(!budget::charge(work,current.len+category.len+1)){return .err(1002);}if(bytes::equal(current,category)){return .ok(i+1);}i+=1;}return .err(1001);
}
fn ref m::Graph graph(ref program::Verified proof,usize body){if(body==0){return program::root(proof);}return program::helper(proof,body-1);}
// Count every slot before allocating any slot storage. IDs are sparse u32
// identities, never allocation lengths. Verified supplies global uniqueness.
fn u32 preflight(ref program::Verified proof,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}var usize total=0;var usize body=0;
    while(body<=program::helper_count(proof)){if(!budget::charge(work,1)){return 1002;}let g=graph(proof,body);let parameters=v::len<m::Value>(m::parameters(g));
        if(parameters>4096-total){return 1002;}total+=parameters;var usize i=0;
        while(i<v::len<m::Operation>(m::operations(g))){if(!budget::charge(work,1)){return 1002;}let op=*v::at<m::Operation>(m::operations(g),i);
            let usize count=if(op.result!=0){1}else{0};let usize found=if(op.found!=0){1}else{0};let usize borrow=if(op.borrow_result!=0){1}else{0};
            if(count+found+borrow>4096-total){return 1002;}total+=count+found+borrow;i+=1;}body+=1;
    }return 0;
}
fn u32 push(ref mut v::Vec<usize> ids,usize id,ref mut usize work){if(!budget::charge(work,1)){return 1002;}return if(v::push<usize>(ids,id).is_ok){0}else{1003};}
fn u32 category(ref mut v::Vec<Span> traps,ref mut bytes::Buffer data,ref u8[] raw,ref mut usize work){
    if(raw.len==0){return 0;}if(raw.len>127 || !budget::charge(work,raw.len+1)){return 1002;}
    var usize at=0;while(at<v::len<Span>(traps)){let span=*v::at<Span>(traps,at);let current=bytes::view(data)[span.start..span.end];if(!budget::charge(work,current.len+raw.len+1)){return 1002;}if(bytes::equal(current,raw)){return 0;}if(budget::before(raw,current)){break;}at+=1;}
    let start=bytes::len(data);if(raw.len>m::byte_limit()-start){return 1002;}
    // Prepay shifts before growing either owned collection.
    if(!budget::charge(work,v::len<Span>(traps)-at+1)){return 1002;}
    if(!bytes::append(data,raw).is_ok){return 1003;}let span=Span{.start,.end=start+raw.len};
    if(!v::push<Span>(traps,span).is_ok){return 1003;}var usize i=v::len<Span>(traps)-1;
    while(i>at){let previous=*v::at<Span>(traps,i-1);{let slot=v::at_mut<Span>(traps,i);*slot=previous;}i-=1;}{let slot=v::at_mut<Span>(traps,at);*slot=span;}return 0;
}
fn ref u8[] span_bytes(ref v::Vec<Span> traps,ref bytes::Buffer data,usize index){let span=*v::at<Span>(traps,index);return bytes::view(data)[span.start..span.end];}
fn u32 append(ref mut bytes::Buffer out,ref u8[] raw,ref mut usize work){
    if(raw.len>m::byte_limit()-bytes::len(out) || !budget::charge(work,raw.len+1)){return 1002;}return if(bytes::append(out,raw).is_ok){0}else{1003};
}
pub fn Result<Plan,u32> build(ref program::Verified proof,ref mut usize work){
    let bounded=preflight(proof,work);if(bounded!=0){return .err(bounded);}
    var v::Vec<Body> bodies=v::empty<Body>();var v::Vec<Span> traps=v::empty<Span>();var data=bytes::empty();var usize body=0;
    while(body<=program::helper_count(proof)){if(!budget::charge(work,1)){return .err(1002);}let g=graph(proof,body);var v::Vec<usize> values=v::empty<usize>();var v::Vec<usize> borrows=v::empty<usize>();var usize i=0;
        while(i<v::len<m::Block>(m::blocks(g))){if(!budget::charge(work,1)){return .err(1002);}let block=*v::at<m::Block>(m::blocks(g),i);var usize j=0;
            while(j<block.parameter_count){let code=push(ref mut values,v::at<m::Value>(m::parameters(g),block.parameter_start+j).id,work);if(code!=0){return .err(code);}j+=1;}j=0;
            while(j<block.operation_count){if(!budget::charge(work,1)){return .err(1002);}let op=*v::at<m::Operation>(m::operations(g),block.operation_start+j);var u32 code=0;
                if(op.result!=0){code=push(ref mut values,op.result,work);}if(code==0 && op.found!=0){code=push(ref mut values,op.found,work);}if(code==0 && op.borrow_result!=0){code=push(ref mut borrows,op.borrow_result,work);}if(code!=0){return .err(code);}j+=1;}
            if(block.term.kind==5){let code=category(ref mut traps,ref mut data,m::data(g)[block.term.category_start..block.term.category_end],work);if(code!=0){return .err(code);}}i+=1;
        }if(!budget::charge(work,1)){return .err(1002);}if(!v::push<Body>(ref mut bodies,Body{.values,.borrows}).is_ok){return .err(1003);}body+=1;
    }
    var output=bytes::empty();var u32 code=append(ref mut output,if(v::len<Span>(ref traps)==0){b"null"}else{b"["},work);if(code!=0){return .err(code);}var usize i=0;
    // Verified categories contain only [a-z0-9-], so Go encoding/json needs
    // no escapes, whitespace, or HTML substitutions for these exact bytes.
    while(i<v::len<Span>(ref traps)){if(i!=0){code=append(ref mut output,b",",work);}if(code==0){code=append(ref mut output,b"\"",work);}if(code==0){code=append(ref mut output,span_bytes(ref traps,ref data,i),work);}if(code==0){code=append(ref mut output,b"\"",work);}if(code!=0){return .err(code);}i+=1;}
    if(v::len<Span>(ref traps)!=0){code=append(ref mut output,b"]",work);if(code!=0){return .err(code);}}return .ok(Plan{.bodies,.traps,.data,.json=output});
}
