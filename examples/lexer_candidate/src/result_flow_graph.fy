module result_flow_graph;
import syntax as syntax;
import name_resolution as names;
import type_state as state;
import tagged_types as tagged;
import std::vec as v;
import std::bytes as bytes;

// Obligations have columns only for exact Result locals/parameters. Nodes:
// 0 control, 1 take, 2 produce, 3 overwrite check, 4 scope check/kill,
// 5 normal exit, 6 unhandled evaluation. Zero successors are terminal.
pub struct Node { u32 kind; usize column; usize start; usize end; usize next; usize alternative; }
pub resource Graph { v::Vec<Node> nodes; v::Vec<usize> columns; usize width; cleanup {} }
struct Context {
    usize current; usize[64] scopes; usize[64] captures; usize scope_depth;
    usize[64] breaks; usize[64] continues; usize[64] loop_base; usize loops;
    usize work; state::Problem problem;
}
fn void fail(ref mut Context c,u32 code,usize start,usize end){if(c.problem.code==0){c.problem=state::fail(code,start,end,0,0);}}
fn usize fresh(ref mut v::Vec<Node> nodes,ref mut Context c,u32 kind,usize column,usize start,usize end){
    if(c.problem.code!=0){return 0;}if(v::len<Node>(nodes)>=8192){fail(c,10,start,end);return 0;}
    if(!v::push<Node>(nodes,Node{.kind,.column,.start,.end,.next=0,.alternative=0}).is_ok){fail(c,12,start,end);return 0;}
    return v::len<Node>(nodes);
}
fn void jump(ref mut v::Vec<Node> nodes,ref mut Context c,usize to){
    if(c.current==0 || c.problem.code!=0){return;}
    let ref mut Node previous=v::at_mut<Node>(nodes,c.current-1);previous.next=to;c.current=0;
}
fn usize emit(ref mut v::Vec<Node> nodes,ref mut Context c,u32 kind,usize column,usize start,usize end){
    if(c.current==0){return 0;}let usize id=fresh(nodes,c,kind,column,start,end);jump(nodes,c,id);c.current=id;return id;
}
fn void fork(ref mut v::Vec<Node> nodes,usize from,usize left,usize right){
    if(from==0){return;}let ref mut Node node=v::at_mut<Node>(nodes,from-1);node.next=left;node.alternative=right;
}
fn usize column(ref v::Vec<usize> columns,usize symbol){return if(symbol==0){0}else{*v::at<usize>(columns,symbol)};}
fn u32 take(ref syntax::Parsed p,ref v::Vec<state::Slot> types,ref v::Vec<usize> columns,ref mut v::Vec<Node> nodes,ref mut Context c,usize id,u32 use){
    if(!tagged::result(state::expressions(types,id-1))){return use;}
    let e=syntax::expression_at(p,id);
    if(e.kind==2){discard emit(nodes,c,1,column(columns,state::uses(types,id-1)),e.start,e.end);}
    if(use==0 || use==3){discard emit(nodes,c,6,0,e.start,e.end);}
    return 2;
}
fn void leave(ref syntax::Parsed p,ref v::Vec<state::Slot> types,ref v::Vec<usize> columns,ref mut v::Vec<Node> nodes,ref mut Context c,usize block){
    var usize i=0;while(i<syntax::block_statement_count(p,block)){
        let s=syntax::statement_at(p,syntax::block_statement(p,block,i));
        if(s.kind==2){let usize col=column(columns,state::entries(types,s.entry-1));if(col!=0){discard emit(nodes,c,4,col,s.start,s.end);}}i+=1;
    }
    i=0;while(i<c.scope_depth){let usize scope=c.scopes[i];let usize col=c.captures[i];
        if(scope==block && col!=0){discard emit(nodes,c,4,col,0,0);}i+=1;
    }
}
fn void block(ref syntax::Parsed p,ref v::Vec<state::Slot> types,ref v::Vec<usize> columns,ref mut v::Vec<Node> nodes,ref mut Context c,usize id,u32 use,usize capture,usize depth){
    if(c.current==0 || c.problem.code!=0){return;}if(depth>=64 || c.scope_depth>=64){fail(c,10,0,0);return;}
    let usize scope_slot=c.scope_depth;c.scopes[scope_slot]=id;c.captures[scope_slot]=capture;c.scope_depth+=1;
    var usize i=0;while(i<syntax::block_statement_count(p,id) && c.current!=0 && c.problem.code==0){
        statement(p,types,columns,nodes,c,syntax::block_statement(p,id,i),depth+1);i+=1;
    }
    let tail=syntax::statement_at(p,id).expression;
    if(tail!=0){expression(p,types,columns,nodes,c,tail,use,depth+1);}
    if(c.current!=0){leave(p,types,columns,nodes,c,id);}c.scope_depth-=1;
}
fn void expression(ref syntax::Parsed p,ref v::Vec<state::Slot> types,ref v::Vec<usize> columns,ref mut v::Vec<Node> nodes,ref mut Context c,usize id,u32 use,usize depth){
    if(id==0 || c.current==0 || c.problem.code!=0){return;}
    let e=syntax::expression_at(p,id);c.work+=1;if(depth>=64 || c.work>1048576){fail(c,10,e.start,e.end);return;}
    if(e.kind==2){discard take(p,types,columns,nodes,c,id,use);return;}
    if(e.kind==12 || e.kind==3){let u32 child_use=if(tagged::result(state::expressions(types,id-1))){2}else{use};
        if(e.kind==12){block(p,types,columns,nodes,c,e.left,child_use,0,depth+1);}else{expression(p,types,columns,nodes,c,e.left,child_use,depth+1);}
        if(c.current!=0){discard take(p,types,columns,nodes,c,id,use);}return;
    }
    if(e.kind==30){
        expression(p,types,columns,nodes,c,e.left,2,depth+1);
        let usize split=emit(nodes,c,0,0,e.start,e.end);
        let usize success=fresh(nodes,c,0,0,e.start,e.end);let usize failure=fresh(nodes,c,5,0,e.start,e.end);
        fork(nodes,split,success,failure);c.current=success;return;
    }
    if(e.kind==8 && state::flags(types,id-1)==2048){expression(p,types,columns,nodes,c,e.left,4,depth+1);return;}
    if(e.kind==13){
        expression(p,types,columns,nodes,c,e.left,0,depth+1);let u32 child_use=if(tagged::result(state::expressions(types,id-1))){2}else{use};
        let usize split=emit(nodes,c,0,0,e.start,e.end);let usize yes=fresh(nodes,c,0,0,e.start,e.end);
        let usize no=fresh(nodes,c,0,0,e.start,e.end);let usize join=fresh(nodes,c,0,0,e.start,e.end);fork(nodes,split,yes,no);
        let branches=syntax::expression_at(p,e.right);
        c.current=yes;block(p,types,columns,nodes,c,branches.left,child_use,0,depth+1);jump(nodes,c,join);
        c.current=no;expression(p,types,columns,nodes,c,branches.right,child_use,depth+1);jump(nodes,c,join);c.current=join;discard take(p,types,columns,nodes,c,id,use);return;
    }
    if(e.kind==15){
        expression(p,types,columns,nodes,c,e.left,4,depth+1);let u32 child_use=if(tagged::result(state::expressions(types,id-1))){2}else{use};
        let usize join=fresh(nodes,c,0,0,e.start,e.end);var usize split=emit(nodes,c,0,0,e.start,e.end);var usize link=e.right;
        while(link!=0 && c.problem.code==0){
            let pair=syntax::expression_at(p,link);let arm=syntax::expression_at(p,pair.left);
            let usize body=fresh(nodes,c,0,0,arm.start,arm.end);let usize next=if(pair.right==0){0}else{fresh(nodes,c,0,0,arm.start,arm.end)};
            fork(nodes,split,body,next);c.current=body;
            let pattern=syntax::expression_at(p,arm.left);let usize name=if(pattern.kind==16){pattern.left}else{0};
            let usize capture=if(name==0){0}else{column(columns,state::uses(types,name-1))};
            if(capture!=0){let n=syntax::expression_at(p,name);discard emit(nodes,c,2,capture,n.start,n.end);}
            block(p,types,columns,nodes,c,arm.right,child_use,capture,depth+1);jump(nodes,c,join);
            split=next;link=pair.right;
        }c.current=join;discard take(p,types,columns,nodes,c,id,use);return;
    }
    if(e.kind==5 && (bytes::equal(syntax::expression_text(p,id),b"&&") || bytes::equal(syntax::expression_text(p,id),b"||"))){
        expression(p,types,columns,nodes,c,e.left,0,depth+1);
        let usize split=emit(nodes,c,0,0,e.start,e.end);let usize rhs=fresh(nodes,c,0,0,e.start,e.end);let usize join=fresh(nodes,c,0,0,e.start,e.end);
        fork(nodes,split,rhs,join);c.current=rhs;expression(p,types,columns,nodes,c,e.right,0,depth+1);jump(nodes,c,join);c.current=join;return;
    }
    if(e.kind==6){var usize i=0;while(i<syntax::call_argument_count(p,id)){expression(p,types,columns,nodes,c,syntax::call_argument(p,id,i),2,depth+1);i+=1;}}
    else if(e.kind==10){var usize i=0;while(i<syntax::record_field_count(p,id)){let field=syntax::expression_at(p,syntax::record_field(p,id,i));expression(p,types,columns,nodes,c,field.left,2,depth+1);i+=1;}}
    else if(e.kind==23 || e.kind==24){var usize i=0;while(i<syntax::array_element_count(p,id)){expression(p,types,columns,nodes,c,syntax::array_element(p,id,i),2,depth+1);i+=1;}}
    else if(e.kind==26){expression(p,types,columns,nodes,c,e.left,0,depth+1);let bounds=syntax::expression_at(p,e.right);expression(p,types,columns,nodes,c,bounds.left,0,depth+1);expression(p,types,columns,nodes,c,bounds.right,0,depth+1);}
    else if(e.kind!=1 && e.kind!=28){expression(p,types,columns,nodes,c,e.left,0,depth+1);expression(p,types,columns,nodes,c,e.right,0,depth+1);}
    if(c.current!=0){discard take(p,types,columns,nodes,c,id,use);}
}
fn void statement(ref syntax::Parsed p,ref v::Vec<state::Slot> types,ref v::Vec<usize> columns,ref mut v::Vec<Node> nodes,ref mut Context c,usize id,usize depth){
    if(id==0 || c.current==0 || c.problem.code!=0){return;}let s=syntax::statement_at(p,id);
    if(depth>=64){fail(c,10,s.start,s.end);return;}
    if(s.kind==2 || s.kind==4){
        if(s.kind==4){let target=syntax::expression_at(p,s.target);
            // Evaluating a place visits receivers/index effects without reading
            // its final Result value as an unhandled temporary.
            if(target.kind==8 || target.kind==9){expression(p,types,columns,nodes,c,target.left,0,depth+1);
                if(target.kind==9){expression(p,types,columns,nodes,c,target.right,0,depth+1);}}
            else if(target.kind!=2){expression(p,types,columns,nodes,c,s.target,0,depth+1);}
        }
        expression(p,types,columns,nodes,c,s.expression,1,depth+1);
        let usize col=column(columns,if(s.kind==2){state::entries(types,s.entry-1)}else{state::uses(types,s.target-1)});
        if(col!=0 && c.current!=0){let e=syntax::expression_at(p,s.expression);if(s.kind==4){discard emit(nodes,c,3,col,s.start,s.end);}discard emit(nodes,c,2,col,e.start,e.end);}return;
    }
    if(s.kind==3){expression(p,types,columns,nodes,c,s.expression,4,depth+1);discard emit(nodes,c,5,0,s.start,s.end);c.current=0;return;}
    if(s.kind==9 || s.kind==10){
        if(c.loops==0){fail(c,13,s.start,s.end);return;}let usize frame=c.loops-1;var usize scope=c.loop_base[frame];
        while(scope<c.scope_depth){let usize scope_id=c.scopes[scope];leave(p,types,columns,nodes,c,scope_id);scope+=1;}
        let usize destination=if(s.kind==9){c.breaks[frame]}else{c.continues[frame]};jump(nodes,c,destination);return;
    }
    if(s.kind==8 || s.kind==14){
        let header=syntax::statement_at(p,if(s.kind==14){s.second}else{id});
        if(s.kind==14){statement(p,types,columns,nodes,c,header.first,depth+1);}
        let usize test=fresh(nodes,c,0,0,s.start,s.end);jump(nodes,c,test);c.current=test;
        expression(p,types,columns,nodes,c,s.expression,0,depth+1);
        let usize split=emit(nodes,c,0,0,s.start,s.end);let usize body=fresh(nodes,c,0,0,s.start,s.end);
        let usize after=fresh(nodes,c,0,0,s.start,s.end);let usize post=if(s.kind==14){fresh(nodes,c,0,0,s.start,s.end)}else{test};fork(nodes,split,body,after);
        if(c.loops>=64){fail(c,10,s.start,s.end);return;}let usize frame=c.loops;let usize base=c.scope_depth;c.breaks[frame]=after;c.continues[frame]=post;c.loop_base[frame]=base;c.loops+=1;
        c.current=body;block(p,types,columns,nodes,c,s.first,0,0,depth+1);jump(nodes,c,post);
        if(s.kind==14){c.current=post;statement(p,types,columns,nodes,c,header.second,depth+1);jump(nodes,c,test);}
        c.loops-=1;c.current=after;
        if(s.kind==14 && header.first!=0){let initial=syntax::statement_at(p,header.first);if(initial.kind==2){let usize col=column(columns,state::entries(types,initial.entry-1));if(col!=0){discard emit(nodes,c,4,col,initial.start,initial.end);}}}return;
    }
    expression(p,types,columns,nodes,c,s.expression,if(s.kind==12){4}else if(s.kind==6){3}else{0},depth+1);
}
pub fn Result<Graph,state::Problem> build(ref names::Resolved resolved,ref v::Vec<state::Slot> types,usize function){
    let ref syntax::Parsed p=names::parsed(resolved);
    var v::Vec<usize> columns=v::empty<usize>();var usize width=0;var usize i=0;
    while(i<=names::symbol_count(resolved)){
        var usize col=0;if(i!=0){let s=names::symbol_at(resolved,i);if(s.owner==function && tagged::result(state::symbols(types,i-1))){width+=1;col=width;}}
        if(!v::push<usize>(ref mut columns,col).is_ok){return .err(state::fail(12,0,0,0,0));}i+=1;
    }
    var v::Vec<Node> nodes=v::empty<Node>();
    var Context c=Context{.current=0,.scopes=[0;64],.captures=[0;64],.scope_depth=0,.breaks=[0;64],.continues=[0;64],.loop_base=[0;64],.loops=0,.work=0,.problem=state::ok()};
    c.current=fresh(ref mut nodes,ref mut c,0,0,0,0);i=1;
    while(i<=names::symbol_count(resolved)){let s=names::symbol_at(resolved,i);let usize col=column(ref columns,i);if(col!=0 && s.kind==2){discard emit(ref mut nodes,ref mut c,2,col,s.start,s.end);}i+=1;}
    let d=syntax::declaration_at(p,names::symbol_at(resolved,function).declaration-1);
    block(p,types,ref columns,ref mut nodes,ref mut c,d.body,4,0,0);discard emit(ref mut nodes,ref mut c,5,0,d.start,d.end);
    if(c.problem.code!=0){return .err(c.problem);}
    return .ok(Graph{.nodes,.columns,.width});
}
pub fn usize count(ref Graph p){return v::len<Node>(ref p.nodes);}
pub fn usize width(ref Graph p){return p.width;}
pub fn Node at(ref Graph p,usize id){return *v::at<Node>(ref p.nodes,id-1);}
