module module_unit_loan_owner_reads;
import module_unit_origin_inputs as inputs;
import module_unit_origin_flow_model as g;
import module_unit_origin_sets as sets;
import module_unit_value_model as m;
import module_unit_loan_model as loan;
import module_unit_loan_live_model as live;
import module_unit_loan_checks as loans;
import module_unit_loan_reference as references;
import std::vec as v;

fn m::Problem selected(usize owner,g::Value retained,usize start,usize end,usize use_start,usize use_end,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut v::Vec<loan::Held> held,ref mut v::Vec<loan::Conflict> conflicts,ref mut g::Context c){
    var usize i=0;while(i<sets::count(entries,members,retained.loan)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let id=sets::member(entries,members,retained.loan,i);if(id<3 || id-2>v::len<loan::Loan>(records)){return g::fail(c,12,start,end);}let r=*v::at<loan::Loan>(records,id-3);if(r.owner==owner && r.mutable){let p=loans::reference_conflict(r.start,r.end,start,end,use_start,use_end,records,held,conflicts,c);if(p.code!=0){return p;}}i+=1;}return g::fail(c,0,0,0);
}
pub fn m::Problem check(ref inputs::Inputs input,ref v::Vec<g::Value> env,usize original,usize owner,usize start,usize end,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut v::Vec<loan::Held> held,ref mut v::Vec<loan::Conflict> conflicts,ref mut g::Context c){
    if(owner==0 || c.probes!=0 || c.place_root==original || v::len<loan::Loan>(records)==0){return g::fail(c,0,0,0);}
    // A consuming resource expression keeps its specific E2105 diagnosis.
    var usize i=0;while(i<v::len<loan::Conflict>(conflicts)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let r=*v::at<loan::Conflict>(conflicts,i);if(r.frame==c.frame && r.code==2105 && r.start<=start && r.end>=end){return g::fail(c,0,0,0);}i+=1;}
    let p=inputs::liveness(input,c.frame);var usize low=0;var usize high=live::expression_count(p);while(low<high){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let mid=low+(high-low)/2;if(live::expression_at(p,mid).source_index<original){low=mid+1;}else{high=mid;}}
    if(low<live::expression_count(p)){let at=live::expression_at(p,low);if(at.checked && at.source_index==original){i=0;while(i<live::use_count(p,at)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let use=live::use_at(p,at,i);if(use.local==0 || use.local>v::len<g::Value>(env)){return g::fail(c,12,start,end);}let checked=selected(owner,*v::at<g::Value>(env,use.local-1),start,end,use.start,use.end,entries,members,records,held,conflicts,c);if(checked.code!=0){return checked;}i+=1;}}}
    i=0;while(i<v::len<loan::Held>(held)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let prior=*v::at<loan::Held>(held,i);let checked=selected(owner,g::Value{.subview=false,.loan=prior.loan,.origin=prior.origin,.allocation=0},start,end,if(prior.store){prior.start}else{start},if(prior.store){prior.end}else{end},entries,members,records,held,conflicts,c);if(checked.code!=0){return checked;}i+=1;}return g::fail(c,0,0,0);
}
// Only direct fields of the first constructor argument are metadata. Blocks
// and callbacks keep their own source nodes and remain ordinary reads.
pub fn Result<usize,m::Problem> metadata_root(ref m::Values body,usize owner_id,usize argument_id,ref mut g::Context c){
    var owner=switch(g::expression(body,owner_id,c)){case .ok(e):{e}case .err(p):{return .err(p);}};var usize depth=0;while(owner.kind==3){if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,owner.start,owner.end));}owner=switch(g::expression(body,owner.left,c)){case .ok(e):{e}case .err(p):{return .err(p);}};depth+=1;}if(owner.kind==21 || owner.kind==22){owner=switch(g::expression(body,owner.left,c)){case .ok(e):{e}case .err(p):{return .err(p);}};}
    let root=switch(references::root(body,owner.source_index,c)){case .ok(e):{e}case .err(p):{return .err(p);}};var argument=switch(g::expression(body,argument_id,c)){case .ok(e):{e}case .err(p):{return .err(p);}};depth=0;while(argument.kind==3){if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,argument.start,argument.end));}argument=switch(g::expression(body,argument.left,c)){case .ok(e):{e}case .err(p):{return .err(p);}};depth+=1;}if(argument.kind!=8 || root.kind!=2 || root.binding==0){return .ok(0);}let selected=switch(references::root(body,argument.source_index,c)){case .ok(e):{e}case .err(p):{return .err(p);}};return .ok(if(selected.kind==2 && selected.binding==root.binding){selected.source_index}else{0});
}
