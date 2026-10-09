module module_unit_value_completion;
import module_unit_value_model as m;
import syntax as syntax;
import std::bytes as bytes;

// Match the reference type checker's syntactic completion gate. This is not a
// CFG proof: loops, plain block statements and switches do not prove a return.
fn Result<bool,m::Problem> expression(ref syntax::Parsed source,m::Frame f,usize id,usize depth,ref mut usize work){
    if(id==0){return .ok(false);}let e=syntax::expression_at(source,id);
    if(depth>=64 || !m::charge(work,1)){return .err(m::fail(f,10,e.start,e.end,0,0));}
    if(e.kind==31){return block(source,f,e.left,depth+1,work);}
    if(e.kind!=13){return .ok(false);}let branches=syntax::expression_at(source,e.right);
    if(branches.kind!=14){return .err(m::fail(f,12,e.start,e.end,0,0));}
    if(branches.right==0){return .ok(false);}let left=switch(block(source,f,branches.left,depth+1,work)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!left){return .ok(false);}
    let other=syntax::expression_at(source,branches.right);
    return if(other.kind==13){expression(source,f,branches.right,depth+1,work)}else if(other.kind==12){block(source,f,other.left,depth+1,work)}else{.err(m::fail(f,12,other.start,other.end,0,0))};
}
fn Result<bool,m::Problem> block(ref syntax::Parsed source,m::Frame f,usize id,usize depth,ref mut usize work){
    if(id==0 || id>syntax::statement_count(source)){return .err(m::fail(f,12,0,0,0,0));}let b=syntax::statement_at(source,id);
    if(depth>=64 || !m::charge(work,1)){return .err(m::fail(f,10,b.start,b.end,0,0));}if(b.kind!=1){return .err(m::fail(f,12,b.start,b.end,0,0));}
    // The reference gate asks whether any direct statement completes the body.
    // Walking the parser's reverse links directly avoids repeated indexed scans.
    var usize link=b.first;var usize count=0;var bool terminated=b.expression!=0;
    while(link!=0){if(!m::charge(work,1)){return .err(m::fail(f,10,b.start,b.end,0,0));}let row=syntax::statement_at(source,link);if(row.kind!=7 || count>=b.second){return .err(m::fail(f,12,row.start,row.end,0,0));}let s=syntax::statement_at(source,row.first);
        if(s.kind==3){terminated=true;}else if(s.kind==11 && s.expression!=0){let e=syntax::expression_at(source,s.expression);if(e.kind==1 && bytes::equal(syntax::expression_text(source,s.expression),b"false")){terminated=true;}}
        if(!terminated){let result=switch(if(s.kind==17){block(source,f,s.first,depth+1,work)}else{expression(source,f,s.expression,depth+1,work)}){case .ok(b):{b}case .err(p):{return .err(p);}};terminated=result;}
        link=row.second;count+=1;
    }if(count!=b.second){return .err(m::fail(f,12,b.start,b.end,0,0));}return .ok(terminated);
}
pub fn Result<bool,m::Problem> check(ref syntax::Parsed source,m::Frame f,bool void_result,ref mut usize work){
    if(f.declaration==0 || f.declaration>syntax::declaration_count(source)){return .err(m::fail(f,12,0,0,0,0));}let d=syntax::declaration_at(source,f.declaration-1);let usize body=if(f.kind==2){d.cleanup_body}else{d.body};if(body==0 || body>syntax::statement_count(source)){return .err(m::fail(f,12,d.start,d.end,0,0));}let b=syntax::statement_at(source,body);
    if(!m::charge(work,1)){return .err(m::fail(f,10,b.start,b.end,0,0));}
    // Function/cleanup bodies are statement blocks in the normative grammar.
    if(body==0 || b.kind!=1 || b.expression!=0){return .err(m::fail(f,12,b.start,b.end,0,0));}if(void_result){return .ok(true);}
    let complete=switch(block(source,f,body,0,work)){case .ok(b):{b}case .err(p):{return .err(p);}};
    return if(complete){.ok(true)}else{.err(m::fail(f,2,b.start,b.end,f.result,0))};
}
