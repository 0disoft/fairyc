module tests::module_unit_loan_live_maps;
import module_unit_loan_live_model as l;
import module_unit_loan_live_maps as maps;
import module_unit_value_model as m;
import std::vec as v;

@test fn void repeated_maps_reuse_exact_rows_without_scanning_all_prior_maps(){
    var l::Context c=l::Context{.frame=m::Frame{.kind=3,.context=0,.module_index=0,.declaration=1,.result=0},.work=0,.rounds=0,.buckets=[0;1024]};
    var v::Vec<l::Use> env=v::empty<l::Use>();assert(v::push<l::Use>(ref mut env,l::Use{.local=1,.start=0,.end=0}).is_ok);
    var v::Vec<l::Header> headers=v::empty<l::Header>();var v::Vec<l::Row> rows=v::empty<l::Row>();
    var usize i=1;while(i<=256){let slot=v::at_mut<l::Use>(ref mut env,0);*slot=l::Use{.local=1,.start=i,.end=i+1};let id=switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(id==i);i+=1;}
    i=1;while(i<=256){let slot=v::at_mut<l::Use>(ref mut env,0);*slot=l::Use{.local=1,.start=i,.end=i+1};let id=switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(id==i);i+=1;}
    assert(v::len<l::Header>(ref headers)==256 && v::len<l::Row>(ref rows)==256 && c.work<4096);
    // The modular hash deliberately collides; source spans still differ.
    let slot=v::at_mut<l::Use>(ref mut env,0);*slot=l::Use{.local=1,.start=65522,.end=65523};
    let collision=switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(collision==257 && v::at<l::Header>(ref headers,0).hash==v::at<l::Header>(ref headers,256).hash);
    var v::Vec<l::Local> locals=v::empty<l::Local>();assert(v::push<l::Local>(ref mut locals,l::Local{.slot=1,.start=0,.end=0,.borrowed=true}).is_ok);let restored=switch(maps::restore(collision,ref locals,ref headers,ref rows,ref mut c)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::at<l::Use>(ref restored,0).start==65522 && v::at<l::Use>(ref restored,0).end==65523);
    let first=v::at_mut<l::Use>(ref mut env,0);*first=l::Use{.local=1,.start=1,.end=2};switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{assert(n==1);}case .err(_):{assert(false);}};
    c.work=1048576;switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10);}};
}

@test fn void compact_live_slots_keep_original_bindings_across_restore_and_kill(){
    var l::Context c=l::Context{.frame=m::Frame{.kind=3,.context=0,.module_index=0,.declaration=1,.result=0},.work=0,.rounds=0,.buckets=[0;1024]};
    var v::Vec<l::Local> locals=v::empty<l::Local>();var usize i=0;
    while(i<256){assert(v::push<l::Local>(ref mut locals,l::Local{.slot=0,.start=i,.end=i+1,.borrowed=false}).is_ok);i+=1;}
    assert(v::push<l::Local>(ref mut locals,l::Local{.slot=1,.start=256,.end=257,.borrowed=true}).is_ok);
    assert(v::push<l::Local>(ref mut locals,l::Local{.slot=0,.start=257,.end=258,.borrowed=false}).is_ok);
    assert(v::push<l::Local>(ref mut locals,l::Local{.slot=2,.start=258,.end=259,.borrowed=true}).is_ok);
    var env=switch(l::empty(ref locals,ref mut c)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(v::len<l::Use>(ref env)==2 && v::at<l::Use>(ref env,0).local==257 && v::at<l::Use>(ref env,1).local==259);
    let first=v::at_mut<l::Use>(ref mut env,0);*first=l::Use{.local=257,.start=300,.end=301};let last=v::at_mut<l::Use>(ref mut env,1);*last=l::Use{.local=259,.start=400,.end=401};
    var v::Vec<l::Header> headers=v::empty<l::Header>();var v::Vec<l::Row> rows=v::empty<l::Row>();
    let map=switch(maps::intern(ref env,ref mut headers,ref mut rows,ref mut c)){case .ok(n):{n}case .err(_):{assert(false);return;}};
    assert(v::at<l::Row>(ref rows,0).local==257 && v::at<l::Row>(ref rows,1).local==259);
    var restored=switch(maps::restore(map,ref locals,ref headers,ref rows,ref mut c)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(l::kill(ref mut restored,258,ref locals,ref mut c).code==0 && v::at<l::Use>(ref restored,1).end==401);
    assert(l::kill(ref mut restored,259,ref locals,ref mut c).code==0 && v::at<l::Use>(ref restored,1).local==259 && v::at<l::Use>(ref restored,1).end==0 && v::at<l::Use>(ref restored,0).end==301);
    // A nearly exhausted allowance can copy two borrowed slots despite the many
    // scalar locals; the unchanged traversal ceiling still applies.
    c.work=1048570;let copied=switch(l::copy(ref env,ref mut c)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::len<l::Use>(ref copied)==2 && c.work==1048572);
}
