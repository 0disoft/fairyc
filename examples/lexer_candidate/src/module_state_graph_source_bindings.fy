module module_state_graph_source_bindings;
import module_state_graph_source_context as context;
import module_state_graph_source_operands as operands;
import module_state_graph_model as m;
import module_state_layout as state;
import project_cache_encoding as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Source lowering only: ID availability here is not a borrow-lifetime proof.
// The independent program verifier checks mutations, helper effects and edges.
fn Result<context::Binding,u32> borrowed(ref syntax::Parsed p,usize expression,ref u8[] name,ref state::Plan layouts,ref v::Vec<context::Binding> env,ref mut context::Counters ids,ref mut bytes::Buffer data,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<context::Span> spans,ref mut usize work){
    if(ids.helper){return .err(1001);}
    let argc=syntax::call_argument_count(p,expression);
    var context::Binding result=context::Binding{.start=0,.end=0,.codec=0,.value=0,.found=0,.borrow=0};
    var m::Operation op=m::Operation{.kind=0,.capability=ids.capability,.field=0,.key=m::absent(),.value=m::absent(),.result=0,.found=0,.cost=1,.callee=0,.borrow_input=0,.borrow_result=0,.view_offset=0,.view_length=0};
    if(bytes::equal(name,b"storage::borrow")){
        if(argc==0 || argc>2 || state::count(layouts)!=1){return .err(1001);}
        let index=switch(operands::field(p,syntax::call_argument(p,expression,0),layouts,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        let layout=state::at(layouts,0);if(index>=layout.field_count){return .err(1001);}
        let field=state::field_at(layouts,layout.field_start+index);
        if(field.kind!=1 && field.kind!=2 || argc!=if(field.kind==2){2}else{1}){return .err(1001);}
        op.field=index;op.kind=if(field.kind==2){8}else{7};
        if(field.kind==2){
            op.key=switch(operands::operand(p,syntax::call_argument(p,expression,1),field.key,env,data,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            if(op.key.input!=0){return .err(1001);}
        }
        op.borrow_result=switch(context::borrow(ids,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        op.found=switch(context::value(ids,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        result.codec=field.value;result.borrow=op.borrow_result;result.found=op.found;
    }else{
        let view=bytes::equal(name,b"storage::view");
        if(!view && !bytes::equal(name,b"storage::copy") || argc!=if(view){3}else{1}){return .err(1001);}
        let argument=switch(operands::ungroup(p,syntax::call_argument(p,expression,0),work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        if(argument==0 || syntax::expression_at(p,argument).kind!=2){return .err(1001);}
        let index=switch(operands::lookup(p,env,syntax::expression_text(p,argument),work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        if(index==0){return .err(1001);}let original=*v::at<context::Binding>(env,index-1);
        if(original.borrow==0){return .err(1001);}result.codec=original.codec;op.borrow_input=original.borrow;
        if(view){
            op.view_offset=switch(operands::u32_literal(p,syntax::call_argument(p,expression,1),work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            op.view_length=switch(operands::u32_literal(p,syntax::call_argument(p,expression,2),work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            if(op.view_length==0){return .err(1001);}op.kind=9;
            op.borrow_result=switch(context::borrow(ids,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            result.borrow=op.borrow_result;result.found=original.found;
        }else{
            op.kind=10;op.result=switch(context::value(ids,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
            result.value=op.result;
        }
    }
    let e=syntax::expression_at(p,expression);
    let code=context::emit(op,context::Span{.start=e.start,.end=e.end},ids,ops,spans,work);
    if(code!=0){return .err(code);}return .ok(result);
}

pub fn u32 bind(ref syntax::Parsed p,usize statement,ref state::Plan layouts,ref mut v::Vec<context::Binding> env,ref mut context::Counters ids,ref mut bytes::Buffer data,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<context::Span> spans,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}
    if(statement==0 || statement>syntax::statement_count(p)){return 1001;}
    let s=syntax::statement_at(p,statement);
    if(s.kind!=2 || s.entry==0 || s.entry>syntax::entry_count(p) || s.expression==0 || s.expression>syntax::expression_count(p)){return 1001;}
    // entry_at locates stable parser IDs; account for its bounded table scan.
    if(!budget::charge(work,syntax::entry_count(p)+1)){return 1002;}let e=syntax::entry_at(p,s.entry-1);
    if(e.kind!=2 || e.name_start>=e.name_end || e.name_end>syntax::source_view(p).len){return 1001;}
    let name=syntax::source_view(p)[e.name_start..e.name_end];
    let existing=switch(operands::lookup(p,env,name,work)){case .ok(value):{value}case .err(code):{return code;}};
    if(existing!=0){return 1001;}
    var usize i=0;while(i<syntax::declaration_count(p)){
        if(!budget::charge(work,1)){return 1002;}let d=syntax::declaration_at(p,i);
        if(d.kind==3){let function=syntax::declaration_name(p,i+1);if(!budget::charge(work,function.len+name.len+1)){return 1002;}if(bytes::equal(function,name)){return 1001;}}i+=1;
    }
    var context::Binding value=context::Binding{.start=e.name_start,.end=e.name_end,.codec=0,.value=0,.found=0,.borrow=0};
    let root=switch(operands::ungroup(p,s.expression,work)){case .ok(value):{value}case .err(code):{return code;}};
    var usize alias=0;
    if(root!=0 && syntax::expression_at(p,root).kind==2){alias=switch(operands::lookup(p,env,syntax::expression_text(p,root),work)){case .ok(value):{value}case .err(code):{return code;}};}
    // Direct calls only, matching Go: grouping is accepted for aliases but does
    // not turn an arbitrary grouped expression into an admitted storage call.
    var bytes::Buffer operation=bytes::empty();
    if(syntax::expression_at(p,s.expression).kind==6){operation=switch(operands::call_name(p,s.expression,work)){case .ok(value):{value}case .err(code):{return code;}};}
    let call=bytes::view(ref operation);
    if(bytes::equal(call,b"storage::borrow") || bytes::equal(call,b"storage::view") || bytes::equal(call,b"storage::copy")){
        value=switch(borrowed(p,s.expression,call,layouts,env,ids,data,ops,spans,work)){case .ok(value):{value}case .err(code):{return code;}};
    }else if(alias!=0 && v::at<context::Binding>(env,alias-1).borrow!=0){
        value=*v::at<context::Binding>(env,alias-1);
    }else if(bytes::equal(call,b"storage::read")){
        let op=switch(operands::access(p,s.expression,layouts,env,ids,data,work)){case .ok(value):{value}case .err(code):{return code;}};
        if((op.kind!=1 && op.kind!=4) || state::count(layouts)!=1){return 1001;}let layout=state::at(layouts,0);if(op.field>=layout.field_count){return 1001;}
        let expression=syntax::expression_at(p,s.expression);let code=context::emit(op,context::Span{.start=expression.start,.end=expression.end},ids,ops,spans,work);if(code!=0){return code;}
        value.value=op.result;value.found=op.found;value.codec=state::field_at(layouts,layout.field_start+op.field).value;
    }else{
        let resolved=switch(operands::reference(p,s.expression,env,work)){case .ok(value):{value}case .err(code):{return code;}};
        value.value=resolved.id;value.codec=resolved.codec;
        if(alias!=0){value.found=v::at<context::Binding>(env,alias-1).found;}
    }
    value.start=e.name_start;value.end=e.name_end;
    if(value.codec<1 || value.codec>7){return 1001;}
    if(e.type_root!=0 && (value.borrow!=0 || operands::codec(p,e.type_root)!=value.codec)){return 1001;}
    if(!budget::charge(work,1)){return 1002;}if(!v::push<context::Binding>(env,value).is_ok){return 1003;}return 0;
}
