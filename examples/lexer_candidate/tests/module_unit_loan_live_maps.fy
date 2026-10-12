module tests::module_unit_loan_live_maps;
import module_unit_loan_live_model as l;
import module_unit_loan_live_maps as maps;
import module_unit_value_model as m;
import std::vec as v;

@test fn void repeated_maps_reuse_exact_rows_without_scanning_all_prior_maps(){
    var l::Context c=l::Context{.frame=m::Frame{.kind=3,.context=0,.module_index=0,.declaration=1,.result=0},.work=0,.rounds=0,.buckets=[0;1024]};
    var v::Vec<l::Use> env=v::empty<l::Use>();assert(v::push<l::Use>(ref mut env,l::Use{.start=0,.end=0}).is_ok);
    var v::Vec<l::Header> headers=v::empty<l::Header>();var v::Vec<l::Row> rows=v::empty<l::Row>();
    var usize i=1;while(i<=256){let slot=v::at_mut<l::Use>(ref mut env,0);*slot=l::Use{.start=i,.end=i+1};let id=switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(id==i);i+=1;}
    i=1;while(i<=256){let slot=v::at_mut<l::Use>(ref mut env,0);*slot=l::Use{.start=i,.end=i+1};let id=switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(id==i);i+=1;}
    assert(v::len<l::Header>(ref headers)==256 && v::len<l::Row>(ref rows)==256 && c.work<4096);
    // The modular hash deliberately collides; source spans still differ.
    let slot=v::at_mut<l::Use>(ref mut env,0);*slot=l::Use{.start=65522,.end=65523};
    let collision=switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(collision==257 && v::at<l::Header>(ref headers,0).hash==v::at<l::Header>(ref headers,256).hash);
    let restored=switch(maps::restore(collision,1,ref headers,ref rows,ref mut c)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::at<l::Use>(ref restored,0).start==65522 && v::at<l::Use>(ref restored,0).end==65523);
    let first=v::at_mut<l::Use>(ref mut env,0);*first=l::Use{.start=1,.end=2};switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{assert(n==1);}case .err(_):{assert(false);}};
    c.work=1048576;switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10);}};
}
