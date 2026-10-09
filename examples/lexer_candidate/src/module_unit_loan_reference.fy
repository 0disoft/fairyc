module module_unit_loan_reference;
import module_unit_origin_inputs as inputs;
import module_unit_origin_flow_model as g;
import module_unit_origin_sets as sets;
import module_unit_loan_model as loan;
import module_unit_loan_live_model as live;
import module_unit_loan_checks as loans;
import module_unit_value_model as m;
import module_unit_values as values;
import module_unit_value_builtin as borrows;
import std::vec as v;

// Tagged values may retain exactly one borrowed leaf, with its mutability.
fn Result<u32,m::Problem> leaf(ref inputs::Inputs input,ref values::Plan primary,usize type,ref mut g::Context c,usize depth){
    if(type==0){return .ok(0);}if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,0,0));}
    let t=inputs::type_at(input,primary,c.frame,type);if(borrows::borrowed(t)){return .ok(if(borrows::mutable(t)){2}else{1});}if(t.kind!=5 && t.kind!=6){return .ok(0);}
    let first=switch(leaf(input,primary,t.first,c,depth+1)){case .ok(k):{k}case .err(p):{return .err(p);}};if(first!=0 || t.kind==5){return .ok(first);}return leaf(input,primary,t.second,c,depth+1);
}
pub fn Result<m::Expression,m::Problem> root(ref m::Values body,usize original,ref mut g::Context c){
    var usize id=original;var usize depth=0;while(id!=0){if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,0,0));}let e=switch(g::expression(body,id,c)){case .ok(e):{e}case .err(p):{return .err(p);}};if(e.kind!=3 && e.kind!=8 && e.kind!=9 && e.kind!=26 && !(e.kind==4 && e.place.kind==3)){return .ok(e);}id=e.left;depth+=1;}return .err(g::fail(c,12,0,0));
}
pub fn m::Problem access(ref inputs::Inputs input,ref values::Plan primary,ref m::Values body,ref v::Vec<g::Value> env,usize original,g::Value value,bool mutable,usize start,usize end,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> loan_records,ref mut v::Vec<loan::Held> held_loans,ref mut v::Vec<loan::Conflict> loan_conflicts,ref mut g::Context c){
    let source=switch(g::expression(body,original,c)){case .ok(e):{e}case .err(p):{return p;}};let held=loans::access(value,mutable,start,end,source.start,source.end,entries,members,loan_records,held_loans,loan_conflicts,c);if(held.code!=0){return held;}if(!mutable || c.probes!=0 || value.origin<=1){return g::fail(c,0,0,0);}
    let p=inputs::liveness(input,c.frame);var usize low=0;var usize high=live::expression_count(p);while(low<high){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let mid=low+(high-low)/2;if(live::expression_at(p,mid).source_index<original){low=mid+1;}else{high=mid;}}
    if(low>=live::expression_count(p)){return g::fail(c,0,0,0);}let at=live::expression_at(p,low);if(at.source_index!=original || !at.checked){return g::fail(c,0,0,0);}var usize i=0;
    while(i<live::use_count(p,at)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let use=live::use_at(p,at,i);if(use.local==0 || use.local>m::local_count(body) || use.local>v::len<g::Value>(env)){return g::fail(c,12,start,end);}
        let local=m::local_at(body,use.local);let kind=switch(leaf(input,primary,local.type,c,0)){case .ok(k):{k}case .err(p):{return p;}};if(kind==1){let prior=*v::at<g::Value>(env,use.local-1);let same=switch(loans::overlap(value.origin,prior.origin,entries,members,c,start,end)){case .ok(b):{b}case .err(p):{return p;}};if(same){let owner=inputs::owner(input,inputs::frame(input,c.frame).local_start+use.local);let checked=loans::reference_conflict(owner.start,owner.end,start,end,use.start,use.end,loan_records,held_loans,loan_conflicts,c);if(checked.code!=0){return checked;}}}i+=1;
    }return g::fail(c,0,0,0);
}
pub fn m::Problem store(ref inputs::Inputs input,ref values::Plan primary,ref m::Values body,ref v::Vec<g::Value> env,usize original,usize start,usize end,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> loan_records,ref mut v::Vec<loan::Held> held_loans,ref mut v::Vec<loan::Conflict> loan_conflicts,ref mut g::Context c){
    // Replacing a descriptor does not write through the old reference.
    var target=switch(g::expression(body,original,c)){case .ok(e):{e}case .err(p):{return p;}};var usize depth=0;while(target.kind==3){if(depth>=64 || !m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}target=switch(g::expression(body,target.left,c)){case .ok(e):{e}case .err(p):{return p;}};depth+=1;}if(target.kind==2){return g::fail(c,0,0,0);}
    let e=switch(root(body,original,c)){case .ok(e):{e}case .err(p):{return p;}};if(e.kind!=2 || e.binding==0 || e.type==0 || !borrows::borrowed(inputs::type_at(input,primary,c.frame,e.type))){return g::fail(c,0,0,0);}if(e.binding>v::len<g::Value>(env)){return g::fail(c,12,start,end);}return access(input,primary,body,env,e.source_index,*v::at<g::Value>(env,e.binding-1),true,start,end,entries,members,loan_records,held_loans,loan_conflicts,c);
}
