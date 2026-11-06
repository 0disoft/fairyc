module module_state_graph_source_flow;
import module_state_graph_source_context as context;
import module_state_graph_source_bindings as bindings;
import module_state_graph_source_operands as operands;
import module_state_graph_model as m;
import module_state_graph_values as values;
import module_state_layout as state;
import project_cache_encoding as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Source production, not proof publication. The independent program factory
// validates effects, types, helper reachability and borrow lifetimes afterwards.
// Internal cursors are block ID + 1; zero is a closed path, not entry block 0.
fn bool expression(ref syntax::Parsed p,usize id){return id!=0 && id<=syntax::expression_count(p);}
fn bool statement(ref syntax::Parsed p,usize id){return id!=0 && id<=syntax::statement_count(p);}

pub fn Result<u32,u32> effects(ref syntax::Parsed p,syntax::Declaration d,ref mut context::Span span,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}
    if(d.attribute_count>1){return .err(1001);}if(d.attribute_count==0){return .ok(0);}
    let a=syntax::attribute_at(p,d.attribute_start);
    if(a.name_start>a.name_end || a.name_end>syntax::source_view(p).len){return .err(1001);}
    if(!budget::charge(work,a.name_end-a.name_start+1)){return .err(1002);}
    if(!bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"effects")){return .err(1001);}
    span.start=a.start;span.end=a.end;var u32 mask=0;var usize i=0;
    while(i<a.argument_count){
        if(!budget::charge(work,1)){return .err(1002);}let arg=syntax::attribute_argument_at(p,a.argument_start+i);let name=syntax::attribute_argument_text(p,a.argument_start+i);
        if(!budget::charge(work,name.len+1)){return .err(1002);}
        let u32 bit=if(bytes::equal(name,b"storage_read")){1}else if(bytes::equal(name,b"storage_write")){2}else{0};
        if(arg.kind!=1 || bit==0 || (mask&bit)!=0){return .err(1001);}mask=mask|bit;i+=1;
    }return .ok(mask);
}

fn Result<v::Vec<context::Binding>,u32> clone(ref v::Vec<context::Binding> env,ref mut usize work){
    var v::Vec<context::Binding> out=v::empty<context::Binding>();var usize i=0;
    while(i<v::len<context::Binding>(env)){
        if(!budget::charge(work,1)){return .err(1002);}if(!v::push<context::Binding>(ref mut out,*v::at<context::Binding>(env,i)).is_ok){return .err(1003);}i+=1;
    }return .ok(out);
}

