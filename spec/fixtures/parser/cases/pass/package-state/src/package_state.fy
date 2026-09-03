module package_state;

@contract_uid("example.token")
state {
    cell u64 total_supply;
    map bytes20 u64 balances;
}
