module module_unit_origin_paths;
import module_unit_value_model as m;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Outcomes: 1 fallthrough, 2 return/trap, 4 break, 8 continue.
// No predicate correlation or at-least-once loop proof is inferred.
pub struct Dead {usize start;usize end;usize maximum;}
pub resource Paths {m::Frame frame;usize start;usize end;usize outcomes;v::Vec<Dead> dead;cleanup{}}
fn m::Problem fail(m::Frame f,u32 code,usize start,usize end){return m::fail(f,code,start,end,0,0);}
fn m::Problem exclude(m::Frame f,usize start,usize end,ref mut v::Vec<Dead> dead,ref mut usize work){
    if(start>=end){return fail(f,0,0,0);}if(!m::charge(work,1) || v::len<Dead>(dead)>=8192){return fail(f,10,start,end);}if(!v::push<Dead>(dead,Dead{.start,.end,.maximum=0}).is_ok){return fail(f,11,start,end);}var usize i=v::len<Dead>(dead)-1;
    while(i>0 && v::at<Dead>(dead,i-1).start>start){if(!m::charge(work,1)){return fail(f,10,start,end);}let old=*v::at<Dead>(dead,i-1);let slot=v::at_mut<Dead>(dead,i);*slot=old;i-=1;}let slot=v::at_mut<Dead>(dead,i);*slot=Dead{.start,.end,.maximum=0};return fail(f,0,0,0);
}
fn Result<usize,m::Problem> list(ref syntax::Parsed source,m::Frame f,usize id,u32 kind,usize depth,ref mut v::Vec<Dead> dead,ref mut usize work){
    var v::Vec<usize> order=v::empty<usize>();var usize link=id;while(link!=0){let e=syntax::expression_at(source,link);if(!m::charge(work,1) || v::len<usize>(ref order)>=1024){return .err(fail(f,10,e.start,e.end));}if(e.kind!=kind){return .err(fail(f,12,e.start,e.end));}if(!v::push<usize>(ref mut order,e.left).is_ok){return .err(fail(f,11,e.start,e.end));}link=e.right;}
    var usize result=1;var usize i=v::len<usize>(ref order);while(i>0){i-=1;let original=*v::at<usize>(ref order,i);let e=syntax::expression_at(source,original);if((result & 1)==0){let p=exclude(f,e.start,e.end,dead,work);if(p.code!=0){return .err(p);}}
        else{let next=switch(expression(source,f,original,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};result=(result & 14) | next;}
    }return .ok(result);
}
fn Result<usize,m::Problem> expression(ref syntax::Parsed source,m::Frame f,usize id,usize depth,ref mut v::Vec<Dead> dead,ref mut usize work){
    if(id==0){return .ok(1);}if(id>syntax::expression_count(source)){return .err(fail(f,12,0,0));}let e=syntax::expression_at(source,id);if(depth>=64 || !m::charge(work,1)){return .err(fail(f,10,e.start,e.end));}
    if(e.kind==12 || e.kind==31){return block(source,f,e.left,depth+1,dead,work);}if(e.kind==1 || e.kind==2 || e.kind==16 || e.kind==17 || e.kind==28 || e.kind==32 || e.kind==33){return .ok(1);}
    let left=switch(expression(source,f,e.left,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};
    if(e.kind==13){let branches=syntax::expression_at(source,e.right);if(branches.kind!=14){return .err(fail(f,12,e.start,e.end));}if((left & 1)==0){let p=exclude(f,branches.start,branches.end,dead,work);return if(p.code==0){.ok(left)}else{.err(p)};}let yes=switch(block(source,f,branches.left,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};let no=switch(expression(source,f,branches.right,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};return .ok((left & 14) | yes | no);}
    if(e.kind==15){var usize link=e.right;var usize result=0;while(link!=0){let row=syntax::expression_at(source,link);if(row.kind!=19 || !m::charge(work,1)){return .err(fail(f,if(row.kind!=19){12}else{10},row.start,row.end));}let arm=syntax::expression_at(source,row.left);if(arm.kind!=18){return .err(fail(f,12,arm.start,arm.end));}let body=syntax::statement_at(source,arm.right);if((left & 1)==0){let p=exclude(f,body.start,body.end,dead,work);if(p.code!=0){return .err(p);}}else{let next=switch(block(source,f,arm.right,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};result=result | next;}link=row.right;}return .ok((left & 14) | if((left & 1)!=0){if(result==0){1}else{result}}else{0});}
    // The right IDs of casts and generic applications belong to the type arena.
    if(e.kind==3 || e.kind==4 || e.kind==8 || e.kind==20 || e.kind==21 || e.kind==22 || e.kind==23){return .ok(left);}if(e.kind==30){return .ok((left & 14) | if((left & 1)!=0){3}else{0});}
    if(e.right==0){return .ok(left);}let right=syntax::expression_at(source,e.right);if((left & 1)==0){let p=exclude(f,right.start,right.end,dead,work);return if(p.code==0){.ok(left)}else{.err(p)};}
    let next=switch(if(e.kind==6){list(source,f,e.right,7,depth+1,dead,work)}else if(e.kind==10){list(source,f,e.right,11,depth+1,dead,work)}else if(e.kind==24){list(source,f,e.right,25,depth+1,dead,work)}else{expression(source,f,e.right,depth+1,dead,work)}){case .ok(n):{n}case .err(p):{return .err(p);}};
    let op=syntax::expression_text(source,id);let bool optional=e.kind==5 && (bytes::equal(op,b"&&") || bytes::equal(op,b"||"));return .ok((left & 14) | next | if(optional){1}else{0});
}
fn Result<usize,m::Problem> statement(ref syntax::Parsed source,m::Frame f,usize id,usize depth,ref mut v::Vec<Dead> dead,ref mut usize work){
    if(id==0){return .ok(1);}let s=syntax::statement_at(source,id);if(depth>=64 || !m::charge(work,1)){return .err(fail(f,10,s.start,s.end));}if(s.kind==9){return .ok(4);}if(s.kind==10){return .ok(8);}if(s.kind==17){return block(source,f,s.first,depth+1,dead,work);}
    if(s.kind==14){let h=syntax::statement_at(source,s.second);let initial=switch(statement(source,f,h.first,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};if((initial & 1)==0){let after=syntax::statement_at(source,h.first);let p=exclude(f,after.end,s.end,dead,work);return if(p.code==0){.ok(initial)}else{.err(p)};}
        let condition=switch(expression(source,f,s.expression,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};let body=syntax::statement_at(source,s.first);var usize exits=0;if((condition & 1)!=0){exits=switch(block(source,f,s.first,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};}else{let p=exclude(f,body.start,body.end,dead,work);if(p.code!=0){return .err(p);}}
        var usize post=0;if(h.second!=0){let at=syntax::statement_at(source,h.second);if((condition & 1)!=0 && (exits & 9)!=0){post=switch(statement(source,f,h.second,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};}else{let p=exclude(f,at.start,at.end,dead,work);if(p.code!=0){return .err(p);}}}return .ok((initial & 14) | (condition & 2) | (exits & 2) | (post & 2) | if((condition & 13)!=0){1}else{0});
    }
    let target=switch(expression(source,f,if(s.kind==4){s.target}else{0},depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};if((target & 1)==0){let e=syntax::expression_at(source,s.expression);let p=exclude(f,e.start,e.end,dead,work);return if(p.code==0){.ok(target)}else{.err(p)};}
    let result=switch(expression(source,f,s.expression,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};if(s.kind==3){return .ok((target & 14) | (result & 14) | if((result & 1)!=0){2}else{0});}
    if(s.kind==11 && s.expression!=0 && bytes::equal(syntax::expression_text(source,s.expression),b"false")){return .ok((target & 14) | (result & 14) | if((result & 1)!=0){2}else{0});}
    if(s.kind==8 || s.kind==18){let b=syntax::statement_at(source,s.first);if((result & 1)==0){let p=exclude(f,b.start,b.end,dead,work);return if(p.code==0){.ok((result & 2) | if((result & 12)!=0){1}else{0})}else{.err(p)};}let body=switch(block(source,f,s.first,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};return .ok((result & 2) | (body & 2) | 1);}
    return .ok((target & 14) | result);
}
fn Result<usize,m::Problem> block(ref syntax::Parsed source,m::Frame f,usize id,usize depth,ref mut v::Vec<Dead> dead,ref mut usize work){
    if(id==0 || id>syntax::statement_count(source)){return .err(fail(f,12,0,0));}let b=syntax::statement_at(source,id);if(b.kind!=1 || depth>=64 || !m::charge(work,1)){return .err(fail(f,if(b.kind!=1){12}else{10},b.start,b.end));}var v::Vec<usize> order=v::empty<usize>();var usize link=b.first;
    while(link!=0){let row=syntax::statement_at(source,link);if(row.kind!=7 || !m::charge(work,1) || v::len<usize>(ref order)>=8192){return .err(fail(f,if(row.kind!=7){12}else{10},row.start,row.end));}if(!v::push<usize>(ref mut order,row.first).is_ok){return .err(fail(f,11,row.start,row.end));}link=row.second;}if(v::len<usize>(ref order)!=b.second){return .err(fail(f,12,b.start,b.end));}
    var usize result=1;var usize i=v::len<usize>(ref order);while(i>0 && (result & 1)!=0){i-=1;let id=*v::at<usize>(ref order,i);let s=syntax::statement_at(source,id);let next=switch(statement(source,f,id,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};result=(result & 14) | next;if((result & 1)==0 && (i>0 || b.expression!=0)){let p=exclude(f,s.end,b.end,dead,work);if(p.code!=0){return .err(p);}}}
    if((result & 1)!=0){let next=switch(expression(source,f,b.expression,depth+1,dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};result=(result & 14) | next;}return .ok(result);
}
pub fn Result<Paths,m::Problem> build(ref syntax::Parsed source,m::Frame f,ref mut usize work){
    if(f.declaration==0 || f.declaration>syntax::declaration_count(source)){return .err(fail(f,12,0,0));}let d=syntax::declaration_at(source,f.declaration-1);let usize body=if(f.kind==2){d.cleanup_body}else{d.body};if(body==0 || body>syntax::statement_count(source)){return .err(fail(f,12,d.start,d.end));}let b=syntax::statement_at(source,body);var v::Vec<Dead> dead=v::empty<Dead>();let outcomes=switch(block(source,f,body,0,ref mut dead,work)){case .ok(n):{n}case .err(p):{return .err(p);}};var usize i=0;var usize maximum=0;
    while(i<v::len<Dead>(ref dead)){if(!m::charge(work,1)){return .err(fail(f,10,b.start,b.end));}let end=v::at<Dead>(ref dead,i).end;if(end>maximum){maximum=end;}let slot=v::at_mut<Dead>(ref mut dead,i);slot.maximum=maximum;i+=1;}return .ok(Paths{.frame=f,.start=b.start,.end=b.end,.outcomes,.dead});
}
pub fn usize outcomes(ref Paths p){return p.outcomes;}
pub fn usize dead_count(ref Paths p){return v::len<Dead>(ref p.dead);}
pub fn Result<bool,m::Problem> live(ref Paths p,usize start,usize end,ref mut usize work){
    if(start<p.start || end>p.end || start>end){return .err(fail(p.frame,12,start,end));}if(!m::charge(work,1)){return .err(fail(p.frame,10,start,end));}var usize low=0;var usize high=v::len<Dead>(ref p.dead);while(low<high){if(!m::charge(work,1)){return .err(fail(p.frame,10,start,end));}let usize mid=low+(high-low)/2;if(v::at<Dead>(ref p.dead,mid).start<=start){low=mid+1;}else{high=mid;}}return .ok(low==0 || v::at<Dead>(ref p.dead,low-1).maximum<end);
}
