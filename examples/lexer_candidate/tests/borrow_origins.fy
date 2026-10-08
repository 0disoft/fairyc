module tests::borrow_origins;
import borrow_origins as origins;
import std::vec as v;
fn usize joined(ref mut v::Vec<origins::Entry> sets,ref mut v::Vec<usize> members,usize a,usize b,ref mut usize work){return switch(origins::join(sets,members,a,b,work)){case .ok(id):{id}case .err(_):{assert(false);0}};}
@test fn void unions_converge_without_dropping_any_owner(){
    var v::Vec<origins::Entry> sets=v::empty<origins::Entry>();var v::Vec<usize> members=v::empty<usize>();var usize work=0;
    let usize ab=joined(ref mut sets,ref mut members,7,3,ref mut work);assert(origins::mixed(ab));
    assert(joined(ref mut sets,ref mut members,3,7,ref mut work)==ab && joined(ref mut sets,ref mut members,ab,7,ref mut work)==ab);
    let usize left=joined(ref mut sets,ref mut members,ab,131072,ref mut work);
    let usize bc=joined(ref mut sets,ref mut members,7,131072,ref mut work);
    let usize right=joined(ref mut sets,ref mut members,3,bc,ref mut work);
    assert(left==right && origins::count(ref sets,left)==3);
    assert(origins::contains(ref sets,ref members,left,3,ref mut work) && origins::contains(ref sets,ref members,left,7,ref mut work) && origins::contains(ref sets,ref members,left,131072,ref mut work));
    assert(!origins::contains(ref sets,ref members,left,4,ref mut work));
    assert(origins::overlap(ref sets,ref members,left,ab,ref mut work) && !origins::overlap(ref sets,ref members,left,9,ref mut work));
    assert(joined(ref mut sets,ref mut members,0,left,ref mut work)==left && joined(ref mut sets,ref mut members,65538,left,ref mut work)==left);
    assert(joined(ref mut sets,ref mut members,65538,65538,ref mut work)==65538);
    assert(origins::count(ref sets,0)==0 && !origins::overlap(ref sets,ref members,0,left,ref mut work));
    work=1048576;assert(switch(origins::join(ref mut sets,ref mut members,1,2,ref mut work)){case .ok(_):{false}case .err(code):{code==2}});
}
