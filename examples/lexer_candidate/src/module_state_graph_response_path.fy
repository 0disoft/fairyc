module module_state_graph_response_path;
import module_state_graph_program as program;
import module_state_graph_model as m;
import module_state_graph_response_common as common;
import std::bytes as bytes;
import std::vec as v;

struct Span{usize start;usize end;}
struct Site{common::Step step;usize next;usize alternative;usize callee;usize exit;usize trap;}
// Only the sealed Verified factory supplies these owned, immutable CFG facts.
pub resource Plan{v::Vec<Site> operations;v::Vec<Site> terms;v::Vec<Span> traps;bytes::Buffer data;usize start;cleanup{}}
pub struct Stack{usize[33] sites;}
pub struct Path{Stack stack;usize depth;usize next;usize alternative;usize exit;usize trap;usize cost;usize operations;}
pub fn usize exit(ref Path path){return path.exit;}
pub fn usize trap(ref Path path){return path.trap;}
pub fn usize cost(ref Path path){return path.cost;}
pub fn usize operations(ref Path path){return path.operations;}
pub fn ref u8[] trap_category(ref Plan plan,usize id){
    if(id==0 || id>v::len<Span>(ref plan.traps)){return bytes::view(ref plan.data)[..0];}
    let span=*v::at<Span>(ref plan.traps,id-1);return bytes::view(ref plan.data)[span.start..span.end];
}
fn bool enter(ref mut common::Work work){
    if(work.structure>16777216 || work.bytes>1073741824){return false;}return common::charge(work,1);
}
fn ref m::Graph graph(ref program::Verified proof,usize body){if(body==0){return program::root(proof);}return program::helper(proof,body-1);}
fn usize first(ref m::Graph g,usize block,usize op_base,usize term_base){
    let b=*v::at<m::Block>(m::blocks(g),block);return if(b.operation_count==0){2147483648 | (term_base+block+1)}else{op_base+b.operation_start+1};
}
fn bool before(ref u8[] a,ref u8[] b){var usize i=0;while(i<a.len && i<b.len){if(a[i]!=b[i]){return a[i]<b[i];}i+=1;}return a.len<b.len;}
fn u32 category(ref mut v::Vec<Span> traps,ref mut bytes::Buffer data,ref u8[] raw,ref mut common::Work work){
    if(!common::charge(work,1)){return 1002;}var usize at=0;
    while(at<v::len<Span>(traps)){
        if(!common::charge(work,1)){return 1002;}let span=*v::at<Span>(traps,at);let old=bytes::view(data)[span.start..span.end];
        if(!common::byte_charge(work,old.len+raw.len)){return 1002;}
        if(bytes::equal(old,raw)){return 0;}if(!common::byte_charge(work,old.len+raw.len)){return 1002;}if(before(raw,old)){break;}at+=1;
    }
    if(!common::charge(work,v::len<Span>(traps)-at+1) || !common::byte_charge(work,raw.len)){return 1002;}
    let start=bytes::len(data);if(!bytes::append(data,raw).is_ok){return 1003;}
    let span=Span{.start,.end=start+raw.len};if(!v::push<Span>(traps,span).is_ok){return 1003;}
    var usize i=v::len<Span>(traps)-1;while(i>at){let old=*v::at<Span>(traps,i-1);{let slot=v::at_mut<Span>(traps,i);*slot=old;}i-=1;}
    {let slot=v::at_mut<Span>(traps,at);*slot=span;}return 0;
}
fn Result<usize,u32> trap_id(ref v::Vec<Span> traps,ref bytes::Buffer data,ref u8[] raw,ref mut common::Work work){
    var usize i=0;while(i<v::len<Span>(traps)){
        if(!common::charge(work,1)){return .err(1002);}let span=*v::at<Span>(traps,i);let old=bytes::view(data)[span.start..span.end];
        if(!common::byte_charge(work,old.len+raw.len)){return .err(1002);}if(bytes::equal(old,raw)){return .ok(i+1);}i+=1;
    }return .err(1001);
}
pub fn Result<Plan,u32> build(ref program::Verified proof,ref mut common::Work work){
    if(!enter(work)){return .err(1002);}let count=program::helper_count(proof)+1;if(count>33){return .err(1002);}
    if(!common::charge(work,99)){return .err(1002);}var usize[33] op_base=[0;33];var usize[33] term_base=[0;33];var usize[33] starts=[0;33];
    var usize ops=0;var usize terms=0;var usize body=0;
    // Preflight all allocation lengths. Sparse SSA IDs never index this plan.
    while(body<count){
        if(!common::charge(work,1)){return .err(1002);}let g=graph(proof,body);let n=v::len<m::Operation>(m::operations(g));let blocks=v::len<m::Block>(m::blocks(g));
        if(n>1024-ops || blocks==0 || blocks>64){return .err(1002);}op_base[body]=ops;term_base[body]=terms;
        starts[body]=first(g,m::header(g).entry,ops,terms);ops+=n;terms+=blocks;body+=1;
    }
    var v::Vec<Span> traps=v::empty<Span>();var data=bytes::empty();body=0;
    while(body<count){
        if(!common::charge(work,1)){return .err(1002);}let g=graph(proof,body);var usize i=0;
        while(i<v::len<m::Block>(m::blocks(g))){
            if(!common::charge(work,1)){return .err(1002);}let t=v::at<m::Block>(m::blocks(g),i).term;
            if(t.kind==5){let code=category(ref mut traps,ref mut data,m::data(g)[t.category_start..t.category_end],work);if(code!=0){return .err(code);}}i+=1;
        }body+=1;
    }
    var v::Vec<Site> operations=v::empty<Site>();var v::Vec<Site> terminators=v::empty<Site>();body=0;
    while(body<count){
        if(!common::charge(work,1)){return .err(1002);}let g=graph(proof,body);var usize i=0;
        while(i<v::len<m::Block>(m::blocks(g))){
            if(!common::charge(work,1)){return .err(1002);}let b=*v::at<m::Block>(m::blocks(g),i);let term=2147483648 | (term_base[body]+i+1);var usize j=0;
            while(j<b.operation_count){
                if(!common::charge(work,1)){return .err(1002);}let index=b.operation_start+j;let op=*v::at<m::Operation>(m::operations(g),index);
                let next=if(j+1==b.operation_count){term}else{op_base[body]+index+2};var usize callee=0;
                if(op.kind==11){if(op.callee==0 || op.callee>=count){return .err(1001);}callee=starts[op.callee];}
                let site=Site{.step=common::Step{.body,.index,.operation=true},.next,.alternative=0,.callee,.exit=0,.trap=0};
                if(!v::push<Site>(ref mut operations,site).is_ok){return .err(1003);}j+=1;
            }
            let t=b.term;var usize next=0;var usize alternative=0;var usize exit=0;var usize trap=0;
            if(t.kind<=2){next=first(g,t.target,op_base[body],term_base[body]);if(t.kind==2){alternative=first(g,t.alternative,op_base[body],term_base[body]);}}
            else{exit=(t.kind-2) as usize;if(t.kind==5){trap=switch(trap_id(ref traps,ref data,m::data(g)[t.category_start..t.category_end],work)){case .ok(id):{id}case .err(code):{return .err(code);}};}}
            let site=Site{.step=common::Step{.body,.index=i,.operation=false},.next,.alternative,.callee=0,.exit,.trap};
            if(!v::push<Site>(ref mut terminators,site).is_ok){return .err(1003);}i+=1;
        }body+=1;
    }return .ok(Plan{.operations,.terms=terminators,.traps,.data,.start=starts[0]});
}
pub fn Path start(ref Plan plan){return Path{.stack=Stack{.sites=[0;33]},.depth=0,.next=plan.start,.alternative=0,.exit=0,.trap=0,.cost=0,.operations=0};}
fn Result<Site,u32> site_at(ref Plan plan,usize id){
    if(id>2147483648){let index=id-2147483649;if(index<v::len<Site>(ref plan.terms)){return .ok(*v::at<Site>(ref plan.terms,index));}}
    else if(id!=0 && id<=v::len<Site>(ref plan.operations)){return .ok(*v::at<Site>(ref plan.operations,id-1));}return .err(1001);
}
pub fn Result<common::Step,u32> advance(ref Plan plan,ref mut Path path,usize site,usize operation,ref mut common::Work work){
    if(!enter(work)){return .err(1002);}if(path.exit!=0 || path.depth>33 || site==0 || (site!=path.next && site!=path.alternative)){return .err(1001);}
    let n=switch(site_at(plan,site)){case .ok(n):{n}case .err(code):{return .err(code);}};
    let usize extra=if(n.step.operation){1}else{0};
    if(path.cost==18446744073709551615 || extra>18446744073709551615-path.operations || operation!=path.operations+extra){return .err(1001);}
    if(n.callee!=0 && path.depth>=33 || n.exit==4 && path.depth==0){return .err(1001);}
    path.cost+=1;path.operations+=extra;path.next=n.next;path.alternative=n.alternative;
    if(n.callee!=0){let depth=path.depth;path.stack.sites[depth]=n.next;path.depth=depth+1;path.next=n.callee;path.alternative=0;}
    else if(n.exit==4){let depth=path.depth-1;let next=path.stack.sites[depth];path.depth=depth;path.next=next;path.alternative=0;}
    else if(n.exit!=0){path.exit=n.exit;path.trap=n.trap;path.next=0;path.alternative=0;}return .ok(n.step);
}
pub fn Result<bool,u32> counters(ref Plan plan,ref Path path,usize status,usize cost,usize operations,ref mut common::Work work){
    if(!enter(work)){return .err(1002);}if(cost==path.cost && operations==path.operations){return .ok(status!=0 || path.exit!=0);}
    if(status==0 || path.exit!=0 || path.cost==18446744073709551615 || cost!=path.cost+1){return .ok(false);}
    var usize i=0;while(i<2){
        if(!common::charge(work,1)){return .err(1002);}let id=if(i==0){path.next}else{path.alternative};i+=1;if(id==0){continue;}
        let n=switch(site_at(plan,id)){case .ok(n):{n}case .err(code):{return .err(code);}};let usize extra=if(n.step.operation){1}else{0};
        if(extra<=18446744073709551615-path.operations && operations==path.operations+extra){return .ok(true);}
    }return .ok(false);
}
