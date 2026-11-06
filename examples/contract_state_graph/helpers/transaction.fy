module transaction;

@contract_uid("example.native-state-helpers")
state {
    cell bool reject;
    cell u64 value;
    map u64 u64 values;
}

@effects(storage_read, storage_write)
fn void store_value() {
    let current = storage::read("value");
    storage::write("values", current, current);
    return;
}

@effects(storage_read, storage_write)
fn void check_and_store() {
    let rejected = storage::read("reject");
    if (rejected) {
        storage::reject();
    }
    store_value();
    return;
}

@effects(storage_read, storage_write)
fn void transaction() {
    storage::write("value", 42);
    check_and_store();
    storage::read("values", 42);
    storage::commit();
}
