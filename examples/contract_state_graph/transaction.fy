module transaction;

@contract_uid("example.native-state-graph")
state {
    cell u64 key;
    cell bool enabled;
    cell u64 value;
    map u64 u64 values;
}

@effects(storage_read, storage_write)
fn void transaction() {
    let key = storage::read("key");
    let enabled = storage::read("enabled");
    let previous = storage::read("values", key);
    if (enabled) {
        if (storage::found(previous)) {
            storage::write("value", previous);
        } else {
            storage::write("value", 42);
        }
    } else {
        storage::delete("value");
    }
    let selected = storage::read("value");
    storage::write("values", key, selected);
    storage::read("values", key);
    storage::commit();
}
