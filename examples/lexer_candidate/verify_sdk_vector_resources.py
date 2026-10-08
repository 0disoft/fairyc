"""Observe resource Vec ownership using the original Fairy SDK and C runtime."""
from pathlib import Path
import verify_sdk_vector as observer

observer.SUITE = "vector-resources"
observer.CASE_OBSERVER = Path(__file__)
observer.SUFFIX += r"""
static int32_t fixture_drop_log[64];
static size_t fixture_drop_size;
void fixture_drop(int32_t n){if(fixture_drop_size>=64)abort();fixture_drop_log[fixture_drop_size++]=n;}
size_t fixture_drops(void){return fixture_drop_size;}
int32_t fixture_drop_at(size_t i){if(i>=fixture_drop_size)abort();return fixture_drop_log[i];}
"""
PRELUDE = observer.FOREIGN + """
unsafe extern C fn void fixture_drop(i32 value);
unsafe extern C fn usize fixture_drops();
unsafe extern C fn i32 fixture_drop_at(usize index);
resource Item{i32 n;cleanup{unsafe{fixture_drop(this.n);}}}
fn Item item(i32 n){return Item{.n=n};}
"""
observer.CASES = {
    "resource-element-borrows": PRELUDE + """
fn i32 main(){
    {
        var v::Vec<Item> x=v::empty<Item>();assert(v::push<Item>(ref mut x,item(11)).is_ok && v::push<Item>(ref mut x,item(23)).is_ok);
        let ref Item a=v::at<Item>(ref x,0);assert(a.n==11);
        let ref mut Item b=v::at_mut<Item>(ref mut x,0);b.n=17;assert(b.n==17);
        assert(v::clear<Item>(ref mut x)==2 && unsafe{fixture_drops()}==2);
        assert(unsafe{fixture_drop_at(0)}==23 && unsafe{fixture_drop_at(1)}==17);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "short-circuit-owned-temporaries": PRELUDE + """
fn i32 main(){
    {
        var v::Vec<Item> x=v::empty<Item>();
        assert(!(false && v::push<Item>(ref mut x,item(11)).is_ok));
        assert(true || v::push<Item>(ref mut x,item(23)).is_ok);
        assert(v::is_empty<Item>(ref x) && unsafe{fixture_drops()}==0);
        unsafe{fixture_fail_next();}assert(!(true && v::push<Item>(ref mut x,item(41)).is_ok));
        assert(v::is_empty<Item>(ref x) && unsafe{fixture_drops()}==1 && unsafe{fixture_drop_at(0)}==41);
        assert(false || v::push<Item>(ref mut x,item(37)).is_ok);
        assert(v::len<Item>(ref x)==1 && unsafe{fixture_drops()}==1);
    }
    assert(unsafe{fixture_live()}==0 && unsafe{fixture_drops()}==2 && unsafe{fixture_drop_at(1)}==37);return 7;
}
""",
    "resource-lifecycle": PRELUDE + """
fn i32 main(){
    {
        var v::Vec<Item> x=v::empty<Item>();assert(v::is_empty<Item>(ref x));
        assert(v::push<Item>(ref mut x,item(11)).is_ok && v::push<Item>(ref mut x,item(23)).is_ok && v::push<Item>(ref mut x,item(37)).is_ok);
        assert(unsafe{fixture_drops()}==0);
        assert(switch(v::pop<Item>(ref mut x)){case .some(n):{n.n==37}case .none:{false}});
        assert(unsafe{fixture_drops()}==1 && unsafe{fixture_drop_at(0)}==37);
        let usize capacity=v::capacity<Item>(ref x);assert(v::reserve<Item>(ref mut x,capacity+1).is_ok);
        assert(unsafe{fixture_drops()}==1 && v::len<Item>(ref x)==2);
        assert(v::truncate<Item>(ref mut x,1)==1);assert(unsafe{fixture_drops()}==2 && unsafe{fixture_drop_at(1)}==23);
        assert(v::clear<Item>(ref mut x)==1 && v::clear<Item>(ref mut x)==0);
        assert(unsafe{fixture_drops()}==3 && unsafe{fixture_drop_at(2)}==11);
        assert(v::push<Item>(ref mut x,item(41)).is_ok);
    }
    assert(unsafe{fixture_live()}==0 && unsafe{fixture_drops()}==4 && unsafe{fixture_drop_at(3)}==41);return 7;
}
""",
    "resource-oom-donor": PRELUDE + """
fn i32 main(){
    {
        var v::Vec<Item> x=v::empty<Item>();unsafe{fixture_fail_next();}
        let Result<usize,Item> failed=v::push<Item>(ref mut x,item(17));
        assert(v::len<Item>(ref x)==0 && v::capacity<Item>(ref x)==0 && unsafe{fixture_drops()}==0);
        let Item donor=switch(failed){case .err(n):{n}case .ok(_):{assert(false);item(0)}};
        assert(donor.n==17 && unsafe{fixture_drops()}==0);assert(v::push<Item>(ref mut x,donor).is_ok);
        let usize capacity=v::capacity<Item>(ref x);var usize i=1;while(i<capacity){assert(v::push<Item>(ref mut x,item(23)).is_ok);i+=1;}
        unsafe{fixture_fail_next();}let Result<usize,Item> growth=v::push<Item>(ref mut x,item(41));
        assert(v::len<Item>(ref x)==capacity && v::capacity<Item>(ref x)==capacity && unsafe{fixture_drops()}==0);
        assert(switch(growth){case .err(n):{n.n==41}case .ok(_):{false}});
        assert(unsafe{fixture_drops()}==1 && unsafe{fixture_drop_at(0)}==41);
        assert(v::clear<Item>(ref mut x)==capacity);assert(unsafe{fixture_drops()}==capacity+1);
        assert(unsafe{fixture_drop_at(capacity)}==17);
    }
    assert(unsafe{fixture_live()}==0);return 7;
}
""",
    "nested-resource-cleanup": PRELUDE + """
resource Diagnostic{Item token;Item message;cleanup{}}
fn Diagnostic diagnostic(i32 token,i32 message){return Diagnostic{.token=item(token),.message=item(message)};}
fn i32 main(){
    {
        var v::Vec<Diagnostic> x=v::empty<Diagnostic>();
        assert(v::push<Diagnostic>(ref mut x,diagnostic(11,12)).is_ok && v::push<Diagnostic>(ref mut x,diagnostic(21,22)).is_ok);
        let usize capacity=v::capacity<Diagnostic>(ref x);assert(v::reserve<Diagnostic>(ref mut x,capacity+1).is_ok);
        assert(unsafe{fixture_drops()}==0);
    }
    assert(unsafe{fixture_live()}==0 && unsafe{fixture_drops()}==4);
    assert(unsafe{fixture_drop_at(0)}==22 && unsafe{fixture_drop_at(1)}==21 && unsafe{fixture_drop_at(2)}==12 && unsafe{fixture_drop_at(3)}==11);return 7;
}
""",
}
observer.TRAPS = {}
observer.REJECT = {
    "retained-resource-borrow": (PRELUDE + "fn i32 main(){var v::Vec<Item> x=v::empty<Item>();assert(v::push<Item>(ref mut x,item(11)).is_ok);let ref Item a=v::at<Item>(ref x,0);discard v::clear<Item>(ref mut x);discard a.n;return 7;}", "code 2101"),
    "moved-donor": (PRELUDE + "fn i32 main(){var v::Vec<Item> x=v::empty<Item>();let Item donor=item(11);discard v::push<Item>(ref mut x,donor);discard donor.n;return 7;}", "phase 5 code 45"),
    "wrong-resource": (PRELUDE + "resource Other{i32 n;cleanup{}}fn i32 main(){var v::Vec<Item> x=v::empty<Item>();discard v::push<Item>(ref mut x,Other{.n=7});return 7;}", "code 1"),
    "stolen-put": (PRELUDE + "fn i32 main(){discard unsafe{storage_put<Item>(null,item(7))};return 7;}", "code 5102"),
}

if __name__ == "__main__":
    observer.main()