// Clone names/codec/presence but never a block-local owner-bound handle.
// Aliases deliberately get separate successor parameters, exactly as Go does.
fn Result<usize,u32> new_block(ref mut v::Vec<context::Binding> env,ref mut context::Counters ids,ref mut v::Vec<m::Value> parameters,ref mut v::Vec<m::Block> blocks,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}let id=v::len<m::Block>(blocks);if(id>=m::block_limit()){return .err(1002);}
    let start=v::len<m::Value>(parameters);var usize i=0;
    while(i<v::len<context::Binding>(env)){
        if(!budget::charge(work,1)){return .err(1002);}var row=*v::at<context::Binding>(env,i);row.borrow=0;
        if(row.value!=0){
            row.value=switch(context::value(ids,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            if(!v::push<m::Value>(parameters,m::Value{.id=row.value,.codec=row.codec}).is_ok){return .err(1003);}
        }
        if(row.found!=0){
            row.found=switch(context::value(ids,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            if(!v::push<m::Value>(parameters,m::Value{.id=row.found,.codec=1}).is_ok){return .err(1003);}
        }
        let slot=v::at_mut<context::Binding>(env,i);*slot=row;i+=1;
    }
    if(!v::push<m::Block>(blocks,m::Block{.id,.parameter_start=start,.parameter_count=v::len<m::Value>(parameters)-start,.operation_start=0,.operation_count=0,.term=m::exit(0)}).is_ok){return .err(1003);}
    return .ok(id+1);
}

fn u32 arguments(ref v::Vec<context::Binding> env,usize count,ref mut v::Vec<usize> args,ref mut usize work){
    if(count>v::len<context::Binding>(env)){return 1001;}var usize i=0;
    while(i<count){
        if(!budget::charge(work,1)){return 1002;}let row=*v::at<context::Binding>(env,i);
        if(row.value!=0 && !v::push<usize>(args,row.value).is_ok){return 1003;}
        if(row.found!=0 && !v::push<usize>(args,row.found).is_ok){return 1003;}i+=1;
    }return 0;
}

fn u32 set_term(usize current,m::Terminator term,ref mut v::Vec<m::Block> blocks,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(current==0 || current>v::len<m::Block>(blocks)){return 1001;}
    var block=*v::at<m::Block>(blocks,current-1);if(block.term.kind!=0){return 1001;}block.term=term;
    let slot=v::at_mut<m::Block>(blocks,current-1);*slot=block;return 0;
}

fn u32 jump(usize current,usize target,ref v::Vec<context::Binding> env,usize prefix,ref mut v::Vec<usize> args,ref mut v::Vec<m::Block> blocks,ref mut usize work){
    if(current==0){return 0;}if(target==0 || target>v::len<m::Block>(blocks)){return 1001;}
    var term=m::exit(1);term.target=target-1;term.argument_start=v::len<usize>(args);
    let code=arguments(env,prefix,args,work);if(code!=0){return code;}term.argument_count=v::len<usize>(args)-term.argument_start;
    return set_term(current,term,blocks,work);
}

// Every binder/access appends to a raw source-traversal stream. Track ownership
// separately: nested true arms may allocate blocks beyond the earlier false arm.
fn u32 tag(usize current,ref v::Vec<m::Operation> ops,ref mut v::Vec<usize> tags,ref mut usize work){
    if(current==0 || v::len<usize>(tags)>v::len<m::Operation>(ops)){return 1001;}
    while(v::len<usize>(tags)<v::len<m::Operation>(ops)){
        if(!budget::charge(work,1)){return 1002;}if(!v::push<usize>(tags,current-1).is_ok){return 1003;}
    }return 0;
}

fn Result<usize,u32> action(ref syntax::Parsed p,usize sid,usize current,ref state::Plan layouts,ref v::Vec<context::Function> helpers,ref v::Vec<context::Binding> env,ref mut context::Counters ids,ref mut bytes::Buffer data,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<context::Span> spans,ref mut context::Span exit,ref mut usize work){
    let s=syntax::statement_at(p,sid);
    let canonical=switch(operands::call_name(p,s.expression,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let name=bytes::view(ref canonical);let argc=syntax::call_argument_count(p,s.expression);var term=m::exit(0);
    if(bytes::equal(name,b"storage::commit") || bytes::equal(name,b"storage::reject")){
        let commit=bytes::equal(name,b"storage::commit");if(argc!=0 || commit && ids.helper){return .err(1001);}term=m::exit(if(commit){3}else{4});
    }else if(bytes::equal(name,b"storage::trap")){
        if(argc!=1){return .err(1001);}let category=switch(operands::string(p,syntax::call_argument(p,s.expression,0),7,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        let checked=values::category(bytes::view(ref category),work);if(checked!=0){return .err(checked);}let length=bytes::len(ref category);let size=bytes::len(data);
        if(size>m::byte_limit() || length>m::byte_limit()-size){return .err(1002);}term=m::exit(5);term.category_start=size;
        let code=budget::raw(data,bytes::view(ref category),work);if(code!=0){return .err(if(code==4){1003}else{1002});}term.category_end=bytes::len(data);
    }else{
        if(bytes::equal(name,b"transaction")){return .err(1001);}var usize callee=0;var usize i=0;
        while(i<v::len<context::Function>(helpers)){
            if(!budget::charge(work,1)){return .err(1002);}let h=*v::at<context::Function>(helpers,i);
            if(h.name.start>h.name.end || h.name.end>syntax::source_view(p).len){return .err(1001);}let declared=syntax::source_view(p)[h.name.start..h.name.end];
            if(!budget::charge(work,declared.len+name.len+1)){return .err(1002);}
            if(bytes::equal(name,declared)){if(callee!=0 || h.index==0 || h.index!=i+1){return .err(1001);}callee=h.index;}i+=1;
        }
        if(callee!=0){
            if(argc!=0){return .err(1001);}let op=m::Operation{.kind=11,.capability=ids.capability,.field=0,.key=m::absent(),.value=m::absent(),.result=0,.found=0,.cost=1,.callee,.borrow_input=0,.borrow_result=0,.view_offset=0,.view_length=0};
            let code=context::emit(op,context::Span{.start=s.start,.end=s.end},ids,ops,spans,work);if(code!=0){return .err(code);}
        }else{
            let op=switch(operands::access(p,s.expression,layouts,env,ids,data,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            let e=syntax::expression_at(p,s.expression);let code=context::emit(op,context::Span{.start=e.start,.end=e.end},ids,ops,spans,work);if(code!=0){return .err(code);}
        }return .ok(current);
    }
    let code=set_term(current,term,blocks,work);if(code!=0){return .err(code);}exit.start=s.start;exit.end=s.end;return .ok(0);
}

fn Result<usize,u32> branch(ref syntax::Parsed p,usize expr,usize current,usize depth,ref state::Plan layouts,ref v::Vec<context::Function> helpers,ref mut v::Vec<context::Binding> env,ref mut context::Counters ids,ref mut bytes::Buffer data,ref mut v::Vec<m::Value> parameters,ref mut v::Vec<usize> args,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<context::Span> spans,ref mut v::Vec<usize> tags,ref mut context::Span exit,ref mut usize work){
    if(!budget::charge(work,1) || depth>=m::block_limit()){return .err(1002);}
    if(current==0 || !expression(p,expr)){return .err(1001);}let e=syntax::expression_at(p,expr);
    if(e.kind!=13 || !expression(p,e.left) || !expression(p,e.right)){return .err(1001);}let arms=syntax::expression_at(p,e.right);
    if(arms.kind!=14 || !statement(p,arms.left) || syntax::statement_at(p,arms.left).kind!=1){return .err(1001);}
    var usize alternative=0;
    if(arms.right!=0){
        if(!expression(p,arms.right)){return .err(1001);}let otherwise=syntax::expression_at(p,arms.right);
        // Go requires block arms; an else-if expression is not an implicit block.
        if(otherwise.kind!=12 || !statement(p,otherwise.left) || syntax::statement_at(p,otherwise.left).kind!=1){return .err(1001);}alternative=otherwise.left;
    }
    let condition=switch(operands::reference(p,e.left,env,work)){case .ok(value):{value}case .err(code):{return .err(code);}};if(condition.codec!=1 || condition.id==0){return .err(1001);}
    let prefix=v::len<context::Binding>(env);
    var yes_env=switch(clone(env,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let yes=switch(new_block(ref mut yes_env,ids,parameters,blocks,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var no_env=switch(clone(env,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let no=switch(new_block(ref mut no_env,ids,parameters,blocks,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var term=m::exit(2);term.condition=condition.id;term.target=yes-1;term.alternative=no-1;term.argument_start=v::len<usize>(args);
    let yes_args=arguments(env,prefix,args,work);if(yes_args!=0){return .err(yes_args);}term.argument_count=v::len<usize>(args)-term.argument_start;
    term.alternate_start=v::len<usize>(args);let no_args=arguments(env,prefix,args,work);if(no_args!=0){return .err(no_args);}term.alternate_count=v::len<usize>(args)-term.alternate_start;
    let closed=set_term(current,term,blocks,work);if(closed!=0){return .err(closed);}
    let yes_end=switch(block(p,arms.left,yes,depth+1,layouts,helpers,ref mut yes_env,ids,data,parameters,args,ops,blocks,spans,tags,exit,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var usize no_end=no;
    if(alternative!=0){no_end=switch(block(p,alternative,no,depth+1,layouts,helpers,ref mut no_env,ids,data,parameters,args,ops,blocks,spans,tags,exit,work)){case .ok(value):{value}case .err(code):{return .err(code);}};}
    if(yes_end==0 && no_end==0){return .ok(0);}
    var joined=switch(clone(env,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let join=switch(new_block(ref mut joined,ids,parameters,blocks,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let yes_jump=jump(yes_end,join,ref yes_env,prefix,args,blocks,work);if(yes_jump!=0){return .err(yes_jump);}
    let no_jump=jump(no_end,join,ref no_env,prefix,args,blocks,work);if(no_jump!=0){return .err(no_jump);}
    var usize i=0;while(i<prefix){if(!budget::charge(work,1)){return .err(1002);}let row=*v::at<context::Binding>(ref joined,i);let slot=v::at_mut<context::Binding>(env,i);*slot=row;i+=1;}
    return .ok(join);
}

fn Result<usize,u32> block(ref syntax::Parsed p,usize body,usize current,usize depth,ref state::Plan layouts,ref v::Vec<context::Function> helpers,ref mut v::Vec<context::Binding> env,ref mut context::Counters ids,ref mut bytes::Buffer data,ref mut v::Vec<m::Value> parameters,ref mut v::Vec<usize> args,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<context::Span> spans,ref mut v::Vec<usize> tags,ref mut context::Span exit,ref mut usize work){
    if(!budget::charge(work,1) || depth>=m::block_limit()){return .err(1002);}if(!statement(p,body)){return .err(1001);}let b=syntax::statement_at(p,body);if(b.kind!=1){return .err(1001);}
    // Parser statement links are reversed. Walk them once, then consume IDs
    // backwards; do not impose an operation limit on zero-operation aliases.
    var v::Vec<usize> statements=v::empty<usize>();var usize link=b.first;
    while(link!=0){
        if(!budget::charge(work,1)){return .err(1002);}if(!statement(p,link) || v::len<usize>(ref statements)>=b.second){return .err(1001);}let row=syntax::statement_at(p,link);
        if(row.kind!=7 || !statement(p,row.first)){return .err(1001);}if(!v::push<usize>(ref mut statements,row.first).is_ok){return .err(1003);}link=row.second;
    }
    if(v::len<usize>(ref statements)!=b.second){return .err(1001);}var usize left=b.second;var usize cursor=current;
    while(left>0){
        if(!budget::charge(work,1)){return .err(1002);}if(cursor==0){return .err(1001);}left-=1;let sid=*v::at<usize>(ref statements,left);let s=syntax::statement_at(p,sid);
        if(s.kind==2){
            let code=bindings::bind(p,sid,layouts,env,ids,data,ops,spans,work);if(code!=0){return .err(code);}let marked=tag(cursor,ops,tags,work);if(marked!=0){return .err(marked);}
        }else if(s.kind==6 && expression(p,s.expression)){
            if(syntax::expression_at(p,s.expression).kind==13){
                cursor=switch(branch(p,s.expression,cursor,depth,layouts,helpers,env,ids,data,parameters,args,ops,blocks,spans,tags,exit,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            }else{
                let previous=cursor;cursor=switch(action(p,sid,cursor,layouts,helpers,env,ids,data,ops,blocks,spans,exit,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
                let marked=tag(previous,ops,tags,work);if(marked!=0){return .err(marked);}
            }
        }else if(s.kind==3 && ids.helper && s.expression==0){
            let code=set_term(cursor,m::exit(6),blocks,work);if(code!=0){return .err(code);}exit.start=s.start;exit.end=s.end;cursor=0;
        }else{return .err(1001);}
    }
    if(b.expression!=0){
        if(cursor==0 || !expression(p,b.expression) || syntax::expression_at(p,b.expression).kind!=13){return .err(1001);}
        return branch(p,b.expression,cursor,depth,layouts,helpers,env,ids,data,parameters,args,ops,blocks,spans,tags,exit,work);
    }return .ok(cursor);
}

// Stable counting scatter is O(operations + blocks). Repack edge argument
// slices in block order too; source DFS and block allocation order can differ.
fn Result<v::Vec<m::Operation>,u32> pack(ref v::Vec<m::Operation> raw,ref v::Vec<usize> tags,ref v::Vec<context::Span> raw_spans,ref mut v::Vec<m::Block> blocks,ref v::Vec<usize> raw_args,ref mut v::Vec<usize> args,ref mut v::Vec<context::Span> spans,ref mut usize work){
    let count=v::len<m::Operation>(raw);let block_count=v::len<m::Block>(blocks);
    if(block_count==0 || block_count>64 || count!=v::len<usize>(tags) || count!=v::len<context::Span>(raw_spans)){return .err(1001);}
    var usize[64] counts=[0;64];var usize[64] cursors=[0;64];var usize i=0;
    var v::Vec<m::Operation> out=v::empty<m::Operation>();var v::Vec<context::Span> packed=v::empty<context::Span>();
    while(i<count){
        if(!budget::charge(work,1)){return .err(1002);}let owner=*v::at<usize>(tags,i);if(owner>=block_count){return .err(1001);}counts[owner]+=1;
        if(!v::push<m::Operation>(ref mut out,*v::at<m::Operation>(raw,i)).is_ok || !v::push<context::Span>(ref mut packed,*v::at<context::Span>(raw_spans,i)).is_ok){return .err(1003);}i+=1;
    }
    var usize offset=0;i=0;
    while(i<block_count){
        if(!budget::charge(work,1)){return .err(1002);}var row=*v::at<m::Block>(blocks,i);if(row.id!=i || row.term.kind==0){return .err(1001);}cursors[i]=offset;row.operation_start=offset;row.operation_count=counts[i];offset+=counts[i];
        var usize edge=0;while(edge<2){
            let start=if(edge==0){row.term.argument_start}else{row.term.alternate_start};let length=if(edge==0){row.term.argument_count}else{row.term.alternate_count};
            if(!m::range(start,length,v::len<usize>(raw_args))){return .err(1001);}let begin=v::len<usize>(args);var usize j=0;
            while(j<length){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(args,*v::at<usize>(raw_args,start+j)).is_ok){return .err(1003);}j+=1;}
            if(edge==0){row.term.argument_start=if(row.term.kind<=2){begin}else{0};}else{row.term.alternate_start=if(row.term.kind==2){begin}else{0};}edge+=1;
        }
        let slot=v::at_mut<m::Block>(blocks,i);*slot=row;i+=1;
    }
    i=0;while(i<count){
        if(!budget::charge(work,1)){return .err(1002);}let owner=*v::at<usize>(tags,i);let at=cursors[owner];cursors[owner]+=1;
        {let slot=v::at_mut<m::Operation>(ref mut out,at);*slot=*v::at<m::Operation>(raw,i);}
        {let slot=v::at_mut<context::Span>(ref mut packed,at);*slot=*v::at<context::Span>(raw_spans,i);}i+=1;
    }
    i=0;while(i<count){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<context::Span>(spans,*v::at<context::Span>(ref packed,i)).is_ok){return .err(1003);}i+=1;}
    return .ok(out);
}

pub fn Result<m::Graph,u32> build(ref syntax::Parsed p,usize declaration,ref state::Plan layouts,ref v::Vec<context::Function> helpers,usize capability,usize remaining,ref mut v::Vec<context::Span> spans,ref mut context::Span exit,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(declaration>=syntax::declaration_count(p) || capability==0 || capability>4294967295 || state::count(layouts)!=1){return .err(1001);}if(remaining>m::operation_limit()){return .err(1002);}
    let d=syntax::declaration_at(p,declaration);if(d.kind!=3 || !statement(p,d.body)){return .err(1001);}let name=syntax::declaration_name(p,declaration+1);if(!budget::charge(work,name.len+1)){return .err(1002);}
    var context::Span attribute=context::Span{.start=0,.end=0};let mask=switch(effects(p,d,ref mut attribute,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var context::Counters ids=context::Counters{.values=0,.borrows=0,.capability,.remaining,.helper=!bytes::equal(name,b"transaction")};
    var v::Vec<context::Binding> env=v::empty<context::Binding>();var bytes::Buffer data=bytes::empty();
    var v::Vec<m::Value> parameters=v::empty<m::Value>();var v::Vec<usize> raw_args=v::empty<usize>();var v::Vec<m::Operation> raw_ops=v::empty<m::Operation>();var v::Vec<m::Block> blocks=v::empty<m::Block>();
    var v::Vec<context::Span> raw_spans=v::empty<context::Span>();var v::Vec<usize> tags=v::empty<usize>();var context::Span last_exit=context::Span{.start=0,.end=0};
    let entry=switch(new_block(ref mut env,ref mut ids,ref mut parameters,ref mut blocks,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    let end=switch(block(p,d.body,entry,0,layouts,helpers,ref mut env,ref mut ids,ref mut data,ref mut parameters,ref mut raw_args,ref mut raw_ops,ref mut blocks,ref mut raw_spans,ref mut tags,ref mut last_exit,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    if(end!=0){return .err(1001);}var v::Vec<usize> args=v::empty<usize>();
    let ops=switch(pack(ref raw_ops,ref tags,ref raw_spans,ref mut blocks,ref raw_args,ref mut args,spans,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    *exit=last_exit;return .ok(m::assemble(m::Header{.layout=0,.capability,.effects=mask,.entry=0},data,parameters,args,ops,blocks));
}
