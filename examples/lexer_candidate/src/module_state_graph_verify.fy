module module_state_graph_verify;
import module_state_graph_model as m;
import module_state_graph_values as values;
import module_state_graph_borrows as borrows;
import module_state_layout as state;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// No public constructor. The proof retains its independent immutable layouts;
// mutating the caller's original layout cannot change a published graph owner.
pub resource Verified{m::Graph graph;state::Plan layouts;cleanup{}}
pub fn ref m::Graph graph(ref Verified proof){return ref proof.graph;}
pub fn ref state::Plan layouts(ref Verified proof){return ref proof.layouts;}
pub fn Result<Verified,u32> verify(m::Graph graph,ref state::Plan layouts,ref mut usize work){
    let code=validate(ref graph,layouts,work);if(code!=0){return .err(code);}
    var usize measured=0;let owned=switch(state::copy(layouts,ref mut measured)){
        case .ok(p):{if(!budget::charge(work,measured)){return .err(1002);}p}
        case .err(code):{if(!budget::charge(work,measured)){return .err(1002);}return .err(if(code==10){1002}else if(code==11){1003}else{1001});}
    };
    return .ok(Verified{.graph,.layouts=owned});
}
fn u32 define_read(ref mut v::Vec<m::Value> all,ref mut v::Vec<m::Value> live,usize id,u32 codec,ref mut usize work){
    let row=m::Value{.id,.codec};let code=values::define(all,row,work);if(code!=0){return code;}if(!budget::charge(work,1)){return 1002;}return if(v::push<m::Value>(live,row).is_ok){0}else{1003};
}
fn u32 operation(m::Operation op,ref m::Graph graph,ref state::Plan layouts,ref mut v::Vec<m::Value> all,ref mut v::Vec<m::Value> live,ref mut v::Vec<m::Value> all_borrows,ref mut borrows::Ledger handles,ref v::Vec<u32> helpers,bool helper,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}let layout=state::at(layouts,m::header(graph).layout);
    if(op.capability!=m::header(graph).capability || op.cost!=1){return 1001;}
    if(op.kind==11){
        if(op.callee==0 || op.callee>v::len<u32>(helpers) || op.field!=0 || !values::absent(op.key) || !values::absent(op.value) || op.result!=0 || op.found!=0 || op.borrow_input!=0 || op.borrow_result!=0 || op.view_offset!=0 || op.view_length!=0){return 1001;}
        let effects=*v::at<u32>(helpers,op.callee-1);if(effects>3 || (m::header(graph).effects&effects)!=effects){return 1001;}
        return if((effects&2)!=0){borrows::invalidate_all(handles,work)}else{0};
    }
    if(op.kind>=7 && op.kind<=10){return borrows::apply(op,graph,layouts,all,live,all_borrows,handles,helper,work);}
    if(op.kind<1 || op.kind>6 || op.field>=layout.field_count || op.callee!=0 || op.borrow_input!=0 || op.borrow_result!=0 || op.view_offset!=0 || op.view_length!=0){return 1001;}
    let field=state::field_at(layouts,layout.field_start+op.field);let read=op.kind==1 || op.kind==4;let write=op.kind==2 || op.kind==5;let cell=op.kind<=3;
    if((field.kind==1)!=cell || (read && (op.result==0 || op.found==0 || op.result==op.found)) || (!read && (op.result!=0 || op.found!=0))){return 1001;}
    let u32 required=if(read){1}else{2};if((m::header(graph).effects&required)!=required){return 1001;}
    if(cell){if(!values::absent(op.key)){return 1001;}}else{let code=values::operand(op.key,field.key,m::data(graph),live,work);if(code!=0){return code;}}
    if(write){let code=values::operand(op.value,field.value,m::data(graph),live,work);if(code!=0){return code;}}else if(!values::absent(op.value)){return 1001;}
    let invalidated=borrows::invalidate(op,graph,handles,work);if(invalidated!=0){return invalidated;}
    if(read){let code=define_read(all,live,op.result,field.value,work);if(code!=0){return code;}return define_read(all,live,op.found,1,work);}return 0;
}
fn u32 edge(ref m::Graph graph,usize target,usize start,usize count,ref v::Vec<m::Value> live,ref mut usize work){
    if(target>=v::len<m::Block>(m::blocks(graph)) || !m::range(start,count,v::len<usize>(m::arguments(graph)))){return 1001;}let block=*v::at<m::Block>(m::blocks(graph),target);
    if(count!=block.parameter_count){return 1001;}var usize i=0;while(i<count){let found=switch(values::lookup(live,*v::at<usize>(m::arguments(graph),start+i),work)){case .ok(t):{t}case .err(code):{return code;}};
        if(found==0 || found!=v::at<m::Value>(m::parameters(graph),block.parameter_start+i).codec){return 1001;}i+=1;}return 0;
}
fn u32 terminator(m::Terminator term,ref m::Graph graph,ref v::Vec<m::Value> live,bool helper,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(term.cost!=1 || term.kind<1 || term.kind>6 || (helper && term.kind==3) || (!helper && term.kind==6)){return 1001;}
    if(term.kind==1 || term.kind==2){if(term.category_start!=0 || term.category_end!=0){return 1001;}
        if(term.kind==1){if(term.condition!=0 || term.alternative!=0 || term.alternate_start!=0 || term.alternate_count!=0){return 1001;}}
        else{let found=switch(values::lookup(live,term.condition,work)){case .ok(t):{t}case .err(code):{return code;}};if(found!=1){return 1001;}}
        let code=edge(graph,term.target,term.argument_start,term.argument_count,live,work);if(code!=0){return code;}
        return if(term.kind==2){edge(graph,term.alternative,term.alternate_start,term.alternate_count,live,work)}else{0};
    }
    if(term.condition!=0 || term.target!=0 || term.argument_start!=0 || term.argument_count!=0 || term.alternative!=0 || term.alternate_start!=0 || term.alternate_count!=0){return 1001;}
    if(term.kind!=5){return if(term.category_start==0 && term.category_end==0){0}else{1001};}
    if(term.category_start>term.category_end || term.category_end>m::data(graph).len){return 1001;}return values::category(m::data(graph)[term.category_start..term.category_end],work);
}
fn u32 topology(ref m::Graph graph,ref mut usize work){
    let count=v::len<m::Block>(m::blocks(graph));var usize[64] incoming=[0;64];var bool[64] reached=[false;64];var usize[64] queue=[0;64];var usize i=0;
    while(i<count){if(!budget::charge(work,1)){return 1002;}let term=v::at<m::Block>(m::blocks(graph),i).term;if(term.kind<=2){incoming[term.target]+=1;if(term.kind==2){incoming[term.alternative]+=1;}}i+=1;}
    if(incoming[m::header(graph).entry]!=0){return 1001;}var usize read=0;var usize written=1;queue[0]=m::header(graph).entry;reached[m::header(graph).entry]=true;
    while(read<written){if(!budget::charge(work,1)){return 1002;}let term=v::at<m::Block>(m::blocks(graph),queue[read]).term;var usize e=0;let usize edges=if(term.kind==2){2}else if(term.kind==1){1}else{0};
        while(e<edges){let target=if(e==0){term.target}else{term.alternative};if(!reached[target]){reached[target]=true;queue[written]=target;written+=1;}e+=1;}read+=1;}
    if(written!=count){return 1001;}read=0;written=0;i=0;while(i<count){if(incoming[i]==0){queue[written]=i;written+=1;}i+=1;}
    while(read<written){if(!budget::charge(work,1)){return 1002;}let term=v::at<m::Block>(m::blocks(graph),queue[read]).term;var usize e=0;let usize edges=if(term.kind==2){2}else if(term.kind==1){1}else{0};
        while(e<edges){let target=if(e==0){term.target}else{term.alternative};incoming[target]-=1;if(incoming[target]==0){queue[written]=target;written+=1;}e+=1;}read+=1;}return if(written==count){0}else{1001};
}
// Detached validity only. This grants neither current-source Complete authority,
// executable state access, artifact restoration nor a cache hit.
pub fn u32 validate(ref m::Graph graph,ref state::Plan layouts,ref mut usize work){
    let helpers=v::empty<u32>();return validate_function(graph,layouts,ref helpers,false,false,work);
}
// Context-aware validation returns no proof. Only the complete program factory
// may bind helper names, reachability and exact transitive effects to authority.
pub fn u32 validate_function(ref m::Graph graph,ref state::Plan layouts,ref v::Vec<u32> helpers,bool exact,bool helper,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}let count=v::len<m::Block>(m::blocks(graph));
    if(count==0 || count>m::block_limit() || v::len<m::Operation>(m::operations(graph))>m::operation_limit() || m::data(graph).len>m::byte_limit()){return 1002;}
    if(m::header(graph).layout>=state::count(layouts) || m::header(graph).entry>=count || m::header(graph).capability==0 || m::header(graph).effects>3){return 1001;}
    var usize measured=0;let layout_code=state::validate_graph(layouts,ref mut measured);if(!budget::charge(work,measured)){return 1002;}if(layout_code!=0){return if(layout_code==10){1002}else if(layout_code==11){1003}else{1001};}
    var usize parameters=0;var usize operations=0;var usize arguments=0;var usize i=0;var v::Vec<m::Value> all=v::empty<m::Value>();var v::Vec<m::Value> all_borrows=v::empty<m::Value>();var u32 required=0;
    // Validate every packed span before an edge can inspect its target block.
    while(i<count){if(!budget::charge(work,1)){return 1002;}let block=*v::at<m::Block>(m::blocks(graph),i);
        if(block.id!=i || block.parameter_start!=parameters || block.operation_start!=operations || !m::range(parameters,block.parameter_count,v::len<m::Value>(m::parameters(graph))) || !m::range(operations,block.operation_count,v::len<m::Operation>(m::operations(graph))) || (i==m::header(graph).entry && block.parameter_count!=0)){return 1001;}
        var usize j=0;while(j<block.parameter_count){let code=values::define(ref mut all,*v::at<m::Value>(m::parameters(graph),parameters+j),work);if(code!=0){return code;}j+=1;}
        parameters+=block.parameter_count;operations+=block.operation_count;let term=block.term;
        if(term.kind<=2){if(term.argument_start!=arguments || !m::range(arguments,term.argument_count,v::len<usize>(m::arguments(graph)))){return 1001;}arguments+=term.argument_count;
            if(term.kind==2){if(term.alternate_start!=arguments || !m::range(arguments,term.alternate_count,v::len<usize>(m::arguments(graph)))){return 1001;}arguments+=term.alternate_count;}}i+=1;
    }
    if(parameters!=v::len<m::Value>(m::parameters(graph)) || operations!=v::len<m::Operation>(m::operations(graph)) || arguments!=v::len<usize>(m::arguments(graph))){return 1001;}
    i=0;while(i<count){let block=*v::at<m::Block>(m::blocks(graph),i);var v::Vec<m::Value> live=v::empty<m::Value>();var handles=borrows::empty();var usize j=0;
        while(j<block.parameter_count){if(!budget::charge(work,1)){return 1002;}if(!v::push<m::Value>(ref mut live,*v::at<m::Value>(m::parameters(graph),block.parameter_start+j)).is_ok){return 1003;}j+=1;}j=0;
        while(j<block.operation_count){let op=*v::at<m::Operation>(m::operations(graph),block.operation_start+j);let code=operation(op,graph,layouts,ref mut all,ref mut live,ref mut all_borrows,ref mut handles,helpers,helper,work);if(code!=0){return code;}
            if(op.kind==1 || op.kind==4 || op.kind==7 || op.kind==8){required|=1;}else if(op.kind==2 || op.kind==3 || op.kind==5 || op.kind==6){required|=2;}else if(op.kind==11){required|=*v::at<u32>(helpers,op.callee-1);}j+=1;}
        let code=terminator(block.term,graph,ref live,helper,work);if(code!=0){return code;}i+=1;
    }if(exact && required!=m::header(graph).effects){return 1001;}return topology(graph,work);
}
