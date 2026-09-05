module transaction;
@contract_uid("example.native-state")
state { cell u64 value; }
@effects(storage_read, storage_write)
fn void transaction() {
    storage::write("value", 42);
    storage::read("value");
    storage::trap("overflow");
}
