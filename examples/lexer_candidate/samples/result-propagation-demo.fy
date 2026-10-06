module main;
fn Result<i64,bool> lookup(bool found) { return if(found){.ok(4000000000)}else{.err(true)}; }
fn Result<bool,bool> positive(bool found) { let i64 value=lookup(found)?;return .ok(value>0); }
fn i32 main() {
    assert(switch(positive(true)){case .ok(value):{value}case .err(_):{false}});
    assert(switch(positive(false)){case .ok(_):{false}case .err(missing):{missing}});
    return 0;
}
