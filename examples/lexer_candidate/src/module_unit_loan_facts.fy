module module_unit_loan_facts;
import module_unit_loan_model as loan;
import module_unit_origin_flow_model as g;
import module_unit_origin_inputs as inputs;
import module_unit_origin_sets as sets;
import module_unit_values as values;
import module_unit_value_builtin as borrows;
import module_unit_value_model as m;
import std::vec as v;

// Fact-only sites never freeze a storage root or add descendant generations.
pub fn Result<g::Value,m::Problem> seed(g::Value value,bool mutable,usize parent,usize start,usize end,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut g::Context c){
    if(value.origin<=1 || value.origin==sets::empty()){return .ok(value);}var usize i=0;var usize id=0;
    while(i<v::len<loan::Loan>(records)){if(!m::charge(ref mut c.work,1)){return .err(g::fail(c,10,start,end));}let r=*v::at<loan::Loan>(records,i);if(r.fact_only && r.origin==value.origin && r.mutable==mutable && r.parent==parent && r.start==start && r.end==end){id=i+3;break;}i+=1;}
    if(id==0){if(v::len<loan::Loan>(records)>=16384 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,start,end));}
        if(!v::push<loan::Loan>(records,loan::Loan{.fact_only=true,.retained=false,.binding=0,.last_start=0,.last_end=0,.escapes=false,.subview=false,.origin=value.origin,.parent,.owner=0,.start,.end,.mutable}).is_ok){return .err(g::fail(c,11,start,end));}id=v::len<loan::Loan>(records)+2;}
    var result=value;result.loan=switch(g::join(entries,members,c,value.loan,id,start,end)){case .ok(n):{n}case .err(p):{return .err(p);}};return .ok(result);
}
pub fn m::Problem retain(g::Value value,usize binding,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut g::Context c,usize start,usize end){
    if(c.probes!=0){return g::fail(c,0,0,0);}var usize i=0;while(i<sets::count(entries,members,value.loan)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let id=sets::member(entries,members,value.loan,i);if(id<3 || id-2>v::len<loan::Loan>(records)){return g::fail(c,12,start,end);}let r=v::at_mut<loan::Loan>(records,id-3);if(!r.retained){r.retained=true;r.binding=binding;}i+=1;}return g::fail(c,0,0,0);
}
pub fn m::Problem used(g::Value value,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut g::Context c,usize start,usize end){
    if(c.probes!=0){return g::fail(c,0,0,0);}var usize i=0;while(i<sets::count(entries,members,value.loan)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let id=sets::member(entries,members,value.loan,i);if(id<3 || id-2>v::len<loan::Loan>(records)){return g::fail(c,12,start,end);}let r=v::at_mut<loan::Loan>(records,id-3);if(r.retained && end>r.last_end){r.last_start=start;r.last_end=end;}i+=1;}return g::fail(c,0,0,0);
}
pub fn Result<bool,m::Problem> mutable_type(ref inputs::Inputs input,ref values::Plan primary,usize type,ref mut g::Context c,usize depth){
    if(type==0){return .ok(false);}if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(g::fail(c,10,0,0));}let t=inputs::type_at(input,primary,c.frame,type);if(borrows::borrowed(t)){return .ok(borrows::mutable(t));}if(t.kind!=5 && t.kind!=6){return .ok(false);}let first=switch(mutable_type(input,primary,t.first,c,depth+1)){case .ok(b):{b}case .err(p):{return .err(p);}};return if(first || t.kind==5){.ok(first)}else{mutable_type(input,primary,t.second,c,depth+1)};
}
pub fn m::Problem escaping(g::Value value,ref inputs::Inputs input,ref mut v::Vec<sets::Entry> entries,ref mut v::Vec<usize> members,ref mut v::Vec<loan::Loan> records,ref mut g::Context c,usize start,usize end){
    if(c.probes!=0 || sets::count(entries,members,value.origin)!=1){return g::fail(c,0,0,0);}let origin=sets::member(entries,members,value.origin,0);
    if(origin!=2){let owner=sets::owner_id(origin);if(owner==0 || owner>inputs::owner_count(input)){return g::fail(c,12,start,end);}let r=inputs::owner(input,owner);if(r.kind!=11 && !((r.kind==1 || r.kind==13) && r.borrowed)){return g::fail(c,0,0,0);}}
    var actual=value;if(sets::count(entries,members,value.loan)==0){actual=switch(seed(value,c.return_mutable,0,start,end,entries,members,records,c)){case .ok(v):{v}case .err(p):{return p;}};}
    var usize i=0;while(i<sets::count(entries,members,actual.loan)){if(!m::charge(ref mut c.work,1)){return g::fail(c,10,start,end);}let id=sets::member(entries,members,actual.loan,i);if(id<3 || id-2>v::len<loan::Loan>(records)){return g::fail(c,12,start,end);}let r=v::at_mut<loan::Loan>(records,id-3);r.retained=true;r.last_start=start;r.last_end=end;r.escapes=true;i+=1;}return g::fail(c,0,0,0);
}
pub fn m::Problem publish(ref v::Vec<loan::Loan> records,ref v::Vec<sets::Entry> entries,ref v::Vec<usize> members,ref mut v::Vec<loan::Fact> facts,usize frame,m::Frame source,ref mut usize work){
    let base=v::len<loan::Fact>(facts);var usize i=0;while(i<v::len<loan::Loan>(records)){if(!m::charge(work,1)){return m::fail(source,10,0,0,0,0);}let r=*v::at<loan::Loan>(records,i);
        if(r.retained){let origin=if(r.origin==0 && r.owner!=0){r.owner+2}else{r.origin};var usize j=0;while(j<sets::count(entries,members,origin)){let leaf=sets::member(entries,members,origin,j);var usize at=base;var bool found=false;
            while(at<v::len<loan::Fact>(facts)){if(!m::charge(work,1)){return m::fail(source,10,r.start,r.end,0,0);}let old=*v::at<loan::Fact>(facts,at);if(old.binding==r.binding && old.origin==leaf && old.mutable==r.mutable && old.start==r.start && old.end==r.end){found=true;let slot=v::at_mut<loan::Fact>(facts,at);if(r.last_end>slot.last_end){slot.last_start=r.last_start;slot.last_end=r.last_end;}slot.escapes=slot.escapes || r.escapes;break;}at+=1;}
            if(!found){if(v::len<loan::Fact>(facts)>=16384 || !m::charge(work,1)){return m::fail(source,10,r.start,r.end,0,0);}if(!v::push<loan::Fact>(facts,loan::Fact{.frame,.binding=r.binding,.origin=leaf,.mutable=r.mutable,.start=r.start,.end=r.end,.last_start=r.last_start,.last_end=r.last_end,.escapes=r.escapes}).is_ok){return m::fail(source,11,r.start,r.end,0,0);}}j+=1;}}
        i+=1;}return m::fail(source,0,0,0,0,0);
}
