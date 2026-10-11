module module_unit_moves;
import module_unit_move_model as g;
import module_unit_move_control as control;
import module_unit_value_model as m;
import module_unit_values as values;
import syntax as syntax;
import std::vec as v;

pub resource Plan {v::Vec<g::Diagnostic> diagnostics;v::Vec<g::Fact> facts;v::Vec<g::Fact> cleanups;usize work;cleanup{}}
pub fn usize count(ref Plan p){return v::len<g::Diagnostic>(ref p.diagnostics);}
pub fn g::Diagnostic at(ref Plan p,usize index){return *v::at<g::Diagnostic>(ref p.diagnostics,index);}
pub fn usize fact_count(ref Plan p){return v::len<g::Fact>(ref p.facts);}
pub fn g::Fact fact_at(ref Plan p,usize index){return *v::at<g::Fact>(ref p.facts,index);}
pub fn usize cleanup_count(ref Plan p){return v::len<g::Fact>(ref p.cleanups);}
pub fn g::Fact cleanup_at(ref Plan p,usize index){return *v::at<g::Fact>(ref p.cleanups,index);}
pub fn usize work(ref Plan p){return p.work;}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources,ref values::Plan primary,usize prior_work){
    var v::Vec<g::Diagnostic> diagnostics=v::empty<g::Diagnostic>();var v::Vec<g::Fact> facts=v::empty<g::Fact>();var usize work=prior_work;var usize frame=0;
    while(frame<values::count(primary)){let body=values::at(primary,frame);let value=m::frame(body);let source=v::at<syntax::Parsed>(sources,value.module_index);let d=syntax::declaration_at(source,value.declaration-1);var g::Context c=g::context(value,frame+1,work,if(value.kind==2){syntax::statement_at(source,d.cleanup_body).end}else{0});var v::Vec<u8> owners=v::empty<u8>();var v::Vec<g::Cell> env=v::empty<g::Cell>();var bool needed=false;var usize local=1;
        while(local<=m::local_count(body)){let n=m::local_at(body,local);let owned=switch(g::owned(primary,n.type,ref mut c,0)){case .ok(b):{b}case .err(p):{return .err(p);}};needed=needed || owned;let state=if(owned && (n.kind==1 || n.kind==10)){g::Cell{.state=1,.move_start=0,.move_end=0}}else{g::empty()};if(!v::push<u8>(ref mut owners,if(owned){1}else{0}).is_ok || !v::push<g::Cell>(ref mut env,state).is_ok){return .err(g::fail(ref c,11,n.start,n.end));}local+=1;}
        // Scalar fields of inert owners introduce no move obligation. Resource
        // owner fields still require traversal, even when the field is scalar.
        var usize expression=1;while(!needed && expression<=m::expression_count(body)){if(!m::charge(ref mut c.work,1)){return .err(g::fail(ref c,10,0,0));}let e=m::expression_at(body,expression);if(e.kind==8 && e.field.id!=0){needed=switch(g::owned(primary,e.field.owner,ref mut c,0)){case .ok(b):{b}case .err(p):{return .err(p);}};}let owned=switch(g::owned(primary,e.type,ref mut c,0)){case .ok(b):{b}case .err(p):{return .err(p);}};needed=needed || owned;expression+=1;}
        if(needed){var v::Vec<g::Edge> edges=v::empty<g::Edge>();let result=switch(g::owned(primary,value.result,ref mut c,0)){case .ok(b):{b}case .err(p):{return .err(p);}};
            let reachable=switch(control::block(sources,primary,body,ref owners,ref mut env,ref mut edges,ref mut diagnostics,ref mut facts,if(value.kind==2){d.cleanup_body}else{d.body},result,0,ref mut c)){case .ok(b):{b}case .err(p):{return .err(p);}};
            if(reachable){let b=syntax::statement_at(source,if(value.kind==2){d.cleanup_body}else{d.body});let dropped=g::cleanup_live(body,ref env,1,b.start,b.end,ref mut facts,ref mut c);if(dropped.code!=0){return .err(dropped);}}
            if(v::len<g::Edge>(ref edges)!=0){return .err(g::fail(ref c,12,d.start,d.end));}
            if(value.kind==2){local=1;while(local<=m::local_count(body)){let n=m::local_at(body,local);let cell=*v::at<g::Cell>(ref env,local-1);if(n.kind==10 && cell.state!=1){let p=g::report(body,local,1102,if(cell.move_end!=0){cell.move_start}else{d.start},if(cell.move_end!=0){cell.move_end}else{d.end},cell,0,0,ref mut diagnostics,ref mut c);if(p.code!=0){return .err(p);}}local+=1;}}
        }work=c.work;frame+=1;
    }var v::Vec<g::Fact> state_facts=v::empty<g::Fact>();var v::Vec<g::Fact> cleanups=v::empty<g::Fact>();var usize i=0;while(i<v::len<g::Fact>(ref facts)){let fact=*v::at<g::Fact>(ref facts,i);if(!m::charge(ref mut work,1)){return .err(m::Problem{.code=10,.module_index=0,.start=fact.start,.end=fact.end,.expected=0,.actual=0});}let appended=if(fact.kind==5){v::push<g::Fact>(ref mut cleanups,fact).is_ok}else{v::push<g::Fact>(ref mut state_facts,fact).is_ok};if(!appended){return .err(m::Problem{.code=11,.module_index=0,.start=fact.start,.end=fact.end,.expected=0,.actual=0});}i+=1;}return .ok(Plan{.diagnostics,.facts=state_facts,.cleanups,.work});
}
