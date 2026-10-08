module hir_byte_loop;
import hir_model as h;
import hir_state as state;
import type_check as typing;
import name_resolution as names;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Normalize the admitted read-only byte loop into owned block/For/Index facts.
// Hidden descriptors have numeric binding IDs; source identifiers are never
// synthesized and SDK/source AST ownership remains untouched.
pub struct Build {usize loop;usize capture;}
fn h::Binding hidden(h::Binding original,h::BindingKind kind,h::Type type){return h::Binding{.link_start=0,.link_end=0,.unsafe_call=false,.must_use=false,.effects=0,.kind,.type,.start=original.start,.end=original.end,.owner=original.owner,.scope=original.scope-1,.module_index=original.module_index,.declaration=0,.signature_start=0,.signature_count=0};}
pub fn bool prepare(ref typing::Typed input,ref mut v::Vec<h::Binding> bindings,ref mut v::Vec<state::Link> links,ref mut state::Context c){
    let parsed=names::parsed(typing::resolved(input));var usize i=1;
    while(i<=syntax::statement_count(parsed)){let source=syntax::statement_at(parsed,i);if(source.kind==18){
        let usize item=state::use(links,source.target);if(item==0){i+=1;continue;}if(v::len<h::Binding>(bindings)>1150){discard state::fail(c,2,source.start,source.end);return false;}
        let original=*v::at<h::Binding>(bindings,item-1);if(original.scope<2 || h::type_code(original.type)!=3){discard state::fail(c,4,source.start,source.end);return false;}
        let h::Binding view_binding=hidden(original,.Let,.Bytes);
        if(!v::push<h::Binding>(bindings,view_binding).is_ok){discard state::fail(c,3,source.start,source.end);return false;}let usize view=v::len<h::Binding>(bindings);
        if(!v::push<h::Binding>(bindings,hidden(original,.Var,.Usize)).is_ok){discard state::fail(c,3,source.start,source.end);return false;}let usize index=v::len<h::Binding>(bindings);
        let ref mut state::Link link=v::at_mut<state::Link>(links,source.target-1);link.iteration_view=view;link.iteration_index=index;
    }i+=1;}return true;
}
fn usize node(ref mut v::Vec<h::Expression> expressions,ref mut state::Context c,syntax::Statement source,h::ExpressionKind kind,h::Type type,h::Operator operator,usize left,usize right,usize binding){
    return state::push_expression(expressions,h::Expression{.kind,.type,.operator,.start=source.start,.end=source.end,.left,.right,.binding,.alternative=0,.block=0,.edge_start=0,.edge_count=0,.payload_start=0,.payload_end=0,.boolean=false},c);
}
fn usize integer(ref mut v::Vec<h::Expression> expressions,ref mut bytes::Buffer payload,ref mut state::Context c,syntax::Statement source,u8 value){
    let usize at=bytes::len(payload);if(!bytes::append_byte(payload,value).is_ok){return state::fail(c,3,source.start,source.end);}
    return state::push_expression(expressions,h::Expression{.kind=.Integer,.type=.Usize,.operator=.None,.start=source.start,.end=source.end,.left=0,.right=0,.binding=0,.alternative=0,.block=0,.edge_start=0,.edge_count=0,.payload_start=at,.payload_end=at+1,.boolean=false},c);
}
fn usize binding(ref mut v::Vec<h::Statement> statements,ref mut state::Context c,syntax::Statement source,usize scope,usize id,usize value){
    return state::push_statement(statements,h::Statement{.kind=.Binding,.start=source.start,.end=source.end,.scope,.binding=id,.expression=value,.target=0,.operator=.None,.body=0,.initial=0,.post=0,.loop_target=0},c);
}
pub fn Build start(syntax::Statement source,usize repeat_start,usize value,ref v::Vec<state::Link> links,ref mut state::Context c,ref mut v::Vec<h::Expression> expressions,ref mut v::Vec<h::Statement> statements,ref mut bytes::Buffer payload){
    let link=*v::at<state::Link>(links,source.target-1);let usize scope=state::scope(c);
    let usize capture=binding(statements,c,source,scope,link.iteration_view,value);
    // Bounds run during capture only. Repeated nodes start at the body so
    // source-span lifetime checks cannot retain a bound-call loan on backedges.
    var syntax::Statement repeated=source;repeated.start=repeat_start;
    let usize zero=integer(expressions,payload,c,repeated,48);let usize initial=binding(statements,c,repeated,scope,link.iteration_index,zero);
    let usize index=node(expressions,c,repeated,.Reference,.Usize,.None,0,0,link.iteration_index);
    let usize view=node(expressions,c,repeated,.Reference,.Bytes,.None,0,0,link.iteration_view);
    let usize length=node(expressions,c,repeated,.Length,.Usize,.None,view,0,0);
    let usize condition=node(expressions,c,repeated,.Binary,.Bool,.Less,index,length,0);
    let usize one=integer(expressions,payload,c,repeated,49);
    let h::Statement post=h::Statement{.kind=.Assignment,.start=repeated.start,.end=repeated.end,.scope,.binding=0,.expression=one,.target=index,.operator=.Plus,.body=0,.initial=0,.post=0,.loop_target=0};
    let usize step=state::push_statement(statements,post,c);
    let h::Statement loop=h::Statement{.kind=.For,.start=repeated.start,.end=repeated.end,.scope,.binding=0,.expression=condition,.target=0,.operator=.None,.body=0,.initial,.post=step,.loop_target=0};
    return Build{.loop=state::push_statement(statements,loop,c),.capture};
}
pub fn usize finish(syntax::Statement source,Build built,usize body,usize parent_scope,ref v::Vec<state::Link> links,ref mut state::Context c,ref mut v::Vec<h::Expression> expressions,ref mut v::Vec<h::Statement> statements,ref mut v::Vec<h::Block> blocks,ref mut v::Vec<usize> edges){
    if(body==0 || built.loop==0 || built.capture==0 || c.problem.code!=0){return 0;}
    let link=*v::at<state::Link>(links,source.target-1);let old=*v::at<h::Block>(blocks,body-1);
    var syntax::Statement repeated=source;repeated.start=old.start;
    let usize view=node(expressions,c,repeated,.Reference,.Bytes,.None,0,0,link.iteration_view);
    let usize index=node(expressions,c,repeated,.Reference,.Usize,.None,0,0,link.iteration_index);
    let usize value=node(expressions,c,repeated,.Index,.U8,.None,view,index,0);
    let usize item=binding(statements,c,repeated,old.scope,link.use,value);
    let usize first=state::reserve_edges(edges,old.edge_count+1,c);if(c.problem.code!=0){return 0;}state::set_edge(edges,first,item);
    var usize i=0;while(i<old.edge_count){let usize child=*v::at<usize>(edges,old.edge_start+i);state::set_edge(edges,first+i+1,child);i+=1;}
    {let ref mut h::Block stored=v::at_mut<h::Block>(blocks,body-1);stored.edge_start=first;stored.edge_count=old.edge_count+1;}
    {let ref mut h::Statement stored=v::at_mut<h::Statement>(statements,built.loop-1);stored.body=body;}
    let usize outer=state::reserve_edges(edges,2,c);if(c.problem.code!=0){return 0;}state::set_edge(edges,outer,built.capture);state::set_edge(edges,outer+1,built.loop);
    if(v::len<h::Block>(blocks)>=8192){return state::fail(c,2,source.start,source.end);}
    if(!v::push<h::Block>(blocks,h::Block{.unsafe_body=false,.start=source.start,.end=source.end,.owner=c.owner,.scope=state::scope(c),.parent_scope,.edge_start=outer,.edge_count=2,.tail=0}).is_ok){return state::fail(c,3,source.start,source.end);}
    let usize wrapper=node(expressions,c,source,.Block,.Void,.None,0,0,0);if(wrapper!=0){let ref mut h::Expression stored=v::at_mut<h::Expression>(expressions,wrapper-1);stored.block=v::len<h::Block>(blocks);}return wrapper;
}
