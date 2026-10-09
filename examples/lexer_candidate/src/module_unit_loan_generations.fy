module module_unit_loan_generations;
import module_unit_loan_model as loan;
import module_unit_loan_checks as loans;
import module_unit_loan_live_model as live;
import module_unit_origin_inputs as inputs;
import module_unit_origin_flow_model as g;
import module_unit_origin_sets as sets;
import module_unit_value_model as m;
import std::vec as v;

pub fn Result<g::Value,m::Problem> child(g::Value value,usize parent,usize start,usize end,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut g::Context c){
    if(value.origin<=1 || value.origin==sets::empty()){return .ok(value);}var usize i=0;var usize found=0;while(i<v::len<loan::Loan>(records)){if(!m::charge(ref mut c.work,1)){return .err(g::fail(c,10,start,end));}let old=*v::at<loan::Loan>(records,i);if(old.subview && old.parent==parent && old.origin==value.origin && old.start==start && old.end==end){found=i+3;break;}i+=1;}
    if(found==0){if(v::len<loan::Loan>(records)>=16384 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,start,end));}if(!v::push<loan::Loan>(records,loan::Loan{.subview=true,.origin=value.origin,.parent,.owner=0,.mutable=true,.start,.end}).is_ok){return .err(g::fail(c,11,start,end));}found=v::len<loan::Loan>(records)+2;}
    let combined=switch(g::join(entries,members,c,value.loan,found,start,end)){case .ok(n):{n}case .err(p):{return .err(p);}};var result=value;result.loan=combined;return .ok(result);
}
fn m::Problem selected(g::Value access,g::Value retained,usize start,usize end,usize use_start,usize use_end,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut v::Vec<loan::Held> held,ref mut v::Vec<loan::Conflict> conflicts,ref mut g::Context c){
    let same=switch(loans::overlap(access.origin,retained.origin,entries,members,c,start,end)){case .ok(b):{b}case .err(p):{return p;}};if(!same){return g::fail(c,0,0,0);}var usize i=0;while(i<sets::count(entries,members,retained.loan)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let id=sets::member(entries,members,retained.loan,i);if(id<3 || id-2>v::len<loan::Loan>(records)){return g::fail(c,12,start,end);}let prior=*v::at<loan::Loan>(records,id-3);let inherited=switch(sets::contains(entries,members,access.loan,id,ref mut c.work)){case .ok(b):{b}case .err(code):{return g::fail(c,code,start,end);}};if(prior.subview && !inherited){let p=loans::reference_conflict(prior.start,prior.end,start,end,use_start,use_end,records,held,conflicts,c);if(p.code!=0){return p;}}i+=1;}return g::fail(c,0,0,0);
}
pub fn m::Problem path(ref inputs::Inputs input,ref v::Vec<g::Value> env,usize original,g::Value value,usize start,usize end,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut v::Vec<loan::Held> held,ref mut v::Vec<loan::Conflict> conflicts,ref mut g::Context c){
    if(c.probes!=0 || value.origin<=1){return g::fail(c,0,0,0);}let p=inputs::liveness(input,c.frame);var usize low=0;var usize high=live::expression_count(p);while(low<high){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let mid=low+(high-low)/2;if(live::expression_at(p,mid).source_index<original){low=mid+1;}else{high=mid;}}
    if(low<live::expression_count(p)){let at=live::expression_at(p,low);if(at.checked && at.source_index==original){var usize i=0;while(i<live::use_count(p,at)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let use=live::use_at(p,at,i);if(use.local==0 || use.local>v::len<g::Value>(env)){return g::fail(c,12,start,end);}let checked=selected(value,*v::at<g::Value>(env,use.local-1),start,end,use.start,use.end,entries,members,records,held,conflicts,c);if(checked.code!=0){return checked;}i+=1;}}}
    var usize i=0;while(i<v::len<loan::Held>(held)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let prior=*v::at<loan::Held>(held,i);let checked=selected(value,g::Value{.subview=false,.loan=prior.loan,.origin=prior.origin,.allocation=0},start,end,start,end,entries,members,records,held,conflicts,c);if(checked.code!=0){return checked;}i+=1;}return g::fail(c,0,0,0);
}
pub fn m::Problem owner_read(usize owner,usize start,usize end,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut v::Vec<loan::Held> held,ref mut v::Vec<loan::Conflict> conflicts,ref mut g::Context c){
    if(owner==0 || c.probes!=0){return g::fail(c,0,0,0);}var usize i=0;while(i<v::len<loan::Held>(held)){let prior=*v::at<loan::Held>(held,i);if(prior.store){var usize j=0;while(j<sets::count(entries,members,prior.loan)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let id=sets::member(entries,members,prior.loan,j);if(id<3 || id-2>v::len<loan::Loan>(records)){return g::fail(c,12,start,end);}let r=*v::at<loan::Loan>(records,id-3);if(r.owner==owner && r.mutable){let checked=loans::reference_conflict(r.start,r.end,start,end,prior.start,prior.end,records,held,conflicts,c);if(checked.code!=0){return checked;}}j+=1;}}i+=1;}return g::fail(c,0,0,0);
}
