module main;

@contract_uid("example.contract-state-declaration")
state {
    cell u64 counter;
    map bytes20 u64 balances;
}

fn i32 main() {
    return 42;
}
