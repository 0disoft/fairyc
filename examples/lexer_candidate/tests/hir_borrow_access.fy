module tests::hir_borrow_access;
import tests::hir_nominal as fixture;
import hir_borrow_access as access;
import borrow_origins as owners;
import hir_model as h;
import std::vec as v;

@test fn void owner_lists_preserve_exhaustive_order_and_mixed_membership(){
    let p=fixture::retained(b"module main;fn i32 main(){var u8[1] a=[1];var u8[1] b=[2];let ref u8[] p=if(true){ref a}else{ref b};return p[0] as i32;}");
    var v::Vec<owners::Entry> sets=v::empty<owners::Entry>();var v::Vec<usize> members=v::empty<usize>();var usize work=0;
    let usize mixed=switch(owners::join(ref mut sets,ref mut members,2,3,ref mut work)){case .ok(id):{id}case .err(_):{assert(false);return;}};
    var v::Vec<usize> origins=v::empty<usize>();var usize id=0;
    while(id<=h::expression_count(ref p)){
        let usize origin=if(id==0){0}else if(id%3==0){mixed}else if(id%3==1){2}else{0};
        assert(v::push<usize>(ref mut origins,origin).is_ok);id+=1;
    }
    let index=switch(access::create(ref p,ref origins,ref sets,ref members,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    var usize selected=0;
    while(selected<5){
        let usize owner=if(selected==0){0}else if(selected==1){2}else if(selected==2){3}else if(selected==3){mixed}else{65538};
        var v::Vec<usize> cursor=v::empty<usize>();assert(access::begin(ref index,ref sets,ref members,owner,ref mut cursor,ref mut work)==0);id=1;
        while(id<=h::expression_count(ref p)){
            let e=h::expression_at(ref p,id);
            let bool expected=h::expression_code(e.kind)==3 && owners::contains(ref sets,ref members,owner,e.binding,ref mut work) || owners::overlap(ref sets,ref members,owner,*v::at<usize>(ref origins,id),ref mut work);
            if(expected){assert(access::next(ref index,ref mut cursor,ref mut work)==id);}id+=1;
        }
        assert(access::next(ref index,ref mut cursor,ref mut work)==0);selected+=1;
    }
    work=1048576;assert(switch(access::create(ref p,ref origins,ref sets,ref members,ref mut work)){case .ok(_):{false}case .err(code):{code==2}});
    var v::Vec<usize> cursor=v::empty<usize>();work=0;assert(access::begin(ref index,ref sets,ref members,mixed,ref mut cursor,ref mut work)==0);work=1048576;assert(access::next(ref index,ref mut cursor,ref mut work)==0 && work>1048576);
}
